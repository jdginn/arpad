package xtouch

import (
	dev "github.com/jdginn/arpad/devices"
	midi "gitlab.com/gomidi/midi/v2"
)

type ledState uint8

const (
	ledOff ledState = iota
	ledOn
	ledFlashing
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
	callbacks []func(bool) error
}

// Bind specifies the callback to run when this button is pressed.
func (b *Button) Bind(nil, callback func(bool) error) {
	b.callbacks = append(b.callbacks, callback)
}

func (b *Button) IsPressed() bool {
	return b.isPressed
}

func (b *Button) SetLEDOff() error {
	return b.d.Send(midi.NoteOn(b.channel, b.key, 0))
}

func (b *Button) SetLEDFlashing() error {
	return b.d.Send(midi.NoteOn(b.channel, b.key, 1))
}

func (b *Button) SetLEDOn() error {
	return b.d.Send(midi.NoteOn(b.channel, b.key, 127))
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewButton(channel, key uint8, callbacks ...func(bool) error) *Button {
	b := &Button{
		d:         x.base,
		channel:   channel,
		key:       key,
		callbacks: callbacks,
	}
	x.base.BindNote(dev.PathNote{channel, key}, func(v bool) error {
		if v {
			b.SetLEDOn()
		} else {
			b.SetLEDOff()
		}
		b.isPressed = v
		for _, e := range b.callbacks {
			e(b.isPressed)
		}
		return nil
	})
	return b
}

type ToggleButton struct {
	*Button

	isToggled bool
	callbacks []func(bool) error
}

func (b *ToggleButton) SetToggle(val bool) error {
	b.isToggled = val
	return nil
}

func (b *ToggleButton) IsToggled() bool {
	return b.isToggled
}

func (b *ToggleButton) Bind(nil, callback func(bool) error) {
	b.callbacks = append(b.callbacks, callback)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewToggleButton(channel, key uint8, callbacks ...func(bool) error) *ToggleButton {
	b := &ToggleButton{
		Button: &Button{
			d:         x.base,
			channel:   channel,
			key:       key,
			callbacks: callbacks,
		},
	}
	x.base.BindNote(dev.PathNote{channel, key}, func(v bool) error {
		b.isPressed = v
		if v {
			b.isToggled = !b.isToggled
			if b.isToggled {
				b.SetLEDOn()
			} else {
				b.SetLEDOff()
			}
			for _, e := range b.callbacks {
				e(b.isToggled)
			}
		}
		return nil
	})
	return b
}
