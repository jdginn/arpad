package main

import (
	"fmt"
	"log/slog"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"

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
	OSC_REAPER_IP   = "0.0.0.0"
	OSC_REAPER_PORT = 9000
	OSC_ARPAD_IP    = "192.168.22.129"
	OSC_ARPAD_PORT  = 9001
)

const DEVICE_TRACKS = 8

func getMidiPorts() (in drivers.In, out drivers.Out, err error) {
	const MIDI_IN = "X-Touch INT"
	const FALLBACK_MIDI_IN = "IAC Driver Bus 1"
	const MIDI_OUT = "X-Touch INT"
	const FALLBACK_MIDI_OUT = "IAC Driver Bus 2"
	const LAST_DITCH_MIDI_OUT = "IAC Driver Bus 1"
	in, err = midi.FindInPort(MIDI_IN)
	if err != nil {
		in, err = midi.FindInPort(FALLBACK_MIDI_IN)
		if err != nil {
			return in, out, fmt.Errorf("could not any midi in port")
		}
		slog.Warn("Midi in: X-Touch not found; fallling back to IAC Driver Bus 1. Is the hardware connected?")
	}
	out, err = midi.FindOutPort(MIDI_OUT)
	if err != nil {
		out, err = midi.FindOutPort(FALLBACK_MIDI_OUT)
		if err != nil {
			out, err = midi.FindOutPort(LAST_DITCH_MIDI_OUT)
			if err != nil {
				return in, out, fmt.Errorf("could not find any midi in port")
			}
			slog.Warn("Midi out: X-Touch not found; fallling back to IAC Driver Bus 1, WHICH LOOPS MIDI BACK IN. This will cause problems. Please configure IAC Driver Bus 2 in Audo MIDI Setup.")
		}
		slog.Warn("Midi out: X-Touch not found; fallling back to IAC Driver Bus 2. Is the hardware connected?")
	}
	return in, out, nil
}

func main() {
	defer midi.CloseDriver()
	in, out, err := getMidiPorts()
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

	time.Sleep(time.Second * 1000)
}
