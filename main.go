package main

import (
	"fmt"
	"math"
	"time"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	dev "github.com/jdginn/arpad/devices"
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
func bind[P, A any](mode Mode, bindable bindable[P, A], path P, callback func(A) error) {
	mLib.Bind(mm.ModeManager, mode, bindable.Bind, path, callback)
}

type setable[T any] interface {
	Set(T) error
}

func set[T any](mode Mode, setable setable[T], val T) error {
	return mLib.Set(mm.ModeManager, mode, setable.Set)(val)
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

	// motu := motulib.NewHTTPDatastore("http://localhost:8888")

	var reaper reaperlib.Reaper

	// Per-channel strip controls are defined here
	for trackNum := int64(1); trackNum <= TOTAL_TRACKS; trackNum++ {
		c := xtouch.Channels[trackNum]

		// XTouch bindings
		//
		// MIX Mode
		bind(MIX, c.Fader, nil, func(args dev.ArgsPitchBend) error {
			return reaper.Track.SendTrackVolume(trackNum, normalizeFader(args.Absolute))
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
	}

	// Mode selection
	bind(MIX, xtouch.EncoderAssign.TRACK, nil, func(b bool) error {
		return mm.SetMode(MIX)
	})
	bind(MIX, xtouch.EncoderAssign.PAN_SURROUND, nil, func(b bool) error {
		return mm.SetMode(RECORD)
	})

	// Layer selection within modes
	bind(MIX|MIX_THIS_TRACK_SENDS, xtouch.View.GLOBAL, nil, func(b bool) error {
		return mm.SetMode(MIX_SENDS_TO_THIS_BUS)
	})

	mm.SetMode(MIX)
	reaper.Run()
	fmt.Println("Reaper is running...")
	go xtouch.Run()
	fmt.Println("Xtouch is running...")

	time.Sleep(time.Second * 1000)
}
