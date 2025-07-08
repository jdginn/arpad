//go:generate go run cmd/generatebindings/main.go
package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"gitlab.com/gomidi/midi/v2"

	// _ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver

	"github.com/jdginn/arpad/devices"
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

// const MIDI_IN = "X-Touch INT"
const MIDI_IN = "IAC Driver Bus 1"

// const MIDI_OUT = "X-Touch EXT"
const MIDI_OUT = "IAC Driver Bus 1"

const (
	OSC_REAPER_IP   = "0.0.0.0"
	OSC_REAPER_PORT = 9000
	OSC_ARPAD_IP    = "192.168.1.146"
	OSC_ARPAD_PORT  = 9001
)

const DEVICE_TRACKS = 8

func normalizeFader(abs int16) float64 {
	return float64(abs) / 4 / float64(math.MaxUint16)
}

func getFirstWildcard(prefix, path string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	// If there are more slashes, split and return the first part
	parts := strings.SplitN(rest, "/", 2)
	return parts[0]
}

type trackData struct {
	x          *xtouchlib.XTouchDefault
	surfaceIdx int64
	reaperIdx  int64
	name       string
	volume     float64
	pan        float64
	mute       bool
	solo       bool
	rec        bool
	sends      map[int]*trackSendData
	rcvs       map[int]*trackSendData
}

func (t *trackData) onStateTransition() (errs error) {
	xt := t.x.Channels[t.surfaceIdx]
	return errors.Join(errs,
		xt.Fader.Set(int16(t.volume)),
		xt.Encoder.Ring.Set(t.pan),
		xt.Mute.SetLED(t.mute),
		xt.Solo.SetLED(t.solo),
		xt.Rec.SetLED(t.rec),
	)
}

type trackSendData struct {
	*trackData
	sendIdx uint64
	rcvIdx  uint64
	vol     float64
	pan     float64
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

	xtouch := xtouchlib.New(devices.NewMidiDevice(in, out))

	// motu := motulib.NewHTTPDatastore("http://localhost:8888")

	reaper := reaperlib.NewReaper(devices.NewOscDevice(OSC_ARPAD_IP, OSC_ARPAD_PORT, OSC_REAPER_IP, OSC_REAPER_PORT))

	trackStates := make(map[int64]*trackData)

	dispatcher := reaper.OscDispatcher()

	// Find and populate our collection of track states
	if err := dispatcher.AddMsgHandler("*", func(m *osc.Message) {
		fmt.Printf("Found %s\n", getFirstWildcard("/track/", m.Address))
		idx, err := strconv.ParseInt(getFirstWildcard("/track/", m.Address), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Failed to find valid wildcard in path %s: %v", m.Address, err))
		}
		if _, ok := trackStates[idx]; !ok {
			trackStates[idx] = &trackData{
				x:         xtouch,
				reaperIdx: idx,
				// TODO: there's an argument that we might need to make sure this runs FIRST but I'm not sure
			}
		}
	}); err != nil {
		panic(err)
	}

	// Reaper bindings
	//
	// MIX Mode
	for trackNum := int64(1); trackNum < DEVICE_TRACKS; trackNum++ {
		c := xtouch.Channels[trackNum]
		Bind(ALL, reaper.Track(trackNum).Volume, func(val float64) error {
			return Stateful(MIX, c.Fader).Set(int16(val))
		})
		// ...
	}
	// Transport
	Bind(ALL, reaper.Play, func(b bool) error {
		return xtouch.Transport.PLAY.SetLED(b)
	})
	Bind(ALL, reaper.Click, func(b bool) error {
		return xtouch.Transport.Click.SetLED(b)
	})

	// XTouch bindings
	//
	// Per-channel strip controls
	for trackNum := int64(1); trackNum < DEVICE_TRACKS; trackNum++ {
		c := xtouch.Channels[trackNum]
		rt := reaper.Track(trackNum)

		// MIX Mode
		Bind(MIX, c.Fader, func(val int16) error {
			return rt.Volume.Set(normalizeFader(val))
		})
		Bind(MIX, c.Mute, func(b bool) error {
			return rt.Mute.Set(b)
		})
		Bind(MIX, c.Solo, func(b bool) error {
			return rt.Solo.Set(b)
		})
		Bind(MIX, c.Rec, func(b bool) error {
			return rt.Recarm.Set(b)
		})
		Bind(MIX, c.Select.On, func(uint8) error {
			return rt.Select.Set(true)
		})
		// ...
	}

	// Transport
	Bind(ALL, xtouch.Transport.PLAY.On, func(uint8) error {
		return reaper.Play.Set(true)
	})
	Bind(ALL, xtouch.Transport.STOP.On, func(uint8) error {
		return reaper.Stop.Set(true)
	})
	Bind(ALL, xtouch.Transport.Click, func(b bool) error {
		return reaper.Click.Set(b)
	})
	Bind(MIX, xtouch.Transport.Solo, func(b bool) error {
		return reaper.Soloreset.Set(b)
	})
	Bind(MIX, xtouch.Transport.REW.On, func(uint8) error {
		return reaper.Rewind.Set(true)
	})
	Bind(ALL, xtouch.Transport.FF.On, func(b uint8) error {
		return reaper.Forward.Set(true)
	})

	// Mode selection
	Bind(MIX, xtouch.EncoderAssign.TRACK.On, func(uint8) error {
		return SetMode(MIX)
	})
	Bind(MIX, xtouch.EncoderAssign.PAN_SURROUND.On, func(uint8) error {
		return SetMode(RECORD)
	})

	// Layer selection within modes
	Bind(MIX|MIX_THIS_TRACK_SENDS, xtouch.View.GLOBAL.On, func(uint8) error {
		return SetMode(MIX_SENDS_TO_THIS_BUS)
	})

	SetMode(MIX)
	reaper.Run()
	fmt.Println("Reaper is running...")
	go xtouch.Run()
	fmt.Println("Xtouch is running...")

	time.Sleep(time.Second * 1000)
}
