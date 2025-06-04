package devicestesting

import (
	"errors"
	"sync"
	"testing"

	"github.com/jdginn/arpad/devices"
	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

// MockMIDIPort implements both drivers.In and drivers.Out interfaces
type MockMIDIPort struct {
	mu sync.Mutex

	// For tracking sent messages
	sentMessages []midi.Message

	// For simulating received messages
	listeners []func(msg midi.Message, timestampms int32)

	// For testing error conditions
	shouldError bool

	isOpen bool
}

func NewMockMIDIPort() *MockMIDIPort {
	return &MockMIDIPort{
		sentMessages: make([]midi.Message, 0),
		listeners:    make([]func(msg midi.Message, timestampms int32), 0),
	}
}

func (m *MockMIDIPort) Open() error {
	m.mu.Lock()
	m.isOpen = true
	m.mu.Unlock()
	return nil
}

func (m *MockMIDIPort) Close() error {
	m.mu.Lock()
	m.isOpen = false
	m.mu.Unlock()
	return nil
}

func (m *MockMIDIPort) IsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isOpen
}

// Number implements drivers.Out and drivers.In
func (m *MockMIDIPort) Number() int {
	return 0
}

// String implements drivers.Out and drivers.In
func (m *MockMIDIPort) String() string {
	return "MockMIDIPort"
}

func (m *MockMIDIPort) Underlying() interface{} {
	return m
}

// Send implements drivers.Out
func (m *MockMIDIPort) Send(data []byte) error {
	if m.shouldError {
		return errors.New("mock send error")
	}

	m.mu.Lock()
	m.sentMessages = append(m.sentMessages, data)
	m.mu.Unlock()
	return nil
}

// SimulateReceive simulates receiving a MIDI message
func (m *MockMIDIPort) SimulateReceive(msg midi.Message) {
	m.mu.Lock()
	listeners := make([]func(msg midi.Message, timestampms int32), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	for _, listener := range listeners {
		listener(msg, 0) // timestamp 0 for simplicity
	}
}

func (m *MockMIDIPort) Listen(onMsg func(msg []byte, milliseconds int32), config drivers.ListenConfig) (stopFn func(), err error) {
	if !m.IsOpen() {
		return nil, errors.New("port not open")
	}

	// Register a listener that converts MIDI messages to bytes
	m.mu.Lock()
	m.listeners = append(m.listeners, func(msg midi.Message, timestampms int32) {
		// Convert the MIDI message to bytes and pass to the onMsg callback
		onMsg(msg, timestampms)
	})
	m.mu.Unlock()

	// Return a stop function that removes this listener
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// Find and remove this specific listener
		for i, l := range m.listeners {
			if l != nil { // We can identify this listener because it's the only one that converts to Raw()
				// Remove the listener by setting to nil
				// (we can't modify the slice while iterating)
				m.listeners[i] = nil
				break
			}
		}

		// Clean up nil listeners
		newListeners := make([]func(msg midi.Message, timestampms int32), 0, len(m.listeners))
		for _, l := range m.listeners {
			if l != nil {
				newListeners = append(newListeners, l)
			}
		}
		m.listeners = newListeners
	}, nil
}

// RegisterListener adds a message listener
func (m *MockMIDIPort) RegisterListener(listener func(msg midi.Message, timestampms int32)) {
	m.mu.Lock()
	m.listeners = append(m.listeners, listener)
	m.mu.Unlock()
}

// GetSentMessages returns all messages that were sent
func (m *MockMIDIPort) GetSentMessages() []midi.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]midi.Message, len(m.sentMessages))
	copy(result, m.sentMessages)
	return result
}

// SetError configures the mock to return errors
func (m *MockMIDIPort) SetError(shouldError bool) {
	m.mu.Lock()
	m.shouldError = shouldError
	m.mu.Unlock()
}

// Bindings pre-wrapped with callback tracking

// MidiDevice wraps a MidiDevice to automatically track all callbacks
type MidiDevice struct {
	*devices.MidiDevice
	Tracker *CallbackTracker
}

// NewTestMidiDevice creates a MidiDevice with a mock port and automatic callback tracking
func NewTestMidiDevice(t *testing.T) (*MidiDevice, *MockMIDIPort) {
	mockPort := NewMockMIDIPort()
	device := devices.NewMidiDevice(mockPort, mockPort)
	return &MidiDevice{
		MidiDevice: device,
		Tracker:    NewCallbackTracker(t),
	}, mockPort
}

// BindCC wraps the original BindCC with automatic callback tracking
func (d *MidiDevice) BindCC(path devices.PathCC, callback func(devices.ArgsCC) error) {
	d.MidiDevice.BindCC(path, WrapCallback(d.Tracker, callback))
}

// BindPitchBend wraps the original BindPitchBend with automatic callback tracking
func (d *MidiDevice) BindPitchBend(path devices.PathPitchBend, callback func(devices.ArgsPitchBend) error) {
	d.MidiDevice.BindPitchBend(path, WrapCallback(d.Tracker, callback))
}

// BindNote wraps the original BindNote with automatic callback tracking
func (d *MidiDevice) BindNote(path devices.PathNote, callback func(bool) error) {
	d.MidiDevice.BindNote(path, WrapCallback(d.Tracker, callback))
}

// BindAfterTouch wraps the original BindAfterTouch with automatic callback tracking
func (d *MidiDevice) BindAfterTouch(path devices.PathAfterTouch, callback func(devices.ArgsAfterTouch) error) {
	d.MidiDevice.BindAfterTouch(path, WrapCallback(d.Tracker, callback))
}
