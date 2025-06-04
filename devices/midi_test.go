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
		inputMessage  midi.Message
		validateState func(*devtest.MidiDevice, *devtest.MockMIDIPort)
	}{
		{
			name: "control change message triggers callback",
			setupBindings: func(d *devtest.MidiDevice) {
				d.BindCC(dev.PathCC{Channel: 1, Controller: 7}, func(args dev.ArgsCC) error {
					fmt.Println("Incrementing callCount!")
					assert.Equal(uint8(64), args.Value)
					return nil
				})
			},
			inputMessage: midi.ControlChange(1, 7, 64),
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalledOnce()
			},
		},
		{
			name: "pitch bend message triggers callback",
			setupBindings: func(d *devtest.MidiDevice) {
				callCount := 0
				d.BindPitchBend(dev.PathPitchBend{Channel: 1}, func(args dev.ArgsPitchBend) error {
					callCount++
					assert.EqualValues(100, args.Relative)
					assert.EqualValues(uint16(8192), args.Absolute)
					return nil
				})
			},
			inputMessage: midi.Pitchbend(1, 100),
			validateState: func(d *devtest.MidiDevice, port *devtest.MockMIDIPort) {
				d.Tracker.AssertCalledOnce()
			},
		},
		{
			name: "note on/off messages trigger callbacks",
			setupBindings: func(d *devtest.MidiDevice) {
				// noteState := false
				d.BindNote(dev.PathNote{Channel: 1, Key: 60}, func(on bool) error {
					// noteState = true
					return nil
				})
			},
			inputMessage: midi.NoteOn(1, 60, 100),
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

			// Simulate receiving the message
			mockPort.SimulateReceive(tt.inputMessage)

			// Allow time for processing
			time.Sleep(50 * time.Millisecond)

			// Validate the results
			tt.validateState(device, mockPort)
		})
	}
}
