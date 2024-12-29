package xtouch

import (
	"fmt"
	"math"

	dev "github.com/jdginn/arpad/devices"

	midi "gitlab.com/gomidi/midi/v2"
)

type Fader struct {
	d dev.MidiDevice

	ChannelNo uint8
}

func (f *Fader) Effect(effect dev.EffectPitchBend) {
	f.d.RegisterPitchBend(uint8(1+f.ChannelNo), effect)
}

func (f *Fader) SetFaderAbsolute(val int16) error {
	return f.d.Send(midi.Pitchbend(uint8(1+f.ChannelNo), val))
}

type LEDState uint8

const (
	OFF LEDState = iota
	ON
	FLASHING
)

type Button struct {
	d dev.MidiDevice

	channel uint8
	key     uint8
}

func (b *Button) Effect(effect dev.EffectNote) {
	b.d.RegisterNote(b.channel, b.key, effect)
}

func (f *Button) SetLED(state LEDState) error {
	switch state {
	case OFF:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 0))
	case ON:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 1))
	case FLASHING:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 127))
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
	d dev.MidiDevice

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
	return s.d.Send(midi.SysEx(b))
}

type Meter struct {
	d dev.MidiDevice

	channel uint8
}

func (m *Meter) SendRelative(val float64) error {
	if val > 1.0 {
		return fmt.Errorf("Invalid val: val must be between 0 and 1.0")
	}
	offset := m.channel*16 + uint8(math.Round(8*val))
	return m.d.Send(midi.AfterTouch(0, offset))
}

type XTouch struct {
	base dev.MidiDevice
}

func (x *XTouch) NewFader(channelNo uint8, effects ...dev.EffectPitchBend) Fader {
	for _, e := range effects {
		x.base.RegisterPitchBend(uint8(1+channelNo), e)
	}
	return Fader{
		d:         x.base,
		ChannelNo: channelNo,
	}
}

func (x *XTouch) NewButton(channel, key uint8, effects ...dev.EffectNote) Button {
	for _, e := range effects {
		x.base.RegisterNote(channel, key, e)
	}
	return Button{
		d:       x.base,
		channel: channel,
		key:     key,
	}
}

func (x *XTouch) NewScribble(channel uint8) Scribble {
	return Scribble{
		x.base,
		channel,
	}
}

func (x *XTouch) NewMeter(channel uint8) Meter {
	return Meter{
		x.base,
		channel,
	}
}

// TODO:
// 7seg Display
// Encoder
// Jog Wheel

type channelStrip struct {
	// Encoder
	Scribble Scribble
	Rec      Button
	Solo     Button
	Mute     Button
	Select   Button
	Meter    Meter
	Fader    Fader
	// 7Seg
	// JogWheel
}

func (x *XTouch) NewChannelStrip(id uint8) channelStrip {
	return channelStrip{
		Scribble: x.NewScribble(id + 20),
		Rec:      x.NewButton(0, id),
		Solo:     x.NewButton(0, id+8),
		Mute:     x.NewButton(0, id+16),
		Select:   x.NewButton(0, id+24),
		Meter:    x.NewMeter(id),
		Fader:    x.NewFader(id + 1),
	}
}

type XTouchDefault struct {
	XTouch

	Channels  []channelStrip
	View      []Button
	Function  []Button
	Transport map[string]Button
}

func New(d dev.MidiDevice) XTouchDefault {
	x := XTouchDefault{
		XTouch:    XTouch{d},
		Channels:  make([]channelStrip, 8),
		View:      make([]Button, 8),
		Function:  make([]Button, 8),
		Transport: make(map[string]Button),
	}
	for i := 0; i < 8; i++ {
		x.Channels[i] = x.NewChannelStrip(uint8(i))
	}
	for i := 0; i < 8; i++ {
		x.Function[i] = x.NewButton(0, 54+uint8(i))
	}
	return x
}

type (
	x *XTouchDefault
)

type XTouchExtender struct {
	XTouch
}
