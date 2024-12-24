package devices

import (
	"fmt"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type EffectCC func(uint8) error

type actionCC struct {
	channel    uint8
	controller uint8
	action     EffectCC
}

type EffectPitchBend func(int16, uint16) error

type actionPitchBend struct {
	channel uint8
	action  EffectPitchBend
}

type EffectNote func(bool) error

type actionNote struct {
	channel uint8
	key     uint8
	action  EffectNote
}

type MidiDevice struct {
	inPort  drivers.In
	outPort drivers.Out

	cc        []actionCC
	pitchBend []actionPitchBend
	note      []actionNote
}

func (f *MidiDevice) RegisterCC(channel, controller uint8, action EffectCC) {
	f.cc = append(f.cc, actionCC{
		channel:    channel,
		controller: controller,
		action:     action,
	})
}

func (f *MidiDevice) RegisterNote(channel, key uint8, action EffectNote) {
	f.note = append(f.note, actionNote{
		channel: channel,
		key:     key,
		action:  action,
	})
}

func (f *MidiDevice) RegisterPitchBend(channel uint8, action EffectPitchBend) {
	f.pitchBend = append(f.pitchBend, actionPitchBend{
		channel: channel,
		action:  action,
	})
}

func (f *MidiDevice) Send(msg midi.Message) error {
	return f.outPort.Send(msg)
}

func (f *MidiDevice) Run() {
	defer midi.CloseDriver()

	//TODO: wtf
	in, err := midi.FindInPort("VMPK")
	if err != nil {
		fmt.Println("can't find VMPK")
		return
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {

		switch msg.Type() {
		case midi.ControlChangeMsg:
			var channel, control, value uint8
			if ok := msg.GetControlChange(&channel, &control, &value); !ok {
				panic("bad")
			}
			for _, action := range f.cc {
				if action.channel == channel && action.controller == control {
					if err := action.action(value); err != nil {
						panic("bad")
					}
					break
				}
			}
		case midi.PitchBendMsg:
			var channel uint8
			var relative int16
			var absolute uint16
			if ok := msg.GetPitchBend(&channel, &relative, &absolute); !ok {
				panic("bad")
			}
			for _, action := range f.pitchBend {
				if action.channel == channel {
					if err := action.action(relative, absolute); err != nil {
						panic("bad")
					}
					break
				}
			}
		case midi.NoteOnMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOn(&channel, &key, &velocity); !ok {
				panic("bad")
			}
			for _, action := range f.note {
				if action.key == key {
					if err := action.action(true); err != nil {
						panic("bad")
					}
				}
			}
		case midi.NoteOffMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOff(&channel, &key, &velocity); !ok {
				panic("bad")
			}
			for _, action := range f.note {
				if action.key == key {
					if err := action.action(false); err != nil {
						panic("bad")
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
