package devices

import (
	"fmt"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type PathCC struct {
	Channel    uint8
	Controller uint8
}

type ArgsCC struct {
	Value uint8
}

type CC struct {
	path     PathCC
	callback func(ArgsCC) error
}

type PathPitchBend struct {
	Channel uint8
}

type ArgsPitchBend struct {
	Relative int16
	Absolute uint16
}

type PitchBend struct {
	path     PathPitchBend
	callback func(ArgsPitchBend) error
}

type PathNote struct {
	Channel uint8
	Key     uint8
}

type Note struct {
	path     PathNote
	callback func(bool) error
}

type PathAfterTouch struct {
	Channel uint8
}

type ArgsAfterTouch struct {
	Pressure uint8
}

type AfterTouch struct {
	path     PathAfterTouch
	callback func(ArgsAfterTouch) error
}

// MidiDevice represents a generic MIDI device and allows registering effects for various messages the device may receive.
type MidiDevice struct {
	inPort  drivers.In
	outPort drivers.Out

	cc         []CC
	pitchBend  []PitchBend
	note       []Note
	aftertouch []AfterTouch
}

func NewMidiDevice(inPort drivers.In, outPort drivers.Out) MidiDevice {
	return MidiDevice{
		inPort:     inPort,
		outPort:    outPort,
		cc:         []CC{},
		pitchBend:  []PitchBend{},
		note:       []Note{},
		aftertouch: []AfterTouch{},
	}
}

func (f *MidiDevice) BindCC(path PathCC, callback func(ArgsCC) error) {
	f.cc = append(f.cc, CC{
		path:     path,
		callback: callback,
	})
}

func (f *MidiDevice) BindNote(path PathNote, callback func(bool) error) {
	f.note = append(f.note, Note{
		path:     path,
		callback: callback,
	})
}

func (f *MidiDevice) BindPitchBend(path PathPitchBend, callback func(ArgsPitchBend) error) {
	f.pitchBend = append(f.pitchBend, PitchBend{
		path:     path,
		callback: callback,
	})
}

func (f *MidiDevice) BindAfterTouch(path PathAfterTouch, callback func(ArgsAfterTouch) error) {
	f.aftertouch = append(f.aftertouch, AfterTouch{
		path:     path,
		callback: callback,
	})
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
			for _, cc := range f.cc {
				if cc.path.Channel == channel && cc.path.Controller == control {
					if err := cc.callback(ArgsCC{value}); err != nil {
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
			for _, pitchbend := range f.pitchBend {
				if pitchbend.path.Channel == channel {
					if err := pitchbend.callback(ArgsPitchBend{relative, absolute}); err != nil {
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
			for _, note := range f.note {
				if note.path.Key == key {
					if err := note.callback(true); err != nil {
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
			for _, note := range f.note {
				if note.path.Key == key {
					if err := note.callback(false); err != nil {
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
			for _, aftertouch := range f.aftertouch {
				if aftertouch.path.Channel == channel {
					if err := aftertouch.callback(ArgsAfterTouch{pressure}); err != nil {
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
