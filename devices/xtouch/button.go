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

func (b *baseButton) SetLEDOff() error {
	return b.d.Note(b.channel, b.key).Set(true) // TODO: fix this to 0
}

func (b *baseButton) SetLEDFlashing() error {
	return b.d.Note(b.channel, b.key).Set(true) // TODO: fix this to 1
}

func (b *baseButton) SetLEDOn() error {
	return b.d.Note(b.channel, b.key).Set(true) // TODO: fix this to 127
}

// Button that executes a function when the button is pressed
//
// Ignores the NoteOff signal
type Button struct {
	baseButton
	callbacks []func(bool) error
}

// Bind specifies the callback to run when this button is pressed.
func (b *Button) Bind(callback func(bool) error) {
	b.callbacks = append(b.callbacks, callback)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewButton(channel, key uint8, callbacks ...func(bool) error) *Button {
	b := &Button{
		baseButton: baseButton{
			d:       x.base,
			channel: channel,
			key:     key,
		},
		callbacks: callbacks,
	}
	x.base.Note(channel, key).Bind(func(v bool) error {
		b.isPressed = v
		if b.isPressed {
			b.SetLEDOn()
			for _, e := range b.callbacks {
				e(b.isPressed)
			}
			return nil
		}
		return nil
	})
	return b
}

// MomentaryButton is a button that responds both to the NoteOn and NoteOff signals
type MomentaryButton struct {
	baseButton

	callbacks []func(bool) error
}

// NewMomentaryButton returns a new button corresponding to the given channel and MIDI key.
//
// NewMomentaryButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewMomentaryButton(channel, key uint8, callbacks ...func(bool) error) *MomentaryButton {
	b := &MomentaryButton{
		baseButton: baseButton{
			d:       x.base,
			channel: channel,
			key:     key,
		},
		callbacks: callbacks,
	}
	x.base.Note(channel, key).Bind(func(v bool) error {
		b.isPressed = v
		if b.isPressed {
			b.SetLEDOn()
		} else {
			b.SetLEDOff()
		}
		for _, e := range b.callbacks {
			e(b.isPressed)
		}
		return nil
	})
	return b
}

// Bind specifies the callback to run when this button is pressed.
func (b *MomentaryButton) Bind(callback func(bool) error) {
	b.callbacks = append(b.callbacks, callback)
}

type ToggleButton struct {
	baseButton

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

func (b *ToggleButton) Bind(callback func(bool) error) {
	b.callbacks = append(b.callbacks, callback)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewToggleButton(channel, key uint8, callbacks ...func(bool) error) *ToggleButton {
	b := &ToggleButton{
		baseButton: baseButton{
			d:       x.base,
			channel: channel,
			key:     key,
		},
		callbacks: callbacks,
	}
	x.base.Note(channel, key).Bind(func(v bool) error {
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
