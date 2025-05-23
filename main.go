package main

import (
	"fmt"
	"math"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	dev "github.com/jdginn/arpad/devices"
	motulib "github.com/jdginn/arpad/devices/motu"
	reaperlib "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

type Mode uint64

const (
	DEFAULT Mode = 1 << iota
	MIX
	MIX_THIS_TRACK_SENDS
	MIX_SENDS_TO_THIS_BUS
	RECORD
	RECORD_THIS_TRACK_SENDS
	RECORD_SENDS_TO_THIS_OUTPUT
	RECORD_SENDS_TO_THIS_AUX
	ALL = 0xFFFFFFFFFFFFFFFF
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
type event struct {
	value   any
	actions []func(any) error
}

// layerObserver is a bundle of all the elements registered for a particular mode
//
// Each element is referenced by a descriptive string name.
type layerObserver struct {
	internal map[any]event
}

func newLayer() layerObserver {
	return layerObserver{
		internal: map[any]event{},
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
	//
	// For now at least, we are YOLOing any bitwise set of modes as a key and then checking if the current mode is a subset of the key
	// when we need to update the layer
	modes map[Mode]layerObserver

	selectedTrackMix    string
	selectedTrackRecord int
}

func NewModeManager() *ModeManager {
	return &ModeManager{
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

	for m, l := range c.modes {
		if m|c.currMode != 0 {
			for _, e := range l.internal {
				for _, a := range e.actions {
					a(e.value)
				}
			}
		}
	}
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
func Bind[P, A any](mm *ModeManager, mode Mode, binder func(P, func(A) error), path P, callback func(A) error) {
	// First, we need to tell the mode manager to call the callback with the cached value any time we toggle to this mode
	elem := mm.modes[mode].internal[path]
	// We need to type-delete V because it allows us to store all of these actions in one slice (nifty!)
	elem.actions = append(elem.actions, func(v any) error {
		// Undo the type deletion here in this closure
		// Now everything is type-safe again
		castV := v.(A)
		return callback(castV)
	})

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

	xtouch := xtouchlib.New(dev.NewMidiDevice(in, out))

	motu := motulib.NewHTTPDatastore("http://localhost:8888")

	reaper := reaperlib.OscServer{}

	m := NewModeManager()

	for trackNum := 0; trackNum < 8; trackNum++ {
		c := xtouch.Channels[trackNum]

		// Scribble strip
		Bind(m, RECORD, motu.BindString, fmt.Sprintf("ext/ibank/%d/name", trackNum), func(s string) error {
			return xtouch.Channels[trackNum].Scribble.SendScribble(xtouchlib.Green, []byte(s), []byte("input"))
		})

		// Fader
		normalizeFader := func(abs uint16) float64 {
			return float64(abs) / 4 / float64(math.MaxUint16)
		}
		Bind(m, RECORD|RECORD_THIS_TRACK_SENDS, c.Fader.Bind, nil, func(args dev.ArgsPitchBend) error {
			return motu.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", trackNum), normalizeFader(args.Absolute))
		})
		Bind(m, MIX, c.Fader.Bind, nil, func(args dev.ArgsPitchBend) error {
			return reaper.SetFloat(fmt.Sprintf("channels/%d/fader", trackNum), normalizeFader(args.Absolute))
		})

		Bind(m, RECORD, motu.BindFloat, fmt.Sprintf("mix/main/%d/matrix/fader", trackNum), func(f float64) error {
			return c.Fader.SetFaderAbsolute(int16(f / 4 * float64(math.MaxUint16)))
		})
		Bind(m, MIX, reaper.BindFloat, fmt.Sprintf("channels/%d/fader", trackNum), func(f float64) error {
			return c.Fader.SetFaderAbsolute(int16(f / 4 * float64(math.MaxUint16)))
		})

		// Encoders
		Bind(m, RECORD, c.Encoder.Bind, nil, func(args dev.ArgsCC) error {
			if c.EncoderButton.IsPressed() {
				return reaper.SetInt(fmt.Sprintf("ext/ibank/0/chan/%d/trim", trackNum), int64(args.Value))
			}
			return reaper.SetInt(fmt.Sprintf("mix/chan/%d/pan", trackNum), int64(args.Value)) // TODO:
		})
		Bind(m, MIX, c.Encoder.Bind, nil, func(args dev.ArgsCC) error {
			if c.EncoderButton.IsPressed() {
				return reaper.SetInt(fmt.Sprintf("channels/%d/trim", trackNum), int64(args.Value)) // TODO:
			}
			return reaper.SetInt(fmt.Sprintf("channels/%d/trim", trackNum), int64(args.Value))
		})

		// Select
		Bind(m, MIX, c.Select.Bind, nil, func(b bool) error {
			if m.selectedTrackMix, err = motu.GetStr("channels/%d/name"); err != nil {
				return err
			}
			return nil
		})
		Bind(m, RECORD, c.Select.Bind, nil, func(b bool) error {
			m.selectedTrackRecord = trackNum
			return c.Select.SetLED(xtouchlib.ON)
		})
		// TODO: bind incoming select from DAW

		// Mute
		Bind(m, RECORD, c.Mute.Bind, nil, func(b bool) error {
			// TODO: need toggle funcionality
			return motu.SetBool(fmt.Sprintf("mix/main/%d/matrix/mute", trackNum), b)
		})
		// TODO: default mode
		Bind(m, RECORD, c.Mute.Bind, nil, func(b bool) error {
			// TODO: need toggle funcionality
			return reaper.SetBool(fmt.Sprintf("channels/%d/mute", trackNum), b)
		})
		Bind(m, RECORD, motu.BindBool, fmt.Sprintf("mix/main/%d/matrix/mute", trackNum), func(b bool) error {
			if b {
				xtouch.Channels[trackNum].Mute.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[trackNum].Mute.SetLED(xtouchlib.OFF)
			}
			return nil
		})
		Bind(m, RECORD, motu.BindBool, fmt.Sprintf("channels/%d/mute", trackNum), func(b bool) error {
			if b {
				xtouch.Channels[trackNum].Mute.SetLED(xtouchlib.ON)
			} else {
				xtouch.Channels[trackNum].Mute.SetLED(xtouchlib.OFF)
			}
			return nil
		})

		//		// Solo
		//		xtouch.Channels[trackNum].Solo.Bind(func(b bool) error {
		//			switch m.currMode {
		//			case RECORD:
		//				return motu.SetBool(fmt.Sprintf("mix/main/%d/matrix/solo", trackNum), b)
		//			default:
		//				return reaper.SetBool(fmt.Sprintf("channels/%d/solo", trackNum), b)
		//			}
		//		})
		//		m.BindBool(RECORD, fmt.Sprintf("mix/main/%d/matrix/solo", trackNum), motu.BindBool, func(b bool) error {
		//			if b {
		//				xtouch.Channels[trackNum].Solo.SetLED(xtouchlib.ON)
		//			} else {
		//				xtouch.Channels[trackNum].Solo.SetLED(xtouchlib.OFF)
		//			}
		//			return nil
		//		})
		//		m.BindBool(MIX, fmt.Sprintf("channels/%d/solo", trackNum), motu.BindBool, func(b bool) error {
		//			if b {
		//				xtouch.Channels[trackNum].Solo.SetLED(xtouchlib.ON)
		//			} else {
		//				xtouch.Channels[trackNum].Solo.SetLED(xtouchlib.OFF)
		//			}
		//			return nil
		//		})
		//
		//		// TODO: is there a better way to provide levels to meters?
		//		m.BindFloat(RECORD, "ext/ibank/0/ch/%d/vlLimit", motu.BindFloat, func(v float64) error {
		//			xtouch.Channels[trackNum].Meter.SendRelative(0.9)
		//			return nil
		//		})
		//		m.BindFloat(MIX, "channels/%d/meter", reaper.RegisterFloat, func(v float64) error { // TODO: path
		//			xtouch.Channels[trackNum].Meter.SendRelative(0.9)
		//			return nil
		//		})
		//		m.BindFloat(RECORD, "ext/ibank/0/ch/%d/vlClip", motu.BindFloat, func(v float64) error {
		//			xtouch.Channels[trackNum].Rec.SetLED(xtouchlib.FLASHING)
		//			return nil
		//		})
		//		m.BindFloat(MIX, "channels/%d/clip", reaper.RegisterFloat, func(v float64) error { // TODO: path
		//			xtouch.Channels[trackNum].Rec.SetLED(xtouchlib.FLASHING)
		//			return nil
		//		})
		//		// TODO: trim on encoders
		//
		//	} // end per-channel assignments
		//
		//	// -----------------------------------
		//	// Select primary mode (MIX vs RECORD)
		//	// -----------------------------------
		//
		//	// Select MIX mode
		//	xtouch.EncoderAssign.TRACK.Bind(func(b bool) error {
		//		if b {
		//			m.currMode = MIX
		//		}
		//		// TODO: make a radio button group
		//		xtouch.EncoderAssign.TRACK.SetLED(xtouchlib.ON)
		//		xtouch.EncoderAssign.PAN_SURROUND.SetLED(xtouchlib.OFF)
		//		return nil
		//	})
		//
		//	// Select RECORD mode
		//	xtouch.EncoderAssign.PAN_SURROUND.Bind(func(b bool) error {
		//		if b {
		//			m.currMode = RECORD
		//		}
		//		xtouch.EncoderAssign.TRACK.SetLED(xtouchlib.OFF)
		//		xtouch.EncoderAssign.PAN_SURROUND.SetLED(xtouchlib.ON)
		//		return nil
		//	})
		//
		//	// -----------------------
		//	// Select fader assignment
		//	// -----------------------
		//
		//	// Set to track levels
		//	xtouch.View.MIDI.Bind(func(b bool) error {
		//		if b {
		//			switch m.currMode {
		//			case MIX:
		//				// no-op
		//			case MIX_THIS_TRACK_SENDS:
		//				m.SetMode(MIX)
		//			case MIX_SENDS_TO_THIS_BUS:
		//				m.SetMode(MIX)
		//			case RECORD:
		//				// no-op
		//			case RECORD_THIS_TRACK_SENDS:
		//				m.SetMode(RECORD)
		//			case RECORD_SENDS_TO_THIS_AUX:
		//				m.SetMode(RECORD)
		//			case RECORD_SENDS_TO_THIS_OUTPUT:
		//				m.SetMode(RECORD)
		//			}
		//		}
		//		xtouch.View.MIDI.SetLED(xtouchlib.ON)
		//		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		//		return nil
		//	})
		//
		//	// Set faders to all sends from this track
		//	xtouch.View.INPUTS.Bind(func(b bool) error {
		//		if b {
		//			switch m.currMode {
		//			case MIX:
		//				m.SetMode(MIX_THIS_TRACK_SENDS)
		//			case MIX_THIS_TRACK_SENDS:
		//				// no-op
		//			case MIX_SENDS_TO_THIS_BUS:
		//				m.SetMode(MIX_THIS_TRACK_SENDS)
		//			case RECORD:
		//				m.SetMode(RECORD_THIS_TRACK_SENDS)
		//			case RECORD_THIS_TRACK_SENDS:
		//				// no-op
		//			case RECORD_SENDS_TO_THIS_AUX:
		//				m.SetMode(RECORD_THIS_TRACK_SENDS)
		//			case RECORD_SENDS_TO_THIS_OUTPUT:
		//				m.SetMode(RECORD_THIS_TRACK_SENDS)
		//			}
		//		}
		//		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		//		xtouch.View.INPUTS.SetLED(xtouchlib.ON)
		//		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		//		return nil
		//	})
		//
		//	// Set faders to all inputs to this bus/aux
		//	xtouch.View.AUDIO_TRACKS.Bind(func(b bool) error {
		//		if b {
		//			switch m.currMode {
		//			case MIX:
		//				m.SetMode(MIX_SENDS_TO_THIS_BUS)
		//			case MIX_THIS_TRACK_SENDS:
		//				m.SetMode(MIX_SENDS_TO_THIS_BUS)
		//			case MIX_SENDS_TO_THIS_BUS:
		//				// no-op
		//			case RECORD:
		//				m.SetMode(RECORD_SENDS_TO_THIS_AUX)
		//			case RECORD_THIS_TRACK_SENDS:
		//				m.SetMode(RECORD_SENDS_TO_THIS_AUX)
		//			case RECORD_SENDS_TO_THIS_AUX:
		//				// no-op
		//			case RECORD_SENDS_TO_THIS_OUTPUT:
		//				m.SetMode(RECORD_SENDS_TO_THIS_AUX)
		//			}
		//		}
		//		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		//		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.ON)
		//		xtouch.View.AUDIO_INST.SetLED(xtouchlib.OFF)
		//		return nil
		//	})
		//
		//	// Set faders to all inputs to this output
		//	xtouch.View.AUDIO_INST.Bind(func(b bool) error {
		//		if b {
		//			switch m.currMode {
		//			case MIX:
		//				// no-op
		//			case MIX_THIS_TRACK_SENDS:
		//				// no-op
		//			case MIX_SENDS_TO_THIS_BUS:
		//				// no-op
		//			case RECORD:
		//				m.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
		//			case RECORD_THIS_TRACK_SENDS:
		//				m.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
		//			case RECORD_SENDS_TO_THIS_AUX:
		//				m.SetMode(RECORD_SENDS_TO_THIS_OUTPUT)
		//			case RECORD_SENDS_TO_THIS_OUTPUT:
		//				// no-op
		//			}
		//		}
		//		xtouch.View.MIDI.SetLED(xtouchlib.OFF)
		//		xtouch.View.INPUTS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_TRACKS.SetLED(xtouchlib.OFF)
		//		xtouch.View.AUDIO_INST.SetLED(xtouchlib.ON)
		//		return nil
		//	})
	}
}
