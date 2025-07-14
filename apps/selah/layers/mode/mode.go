package mode

import (
	"errors"
	"sync"
)

type Mode uint64

const (
	DEFAULT Mode = 0
	MIX          = 1<<iota - 1
	MIX_SELECTED_TRACK_SENDS
	MIX_SELECTED_TRACK_RECEIVES
	RECORD
	RECORD_SELECTED_TRACK_SENDS
	RECORD_SELECTED_OUTPUT_RECEIVES
	RECORD_SELECTED_AUX_RECEIVES
	ALL = 0xFFFFFFFFFFFFFFFF
)

// bindable represents an endpoint that can have a callback bound to it to monitor state changes.
// The callback will be invoked whenever the endpoint's value changes.
type bindable[A any] interface {
	Bind(func(A) error)
}

type setable[T any] interface {
	Set(T) error
}

type callbackEvent struct {
	mode     Mode
	callback func() error
}

// registry tracks events that need to be performed upon state transition
//
// 1. setables: tracks the last known value for decorated setables and calls the relevant ones
// with that value upon state transition In this way, setables have automatically-managed state.
//
// 2. callbacks: tracks a list of callbacks to invoke upon state transition. Callbacks are opaque
// to this package and have no automatic state management; the user is responsible for managing
// any state on their own.
type registry struct {
	mu        sync.Mutex
	currMode  Mode
	callbacks []callbackEvent
}

var reg = &registry{}

func OnTransition(mode Mode, callback func() error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.callbacks = append(reg.callbacks, callbackEvent{mode: mode, callback: callback})
}

func SetMode(newMode Mode) (errs error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if newMode == reg.currMode {
		return
	}
	for _, callback := range reg.callbacks {
		if callback.mode == newMode {
			if err := callback.callback(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	reg.currMode = newMode
	return errs
}

func CurrMode() Mode {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return reg.currMode
}

func FilterSet[T any](mode Mode, s setable[T]) setable[T] {
	return &statefulSetable[T]{mode: mode, target: s}
}

// statefulSetable wraps a setable and adds mode-awareness
type statefulSetable[T any] struct {
	mode   Mode
	target setable[T]
}

func (s *statefulSetable[T]) Set(val T) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if (s.mode & reg.currMode) != 0 {
		return s.target.Set(val)
	}
	return nil
}

func FilterBind[T any](mode Mode, b bindable[T]) bindable[T] {
	return &statefulBindable[T]{mode: mode, target: b}
}

// statefulBindable wraps a bindable and adds mode-awareness
type statefulBindable[T any] struct {
	mode   Mode
	target bindable[T]
}

func (s *statefulBindable[T]) Bind(callback func(T) error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if (s.mode & reg.currMode) != 0 {
		s.target.Bind(callback)
	}
}
