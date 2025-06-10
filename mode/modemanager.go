package mode

import (
	"golang.org/x/exp/constraints"
)

// event holds a collection of events that should all update the same underlying value.
//
// The value is cached.
type event struct {
	value any
}

// layerObserver is a bundle of all the elements registered for a particular mode
//
// Each element is referenced by a descriptive string name.
type layerObserver struct {
	internal map[any]*event
}

func newLayer() *layerObserver {
	return &layerObserver{
		internal: map[any]*event{},
	}
}

func (l *layerObserver) get(key any) *event {
	if _, ok := l.internal[key]; !ok {
		l.internal[key] = &event{}
	}
	return l.internal[key]
}

// ModeManager sets the current mode and manages which effects are active depending on the active mode.
//
// Effects of elements in the currently active mode take immediate effect.
//
// All inactive modes' effects will not be run until that mode is activated. ModeManager maintains a copy of the value of each element so that the effects
// can be applied with the correct value immediately when the mode is activated.
type ModeManager[M constraints.Integer] struct {
	currMode M
	// For updating devices when we switch modes
	//
	// For now at least, we are YOLOing any bitwise set of modes as a key and then checking if the current mode is a subset of the key
	// when we need to update the layer
	modes map[M]*layerObserver

	selectedTrackMix    string
	selectedTrackRecord int
}

func NewModeManager[M constraints.Integer](startingMode M) *ModeManager[M] {
	return &ModeManager[M]{
		currMode: startingMode,
		// These modes are hand-written since the list of modes does not change often.
		modes: map[M]*layerObserver{},
	}
}

// SetMode sets the currently active mode.
//
// If the new mode is not the same as the current mode, run each effect of each element with its cached value.
func (c *ModeManager[M]) SetMode(mode M) {
	if c.currMode == mode {
		return
	}
	c.currMode = mode
	// Run any actions associated with this mode to update devices to match
	// values stored for this mode while we were in a different mode
	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
}

// getMode gets the requested mode, initializing it if it does not exist.
func (c *ModeManager[M]) getMode(mode M) *layerObserver {
	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
	return c.modes[mode]
}

// Bind binds the callback to the binding site and adds a guard to ensure that the callback is only called for the specified mode
//
// The most important thing about this function is that defines its generic types from the types in the passed binder function.
// This function will come from the device we are binding to, and it is that device's responsibility to tell us the types it expects.
// Once you have provided the bind function, the language server knows the types you need to provide for the path and callback. This ensures type-safety
// and gives the language server maximum latitude to help you write a valid callback.
//
// There are some functional shenanigans to support proper binding to the device and proper automatic callbacks on mode change within the mode manager.
// Note that this function doesn't care about the types of the bind path or args to the callback as long as the bind function accepts that pair.
func Bind[P, A any, M constraints.Integer](mm *ModeManager[M], mode M, binder func(P, func(A) error), path P, callback func(A) error) {
	// Now bind the callback to the device, but first wrap it in another closure with guards to only run this for the specified mode.
	binder(
		path,
		func(args A) error {
			if mm.currMode|mode != 0 {
				return callback(args)
			}
			return nil
		},
	)
}
