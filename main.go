package main

import (
	"fmt"
	"math"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	"github.com/jdginn/arpad/devices"
	"github.com/jdginn/arpad/devices/motu"
	"github.com/jdginn/arpad/devices/reaper"
	"github.com/jdginn/arpad/devices/xtouch"
)

type Mode int

const (
	RECORD Mode = iota
	MIX
)

// event holds a collection of events that should all update the same underlying value.
//
// The value is cached.
type event[T devices.BaseTypes] struct {
	value   T
	actions []devices.Callback[T]
}

// layerObserver is a bundle of all the elements registered for a particular mode
//
// Each element is referenced by a descriptive string name.
type layerObserver struct {
	ints    map[string]event[int64]
	floats  map[string]event[float64]
	strings map[string]event[string]
	bools   map[string]event[bool]
}

func newLayer() layerObserver {
	return layerObserver{
		ints:    map[string]event[int64]{},
		floats:  map[string]event[float64]{},
		strings: map[string]event[string]{},
		bools:   map[string]event[bool]{},
	}
}

// ModeManager sets the current mode and manages which effects are active depending on the active mode.
//
// Effects of elements in the currently active mode take immediate effect.
//
// All inactive modes' effects will not be run until that mode is activated. ModeManager maintains a copy of the value of each element so that the effects
// can be applied with the correct value immediately when the mode is activated.
type ModeManager struct {
	currMode Mode
	// For updating devices when we switch modes
	modes map[Mode]layerObserver
}

func NewModeManager() ModeManager {
	return ModeManager{
		currMode: MIX,
		// These modes are hand-written since the list of modes does not change often.
		modes: map[Mode]layerObserver{
			MIX:    newLayer(),
			RECORD: newLayer(),
		},
	}
}

// SetMode sets the currently active mode.
//
// If the new mode is not the same as the current mode, run each effect of each element with its cached value.
func (c *ModeManager) SetMode(mode Mode) {
	if c.currMode == mode {
		return
	}
	c.currMode = mode
	// Run any actions associated with this mode to update devices to match
	// values stored for this mode while we were in a different mode
	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
	for _, e := range c.modes[c.currMode].ints {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].floats {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].strings {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].bools {
		for _, a := range e.actions {
			a(e.value)
		}
	}
}

func (c *ModeManager) RegisterInt(mode Mode, key string, foreignRegister func(string, devices.Callback[int64]), effect devices.Callback[int64]) devices.Callback[int64] {
	foreignRegister(key, effect)

	elem := c.modes[mode].ints[key]
	elem.actions = append(elem.actions, effect)

	return func(v int64) error {
		elem.value = v
		if c.currMode == mode {
			return effect(v)
		}
		return nil
	}
}

func (c *ModeManager) RegisterFloat(mode Mode, key string, r func(string, devices.Callback[float64]), e devices.Callback[float64]) {
	elem := c.modes[mode].floats[key]
	elem.actions = append(elem.actions, e)

	r(key, func(v float64) error {
		elem.value = v
		if c.currMode == mode {
			return e(v)
		}
		return nil
	})
}

func main() {
	defer midi.CloseDriver()
	fmt.Printf("outports:\n" + midi.GetOutPorts().String() + "\n")

	in, err := midi.FindInPort("IAC Driver Bus 1")
	if err != nil {
		panic(err)
	}
	fmt.Println(in)
	out, err := midi.FindOutPort("IAC Driver Bus 1")
	if err != nil {
		panic(err)
	}
	fmt.Println(out)

	x := xtouch.New(devices.NewMidiDevice(in, out))

	m := motu.NewHTTPDatastore("http://localhost:8888")

	r := reaper.OscServer{}

	c := NewModeManager()

	for i := 0; i < 8; i++ {
		x.Channels[i].Fader.Bind(func(rel int16, abs uint16) error {
			normalized := float64(abs) / 4 / float64(math.MaxUint16)
			switch c.currMode {
			case RECORD:
				return m.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i), normalized)
			default:
				return r.SetFloat(fmt.Sprintf("channels/%d/fader", i), normalized) // TODO:
			}
		})
		x.Channels[i].Mute.Bind(func(b bool) error {
			switch c.currMode {
			case RECORD:
				return m.SetBool(fmt.Sprintf("mix/main/%d/matrix/mute", i), b)
			default:
				return r.SetBool(fmt.Sprintf("channels/%d/mute", i), b)
			}
		})
		x.Channels[i].Solo.Bind(func(b bool) error {
			switch c.currMode {
			case RECORD:
				return m.SetBool(fmt.Sprintf("mix/main/%d/matrix/solo", i), b)
			default:
				return r.SetBool(fmt.Sprintf("channels/%d/solo", i), b)
			}
		})

		m.BindFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i),
			func(v float64) error {
				x.Channels[i].Fader.SetFaderAbsolute(int16(v / 4 * float64(math.MaxUint16)))
				return nil
			})
		// TODO: is there a better way to provide levels to meters?
		c.RegisterFloat(RECORD, "ext/ibank/0/ch/%d/vlLimit", m.BindFloat, func(v float64) error {
			x.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		c.RegisterFloat(MIX, "channels/%d/meter", r.RegisterFloat, func(v float64) error { // TODO: path
			x.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		c.RegisterFloat(RECORD, "ext/ibank/0/ch/%d/vlClip", m.BindFloat, func(v float64) error {
			x.Channels[i].Rec.SetLED(xtouch.FLASHING)
			return nil
		})
		c.RegisterFloat(MIX, "channels/%d/clip", r.RegisterFloat, func(v float64) error { // TODO: path
			x.Channels[i].Rec.SetLED(xtouch.FLASHING)
			return nil
		})
		// TODO: trim on encoders
	}

	x.Function[0].Bind(func(b bool) error {
		switch c.currMode {
		case MIX:
			c.SetMode(RECORD)
		case RECORD:
			c.SetMode(MIX)
		}
		return nil
	})
}
