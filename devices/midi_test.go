package devices

import (
	"fmt"
	"testing"
	"time"

	h "github.com/jdginn/arpad/devices/midi_harness"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
)

func TestMidiDevice(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name          string
		setupBindings func(*MidiDevice) map[string]any
		inputMessage  midi.Message
		validateState func(map[string]any, *MidiDevice, *h.MockMIDIPort)
	}{
		{
			name: "control change message triggers callback",
			setupBindings: func(dev *MidiDevice) map[string]any {
				callCount := 0
				dev.BindCC(PathCC{Channel: 1, Controller: 7}, func(args ArgsCC) error {
					fmt.Println("Incrementing callCount!")
					callCount++
					assert.Equal(uint8(64), args.Value)
					return nil
				})
				return map[string]any{"calls": callCount}
			},
			inputMessage: midi.ControlChange(1, 7, 64),
			validateState: func(locals map[string]any, dev *MidiDevice, port *h.MockMIDIPort) {
				assert.Equal(1, locals["calls"])
			},
		},
		{
			name: "pitch bend message triggers callback",
			setupBindings: func(dev *MidiDevice) map[string]any {
				callCount := 0
				dev.BindPitchBend(PathPitchBend{Channel: 1}, func(args ArgsPitchBend) error {
					callCount++
					assert.Equal(100, args.Relative)
					assert.Equal(uint16(8192), args.Absolute)
					return nil
				})
				return map[string]any{}
			},
			inputMessage: midi.Pitchbend(1, 100),
			validateState: func(locals map[string]any, dev *MidiDevice, port *h.MockMIDIPort) {
				// Additional state validation if needed
			},
		},
		{
			name: "note on/off messages trigger callbacks",
			setupBindings: func(dev *MidiDevice) map[string]any {
				// noteState := false
				dev.BindNote(PathNote{Channel: 1, Key: 60}, func(on bool) error {
					// noteState = true
					return nil
				})
				return map[string]any{}
			},
			inputMessage: midi.NoteOn(1, 60, 100),
			validateState: func(locals map[string]any, dev *MidiDevice, port *h.MockMIDIPort) {
				// Simulate note off and verify state
				port.SimulateReceive(midi.NoteOff(1, 60))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort := h.NewMockMIDIPort()
			device := NewMidiDevice(mockPort, mockPort)

			// Setup bindings
			locals := tt.setupBindings(device)

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
			tt.validateState(locals, device, mockPort)
		})
	}
}
