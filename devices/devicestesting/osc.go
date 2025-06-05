package devicestesting

import (
	"fmt"
	"testing"

	"github.com/hypebeast/go-osc/osc"
	"github.com/jdginn/arpad/devices"
)

type MockOscClient struct {
	sentMessages []*osc.Message
}

func (m *MockOscClient) Send(msg *osc.Message) error {
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

type MockOscServer struct {
	running bool
}

func (m *MockOscServer) ListenAndServe() error {
	m.running = true
	return nil
}

type MockOscDispatcher struct {
	handlers map[string][]func(*osc.Message) // Change to slice of handlers
}

func (m *MockOscDispatcher) AddMsgHandler(addr string, handler func(*osc.Message)) {
	if m.handlers[addr] == nil {
		m.handlers[addr] = make([]func(*osc.Message), 0)
	}
	m.handlers[addr] = append(m.handlers[addr], handler)
}

func (m *MockOscDispatcher) SimulateMessage(addr string, args ...interface{}) {
	if handlers, ok := m.handlers[addr]; ok {
		msg := osc.NewMessage(addr, args...)
		for _, handler := range handlers { // Call all handlers for this address
			handler(msg)
		}
	}
}

type TestOscDevice struct {
	*devices.OscDevice
	mockClient     *MockOscClient
	mockServer     *MockOscServer
	mockDispatcher *MockOscDispatcher
	Tracker        *CallbackTracker
}

func NewTestOscDevice(t *testing.T) *TestOscDevice {
	mockClient := &MockOscClient{}
	mockServer := &MockOscServer{}
	mockDispatcher := &MockOscDispatcher{
		handlers: make(map[string][]func(*osc.Message)),
	}

	device := devices.NewOscDevice(mockClient, mockServer, mockDispatcher)

	return &TestOscDevice{
		OscDevice:      device,
		mockClient:     mockClient,
		mockServer:     mockServer,
		mockDispatcher: mockDispatcher,
		Tracker:        NewCallbackTracker(t),
	}
}

// Test helper methods
func (d *TestOscDevice) SimulateMessage(addr string, args ...interface{}) {
	d.mockDispatcher.SimulateMessage(addr, args...)
}

func (d *TestOscDevice) GetSentMessages() []*osc.Message {
	return d.mockClient.sentMessages
}

func (d *TestOscDevice) BindInt(addr string, callback func(int64) error) CallbackHandle {
	handle := d.Tracker.RegisterCallback(fmt.Sprintf("int binding for %s", addr))
	d.OscDevice.BindInt(addr, WrapCallback(d.Tracker, handle, callback))
	return handle
}

func (d *TestOscDevice) BindFloat(addr string, callback func(float64) error) {
	handle := d.Tracker.RegisterCallback(fmt.Sprintf("float binding for %s", addr))
	d.OscDevice.BindFloat(addr, WrapCallback(d.Tracker, handle, callback))
}

func (d *TestOscDevice) BindString(addr string, callback func(string) error) {
	handle := d.Tracker.RegisterCallback(fmt.Sprintf("string binding for %s", addr))
	d.OscDevice.BindString(addr, WrapCallback(d.Tracker, handle, callback))
}

func (d *TestOscDevice) BindBool(addr string, callback func(bool) error) {
	handle := d.Tracker.RegisterCallback(fmt.Sprintf("bool binding for %s", addr))
	d.OscDevice.BindBool(addr, WrapCallback(d.Tracker, handle, callback))
}
