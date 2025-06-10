package main

import (
	"fmt"
	"math"
	"time"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	dev "github.com/jdginn/arpad/devices"
	motulib "github.com/jdginn/arpad/devices/motu"
	reaperlib "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
	mLib "github.com/jdginn/arpad/mode"
)

// Modes:
//
// Mix: baseline mode for interacting with the DAW. Primarily mimics traditional MCU features. Supports submodes (triggered by "Flip" button based on context):
// -> This track sends on fader
// -> This send all input tracks
//
//	In this mode, the 9th fader controls the level for this send
//
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
// # Timecode display lists the current mode using ascii-to-7seg characters
//
// # Mode selection is mapped to Encoder Assign
//
// The following buttons are active in every mode:
// - Modify, Utility, Automation, Transport buttons and jog wheel control the DAW at all times.
// - Talkback (mapped to Display button)
// - Global mute (mapped to Global View)
// - Control room monitoring selection (main monitors, mono mixcube, nearfield monitors, headphones-only, other?) mapped to View bttons (excluding Global View)
// - Per-channel record arm always controls the DAW
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

type ModeManager struct {
	*mLib.ModeManager[Mode]

	// Extra metadata we want to keep track of that is not actually part of mode management logic
	selectedTrackRecord string
	selectedTrackMix    int64
}

var mm *ModeManager

func init() {
	mm = &ModeManager{mLib.NewModeManager[Mode](MIX), "", 0}
}

type bindable[P, A any] interface {
	Bind(P, func(A) error)
}

// This is just eliding the first argument because it never changes and visual clarity is at a premium
func bind[P, A any](mode Mode, binder bindable[P, A], path P, callback func(A) error) {
	mLib.Bind(mm.ModeManager, mode, binder.Bind, path, callback)
}

const MIDI_IN = "IAC Driver Bus 1"

// const MIDI_IN = "X-Touch INT"
const MIDI_OUT = "IAC Driver Bus 1"

// const MIDI_OUT = "X-Touch EXT"

const TOTAL_TRACKS = 8

func normalizeFader(abs uint16) float64 {
	return float64(abs) / 4 / float64(math.MaxUint16)
}

