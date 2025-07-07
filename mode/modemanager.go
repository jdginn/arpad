package mode

import (
	"errors"
	"reflect"
	"sync"
)

type Mode int

// bindable represents an endpoint that can have a callback bound to it to monitor state changes.
// The callback will be invoked whenever the endpoint's value changes.
type bindable[A any] interface {
	Bind(func(A) error)
}

type setable[T any] interface {
	Set(T) error
}

// registry tracks the last value sent for each (mode, setable)
type registryKey struct {
	mode   Mode
	target any
}
type registry struct {
	mu       sync.Mutex
	currMode Mode
	values   map[registryKey]any
}

var reg = &registry{values: make(map[registryKey]any)}

func SetMode(newMode Mode) (errs error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.currMode = newMode
	for k, val := range reg.values {
		if k.mode == newMode {
			if s, ok := k.target.(setable[any]); ok {
				err := s.Set(val)
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
		}
	}
	return errs
}

// statefulSetable wraps a setable and adds mode-awareness
type statefulSetable[T any] struct {
	mode   Mode
	target setable[T]
}

func (s *statefulSetable[T]) Set(val T) error {
	reg.mu.Lock()
	k := registryKey{s.mode, s.target}
	prev, ok := reg.values[k]
	reg.mu.Unlock()
	if ok && reflect.DeepEqual(prev, val) {
		return nil // Not dirty, don't resend
	}
	if err := s.target.Set(val); err != nil {
		return err
	}
	reg.mu.Lock()
	reg.values[k] = val
	reg.mu.Unlock()
	return nil
}

// Factory function
func Stateful[T any](mode Mode, s setable[T]) setable[T] {
	return &statefulSetable[T]{mode: mode, target: s}
}

func Bind[A any](mode Mode, binder bindable[A], callback func(A) error) {
	// Now bind the callback to the device, but first wrap it in another closure with guards to only run this for the specified mode.
	binder.Bind(
		func(args A) error {
			if reg.currMode|mode != 0 {
				return callback(args)
			}
			return nil
		},
	)
}
