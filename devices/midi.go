package devices

import (
	"fmt"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

// CallbackCC operates on a control change message.
type CallbackCC func(uint8) error

type eventCC struct {
	channel    uint8
	controller uint8
	callback   CallbackCC
}

// CallbackPitchBend operates on a pitch bend message.
type CallbackPitchBend func(int16, uint16) error

type eventPitchBend struct {
	channel  uint8
	callback CallbackPitchBend
}

// CallbackNote operates on the result of a note message.
type CallbackNote func(bool) error

type eventNote struct {
	channel  uint8
	key      uint8
	callback CallbackNote
}

// CallbackAftertouchtoperates on an AfterTouch message.
type CallbackAftertouch func(uint8) error

type eventAfterTouch struct {
	channel  uint8
	callback CallbackAftertouch
}

// MidiDevice represents a generic MIDI device and allows registering effects for various messages the device may receive.
type MidiDevice struct {
	inPort  drivers.In
	outPort drivers.Out

	cc         []eventCC
	pitchBend  []eventPitchBend
	note       []eventNote
	aftertouch []eventAfterTouch
}

func NewMidiDevice(inPort drivers.In, outPort drivers.Out) MidiDevice {
	return MidiDevice{
		inPort:     inPort,
		outPort:    outPort,
		cc:         []eventCC{},
		pitchBend:  []eventPitchBend{},
		note:       []eventNote{},
		aftertouch: []eventAfterTouch{},
	}
}

// BindCC specifies an callback that should be run each time a control change message is received for the specifeid channel and controller.
func (f *MidiDevice) BindCC(channel, controller uint8, callback CallbackCC) {
	f.cc = append(f.cc, eventCC{
		channel:    channel,
		controller: controller,
		callback:   callback,
	})
}

// BindNote specifies an callback that should be run each time a note message is received for the specifeid channel and key.
func (f *MidiDevice) BindNote(channel, key uint8, callback CallbackNote) {
	f.note = append(f.note, eventNote{
		channel:  channel,
		key:      key,
		callback: callback,
	})
}

// BindPitchBend specifies an callback that should be run each time a pitchbend message is received for the specified channel.
func (f *MidiDevice) BindPitchBend(channel uint8, callback CallbackPitchBend) {
	f.pitchBend = append(f.pitchBend, eventPitchBend{
		channel:  channel,
		callback: callback,
	})
}

// RegisterPitchBend specifies an callback that should be run each time a channel pressure message is received for the specified channel.
func (f *MidiDevice) BindChannelPressure(channel uint8, callback CallbackAftertouch) {
}

// Send sends a message to this device's outPort.
func (f *MidiDevice) Send(msg midi.Message) error {
	return f.outPort.Send(msg)
}

// Run starts this device and causes it to listen and respond to incoming MIDI messages.
//
// For any message with an effect registered, that effect will be run each time such a message is received.
func (f *MidiDevice) Run() {
	defer midi.CloseDriver()

	in, err := midi.FindInPort("VMPK")
	if err != nil {
		fmt.Println("can't find VMPK:", err)
		return
	}

	var stop func()
	stop, err = midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		switch msg.Type() {
		case midi.ControlChangeMsg:
			var channel, control, value uint8
			if ok := msg.GetControlChange(&channel, &control, &value); !ok {
				fmt.Println("failed to parse Control Change message:", err)
				return
			}
			for _, callback := range f.cc {
				if callback.channel == channel && callback.controller == control {
					if err := callback.callback(value); err != nil {
						fmt.Println("failed to process Control Change:", err)
					}
					break
				}
			}
		case midi.PitchBendMsg:
			var channel uint8
			var relative int16
			var absolute uint16
			if ok := msg.GetPitchBend(&channel, &relative, &absolute); !ok {
				fmt.Println("failed to parse Pitch Bend message:", err)
				return
			}
			for _, callback := range f.pitchBend {
				if callback.channel == channel {
					if err := callback.callback(relative, absolute); err != nil {
						fmt.Println("failed to process Pitch Bend:", err)
					}
					break
				}
			}
		case midi.NoteOnMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOn(&channel, &key, &velocity); !ok {
				fmt.Println("failed to parse Note On message:", err)
				return
			}
			for _, callback := range f.note {
				if callback.key == key {
					if err := callback.callback(true); err != nil {
						fmt.Println("failed to process Note On:", err)
					}
				}
			}
		case midi.NoteOffMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOff(&channel, &key, &velocity); !ok {
				fmt.Println("failed to parse Note Off message:", err)
				return
			}
			for _, callback := range f.note {
				if callback.key == key {
					if err := callback.callback(false); err != nil {
						fmt.Println("failed to process Note Off:", err)
					}
				}
			}
		case midi.AfterTouchMsg:
			var channel, pressure uint8
			if ok := msg.GetAfterTouch(&channel, &pressure); !ok {
				fmt.Println("failed to parse After Touch message:", err)
				return
			}
			for _, callback := range f.aftertouch {
				if callback.channel == channel {
					if err := callback.callback(pressure); err != nil {
						fmt.Println("failed to process After Touch:", err)
					}
				}
			}
		}
	}, midi.UseSysEx())
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	stop()
}
