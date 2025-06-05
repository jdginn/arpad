package devicestesting

import (
	"fmt"
	"sync"
	"testing"
)

type CallbackTracker struct {
	t                 *testing.T
	mu                sync.Mutex
	totalCalls        int
	callsByHandle     map[int]int
	callOrder         []int
	descriptions      map[int]string
	nextID            int
	registrationOrder []int // New field to track registration order
}

func NewCallbackTracker(t *testing.T) *CallbackTracker {
	return &CallbackTracker{
		t:                 t,
		callsByHandle:     make(map[int]int),
		callOrder:         make([]int, 0),
		descriptions:      make(map[int]string),
		registrationOrder: make([]int, 0),
	}
}

func (t *CallbackTracker) RegisterCallback(description string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	handle := t.nextID
	t.nextID++
	t.descriptions[handle] = description
	t.callsByHandle[handle] = 0
	t.registrationOrder = append(t.registrationOrder, handle)
	return handle
}

func (t *CallbackTracker) recordCall(handle int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.totalCalls++
	t.callsByHandle[handle]++
	t.callOrder = append(t.callOrder, handle)
}

// WrapCallback wraps a callback function to track its invocations
// The wrapped function will have the same signature as the original
func WrapCallback[T any](tracker *CallbackTracker, handle int, callback func(T) error) func(T) error {
	return func(val T) error {
		tracker.recordCall(handle)
		return callback(val)
	}
}

// Get a callback handle by its registration order
func (t *CallbackTracker) GetNthBinding(n int) int {
	if n >= len(t.registrationOrder) {
		t.t.Fatalf("GetNthBinding: index %d out of range (only %d bindings registered)",
			n, len(t.registrationOrder))
	}
	return t.registrationOrder[n]
}

// Assert callbacks were called in their registration order
func (t *CallbackTracker) AssertBindingsCalledInRegistrationOrder(msgAndArgs ...any) {
	msg := formatMessage("callbacks not called in registration order", msgAndArgs...)
	t.AssertCallOrder(t.registrationOrder, msg)
}

// Assert total number of calls matches expected
func (t *CallbackTracker) AssertCalled(n int, msgAndArgs ...any) {
	if t.totalCalls != n {
		msg := formatMessage("unexpected number of total calls", msgAndArgs...)
		t.t.Errorf("%s: expected %d calls, got %d", msg, n, t.totalCalls)
	}
}

func (t *CallbackTracker) AssertCallbackCalled(handle int, expectedCalls int, msgAndArgs ...any) {
	if actual := t.callsByHandle[handle]; actual != expectedCalls {
		msg := formatMessage("unexpected number of calls", msgAndArgs...)
		desc := t.descriptions[handle]
		t.t.Errorf("%s: callback %d (%s): expected %d calls, got %d",
			msg, handle, desc, expectedCalls, actual)
	}
}

func (t *CallbackTracker) AssertCallOrder(handles []int, msgAndArgs ...any) {
	msg := formatMessage("incorrect call order", msgAndArgs...)
	if len(handles) != len(t.callOrder) {
		t.t.Errorf("%s: expected %d calls, got %d",
			msg, len(handles), len(t.callOrder))
		return
	}
	for i, expected := range handles {
		if actual := t.callOrder[i]; actual != expected {
			t.t.Errorf("%s: at position %d: expected callback %d (%s), got %d (%s)",
				msg, i, expected, t.descriptions[expected],
				actual, t.descriptions[actual])
		}
	}
}

// AssertNotCalled asserts that the callback was never called
func (ct *CallbackTracker) AssertNotCalled(handle int, msgAndArgs ...any) {
	ct.AssertCallbackCalled(handle, 0, msgAndArgs...)
}

// Helper function to format messages consistently
func formatMessage(defaultMsg string, msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return defaultMsg
	}

	// If only one argument and it's a string, use it directly
	if len(msgAndArgs) == 1 {
		if msg, ok := msgAndArgs[0].(string); ok {
			return msg
		}
	}

	// Otherwise format using fmt.Sprintf
	return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
}
