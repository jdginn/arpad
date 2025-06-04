package devicestesting

import (
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
	handlers map[string]func(*osc.Message)
}

func (m *MockOscDispatcher) AddMsgHandler(addr string, handler func(*osc.Message)) {
	m.handlers[addr] = handler
}

func (m *MockOscDispatcher) SimulateMessage(addr string, args ...interface{}) {
	if handler, ok := m.handlers[addr]; ok {
		msg := osc.NewMessage(addr, args...)
		handler(msg)
	}
}

type TestOscDevice struct {
	*devices.OscDevice
	mockClient     *MockOscClient
	mockServer     *MockOscServer
	mockDispatcher *MockOscDispatcher
	tracker        *CallbackTracker
}

func NewTestOscDevice(t *testing.T) *TestOscDevice {
	mockClient := &MockOscClient{}
	mockServer := &MockOscServer{}
	mockDispatcher := &MockOscDispatcher{
		handlers: make(map[string]func(*osc.Message)),
	}

	device := devices.NewOscDevice(mockClient, mockServer, mockDispatcher)

	return &TestOscDevice{
		OscDevice:      device,
		mockClient:     mockClient,
		mockServer:     mockServer,
		mockDispatcher: mockDispatcher,
		tracker:        NewCallbackTracker(t),
	}
}

// Test helper methods
func (d *TestOscDevice) SimulateMessage(addr string, args ...interface{}) {
	d.mockDispatcher.SimulateMessage(addr, args...)
}

func (d *TestOscDevice) GetSentMessages() []*osc.Message {
	return d.mockClient.sentMessages
}

// Bindings pre-wrapped with callback tracking
func (d *TestOscDevice) BindInt(addr string, callback func(int64) error) {
	d.OscDevice.BindInt(addr, WrapCallback(d.tracker, callback))
}

func (d *TestOscDevice) BindFloat(addr string, callback func(float64) error) {
	d.OscDevice.BindFloat(addr, WrapCallback(d.tracker, callback))
}

func (d *TestOscDevice) BindString(addr string, callback func(string) error) {
	d.OscDevice.BindString(addr, WrapCallback(d.tracker, callback))
}

func (d *TestOscDevice) BindBool(addr string, callback func(bool) error) {
	d.OscDevice.BindBool(addr, WrapCallback(d.tracker, callback))
}
