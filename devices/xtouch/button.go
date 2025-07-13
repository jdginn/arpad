package xtouch

import (
	dev "github.com/jdginn/arpad/devices"
)

type ledState uint8

const (
	ledOff ledState = iota
	ledOn
	ledFlashing
)

type baseButton struct {
	// TODO: this needs a mutex...
	d *dev.MidiDevice

	channel uint8
	key     uint8

	isPressed bool
}

func (b *baseButton) IsPressed() bool {
	return b.isPressed
}

func (b *baseButton) SetLED(val bool) error {
	if val {
		return b.SetLEDOn()
	}
	return b.SetLEDOff()
}

func (b *baseButton) SetLEDOff() error {
	return b.d.Note(b.channel, b.key).On.Set(0)
}

func (b *baseButton) SetLEDFlashing() error {
	return b.d.Note(b.channel, b.key).On.Set(1)
}

func (b *baseButton) SetLEDOn() error {
	return b.d.Note(b.channel, b.key).On.Set(127)
}

// TODO: add FlashOnce

// Button that executes a function when the button is pressed
type Button struct {
	*baseButton
	On  *buttonOn
	Off *buttonOff
}

type buttonOn struct {
	*Button
	callbacks []func(uint8) error
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOn) Bind(callback func(uint8) error) {
	// b.callbacks = append(b.callbacks, callback)
	b.baseButton.d.Note(b.channel, b.key).On.Bind(callback)
}

type buttonOff struct {
	*Button
	callbacks []func() error
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOff) Bind(callback func() error) {
	// b.callbacks = append(b.callbacks, callback)
	b.baseButton.d.Note(b.channel, b.key).Off.Bind(callback)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
func (x *XTouch) NewButton(channel, key uint8) *Button {
	b := &Button{
		baseButton: &baseButton{
			d:       x.base,
			channel: channel,
			key:     key,
		},
	}
	b.On = &buttonOn{Button: b}
	b.Off = &buttonOff{Button: b}
	return b
}
