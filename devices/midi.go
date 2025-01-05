package devices

import (
	"fmt"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

// EffectCC defines a callback function that operates as a result of a control change message.
type EffectCC func(uint8) error

type actionCC struct {
	channel    uint8
	controller uint8
	action     EffectCC
}

// EffectPitchBend defines a callback function that operates as a result of a pitch bend message.
type EffectPitchBend func(int16, uint16) error

type actionPitchBend struct {
	channel uint8
	action  EffectPitchBend
}

// EffectNote defines a callback function that operates as a result of a note message.
type EffectNote func(bool) error

type actionNote struct {
	channel uint8
	key     uint8
	action  EffectNote
}

// EffectNote defines a callback function that operates as a result of an AfterTouch message.
type EffectAftertouch func(uint8) error

type actionAfterTouch struct {
	channel uint8
	action  EffectAftertouch
}

// MidiDevice represents a generic MIDI device and allows registering effects for various messages the device may receive.
type MidiDevice struct {
	inPort  drivers.In
	outPort drivers.Out

	cc         []actionCC
	pitchBend  []actionPitchBend
	note       []actionNote
	aftertouch []actionAfterTouch
}

func NewMidiDevice(inPort drivers.In, outPort drivers.Out) MidiDevice {
	return MidiDevice{
		inPort:     inPort,
		outPort:    outPort,
		cc:         []actionCC{},
		pitchBend:  []actionPitchBend{},
		note:       []actionNote{},
		aftertouch: []actionAfterTouch{},
	}
}

// RegisterNote specifies an action that should be run each time a control change message is received for the specifeid channel and controller.
func (f *MidiDevice) RegisterCC(channel, controller uint8, action EffectCC) {
	f.cc = append(f.cc, actionCC{
		channel:    channel,
		controller: controller,
		action:     action,
	})
}

// RegisterNote specifies an action that should be run each time a note message is received for the specifeid channel and key.
func (f *MidiDevice) RegisterNote(channel, key uint8, action EffectNote) {
	f.note = append(f.note, actionNote{
		channel: channel,
		key:     key,
		action:  action,
	})
}

// RegisterPitchBend specifies an action that should be run each time a pitchbend message is received for the specified channel.
func (f *MidiDevice) RegisterPitchBend(channel uint8, action EffectPitchBend) {
	f.pitchBend = append(f.pitchBend, actionPitchBend{
		channel: channel,
		action:  action,
	})
}

// RegisterPitchBend specifies an action that should be run each time a channel pressure message is received for the specified channel.
func (f *MidiDevice) RegisterChannelPressure(channel uint8, action EffectAftertouch) {
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
			for _, action := range f.cc {
				if action.channel == channel && action.controller == control {
					if err := action.action(value); err != nil {
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
			for _, action := range f.pitchBend {
				if action.channel == channel {
					if err := action.action(relative, absolute); err != nil {
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
			for _, action := range f.note {
				if action.key == key {
					if err := action.action(true); err != nil {
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
			for _, action := range f.note {
				if action.key == key {
					if err := action.action(false); err != nil {
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
			for _, action := range f.aftertouch {
				if action.channel == channel {
					if err := action.action(pressure); err != nil {
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
