package devicestesting

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CallbackHandle represents an opaque identifier for a registered callback
type CallbackHandle struct {
	id int
}

type CallbackTracker struct {
	t             *testing.T
	mu            sync.Mutex
	totalCalls    int
	callsByHandle map[CallbackHandle]int
	callOrder     []CallbackHandle
	descriptions  map[CallbackHandle]string
	nextID        int
}

// NewCallbackTracker creates a new CallbackTracker for use in tests
func NewCallbackTracker(t *testing.T) *CallbackTracker {
	return &CallbackTracker{
		t:             t,
		callsByHandle: make(map[CallbackHandle]int),
		callOrder:     make([]CallbackHandle, 0),
		descriptions:  make(map[CallbackHandle]string),
	}
}

func (t *CallbackTracker) RegisterCallback(description string) CallbackHandle {
	t.mu.Lock()
	defer t.mu.Unlock()
	handle := CallbackHandle{id: t.nextID}
	t.nextID++
	t.descriptions[handle] = description
	t.callsByHandle[handle] = 0
	return handle
}

func (t *CallbackTracker) recordCall(handle CallbackHandle) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.totalCalls++
	t.callsByHandle[handle]++
	t.callOrder = append(t.callOrder, handle)
}

// WrapCallback wraps a callback function to track its invocations
// The wrapped function will have the same signature as the original
func WrapCallback[T any](tracker *CallbackTracker, handle CallbackHandle, callback func(T) error) func(T) error {
	return func(val T) error {
		tracker.recordCall(handle)
		return callback(val)
	}
}

// New methods use CallbackHandle instead of raw int
func (t *CallbackTracker) AssertCallbackCalled(handle CallbackHandle, expectedCalls int) {
	if actual := t.callsByHandle[handle]; actual != expectedCalls {
		desc := t.descriptions[handle]
		t.t.Errorf("callback %d (%s): expected %d calls, got %d",
			handle.id, desc, expectedCalls, actual)
	}
}

func (t *CallbackTracker) AssertCallOrder(handles []CallbackHandle) {
	if len(handles) != len(t.callOrder) {
		t.t.Errorf("call order: expected %d calls, got %d",
			len(handles), len(t.callOrder))
		return
	}
	for i, expected := range handles {
		if actual := t.callOrder[i]; actual != expected {
			t.t.Errorf("call order at position %d: expected callback %d (%s), got %d (%s)",
				i, expected.id, t.descriptions[expected],
				actual.id, t.descriptions[actual])
		}
	}
}

// AssertCalled asserts that the callback was called exactly n times
func (ct *CallbackTracker) AssertCalled(expectedCalls int, msg ...any) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	assert.Equal(ct.t, expectedCalls, ct.totalCalls, msg...)
}

// AssertNotCalled asserts that the callback was never called
func (ct *CallbackTracker) AssertNotCalled(msg ...any) {
	ct.AssertCalled(0, msg)
}
