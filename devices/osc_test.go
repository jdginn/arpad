package devices_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/jdginn/arpad/devices/devicestesting"
	devtest "github.com/jdginn/arpad/devices/devicestesting"
	"github.com/stretchr/testify/assert"
)

func TestOscDevice(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name          string
		setupBindings func(*devtest.TestOscDevice)
		messages      []struct {
			addr string
			args []interface{}
		}
		validateState func(*devtest.TestOscDevice)
	}{
		{
			name: "int binding handles various numeric types",
			setupBindings: func(d *devtest.TestOscDevice) {
				var receivedValues []int64
				d.BindInt("/test/int", func(val int64) error {
					receivedValues = append(receivedValues, val)
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/int", []interface{}{42}},            // int
				{"/test/int", []interface{}{float64(42.0)}}, // float64
				{"/test/int", []interface{}{"42"}},          // string
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(3, "callback should be called for each message")
			},
		},
		{
			name: "float binding handles various float types",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindFloat("/test/float", func(val float64) error {
					assert.InDelta(42.5, val, 0.001, "incorrect float value")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/float", []interface{}{float64(42.5)}}, // float64
				{"/test/float", []interface{}{float32(42.5)}}, // float32
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "callback should be called for each message")
			},
		},
		{
			name: "float binding handles int",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindFloat("/test/float", func(val float64) error {
					assert.InDelta(42.0, val, 0.001, "incorrect float value")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/float", []interface{}{42}}, // int
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(1, "callback should be called for each message")
			},
		},
		{
			name: "string binding handles type conversions",
			setupBindings: func(d *devtest.TestOscDevice) {
				var receivedValues []string
				d.BindString("/test/string", func(val string) error {
					receivedValues = append(receivedValues, val)
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/string", []interface{}{"test"}},        // string
				{"/test/string", []interface{}{42}},            // int
				{"/test/string", []interface{}{float64(42.5)}}, // float64
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(3, "callback should be called for each message")
			},
		},
		{
			name: "bool binding handles truthy/falsy values",
			setupBindings: func(d *devtest.TestOscDevice) {
				var receivedValues []bool
				d.BindBool("/test/bool", func(val bool) error {
					receivedValues = append(receivedValues, val)
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/bool", []interface{}{true}},         // bool true
				{"/test/bool", []interface{}{1}},            // int truthy
				{"/test/bool", []interface{}{0}},            // int falsy
				{"/test/bool", []interface{}{"true"}},       // string true
				{"/test/bool", []interface{}{float64(1.0)}}, // float truthy
				{"/test/bool", []interface{}{float64(0.0)}}, // float falsy
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(6, "callback should be called for each message")
			},
		},
		{
			name: "multiple bindings on same address are all called",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/multi", func(val int64) error {
					assert.Equal(int64(42), val, "incorrect value in first binding")
					return nil
				})
				d.BindInt("/test/multi", func(val int64) error {
					assert.Equal(int64(42), val, "incorrect value in second binding")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/multi", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "both callbacks should be executed")
			},
		},
		{
			name: "message to wrong address does not trigger callback",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/correct", func(val int64) error {
					assert.Fail("callback should not be called for wrong address")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/wrong", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(0, "callback should not be called for wrong address")
			},
		},
		{
			name: "multiple bindings of different types on different addresses",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/int", func(val int64) error {
					assert.Equal(int64(42), val, "incorrect int value")
					return nil
				})
				d.BindFloat("/test/float", func(val float64) error {
					assert.InDelta(42.5, val, 0.001, "incorrect float value")
					return nil
				})
				d.BindString("/test/string", func(val string) error {
					assert.Equal("test", val, "incorrect string value")
					return nil
				})
				d.BindBool("/test/bool", func(val bool) error {
					assert.True(val, "incorrect bool value")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/int", []interface{}{42}},
				{"/test/float", []interface{}{42.5}},
				{"/test/string", []interface{}{"test"}},
				{"/test/bool", []interface{}{true}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(4, "all callbacks should be executed")
			},
		},
		{
			name: "handlers are called in registration order",
			setupBindings: func(d *devtest.TestOscDevice) {
				sequence := make([]int, 0)

				d.BindInt("/test/sequence", func(val int64) error {
					sequence = append(sequence, 1)
					assert.Equal([]int{1}, sequence, "first handler should be called first")
					return nil
				})
				d.BindInt("/test/sequence", func(val int64) error {
					sequence = append(sequence, 2)
					assert.Equal([]int{1, 2}, sequence, "second handler should be called second")
					return nil
				})
				d.BindInt("/test/sequence", func(val int64) error {
					sequence = append(sequence, 3)
					assert.Equal([]int{1, 2, 3}, sequence, "third handler should be called third")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/sequence", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(3, "all three handlers should be called")
			},
		},
		{
			name: "error in handler does not prevent subsequent handlers",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/errors", func(val int64) error {
					return fmt.Errorf("intentional error")
				})
				d.BindInt("/test/errors", func(val int64) error {
					return nil // should still be called
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/errors", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "both handlers should be called despite error")
			},
		},
		{
			name: "handlers with different types on same address",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/multi", func(val int64) error {
					assert.Equal(int64(42), val)
					return nil
				})
				d.BindFloat("/test/multi", func(val float64) error {
					assert.Equal(float64(42), val)
					return nil
				})
				d.BindString("/test/multi", func(val string) error {
					assert.Equal("42", val)
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/multi", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCallbackCalled(0, 1, "all type handlers should be called")
				d.Tracker.AssertCallbackCalled(1, 1, "all type handlers should be called")
				d.Tracker.AssertCallbackCalled(2, 1, "all type handlers should be called")
			},
		},
		{
			name: "binding cleanup and reuse",
			setupBindings: func(d *devtest.TestOscDevice) {
				// Add initial binding
				d.BindInt("/test/reuse", func(val int64) error {
					return nil
				})

				// Replace with new binding
				d.BindInt("/test/reuse", func(val int64) error {
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/reuse", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "both original and new binding should be called")
			},
		},
		{
			name: "message ordering is preserved with multiple addresses",
			setupBindings: func(d *devtest.TestOscDevice) {
				sequence := make([]string, 0)

				d.BindInt("/test/first", func(val int64) error {
					sequence = append(sequence, "first")
					return nil
				})
				d.BindInt("/test/second", func(val int64) error {
					sequence = append(sequence, "second")
					assert.Equal([]string{"first", "second"}, sequence, "messages should be processed in order")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/first", []interface{}{1}},
				{"/test/second", []interface{}{2}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "both handlers should be called in order")
			},
		},
		{
			name: "large number of bindings handled correctly",
			setupBindings: func(d *devtest.TestOscDevice) {
				for i := 0; i < 100; i++ {
					address := fmt.Sprintf("/test/many/%d", i)
					d.BindInt(address, func(val int64) error {
						return nil
					})
				}
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/many/0", []interface{}{42}},
				{"/test/many/50", []interface{}{42}},
				{"/test/many/99", []interface{}{42}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(3, "handlers for specific addresses should be called")
			},
		},
		{
			name: "wildcard address patterns",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/*", func(val int64) error {
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/one", []interface{}{1}},
				{"/test/two", []interface{}{2}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				// Note: This test might need modification depending on whether your OSC implementation
				// supports address patterns. If it doesn't, this test should verify that exact
				// matches are required.
				d.Tracker.AssertCalled(0, "wildcard patterns should not match if not supported")
			},
		},
		// {
		// 	name: "empty message arguments",
		// 	setupBindings: func(d *devtest.TestOscDevice) {
		// 		d.BindInt("/test/empty", func(val int64) error {
		// 			assert.Fail("callback should not be called with empty arguments")
		// 			return nil
		// 		})
		// 	},
		// 	messages: []struct {
		// 		addr string
		// 		args []interface{}
		// 	}{
		// 		{"/test/empty", []interface{}{}},
		// 	},
		// 	validateState: func(d *devtest.TestOscDevice) {
		// 		d.Tracker.AssertCalled(0, "callback should not be called for empty arguments")
		// 	},
		// },
		{
			name: "nil argument handling",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindString("/test/nil", func(val string) error {
					assert.Equal("", val, "nil should be converted to empty string")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/nil", []interface{}{nil}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(1, "callback should handle nil argument")
			},
		},
		{
			name: "multiple arguments ignored",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/multi-args", func(val int64) error {
					assert.Equal(int64(42), val, "only first argument should be used")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/multi-args", []interface{}{42, 43, 44}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(1, "callback should be called with first argument only")
			},
		},
		{
			name: "very large numbers",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/large", func(val int64) error {
					assert.Equal(int64(math.MaxInt64), val, "should handle maximum int64")
					return nil
				})
				d.BindFloat("/test/large-float", func(val float64) error {
					assert.Equal(math.MaxFloat64, val, "should handle maximum float64")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/large", []interface{}{int64(math.MaxInt64)}},
				{"/test/large-float", []interface{}{math.MaxFloat64}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(2, "callbacks should handle maximum values")
			},
		},
		{
			name: "rapid message sequence",
			setupBindings: func(d *devtest.TestOscDevice) {
				counter := 0
				d.BindInt("/test/rapid", func(val int64) error {
					counter++
					assert.Equal(int64(counter), val, "messages should be processed in order")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/rapid", []interface{}{1}},
				{"/test/rapid", []interface{}{2}},
				{"/test/rapid", []interface{}{3}},
				{"/test/rapid", []interface{}{4}},
				{"/test/rapid", []interface{}{5}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(5, "all rapid messages should be processed")
			},
		},
		{
			name: "recursive message generation",
			setupBindings: func(d *devtest.TestOscDevice) {
				recursionCount := 0
				maxRecursion := 3

				d.BindInt("/test/recursive", func(val int64) error {
					if recursionCount < maxRecursion {
						recursionCount++
						// Simulate sending another message from within the callback
						d.SimulateMessage("/test/recursive", recursionCount)
					}
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/recursive", []interface{}{0}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(4, "should handle recursive message generation")
			},
		},
		{
			name: "zero values for all types",
			setupBindings: func(d *devtest.TestOscDevice) {
				d.BindInt("/test/zero", func(val int64) error {
					assert.Equal(int64(0), val, "should handle zero int")
					return nil
				})
				d.BindFloat("/test/zero", func(val float64) error {
					assert.Equal(0.0, val, "should handle zero float")
					return nil
				})
				d.BindString("/test/zero", func(val string) error {
					assert.Equal("0", val, "should handle empty string")
					return nil
				})
				d.BindBool("/test/zero", func(val bool) error {
					assert.False(val, "should handle false bool")
					return nil
				})
			},
			messages: []struct {
				addr string
				args []interface{}
			}{
				{"/test/zero", []interface{}{0}},
			},
			validateState: func(d *devtest.TestOscDevice) {
				d.Tracker.AssertCalled(4, "all zero value handlers should be called")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := devtest.NewTestOscDevice(t)

			// Setup bindings
			tt.setupBindings(device)

			// Start the device
			go func() {
				device.Run()
			}()

			// Allow time for server to start
			time.Sleep(50 * time.Microsecond)

			// Send all messages
			for _, msg := range tt.messages {
				device.SimulateMessage(msg.addr, msg.args...)
				// Small delay between messages
				time.Sleep(50 * time.Microsecond)
			}

			// Allow time for processing
			time.Sleep(50 * time.Microsecond)

			// Validate the results
			tt.validateState(device)
		})
	}
}

func TestOscDevice_ResourceManagement(t *testing.T) {
	device := devicestesting.NewTestOscDevice(t)

	t.Run("MultipleBindings", func(t *testing.T) {
		// Test binding multiple handlers to different addresses
		handlers := make(map[string]bool)
		for i := 0; i < 100; i++ {
			addr := fmt.Sprintf("/test/binding/%d", i)
			handlers[addr] = false

			device.BindInt(addr, func(val int64) error {
				handlers[addr] = true
				return nil
			})
		}

		// Verify all handlers can be triggered
		for addr := range handlers {
			device.SimulateMessage(addr, int32(42))
			if !handlers[addr] {
				t.Errorf("Handler for %s was not called", addr)
			}
		}
	})

	t.Run("ServerLifecycle", func(t *testing.T) {
		// Test server startup
		if err := device.Run(); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	})
}

func TestOscDevice_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	device := devicestesting.NewTestOscDevice(t)
	messageCount := 10000

	t.Run("HighThroughput", func(t *testing.T) {
		received := make(chan bool, messageCount)

		device.BindInt("/test/perf", func(val int64) error {
			received <- true
			return nil
		})

		start := time.Now()

		// Send many messages rapidly
		for i := 0; i < messageCount; i++ {
			device.SimulateMessage("/test/perf", int32(i))
		}

		// Wait for all messages to be processed
		for i := 0; i < messageCount; i++ {
			select {
			case <-received:
				// Expected
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for messages to be processed")
			}
		}

		duration := time.Since(start)
		t.Logf("Processed %d messages in %v (%v per message)",
			messageCount, duration, duration/time.Duration(messageCount))
	})
}
