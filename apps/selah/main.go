package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2"

	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver

	"github.com/jdginn/arpad/devices"
	reaperlib "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"

	"github.com/jdginn/arpad/apps/selah/layers"
	. "github.com/jdginn/arpad/apps/selah/layers/mode"
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
	MIDI_IN  = "X-Touch INT"
	MIDI_OUT = "X-Touch INT"
)

// const (
// 	MIDI_IN  = "IAC Driver Bus 1"
// 	MIDI_OUT = "IAC Driver Bus 2"
// )

const (
	OSC_REAPER_IP   = "0.0.0.0"
	OSC_REAPER_PORT = 9000
	OSC_ARPAD_IP    = "192.168.22.129"
	OSC_ARPAD_PORT  = 9001
)

const DEVICE_TRACKS = 8

func intToNormFloat(abs int16) float64 {
	return float64(abs) / 4 / float64(math.MaxUint16)
}

func normFloatToInt(norm float64) int16 {
	return int16((norm - 0.5) * float64(math.MaxInt16))
}

func getFirstWildcard(prefix, path string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	// If there are more slashes, split and return the first part
	parts := strings.SplitN(rest, "/", 2)
	return parts[0], true
}

type bindable[A any] interface {
	Bind(func(A) error)
}

type setable[T any] interface {
	Set(T) error
}

func link[T any](b bindable[T], s setable[T]) {
	b.Bind(func(v T) error { return s.Set(v) })
}

func int16ToNormFloat(val int16) float64 {
	return float64(val) / float64(math.MaxInt16) * 2
}

func main() {
	defer midi.CloseDriver()
	fmt.Printf("outports:\n" + midi.GetOutPorts().String() + "\n")

	in, err := midi.FindInPort(MIDI_IN)
	if err != nil {
		panic(err)
	}
	out, err := midi.FindOutPort(MIDI_OUT)
	if err != nil {
		panic(err)
	}
	xtouch := xtouchlib.New(devices.NewMidiDevice(in, out))

	reaper := reaperlib.NewReaper(devices.NewOscDevice(OSC_ARPAD_IP, OSC_ARPAD_PORT, OSC_REAPER_IP, OSC_REAPER_PORT, reaperlib.NewDispatcher()))

	layers.NewEncoderAssign(xtouch)
	layers.NewTrackManager(xtouch, reaper)
	SetMode(MIX)

	go reaper.Run()
	fmt.Println("Reaper is running...")
	go xtouch.Run()
	fmt.Println("Xtouch is running...")

	// go func() {
	// 	for {
	// 		fmt.Print("A\n")
	// 		reaper.Track(1).Volume.Set(5)
	// 		reaper.Track(2).Volume.Set(0)
	// 		time.Sleep(time.Second)
	// 		fmt.Print("B\n")
	// 		reaper.Track(1).Volume.Set(0)
	// 		reaper.Track(2).Volume.Set(0.75)
	// 		time.Sleep(time.Second)
	// 	}
	// }()

	time.Sleep(time.Second * 1000)
}
