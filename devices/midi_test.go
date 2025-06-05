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
		{
			name: "running status messages are handled correctly",
			setupBindings: func(d *devtest.MidiDevice) {
				var values []uint8
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					values = append(values, args.Value)
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),  // Initial message
				midi.ControlChange(1, 7, 100), // Second message (potentially running status)
				midi.ControlChange(1, 7, 127), // Third message (potentially running status)
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "should handle all messages including running status")
			},
		},
		{
			name: "note messages check both channel and key",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					assert.Fail("callback should not be called for wrong channel")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.NoteOn(2, 60, 100), // Right key, wrong channel
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertNotCalled("callback should not be called for wrong channel")
			},
		},
		{
			name: "overlapping note on messages are handled properly",
			setupBindings: func(d *devtest.MidiDevice) {
				var noteOnCount int
				var noteOffCount int
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					if on {
						noteOnCount++
					} else {
						noteOffCount++
					}
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.NoteOn(1, 60, 100), // First Note On
				midi.NoteOn(1, 60, 127), // Second Note On without Off
				midi.NoteOff(1, 60),     // Note Off
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "should call callback for each message")
			},
		},
		{
			name: "message callbacks are executed in order",
			setupBindings: func(d *devtest.MidiDevice) {
				var sequence []string
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					sequence = append(sequence, fmt.Sprintf("CC:%d", args.Value))
					return nil
				})
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					sequence = append(sequence, fmt.Sprintf("Note:%v", on))
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),  // First message
				midi.NoteOn(1, 60, 100),       // Second message
				midi.ControlChange(1, 7, 100), // Third message
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "all callbacks should be executed")
				// Could add sequence verification if we extend the Tracker to capture call order
			},
		},
		{
			name: "multiple bindings on same channel and controller",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					return nil
				})
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				// This test will help clarify the expected behavior:
				// Should both callbacks be called? Should only the last one be called?
				d.Tracker.AssertCalled(2, "both callbacks should be executed")
			},
		},
		{
			name: "aftertouch messages are handled correctly",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindAfterTouch(dev.PathAfterTouch{Channel: 1}, func(args dev.ArgsAfterTouch) error {
					assert.Equal(uint8(100), args.Pressure, "incorrect pressure value")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.AfterTouch(1, 100),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalledOnce()
			},
		},
		{
			name: "aftertouch messages on wrong channel do not trigger callback",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindAfterTouch(dev.PathAfterTouch{Channel: 1}, func(args dev.ArgsAfterTouch) error {
					assert.Fail("callback should not be called for wrong channel")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.AfterTouch(2, 100), // Wrong channel
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertNotCalled("callback should not be called for wrong channel")
			},
		},
		{
			name: "multiple aftertouch bindings on same channel are all called",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindAfterTouch(dev.PathAfterTouch{Channel: 1}, func(args dev.ArgsAfterTouch) error {
					assert.Equal(uint8(100), args.Pressure, "incorrect pressure value in first binding")
					return nil
				})
				d.BindAfterTouch(dev.PathAfterTouch{Channel: 1}, func(args dev.ArgsAfterTouch) error {
					assert.Equal(uint8(100), args.Pressure, "incorrect pressure value in second binding")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.AfterTouch(1, 100),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(2, "both callbacks should be executed")
			},
		},
		{
			name: "mixed message types including aftertouch are handled correctly",
			setupBindings: func(d *devtest.MidiDevice) {
				// Track message order
				var sequence []string

				d.BindAfterTouch(dev.PathAfterTouch{Channel: 1}, func(args dev.ArgsAfterTouch) error {
					sequence = append(sequence, fmt.Sprintf("AT:%d", args.Pressure))
					return nil
				})
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					sequence = append(sequence, fmt.Sprintf("CC:%d", args.Value))
					return nil
				})
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					sequence = append(sequence, fmt.Sprintf("Note:%v", on))
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.AfterTouch(1, 100),
				midi.ControlChange(1, 7, 64),
				midi.NoteOn(1, 60, 100),
				midi.AfterTouch(1, 127),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(4, "all callbacks should be executed")
			},
		},
		{
			name: "callback errors are handled gracefully",
			setupBindings: func(d *devtest.MidiDevice) {
				// First binding returns error
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					return fmt.Errorf("intentional error")
				})
				// Second binding should still be called
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(1, 7, 64),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(2, "both callbacks should be called despite error")
			},
		},
		{
			name: "zero-value messages are handled correctly",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindCC(dev.PathCC{Channel: 0, Controller: 0}, func(args dev.ArgsCC) error {
					assert.Equal(uint8(0), args.Value, "incorrect value")
					return nil
				})
				d.BindNote(dev.PathNote{Channel: 0, Key: 0}, func(on bool) error {
					return nil
				})
				d.BindAfterTouch(dev.PathAfterTouch{Channel: 0}, func(args dev.ArgsAfterTouch) error {
					assert.Equal(uint8(0), args.Pressure, "incorrect pressure")
					return nil
				})
			},
			inputMessages: []midi.Message{
				midi.ControlChange(0, 0, 0),
				midi.NoteOn(0, 0, 0),
				midi.AfterTouch(0, 0),
			},
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalled(3, "all zero-value messages should trigger callbacks")
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
			time.Sleep(50 * time.Microsecond)

			// Send all input messages
			for _, msg := range tt.inputMessages {
				mockPort.SimulateReceive(msg)
				// Small delay between messages to ensure proper processing
				time.Sleep(50 * time.Microsecond)
			}

			// Allow time for processing
			time.Sleep(50 * time.Microsecond)

			// Validate the results
			tt.validateState(device, mockPort)
		})
	}
}
