//go:generate go run cmd/generatebindings/main.go
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
	. "github.com/jdginn/arpad/mode"
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

const MIDI_IN = "IAC Driver Bus 1"

// const MIDI_IN = "X-Touch INT"
const MIDI_OUT = "IAC Driver Bus 1"

// const MIDI_OUT = "X-Touch EXT"

const TOTAL_TRACKS = 8

func normalizeFader(abs int16) float64 {
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
		Bind(MIX, c.Fader, func(val int16) error {
			return reaper.Track(trackNum).Volume.Set(normalizeFader(val))
		})
		Bind(MIX, c.Mute, func(b bool) error {
			return reaper.Track(trackNum).Mute.Set(b)
		})
		Bind(MIX, c.Solo, func(b bool) error {
			return reaper.Track(trackNum).Solo.Set(b)
		})
		Bind(MIX, c.Rec, func(b bool) error {
			return reaper.Track(trackNum).Recarm.Set(b)
		})
		Bind(MIX, c.Select, func(b bool) error {
			return reaper.Track(trackNum).Select.Set(b)
		})
		// ...
	}

	// Reaper bindings
	//
	// MIX Mode
	for trackNum := int64(1); trackNum <= TOTAL_TRACKS; trackNum++ {
		c := xtouch.Channels[trackNum]
		Bind(MIX, reaper.Track(trackNum).Volume, func(val float64) error {
			return Stateful(MIX, c.Fader).Set(int16(val))
		})
		// ...
	}

	// Mode selection
	Bind(MIX, xtouch.EncoderAssign.TRACK, func(b bool) error {
		return SetMode(MIX)
	})
	Bind(MIX, xtouch.EncoderAssign.PAN_SURROUND, func(b bool) error {
		return SetMode(RECORD)
	})

	// Layer selection within modes
	Bind(MIX|MIX_THIS_TRACK_SENDS, xtouch.View.GLOBAL, func(b bool) error {
		return SetMode(MIX_SENDS_TO_THIS_BUS)
	})

	SetMode(MIX)
	reaper.Run()
	fmt.Println("Reaper is running...")
	go xtouch.Run()
	fmt.Println("Xtouch is running...")

	time.Sleep(time.Second * 1000)
}
