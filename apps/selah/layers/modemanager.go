package layers

import (
	"errors"
	"sync"

	reaper "github.com/jdginn/arpad/devices/reaper"
	"github.com/jdginn/arpad/devices/xtouch"
)

type callbackEvent struct {
	mode     Mode
	callback func() error
}

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

type Manager struct {
	mu sync.Mutex

	x *xtouch.XTouchDefault
	r *reaper.Reaper

	currMode  Mode
	callbacks []callbackEvent
}

func NewManager(x *xtouch.XTouchDefault, r *reaper.Reaper) *Manager {
	m := &Manager{
		x:         x,
		r:         r,
		currMode:  MIX,
		callbacks: make([]callbackEvent, 0),
	}
	return m
}

func (m *Manager) SetMode(newMode Mode) (errs error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if newMode == m.currMode {
		return
	}
	for _, callback := range m.callbacks {
		if callback.mode == newMode {
			if err := callback.callback(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	m.currMode = newMode
	return errs
}

func (m *Manager) CurrMode() Mode {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currMode
}

func (m *Manager) OnTransition(mode Mode, callback func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callbackEvent{mode: mode, callback: callback})
}
