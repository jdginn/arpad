package xtouch

import (
	"fmt"

	dev "github.com/jdginn/arpad/devices"

	midi "gitlab.com/gomidi/midi/v2"
)

type Fader struct {
	dev.MidiDevice

	ChannelNo uint8
}

func (f *Fader) SetFaderAbsolute(val int16) error {
	return f.Send(midi.Pitchbend(uint8(1+f.ChannelNo), val))
}

type LEDState uint8

const (
	OFF LEDState = iota
	ON
	FLASHING
)

type Button struct {
	dev.MidiDevice

	channel uint8
	key     uint8
}

func (f *Button) SetLED(state LEDState) error {
	switch state {
	case OFF:
		return f.Send(midi.NoteOn(f.channel, f.key, 0))
	case ON:
		return f.Send(midi.NoteOn(f.channel, f.key, 1))
	case FLASHING:
		return f.Send(midi.NoteOn(f.channel, f.key, 127))
	default:
		return fmt.Errorf("Unrecognized LED state")
	}
}

type ScribbleColor int

const (
	Off       ScribbleColor = 0x00
	Red       ScribbleColor = 0x01
	Green     ScribbleColor = 0x02
	Yellow    ScribbleColor = 0x03
	Blue      ScribbleColor = 0x04
	Pink      ScribbleColor = 0x05
	Cyan      ScribbleColor = 0x06
	White     ScribbleColor = 0x07
	RedInv    ScribbleColor = 0x41
	GreenInv  ScribbleColor = 0x42
	YellowInv ScribbleColor = 0x43
	BlueInv   ScribbleColor = 0x44
	PinkInv   ScribbleColor = 0x45
	CyanInv   ScribbleColor = 0x46
	WhiteInv  ScribbleColor = 0x47
)

type SysExHeader []byte

var HeaderScribble SysExHeader = []byte{0x00, 0x00, 0x66, 0x58}

type Scribble struct {
	dev.MidiDevice

	channel uint8
}

// TODO: consider making this take strings instead of []byte?
func (s *Scribble) SendScribble(color ScribbleColor, msgTop, msgBottom []byte) error {
	// TODO: check msg for length, support best-effort truncation?
	b := make([]byte, 0, 20)
	b = append(HeaderScribble, byte(s.channel))
	b = append(b, byte(color))
	b = append(b, msgTop...)
	b = append(b, msgBottom...)
	return s.Send(midi.SysEx(b))
}

type XTouch struct {
	base dev.MidiDevice
}

func (x *XTouch) NewFader(channelNo uint8, effects ...dev.EffectPitchBend) Fader {
	for _, e := range effects {
		x.base.RegisterPitchBend(uint8(1+channelNo), e)
	}
	return Fader{
		MidiDevice: x.base,
		ChannelNo:  channelNo,
	}
}

func (x *XTouch) NewButton(channel, key uint8, effects ...dev.EffectNote) Button {
	for _, e := range effects {
		x.base.RegisterNote(channel, key, e)
	}
	return Button{
		MidiDevice: x.base,
		channel:    channel,
		key:        key,
	}
}

func (x *XTouch) NewScribble(channel uint8) Scribble {
	return Scribble{
		x.base,
		channel,
	}
}

// types:
// 7seg Display
// Meter
// Encoder
// Jog Wheel

type channelStrip struct {
	// Encoder
	Scribble Scribble
	Rec      Button
	Solo     Button
	Mute     Button
	Select   Button
	// Meter
	Fader Fader
}

type Builder struct {
	dev.MidiDevice

	faders    map[string]Fader
	channels  []channelStrip
	transport map[string]Button
	view      map[string]Button
	functions map[string]Button
}

type XTouchDefault struct {
	XTouch
}

type XTouchExtender struct {
	XTouch
}
