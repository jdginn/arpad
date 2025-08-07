package devices

import (
	"log/slog"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"

	"github.com/jdginn/arpad/logging"
)

var midiInLog, midiOutLog *slog.Logger

func init() {
	midiInLog = logging.Get(logging.MIDI_IN)
	midiOutLog = logging.Get(logging.MIDI_OUT)
}

// MidiDevice represents a generic MIDI device and allows registering effects for various messages the device may receive.
type MidiDevice struct {
	inPort  drivers.In
	outPort drivers.Out

	SysEx *sysEx

	cc         []*cC
	pitchBend  []*pitchBend
	noteOn     []*noteOn
	noteOff    []*noteOff
	aftertouch []*afterTouch
	sysex      []*sysExMatch
}

func (f *MidiDevice) CC(channel, controller uint8) *cC {
	return &cC{
		device:     f,
		channel:    channel,
		controller: controller,
	}
}

func (f *MidiDevice) PitchBend(channel uint8) *pitchBend {
	return &pitchBend{
		device:  f,
		channel: channel,
	}
}

func (f *MidiDevice) Note(channel, key uint8) *note {
	return &note{
		On: &noteOn{
			device:  f,
			channel: channel,
			key:     key,
		},
		Off: &noteOff{
			device:  f,
			channel: channel,
			key:     key,
		},
	}
}

func (f *MidiDevice) Aftertouch(channel uint8) *afterTouch {
	return &afterTouch{
		device:  f,
		channel: channel,
	}
}

type cC struct {
	device     *MidiDevice
	channel    uint8
	controller uint8
	callback   func(value uint8) error
}

func (ep *cC) Bind(callback func(value uint8) error) {
	ep.callback = callback
	ep.device.cc = append(ep.device.cc, ep)
}

func (ep *cC) Set(value uint8) error {
	midiOutLog.Debug("Sending Control Change", "channel", ep.channel, "controller", ep.controller, "value", value)
	return ep.device.outPort.Send(midi.ControlChange(ep.channel, ep.controller, value))
}

type pitchBend struct {
	device   *MidiDevice
	channel  uint8
	callback func(uint16) error
}

func (ep *pitchBend) Bind(callback func(uint16) error) {
	ep.callback = callback
	ep.device.pitchBend = append(ep.device.pitchBend, ep)
}

func (ep *pitchBend) Set(value uint16) error {
	midiOutLog.Debug("Sending Pitch Bend", "channel", ep.channel, "value", value)
	return ep.device.outPort.Send(midi.Pitchbend(ep.channel, int16(value-0x2000)))
}

type note struct {
	On  *noteOn
	Off *noteOff
}

type noteOn struct {
	device   *MidiDevice
	channel  uint8
	key      uint8
	callback func(uint8) error
}

func (ep *noteOn) Bind(callback func(uint8) error) {
	ep.callback = callback
	ep.device.noteOn = append(ep.device.noteOn, ep)
}

func (ep *noteOn) Set(velocity uint8) error {
	midiOutLog.Debug("Sending Note On", "channel", ep.channel, "key", ep.key, "velocity", velocity)
	return ep.device.outPort.Send(midi.NoteOn(ep.channel, ep.key, velocity))
}

type noteOff struct {
	device   *MidiDevice
	channel  uint8
	key      uint8
	callback func() error
}

func (ep *noteOff) Bind(callback func() error) {
	ep.callback = callback
	ep.device.noteOff = append(ep.device.noteOff, ep)
}

func (ep *noteOff) Set() error {
	midiOutLog.Debug("Sending Note Off", "channel", ep.channel, "key", ep.key)
	return ep.device.outPort.Send(midi.NoteOff(ep.channel, ep.key))
}

type afterTouch struct {
	device   *MidiDevice
	channel  uint8
	callback func(uint8) error
}

func (ep *afterTouch) Bind(callback func(uint8) error) {
	ep.callback = callback
	ep.device.aftertouch = append(ep.device.aftertouch, ep)
}

func (ep *afterTouch) Set(value uint8) error {
	midiOutLog.Debug("Sending After Touch", "channel", ep.channel, "value", value)
	return ep.device.outPort.Send(midi.AfterTouch(ep.channel, value))
}

type sysEx struct {
	device *MidiDevice
}

func (ep *sysEx) Match(pattern []byte) *sysExMatch {
	return &sysExMatch{
		pattern: pattern,
		device:  ep.device,
	}
}

func (ep *sysEx) Set(value []byte) error {
	midiOutLog.Debug("Sending SysEx", "data", value)
	return ep.device.outPort.Send(value)
}

type sysExMatch struct {
	pattern  []byte
	device   *MidiDevice
	callback func([]byte) error
}

func (ep *sysExMatch) Bind(callback func([]byte) error) {
	ep.callback = callback
	ep.device.sysex = append(ep.device.sysex, ep)
}

