package main

import (
	"fmt"
	"math"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	"github.com/jdginn/arpad/devices"
	motulib "github.com/jdginn/arpad/devices/motu"
	reaperlib "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

type Mode int

const (
	MIX Mode = iota
	MIX_THIS_TRACK_SENDS
	MIX_SENDS_TO_THIS_BUS
	RECORD
	RECORD_THIS_TRACK_SENDS
	RECORD_SENDS_TO_THIS_OUTPUT
	RECORD_SENDS_TO_THIS_AUX
)

// Modes:
//
// Mix: baseline mode for interacting with the DAW. Primarily mimics traditional MCU features. Supports submodes (triggered by "Flip" button based on context):
// -> This track sends on fader
// -> This send all input tracks
// 		In this mode, the 9th fader controls the level for this send
// In Mix mode, encoders are mapped to pan by default. Push in to trim gain.
// -> Marker mode (while holding marker button)
// Lists all markers on the scribble strips and allows jumping to marker by clicking in the encoder
//
// Record: baseline mode for interacting with the audio interface's internal mixer. Attempts to mimic the behavior of a large-format analog recording console. Supports submodes (triggered by "Flip" button based on context):
// -> This track sends to fader (separately label effect auxes vs. outputs)
// -> This output all input tracks
// -> This aux all input tracks
// In record mode, encoders are mapped to gain by default. Push in to control pan.
//
// Timecode display lists the current mode using ascii-to-7seg characters
//
// Mode selection is mapped to Encoder Assign
//
// The following buttons are active in every mode:
// - Modify, Utility, Automation, Transport buttons and jog wheel control the DAW at all times.
// - Talkback (mapped to Display button)
// - Global mute (mapped to Global View)
// - Control room monitoring selection (main monitors, mono mixcube, nearfield monitors, headphones-only, other?) mapped to View bttons (excluding Global View)
// - Per-channel record arm always controls the DAW

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

	selectedTrackMix    string
	selectedTrackRecord int
}

