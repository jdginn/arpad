package devices

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
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

	mu         sync.RWMutex
	cc         map[*cC]struct{}
	pitchBend  map[*pitchBend]struct{}
	noteOn     map[*noteOn]struct{}
	noteOff    map[*noteOff]struct{}
	aftertouch map[*afterTouch]struct{}
	sysex      map[*sysExMatch]struct{}
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

func (ep *cC) Bind(callback func(value uint8) error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.cc[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.cc, ep)
		ep.device.mu.Unlock()
	}
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

func (ep *pitchBend) Bind(callback func(uint16) error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.pitchBend[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.pitchBend, ep)
		ep.device.mu.Unlock()
	}
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

func (ep *noteOn) Bind(callback func(uint8) error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.noteOn[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.noteOn, ep)
		ep.device.mu.Unlock()
	}
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

func (ep *noteOff) Bind(callback func() error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.noteOff[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.noteOff, ep)
		ep.device.mu.Unlock()
	}
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

func (ep *afterTouch) Bind(callback func(uint8) error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.aftertouch[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.aftertouch, ep)
		ep.device.mu.Unlock()
	}
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

func byteSliceToHexLiteral(b []byte) string {
	var sb strings.Builder
	sb.WriteString("[]byte{")
	for i, v := range b {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("0x%02x", v))
	}
	sb.WriteString("}")
	return sb.String()
}

func (ep *sysEx) Set(value []byte) error {
	midiOutLog.Debug("Sending SysEx", "bytes", byteSliceToHexLiteral(value))
	return ep.device.outPort.Send(value)
}

func (ep *sysEx) SetSilent(value []byte) error {
	return ep.device.outPort.Send(value)
}

type sysExMatch struct {
	pattern  []byte
	device   *MidiDevice
	callback func([]byte) error
}

func (ep *sysExMatch) Bind(callback func([]byte) error) func() {
	ep.callback = callback
	ep.device.mu.Lock()
	ep.device.sysex[ep] = struct{}{}
	ep.device.mu.Unlock()
	return func() {
		ep.device.mu.Lock()
		delete(ep.device.sysex, ep)
		ep.device.mu.Unlock()
	}
}

func NewMidiDevice(inPort drivers.In, outPort drivers.Out) *MidiDevice {
	d := &MidiDevice{
		inPort:  inPort,
		outPort: outPort,
		SysEx: &sysEx{
			device: &MidiDevice{},
		},
		cc:         make(map[*cC]struct{}),
		pitchBend:  make(map[*pitchBend]struct{}),
		noteOn:     make(map[*noteOn]struct{}),
		noteOff:    make(map[*noteOff]struct{}),
		aftertouch: make(map[*afterTouch]struct{}),
		sysex:      make(map[*sysExMatch]struct{}),
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
			f.mu.RLock()
			for cc := range f.cc {
				if cc.channel == channel && cc.controller == control {
					if err := cc.callback(value); err != nil {
						midiInLog.Error("failed to process Control Change:", err)
					}
				}
			}
			f.mu.RUnlock()
		case midi.PitchBendMsg:
			var channel uint8
			var relative int16
			var absolute uint16
			if ok := msg.GetPitchBend(&channel, &relative, &absolute); !ok {
				midiInLog.Error("failed to parse Pitch Bend message:", err)
				return
			}
			midiInLog.Debug("received Pitch Bend message", "channel", channel, "absolute", absolute, "timestamp", timestampms)
			f.mu.RLock()
			for pitchbend := range f.pitchBend {
				if pitchbend.channel == channel {
					if err := pitchbend.callback(absolute); err != nil {
						midiInLog.Error("failed to process Pitch Bend:", err)
					}
				}
			}
			f.mu.RUnlock()
		case midi.NoteOnMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOn(&channel, &key, &velocity); !ok {
				midiInLog.Error("failed to parse Note On message:", err)
				return
			}
			midiInLog.Debug("received Note On message", "channel", channel, "key", key, "velocity", velocity, "timestamp", timestampms)
			f.mu.RLock()
			for note := range f.noteOn {
				if note.key == key && note.channel == channel {
					if err := note.callback(velocity); err != nil {
						midiInLog.Error("failed to process Note On:", err)
					}
				}
			}
			f.mu.RUnlock()
		case midi.NoteOffMsg:
			var channel, key, velocity uint8
			if ok := msg.GetNoteOff(&channel, &key, &velocity); !ok {
				midiInLog.Error("failed to parse Note Off message:", err)
				return
			}
			midiInLog.Debug("received Note Off message", "channel", channel, "key", key, "velocity", velocity, "timestamp", timestampms)
			f.mu.RLock()
			for note := range f.noteOff {
				if note.key == key && note.channel == channel {
					if err := note.callback(); err != nil {
						midiInLog.Error("failed to process Note Off:", err)
					}
				}
			}
			f.mu.RUnlock()
		case midi.AfterTouchMsg:
			var channel, pressure uint8
			if ok := msg.GetAfterTouch(&channel, &pressure); !ok {
				midiInLog.Error("failed to parse After Touch message:", err)
				return
			}
			midiInLog.Debug("received After Touch message", "channel", channel, "pressure", pressure, "timestamp", timestampms)
			f.mu.RLock()
			for aftertouch := range f.aftertouch {
				if aftertouch.channel == channel {
					if err := aftertouch.callback(pressure); err != nil {
						midiInLog.Error("failed to process After Touch:", err)
					}
				}
			}
			f.mu.RUnlock()
		case midi.SysExMsg:
			var data []byte
			if ok := msg.GetSysEx(&data); !ok {
				midiInLog.Error("failed to parse SysEx message:", err)
				return
			}
			midiInLog.Debug("received SysEx message", "data", data, "timestamp", timestampms)
			f.mu.RLock()
			for sysex := range f.sysex {
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
			f.mu.RUnlock()
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
