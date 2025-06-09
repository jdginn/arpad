package xtouch

import (
	dev "github.com/jdginn/arpad/devices"
	midi "gitlab.com/gomidi/midi/v2"
)

type LEDState uint8

const (
	OFF LEDState = iota
	ON
	FLASHING
)

// Button represents a button on an xtouch controller.
//
// Buttons send MIDI notes on a specified channel and key.
// Buttons incorporate LEDs, which can either be off, on, or flashing.
//
// TODO: need toggle vs. momentary functionality
type Button struct {
	// TODO: this needs a mutex...
	d *dev.MidiDevice

	channel uint8
	key     uint8

	isPressed bool
}

// Bind specifies the callback to run when this button is pressed.
func (b *Button) Bind(nil, callback func(bool) error) {
	b.d.BindNote(dev.PathNote{b.channel, b.key}, func(v bool) error {
		return callback(v)
	})
}

func (b *Button) IsPressed() bool {
	return b.isPressed
}

func (f *Button) SetLEDOff() error {
	return f.d.Send(midi.NoteOn(f.channel, f.key, 0))
}

func (f *Button) SetLEDFlashing() error {
	return f.d.Send(midi.NoteOn(f.channel, f.key, 1))
}

func (f *Button) SetLEDOn(state LEDState) error {
	return f.d.Send(midi.NoteOn(f.channel, f.key, 127))
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewButton(channel, key uint8, callbacks ...func(bool) error) *Button {
	b := &Button{
		d:       x.base,
		channel: channel,
		key:     key,
	}
	x.base.BindNote(dev.PathNote{channel, key}, func(v bool) error {
		b.isPressed = v
		return nil
	})
	for _, e := range callbacks {
		x.base.BindNote(dev.PathNote{channel, key}, e)
	}
	return b
}

type ToggleButton struct {
	b *Button

	isToggled bool
}

func (b *ToggleButton) SetToggle(val bool) error {
	b.isToggled = val
	return nil
}

func (b *ToggleButton) IsToggled() bool {
	return b.isToggled
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewToggleButton(channel, key uint8, callbacks ...func(bool) error) ToggleButton {
	b := ToggleButton{
		b: &Button{
			d:       x.base,
			channel: channel,
			key:     key,
		},
	}
	x.base.BindNote(dev.PathNote{channel, key}, func(v bool) error {
		b.b.isPressed = v
		if v {
			b.isToggled = !b.isToggled
			for _, e := range callbacks {
				e(b.isToggled)
			}
		}
		return nil
	})
	return b
}