func NewModeManager() ModeManager {
	return ModeManager{
		currMode: MIX,
		// These modes are hand-written since the list of modes does not change often.
		modes: map[Mode]layerObserver{
			MIX:                         newLayer(),
			MIX_THIS_TRACK_SENDS:        newLayer(),
			MIX_SENDS_TO_THIS_BUS:       newLayer(),
			RECORD:                      newLayer(),
			RECORD_THIS_TRACK_SENDS:     newLayer(),
			RECORD_SENDS_TO_THIS_AUX:    newLayer(),
			RECORD_SENDS_TO_THIS_OUTPUT: newLayer(),
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

func (c *ModeManager) BindInt(mode Mode, key string, foreignRegister func(string, devices.Callback[int64]), callback devices.Callback[int64]) devices.Callback[int64] {
	foreignRegister(key, callback)

	elem := c.modes[mode].ints[key]
	elem.actions = append(elem.actions, callback)

	return func(v int64) error {
		elem.value = v
		if c.currMode == mode {
			return callback(v)
		}
		return nil
	}
}

func (c *ModeManager) BindFloat(mode Mode, key string, r func(string, devices.Callback[float64]), callback devices.Callback[float64]) {
	elem := c.modes[mode].floats[key]
	elem.actions = append(elem.actions, callback)

	r(key, func(v float64) error {
		elem.value = v
		if c.currMode == mode {
			return callback(v)
		}
		return nil
	})
}

func (c *ModeManager) BindBool(mode Mode, key string, r func(string, devices.Callback[bool]), callback devices.Callback[bool]) {
	elem := c.modes[mode].bools[key]
	elem.actions = append(elem.actions, callback)

	r(key, func(v bool) error {
		elem.value = v
		if c.currMode == mode {
			return callback(v)
		}
		return nil
	})
}

func (c *ModeManager) BindString(mode Mode, key string, r func(string, devices.Callback[string]), callback devices.Callback[string]) {
	elem := c.modes[mode].strings[key]
	elem.actions = append(elem.actions, callback)

	r(key, func(v string) error {
		elem.value = v
		if c.currMode == mode {
			return callback(v)
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

	xtouch := xtouchlib.New(devices.NewMidiDevice(in, out))

	motu := motulib.NewHTTPDatastore("http://localhost:8888")

	reaper := reaperlib.OscServer{}

	modes := NewModeManager()

	for i := 0; i < 8; i++ {
		// Scribble strip
		modes.BindString(RECORD, fmt.Sprintf("ext/ibank/%d/name", i), motu.BindString, func(s string) error {
			return xtouch.Channels[i].Scribble.SendScribble(xtouchlib.Green, []byte(s), []byte("input"))
		})

		// Fader
		xtouch.Channels[i].Fader.Bind(func(rel int16, abs uint16) error {
			normalized := float64(abs) / 4 / float64(math.MaxUint16)
			switch modes.currMode {
			case RECORD:
				return motu.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i), normalized)
			case RECORD_THIS_TRACK_SENDS:
				return motu.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i), normalized)
			case MIX:
				return reaper.SetFloat(fmt.Sprintf("channels/%d/fader", i), normalized) // TODO:
			}
			return nil
		})
		modes.BindFloat(RECORD, fmt.Sprintf("mix/main/%d/matrix/fader"), motu.BindFloat, func(f float64) error {
			return xtouch.Channels[i].Fader.SetFaderAbsolute(int16(f / 4 * float64(math.MaxUint16)))
		})
		modes.BindFloat(MIX, fmt.Sprintf("channels/%d/fader"), motu.BindFloat, func(f float64) error {
			return xtouch.Channels[i].Fader.SetFaderAbsolute(int16(f / 4 * float64(math.MaxUint16)))
		})

		// Encoders
		xtouch.Channels[i].Encoder.Bind(func(u uint8) error {
			switch modes.currMode {
			case RECORD:
				if xtouch.Channels[i].EncoderButton.IsPressed() {
					return reaper.SetInt(fmt.Sprintf("mix/chan/%d/pan", i), int64(u)) // TODO:
				}
				return reaper.SetInt(fmt.Sprintf("ext/ibank/0/chan/%d/trim", i), int64(u))
			case MIX:
				if xtouch.Channels[i].EncoderButton.IsPressed() {
					return reaper.SetInt(fmt.Sprintf("channels/%d/trim", i), int64(u)) // TODO:
				}
				return reaper.SetInt(fmt.Sprintf("channels/%d/pan", i), int64(u))
			}
			return nil
		})

		// Select
		xtouch.Channels[i].Select.Bind(func(b bool) error {
			switch modes.currMode {
			case MIX:
				modes.selectedTrackMix, err = motu.GetStr("channels/%d/name")
				if err != nil {
					return err
				}
			case RECORD:
				modes.selectedTrackRecord = i
				xtouch.Channels[i].Select.SetLED(xtouchlib.ON)
				// TODO: turn off other LEDs
			}
			return nil
		})
		// TODO: bind incoming select from DAW

		// Mute
		xtouch.Channels[i].Mute.Bind(func(b bool) error {
			// TODO: need toggle funcionality
			switch modes.currMode {
			case RECORD:
				return motu.SetBool(fmt.Sprintf("mix/main/%d/matrix/mute", i), b)
			default:
				return reaper.SetBool(fmt.Sprintf("channels/%d/mute", i), b)
			}
		})
		modes.BindBool(RECORD, fmt.Sprintf("mix/main/%d/matrix/mute", i), motu.BindBool, func(b bool) error {
			if b {
				xtouch.Channels[i].Mute.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[i].Mute.SetLED(xtouchlib.OFF)
			}
			return nil
		})
		modes.BindBool(MIX, fmt.Sprintf("channels/%d/mute", i), motu.BindBool, func(b bool) error {
			if b {
				xtouch.Channels[i].Mute.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[i].Mute.SetLED(xtouchlib.OFF)
			}
			return nil
		})

		// Solo
		xtouch.Channels[i].Solo.Bind(func(b bool) error {
			switch modes.currMode {
			case RECORD:
				return motu.SetBool(fmt.Sprintf("mix/main/%d/matrix/solo", i), b)
			default:
				return reaper.SetBool(fmt.Sprintf("channels/%d/solo", i), b)
			}
		})
		modes.BindBool(RECORD, fmt.Sprintf("mix/main/%d/matrix/solo", i), motu.BindBool, func(b bool) error {
			if b {
				xtouch.Channels[i].Solo.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[i].Solo.SetLED(xtouchlib.OFF)
			}
			return nil
		})
		modes.BindBool(MIX, fmt.Sprintf("channels/%d/solo", i), motu.BindBool, func(b bool) error {
			if b {
				xtouch.Channels[i].Solo.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[i].Solo.SetLED(xtouchlib.OFF)
			}
			return nil
		})

		// TODO: is there a better way to provide levels to meters?
		modes.BindFloat(RECORD, "ext/ibank/0/ch/%d/vlLimit", motu.BindFloat, func(v float64) error {
			xtouch.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		modes.BindFloat(MIX, "channels/%d/meter", reaper.RegisterFloat, func(v float64) error { // TODO: path
			xtouch.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		modes.BindFloat(RECORD, "ext/ibank/0/ch/%d/vlClip", motu.BindFloat, func(v float64) error {
			xtouch.Channels[i].Rec.SetLED(xtouchlib.FLASHING)
			return nil
		})
		modes.BindFloat(MIX, "channels/%d/clip", reaper.RegisterFloat, func(v float64) error { // TODO: path
			xtouch.Channels[i].Rec.SetLED(xtouchlib.FLASHING)
			return nil
		})
		// TODO: trim on encoders

	} // end per-channel assignments

	// -----------------------------------
	// Select primary mode (MIX vs RECORD)
	// -----------------------------------

	// Select MIX mode
	xtouch.EncoderAssign.TRACK.Bind(func(b bool) error {
		if b {
			modes.currMode = MIX
		}
		// TODO: make a radio button group
		xtouch.EncoderAssign.TRACK.SetLED(xtouchlib.ON)
		xtouch.EncoderAssign.PAN_SURROUND.SetLED(xtouchlib.OFF)
		return nil
	})

	// Select RECORD mode
	xtouch.EncoderAssign.PAN_SURROUND.Bind(func(b bool) error {
		if b {
			modes.currMode = RECORD
		}
		xtouch.EncoderAssign.TRACK.SetLED(xtouchlib.OFF)
		xtouch.EncoderAssign.PAN_SURROUND.SetLED(xtouchlib.ON)
		return nil
	})

	// -----------------------
	// Select fader assignment
	// -----------------------

	// Set to track levels
	xtouch.View.MIDI.Bind(func(b bool) error {
		if b {
			switch modes.currMode {
			case MIX:
				// no-op
			case MIX_THIS_TRACK_SENDS:
				modes.SetMode(MIX)
			case MIX_SENDS_TO_THIS_BUS:
				modes.SetMode(MIX)
			case RECORD:
				// no-op
			case RECORD_THIS_TRACK_SENDS:
				modes.SetMode(RECORD)
			case RECORD_SENDS_TO_THIS_AUX:
				modes.SetMode(RECORD)
			case RECORD_SENDS_TO_THIS_OUTPUT:
				modes.SetMode(RECORD)
			}
		}
		xtouch.View.MIDI.SetLED(xtouchlib.ON)
		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		return nil
	})

	// Set faders to all sends from this track
	xtouch.View.INPUTS.Bind(func(b bool) error {
		if b {
			switch modes.currMode {
			case MIX:
				modes.SetMode(MIX_THIS_TRACK_SENDS)
			case MIX_THIS_TRACK_SENDS:
				// no-op
			case MIX_SENDS_TO_THIS_BUS:
				modes.SetMode(MIX_THIS_TRACK_SENDS)
			case RECORD:
				modes.SetMode(RECORD_THIS_TRACK_SENDS)
			case RECORD_THIS_TRACK_SENDS:
				// no-op
			case RECORD_SENDS_TO_THIS_AUX:
				modes.SetMode(RECORD_THIS_TRACK_SENDS)
			case RECORD_SENDS_TO_THIS_OUTPUT:
				modes.SetMode(RECORD_THIS_TRACK_SENDS)
			}
		}
		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		xtouch.View.INPUTS.SetLED(xtouchlib.ON)
		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		return nil
	})

	// Set faders to all inputs to this bus/aux
	xtouch.View.AUDIO_TRACKS.Bind(func(b bool) error {
		if b {
			switch modes.currMode {
			case MIX:
				modes.SetMode(MIX_SENDS_TO_THIS_BUS)
			case MIX_THIS_TRACK_SENDS:
				modes.SetMode(MIX_SENDS_TO_THIS_BUS)
			case MIX_SENDS_TO_THIS_BUS:
				// no-op
			case RECORD:
				modes.SetMode(RECORD_SENDS_TO_THIS_AUX)
			case RECORD_THIS_TRACK_SENDS:
				modes.SetMode(RECORD_SENDS_TO_THIS_AUX)
			case RECORD_SENDS_TO_THIS_AUX:
				// no-op
			case RECORD_SENDS_TO_THIS_OUTPUT:
				modes.SetMode(RECORD_SENDS_TO_THIS_AUX)
			}
		}
		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.ON)
		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		return nil
	})

	// Set faders to all inputs to this output
	xtouch.View.AUDIO_INST.Bind(func(b bool) error {
		if b {
			switch modes.currMode {
			case MIX:
				// no-op
			case MIX_THIS_TRACK_SENDS:
				// no-op
			case MIX_SENDS_TO_THIS_BUS:
				// no-op
			case RECORD:
				modes.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
			case RECORD_THIS_TRACK_SENDS:
				modes.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
			case RECORD_SENDS_TO_THIS_AUX:
				modes.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
			case RECORD_SENDS_TO_THIS_OUTPUT:
				// no-op
			}
		}
		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		xtouch.View.AUDIO_INST.SetLED(xtouchlib.ON)
		return nil
	})
}
