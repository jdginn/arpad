package xtouch

import (
	dev "github.com/jdginn/arpad/devices"
)

// Button that executes a function when the button is pressed
type Button struct {
	d *dev.MidiDevice

	channel uint8
	key     uint8

	On  *buttonOn
	Off *buttonOff
	LED *led
}

type buttonOn struct {
	*Button
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOn) Bind(callback func() error) func() {
	return b.d.Note(b.channel, b.key).On.Bind(func(v uint8) error {
		if v == 127 {
			return callback()
		}
		return nil
	})
}

type buttonOff struct {
	*Button
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOff) Bind(callback func() error) func() {
	return b.d.Note(b.channel, b.key).On.Bind(func(v uint8) error {
		if v == 0 {
			return callback()
		}
		return nil
	})
}

type led struct {
	On       *ledOn
	Off      *ledOff
	Flashing *ledFlashing
}

func (l *led) Set(val bool) error {
	if val {
		return l.On.Set()
	}
	return l.Off.Set()
}

type ledOn struct {
	*Button
}

func (l *ledOn) Set() error {
	return l.d.Note(l.channel, l.key).On.Set(127)
}

type ledOff struct {
	*Button
}

func (l *ledOff) Set() error {
	return l.d.Note(l.channel, l.key).On.Set(0)
}

type ledFlashing struct {
	*Button
}

func (l *ledFlashing) SetF() error {
	return l.d.Note(l.channel, l.key).On.Set(1)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
func (x *XTouch) NewButton(channel, key uint8) *Button {
	b := &Button{
		d:       x.base,
		channel: channel,
		key:     key,
	}
	b.On = &buttonOn{Button: b}
	b.Off = &buttonOff{Button: b}
	b.LED = &led{
		On:       &ledOn{Button: b},
		Off:      &ledOff{Button: b},
		Flashing: &ledFlashing{Button: b},
	}
	return b
}