func main() {
	defer midi.CloseDriver()
	fmt.Printf("outports:\n" + midi.GetOutPorts().String() + "\n")

	in, err := midi.FindInPort(MIDI_IN)
	if err != nil {
		panic(err)
	}
	fmt.Println(in)
	out, err := midi.FindOutPort(MIDI_OUT)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)

	xtouch := xtouchlib.New(dev.NewMidiDevice(in, out))

	motu := motulib.NewHTTPDatastore("http://localhost:8888")

	var reaper reaperlib.Reaper

	//

	// Per-channel strip controls are defined here
	for trackNum := int64(1); trackNum <= TOTAL_TRACKS; trackNum++ {
		c := xtouch.Channels[trackNum]

		// XTouch bindings
		//
		// MIX Mode
		bind(MIX, c.Fader, nil, func(args dev.ArgsPitchBend) error {
			reaper.Track.SendTrackVolume(trackNum, normalizeFader(args.Absolute))
			return nil
		})
		bind(MIX, c.Mute, nil, func(b bool) error {
			return reaper.Track.SendTrackMute(trackNum, b)
		})
		bind(MIX, c.Solo, nil, func(b bool) error {
			return reaper.Track.SendTrackSolo(trackNum, b)
		})
		bind(MIX, c.Rec, nil, func(b bool) error {
			return reaper.Track.SendTrackRecArm(trackNum, b)
		})
		bind(MIX, c.Select, nil, func(b bool) error {
			return reaper.Track.SendTrackSelect(trackNum, b)
		})

		// Scribble strip
		bind(RECORD, motu.BindString, fmt.Sprintf("ext/ibank/%d/name", trackNum), func(s string) error {
			return xtouch.Channels[trackNum].Scribble.SendScribble(xtouchlib.Green, []byte(s), []byte("input"))
		})

		// Fader
		normalizeFader := func(abs uint16) float64 {
			return float64(abs) / 4 / float64(math.MaxUint16)
		}
		bind(RECORD|RECORD_THIS_TRACK_SENDS, c.Fader.BindMove, nil, func(args dev.ArgsPitchBend) error {
			fmt.Printf("XTOUCHÉ RECORD: track %d: %f\n", args.Absolute)
			return motu.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", trackNum), normalizeFader(args.Absolute))
		})
		bind(MIX, c.Fader.BindMove, nil, func(args dev.ArgsPitchBend) error {
			fmt.Printf("XTOUCH MIX: track %d: %f\n", args.Absolute)
			return reaper.Track.SendTrackVolume(trackNum, normalizeFader(args.Absolute))
		})

		bind(RECORD, motu.BindFloat, fmt.Sprintf("mix/main/%d/matrix/fader", trackNum), func(f float64) error {
			fmt.Printf("RECORD: track %d: %f\n", trackNum, f)
			return reaper.Track.SendTrackVolume(trackNum+1, f)
			// return c.Fader.SetFaderAbsolute(int16(f / 4 * float64(math.MaxUint16)))
		})

		bind(MIX, reaper.Track.BindTrackVolume, trackNum, func(f float64) error {
			fmt.Printf("MIX: track %d: %f\n", trackNum, f)
			return reaper.Track.SendTrackVolume(trackNum+1, f)
			// return c.Fader.SetFaderAbsolute(int16(f / 4 * math.MaxUint16))
		})

		// Encoders
		bind(RECORD, c.Encoder.Bind, nil, func(args dev.ArgsCC) error {
			if c.EncoderButton.IsPressed() {
				return motu.SetInt(fmt.Sprintf("ext/ibank/0/chan/%d/trim", trackNum), int64(args.Value))
			}
			return motu.SetInt(fmt.Sprintf("mix/chan/%d/pan", trackNum), int64(args.Value)) // TODO:
		})
		bind(MIX, c.Encoder.Bind, nil, func(args dev.ArgsCC) error {
			if c.EncoderButton.IsPressed() {
				// TODO
			}
			return reaper.Track.SendTrackPan(trackNum, float64(args.Value)) // TODO: need to normalize this?
		})

		// Select
		bind(RECORD, c.Select.Bind, nil, func(b bool) error {
			if mm.selectedTrackRecord, err = motu.GetStr("channels/%d/name"); err != nil {
				return err
			}
			return nil
		})
		// bind(MIX, c.Select.Bind, nil, func(b bool) error {
		// 	mm.selectedTrackMix = trackNum
		// 	return c.Select.SetLEDOn(xtouchlib.ON)
		// })
		// // TODO: bind incoming select from DAW

		// Mute
		bind(RECORD, c.Mute.Bind, nil, func(b bool) error {
			// TODO: need toggle funcionality
			return motu.SetBool(fmt.Sprintf("mix/main/%d/matrix/mute", trackNum), b)
		})
		// TODO: default mode
		bind(MIX, c.Mute.Bind, nil, func(b bool) error {
			// TODO: need toggle funcionality
			return reaper.Track.SendTrackMute(trackNum, b)
		})
		// bind(RECORD, motu.BindBool, fmt.Sprintf("mix/main/%d/matrix/mute", trackNum), func(b bool) error {
		// 	if b {
		// 		xtouch.Channels[trackNum].Mute.SetLEDOn(xtouchlib.ON)
		// 	} else {
		// 		xtouch.Channels[trackNum].Mute.SetLEDOn(xtouchlib.OFF)
		// 	}
		// 	return nil
		// })
		// bind(RECORD, motu.BindBool, fmt.Sprintf("channels/%d/mute", trackNum), func(b bool) error {
		// 	if b {
		// 		xtouch.Channels[trackNum].Mute.SetLEDOn(xtouchlib.ON)
		// 	} else {
		// 		xtouch.Channels[trackNum].Mute.SetLEDOn(xtouchlib.OFF)
		// 	}
		// 	return nil
		// })

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
	mm.SetMode(MIX)
	reaper.Run()
	fmt.Println("Reaper is running...")
	go xtouch.Run()
	fmt.Println("Xtouch is running...")

	time.Sleep(time.Second * 1000)

	// for {
	// 	fmt.Println("0")
	// 	reaper.Track.SendTrackVolumeDb(1, -10)
	// 	time.Sleep(time.Second * 10)
	// 	fmt.Println("1")
	// 	reaper.Track.SendTrackVolumeDb(1, 10)
	// 	time.Sleep(time.Second * 10)
	// }
}
