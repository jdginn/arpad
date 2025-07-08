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
	baseButton
	On  *buttonOn
	Off *buttonOff
}

type buttonOn struct {
	*Button
	callbacks []func(uint8) error
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOn) Bind(callback func(uint8) error) {
	b.callbacks = append(b.callbacks, callback)
}

type buttonOff struct {
	*Button
	callbacks []func() error
}

// Bind specifies the callback to run when this button is pressed.
func (b *buttonOff) Bind(callback func() error) {
	b.callbacks = append(b.callbacks, callback)
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewButton(channel, key uint8) *Button {
	b := &Button{
		baseButton: baseButton{
			d:       x.base,
			channel: channel,
			key:     key,
		},
	}
	b.On = &buttonOn{Button: b}
	b.Off = &buttonOff{Button: b}
	x.base.Note(channel, key).On.Bind(func(velocity uint8) error {
		b.isPressed = true
		b.SetLEDOn()
		for _, e := range b.On.callbacks {
			e(velocity)
		}
		return nil
	})
	x.base.Note(channel, key).Off.Bind(func() error {
		b.isPressed = true
		b.SetLEDOff()
		for _, e := range b.Off.callbacks {
			e()
		}
		return nil
	})
	return b
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
	x.base.Note(channel, key).On.Bind(func(velocity uint8) error {
		b.isToggled = !b.isToggled
		if b.isToggled {
			b.SetLEDOn()
		} else {
			b.SetLEDOff()
		}
		for _, e := range b.callbacks {
			e(b.isToggled)
		}
		return nil
	})
	return b
}
