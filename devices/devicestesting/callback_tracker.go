package devicestesting

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CallbackTracker helps track and verify callback invocations in tests
type CallbackTracker struct {
	mu       sync.Mutex
	calls    int
	lastArgs []interface{}
	t        *testing.T
}

// NewCallbackTracker creates a new CallbackTracker for use in tests
func NewCallbackTracker(t *testing.T) *CallbackTracker {
	return &CallbackTracker{
		t:        t,
		lastArgs: make([]interface{}, 0),
	}
}

// WrapCallback wraps a callback function to track its invocations
// The wrapped function will have the same signature as the original
func WrapCallback[T any](ct *CallbackTracker, callback func(T) error) func(T) error {
	return func(arg T) error {
		ct.mu.Lock()
		ct.calls++
		ct.lastArgs = append(ct.lastArgs, arg)
		ct.mu.Unlock()

		if callback != nil {
			return callback(arg)
		}
		return nil
	}
}

// AssertCalled asserts that the callback was called exactly n times
func (ct *CallbackTracker) AssertCalled(expectedCalls int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	assert.Equal(ct.t, expectedCalls, ct.calls,
		"callback should have been called %d times, got %d", expectedCalls, ct.calls)
}

// AssertCalledOnce asserts that the callback was called exactly once
func (ct *CallbackTracker) AssertCalledOnce() {
	ct.AssertCalled(1)
}

// AssertNotCalled asserts that the callback was never called
func (ct *CallbackTracker) AssertNotCalled() {
	ct.AssertCalled(0)
}

// GetLastArgs returns the arguments from the last callback invocation
// Returns nil if never called
func (ct *CallbackTracker) GetLastArgs() []interface{} {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if len(ct.lastArgs) == 0 {
		return nil
	}
	return ct.lastArgs[len(ct.lastArgs)-1:]
}

// Reset resets the call counter and args history
func (ct *CallbackTracker) Reset() {
	ct.mu.Lock()
	ct.calls = 0
	ct.lastArgs = make([]interface{}, 0)
	ct.mu.Unlock()
}