func NewMidiDevice(inPort drivers.In, outPort drivers.Out) *MidiDevice {
	d := &MidiDevice{
		inPort:  inPort,
		outPort: outPort,
		SysEx: &sysEx{
			device: &MidiDevice{},
		},
		cc:         []*cC{},
		pitchBend:  []*pitchBend{},
		noteOn:     []*noteOn{},
		noteOff:    []*noteOff{},
		aftertouch: []*afterTouch{},
		sysex:      []*sysExMatch{},
	}
	d.SysEx = &sysEx{device: d}
	return d
}

// Run starts this device and causes it to listen and respond to incoming MIDI messages.
//
// For any message with an effect registered, that effect will be run each time such a message is received.

func (d *MidiDevice) Run() {
	midiInLog.Info("Starting MIDI device", "inPort", d.inPort.String(), "outPort", d.outPort.String())
	go d.run()
}

func (f *MidiDevice) run() {
	f.inPort.Open()
	defer f.inPort.Close()
	f.outPort.Open()
	defer f.outPort.Close()

	var err error
	var stop func()

	stop, err = midi.ListenTo(f.inPort, func(msg midi.Message, timestampms int32) {
		switch msg.Type() {
		case midi.ControlChangeMsg:
			var channel, control, value uint8
			if ok := msg.GetControlChange(&channel, &control, &value); !ok {
				midiInLog.Error("failed to parse Control Change message:", err)
				return
			}
			midiInLog.Debug("received Control Change message", "channel", channel, "control", control, "value", value, "timestamp", timestampms)
			for _, cc := range f.cc {
				if cc.channel == channel && cc.controller == control {
					if err := cc.callback(value); err != nil {
						midiInLog.Error("failed to process Control Change:", err)
					}
				}
			}
		case midi.PitchBendMsg:
			var channel uint8
			var relative int16
			var absolute uint16
			if ok := msg.GetPitchBend(&channel, &relative, &absolute); !ok {
				midiInLog.Error("failed to parse Pitch Bend message:", err)
				return
			}
			midiInLog.Debug("received Pitch Bend message", "channel", channel, "absolute", absolute, "timestamp", timestampms)
			for _, pitchbend := range f.pitchBend {
				if pitchbend.channel == channel {
					if err := pitchbend.callback(absolute); err != nil {
						midiInLog.Error("failed to process Pitch Bend:", err)
					}
				}
			}
		case midi.NoteOnMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOn(&channel, &key, &velocity); !ok {
				midiInLog.Error("failed to parse Note On message:", err)
				return
			}
			midiInLog.Debug("received Note On message", "channel", channel, "key", key, "velocity", velocity, "timestamp", timestampms)
			for _, note := range f.noteOn {
				if note.key == key && note.channel == channel {
					if err := note.callback(velocity); err != nil {
						midiInLog.Error("failed to process Note On:", err)
					}
				}
			}
		case midi.NoteOffMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOff(&channel, &key, &velocity); !ok {
				midiInLog.Error("failed to parse Note Off message:", err)
				return
			}
			midiInLog.Debug("received Note Off message", "channel", channel, "key", key, "velocity", velocity, "timestamp", timestampms)
			for _, note := range f.noteOff {
				if note.key == key && note.channel == channel {
					if err := note.callback(); err != nil {
						midiInLog.Error("failed to process Note Off:", err)
					}
				}
			}
		case midi.AfterTouchMsg:
			var channel, pressure uint8
			if ok := msg.GetAfterTouch(&channel, &pressure); !ok {
				midiInLog.Error("failed to parse After Touch message:", err)
				return
			}
			midiInLog.Debug("received After Touch message", "channel", channel, "pressure", pressure, "timestamp", timestampms)
			for _, aftertouch := range f.aftertouch {
				if aftertouch.channel == channel {
					if err := aftertouch.callback(pressure); err != nil {
						midiInLog.Error("failed to process After Touch:", err)
					}
				}
			}
		case midi.SysExMsg:
			var data []byte
			if ok := msg.GetSysEx(&data); !ok {
				midiInLog.Error("failed to parse SysEx message:", err)
				return
			}
			// midiInLog.Debug("received SysEx message", "data", data, "timestamp", timestampms)
			for _, sysex := range f.sysex {
				// Check if the message matches the pattern
				//
				// NOTE: currently, we check for directly matching patterns; this won't work with variable arguments embedded into the data
				if len(data) >= len(sysex.pattern) {
					matches := true
					for i, b := range sysex.pattern {
						if data[i] != b {
							matches = false
							break
						}
					}
					if matches {
						if err := sysex.callback(data); err != nil {
							midiInLog.Error("failed to process SysEx:", err)
						}
					}
				}
			}
		}
	}, midi.UseSysEx())
	if err != nil {
		midiInLog.Error("ERROR: %s\n", err)
		return
	}

	// TODO: put this in a goroutine instead of sleeping!
	time.Sleep(time.Second * 1000)

	stop()
}
