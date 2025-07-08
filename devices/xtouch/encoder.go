package xtouch

import (
	"fmt"
	"math"

	dev "github.com/jdginn/arpad/devices"
)

// EncoderDirection represents the rotation direction of an encoder
type EncoderDirection uint8

const (
	// EncoderClockwise indicates clockwise rotation (value 1)
	EncoderClockwise EncoderDirection = 1

	// EncoderCounterClockwise indicates counterclockwise rotation (value 65)
	EncoderCounterClockwise EncoderDirection = 65
)

type Encoder struct {
	d *dev.MidiDevice

	channel     uint8
	encoderCC   uint8 // CC 16-23 for encoder rotation
	buttonCC    uint8 // CC 32-29 for button press
	ledRingLow  uint8 // For CC 48-55
	ledRingHigh uint8 // For CC 56-63

	Ring ring
}

// TODO: binding needs to be a little more carefully thought through...
func (e *Encoder) Bind(callback func(uint8) error) {
	e.d.CC(e.channel, e.encoderCC).Bind(callback)
}

type ring struct {
	*Encoder
	AllSegments      ringSetAllSegments
	ClearAllSegments ringClearAllSegments
}

// SetLEDRingRelative sets the encoder LED ring based on a relative float value [0.0, 1.0].
// The sweep animates smoothly from left to right, using bit patterns to interpolate between segments.
func (e *ring) Set(v float64) error {
	var lowValue, highValue uint8

	// Clamp value
	if v < 0.0 {
		v = 0.0
	}
	if v > 1.0 {
		v = 1.0
	}

	// There are 26 steps in the sweep from left to right
	const sweepSteps = 26

	// LED ring represented as follows:
	// lowValue defines 6 left segments AND center segment
	// highValue defines 6 right segments
	//
	// Segments are lit in accordance with the bitwise value, thus:
	// 0 -> no LEDs lit
	// 1 -> only the left-most LED is lit
	// 3 -> left-most 2 LEDs are lit
	// 127 -> all LEDs lit
	//
	// To represent a smooth sweep from left to right, instead of only illuminating
	// one segment at a time, we can interpolate by illuminating two side by side segments
	// to suggest an "in-between" value.
	//
	// For value 0, keep the left-most segment illuminated so that it is clear the value is
	// low, rather than suggesting that the control is turned off.
	//
	// Here is the full sequence to sweep from left to right:
	// lowValue:  1, 3, 2, 6, 5, 4, 12, 8, 24, 16, 48, 32, 96, 64, 64, 0, 0, 0, 0, 0,  0, 0,  0,  0,  0,  0
	// highValue: 0, 0, 0, 0, 0, 0,  0, 0,  0,  0,  0,  0,  0,  0,  1, 3, 2, 6, 5, 4, 12, 8, 24, 16, 48, 32

	lowPattern := [sweepSteps]uint8{
		1, 3, 2, 6, 5, 4, 12, 8, 24, 16, 48, 32, 96, 64, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	highPattern := [sweepSteps]uint8{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 3, 2, 6, 5, 4, 12, 8, 24, 16, 48, 32,
	}

	// Map v in [0.0,1.0] to sweepSteps
	step := int(math.Round(v * float64(sweepSteps-1)))

	lowValue = lowPattern[step]
	highValue = highPattern[step]

	// Send both CC messages
	if err := e.d.CC(e.channel, e.ledRingLow).Set(lowValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}
	if err := e.d.CC(e.channel, e.ledRingLow).Set(highValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}

	return nil
}

type ringSetAllSegments struct {
	*Encoder
}

func (e *ringSetAllSegments) Set() error {
	const lowValue uint8 = 0 // TODO: check this
	const highValue uint8 = 127
	if err := e.d.CC(e.channel, e.ledRingLow).Set(lowValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}
	if err := e.d.CC(e.channel, e.ledRingHigh).Set(highValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}
	return nil
}

type ringClearAllSegments struct {
	*Encoder
}

func (e *ringClearAllSegments) Set() error {
	const lowValue uint8 = 0
	const highValue uint8 = 0
	if err := e.d.CC(e.channel, e.ledRingLow).Set(lowValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}
	if err := e.d.CC(e.channel, e.ledRingHigh).Set(highValue); err != nil {
		return fmt.Errorf("failed to set low LED ring value: %v", err)
	}
	return nil
}
