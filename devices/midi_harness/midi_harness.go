package midi_harness

import (
	"errors"
	"sync"

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
	return func() {}, nil
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
