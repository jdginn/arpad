package devices_test

import (
	"fmt"
	"testing"
	"time"

	dev "github.com/jdginn/arpad/devices"
	devtest "github.com/jdginn/arpad/devices/devicestesting"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
)

func TestMidiDevice(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name          string
		setupBindings func(*devtest.MidiDevice)
		inputMessages []midi.Message
		validateState func(*devtest.MidiDevice, *devtest.MockMIDIPort)
	}{
		{
			name: "cc message on wrong channel does not trigger callback",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					assert.Fail("callback should not be called for wrong channel")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(2, 7, 64), // Wrong channel
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertNotCalled("callback should not be called for wrong channel")
			},
		},
		{
			name: "cc message on wrong controller number does not trigger callback",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					assert.Fail("callback should not be called for wrong controller")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 8, 64), // Wrong controller number
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertNotCalled("callback should not be called for wrong controller")
			},
		},
		{
			name: "multiple matching messages trigger callback multiple times",
			setupBindings: func(d *devtest.MidiDevice) {
				callCount := 0
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					callCount++
					assert.Equal(uint8(64), args.Value, fmt.Sprintf("incorrect value on call %d", callCount))
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),
				midi.ControlChange(1, 7, 64),
				midi.ControlChange(1, 7, 64),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "callback should be called exactly 3 times")
			},
		},
		{
			name: "mixed matching and non-matching messages only trigger callback for matches",
			setupBindings: func(d *devtest.MidiDevice) {
				callCount := 0
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					callCount++
					assert.Equal(uint8(64), args.Value, fmt.Sprintf("incorrect value on call %d", callCount))
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64), // Should trigger
				midi.ControlChange(2, 7, 64), // Wrong channel
				midi.ControlChange(1, 8, 64), // Wrong controller
				midi.ControlChange(1, 7, 64), // Should trigger
				midi.ProgramChange(1, 5),     // Wrong message type
				midi.ControlChange(1, 7, 64), // Should trigger
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "callback should be called exactly 3 times")
			},
		},
		{
			name: "multiple bindings work independently",
			setupBindings: func(d *devtest.MidiDevice) {
				// First binding - CC messages
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					assert.Equal(uint8(64), args.Value, "incorrect CC value")
					return nil
				})

				// Second binding - Note messages
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					assert.True(on, "note should be on")
					return nil
				})

				// Third binding - Different CC messages
				d.BindCC(dev.PathCC{Channel: 2, Controller: 8}, func(args dev.ArgsCC) error {
					assert.Equal(uint8(100), args.Value, "incorrect CC value on second binding")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),  // Should trigger first CC binding
				midi.NoteOn(1, 60, 100),       // Should trigger note binding
				midi.ControlChange(2, 8, 100), // Should trigger second CC binding
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "all three callbacks should be called once each")
			},
		},
		{
			name: "note on and off messages are handled correctly",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					if on {
						assert.True(on, "note should be on")
					} else {
						assert.False(on, "note should be off")
					}
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.NoteOn(1, 60, 100), // Should trigger with on=true
				midi.NoteOff(1, 60),     // Should trigger with on=false
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(2, "callback should be called for both note on and note off")
			},
		},
		{
			name: "pitch bend messages are handled with correct values",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindPitchBend(dev.PathPitchBend{Channel: 1}, func(args dev.ArgsPitchBend) error {
					assert.Equal(int16(100), args.Relative, "incorrect relative value")
					assert.Equal(uint16(8192+100), args.Absolute, "incorrect absolute value")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.Pitchbend(1, 100),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalledOnce()
			},
		},
		{
			name: "pitch bend messages are handled with correct values",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindPitchBend(dev.PathPitchBend{Channel: 1}, func(args dev.ArgsPitchBend) error {
					assert.Equal(int16(0), args.Relative, "incorrect relative value")
					assert.Equal(uint16(8192), args.Absolute, "incorrect absolute value")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.Pitchbend(1, 0),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalledOnce()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, mockPort := devtest.NewTestMidiDevice(t)

			// Setup bindings
			tt.setupBindings(device)

			// Start listening in a goroutine
			go func() {
				device.Run()
			}()

			// Allow time for goroutine to start
			time.Sleep(50 * time.Millisecond)

			// Send all input messages
			for _, msg := range tt.inputMessages {
				mockPort.SimulateReceive(msg)
				// Small delay between messages to ensure proper processing
				time.Sleep(10 * time.Millisecond)
			}

			// Allow time for processing
			time.Sleep(50 * time.Millisecond)

			// Validate the results
			tt.validateState(device, mockPort)
		})
	}
}
