package devices

import (
	"fmt"

	midi "gitlab.com/gomidi/midi/v2"
)

type XTouch struct {
	base MidiDevice
}

type XTouchFader struct {
	MidiDevice

	sendToDevice func(msg midi.Message) error

	Channel               int
	scribbleColor         ScribbleColor
	scribbleMessageTop    []byte
	scribbleMessageBottom []byte
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

func (d *XTouch) GetNumFaders() int {
	return 8 //TODO:
}

// TODO support aggregating devices

func (d *XTouch) Fader(i int) (XTouchFader, error) {
	if i > d.GetNumFaders()-1 {
		return XTouchFader{}, fmt.Errorf("Fader %d out of range %d", i, d.GetNumFaders())
	}
	return XTouchFader{MidiDevice: d.base, Channel: i}, nil
}

func (f *XTouchFader) SendScribble() error {
	b := make([]byte, 0, 20)
	b = append(HeaderScribble, byte(f.Channel))
	b = append(b, byte(f.scribbleColor))
	b = append(b, f.scribbleMessageTop...)
	b = append(b, f.scribbleMessageBottom...)
	return f.sendToDevice(midi.SysEx(b))
}

func (f *XTouchFader) SetScribbleColor(color ScribbleColor) error {
	f.scribbleColor = color
	return f.SendScribble()
}

func (f *XTouchFader) SetScribbleMessageTop(m string) error {
	// TODO: checking; downcasting
	f.scribbleMessageTop = []byte(m)
	return f.SendScribble()
}

func (f *XTouchFader) SetScribbleMessageBottom(m string) error {
	// TODO: checking; downcasting
	f.scribbleMessageBottom = []byte(m)
	return f.SendScribble()
}

func (f *XTouchFader) RegisterFaderMove(effect EffectPitchBend) {
	f.pitchBend = append(f.pitchBend, actionPitchBend{
		channel: uint8(1 + f.Channel),
		action:  effect,
	})
}

func (f *XTouchFader) SetFaderAbsolute(val int16) error {
	return f.sendToDevice(midi.Pitchbend(uint8(1+f.Channel), val))
}
