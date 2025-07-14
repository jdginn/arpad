package layers

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/hypebeast/go-osc/osc"

	. "github.com/jdginn/arpad/apps/selah/layers/mode"
	reaper "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

const (
	FADER_EPSILON float64 = 0.001
	NUM_CHANNELS  int64   = 8
)

// get returns the first element in the slice for which the predicate returns true.
// If no such element exists, it returns the zero value of T and false.
func get[T any](s []T, predicate func(T) bool) (T, bool) {
	for _, v := range s {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

type trackMapping struct {
	reaperIdx  int64
	surfaceIdx int64
}

type TrackManager struct {
	x             *xtouchlib.XTouchDefault
	r             *reaper.Reaper
	logicalTracks []*TrackData
	trackMappings map[int64]int64
	selectedTrack *TrackData
}

func (m *TrackManager) getTrack(surfaceIdx int64) (*TrackData, bool) {
	reaperIdx, ok := m.trackMappings[surfaceIdx]
	if !ok {
		return &TrackData{}, false
	}
	trackData, ok := get(m.logicalTracks, func(track *TrackData) bool {
		return track.reaperIdx == reaperIdx
	})
	if !ok {
		return &TrackData{}, false
	}
	return trackData, true
}

func (m *TrackManager) AddHardwareTrack(idx int64) {
	// Select
	m.x.Channels[idx].Select.On.Bind(func() (errs error) {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				errs = errors.Join(errs, m.r.Track(m.selectedTrack.reaperIdx).Select.Set(false))
				m.selectedTrack = t
				errs = errors.Join(errs, m.r.Track(t.reaperIdx).Select.Set(true))
			}
		}
		return nil
	})
	// REC
	m.x.Channels[idx].Rec.On.Bind(func() error {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				t.rec = !t.rec
				return m.r.Track(t.reaperIdx).Recarm.Set(t.rec)
			}
		}
		return nil
	})
	// SOLO
	m.x.Channels[idx].Solo.On.Bind(func() error {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				t.solo = !t.solo
				return m.r.Track(t.reaperIdx).Solo.Set(t.solo)
			}
		}
		return nil
	})
	// MUTE
	m.x.Channels[idx].Mute.On.Bind(func() error {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				t.mute = !t.mute
				return m.r.Track(t.reaperIdx).Mute.Set(t.mute)
			}
		}
		return nil
	})
	// Fader
	m.x.Channels[idx].Fader.Bind(func(v uint16) error {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				fmt.Println("Raw value %d", v)
				newVal := intToNormFloat(v)
				fmt.Println("Normalized value %f", newVal)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-t.volume) < FADER_EPSILON {
					t.volume = newVal
					return nil
				}
				t.volume = newVal
				err := m.r.Track(t.reaperIdx).Volume.Set(t.volume)
				if err != nil {
					panic(err)
				}
				return err
			case MIX_SELECTED_TRACK_SENDS:
				// TODO: this indexing is wrong; we need to find send idx from path wildcard values
				newVal := intToNormFloat(v)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-t.sends[idx].volume) < FADER_EPSILON {
					t.sends[idx].volume = newVal
					return nil
				}
				t.sends[idx].volume = newVal
				return m.r.Track(t.reaperIdx).Send(idx).Volume.Set(t.volume)
			}
		}
		return nil
	})
	// Pan
	m.x.Channels[idx].Encoder.Bind(func(v uint8) error {
		if t, ok := m.getTrack(idx); ok {
			switch CurrMode() {
			case MIX:
				t.pan = float64(v) / float64(math.MaxUint8)
				return m.r.Track(t.reaperIdx).Pan.Set(t.pan)
			case MIX_SELECTED_TRACK_SENDS:
				newVal := float64(v) / float64(math.MaxUint8)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				t.pan = newVal
				return m.r.Track(t.reaperIdx).Send(idx).Pan.Set(t.volume)
			}
		}
		return nil
	})
}

func (t *TrackManager) listenForNewTracks() {
	// Find and populate our collection of track states
	//
	// TODO: don't panic here
	if err := t.r.OscDispatcher().AddMsgHandler("/track/@/*", func(m *osc.Message) {
		// TODO: this seems a bit brittle
		idx, err := strconv.ParseInt(m.Arguments[1].(string), 10, 64)
		if err != nil {
			return
		}
		if _, exists := get(t.logicalTracks, func(track *TrackData) bool {
			return track.reaperIdx == idx
		}); !exists {
			t.logicalTracks = append(t.logicalTracks, NewTrackData(
				t,
				idx,
				idx-1,
			))
			t.trackMappings[idx-1] = idx
		}
	}); err != nil {
		panic(err)
	}
	// Update track send info
	if err := t.r.OscDispatcher().AddMsgHandler("/track/@/send/@/*", func(m *osc.Message) {
		trackIdx, err := strconv.ParseInt(m.Arguments[1].(string), 10, 64)
		if err != nil {
			return
		}
		sendIdx, err := strconv.ParseInt(m.Arguments[2].(string), 10, 64)
		if err != nil {
			return
		}
		if _, exists := get(t.logicalTracks, func(track *TrackData) bool {
			return track.reaperIdx == trackIdx
		}); !exists {
			t.logicalTracks = append(t.logicalTracks, NewTrackData(
				t,
				trackIdx,
				trackIdx-1,
			))
		}
		track, _ := get(t.logicalTracks, func(track *TrackData) bool {
			return track.reaperIdx == trackIdx
		})
		track.sends[sendIdx] = NewTrackSendData(track, sendIdx, 0)
	}); err != nil {
		panic(err)
	}
	// Update selected track
	if err := t.r.OscDispatcher().AddMsgHandler("/track/@/select", func(m *osc.Message) {
		trackIdx, err := strconv.ParseInt(m.Arguments[1].(string), 10, 64)
		if err != nil {
			return
		}
		if selectedTarck, ok := get(t.logicalTracks, func(track *TrackData) bool {
			return track.reaperIdx == trackIdx
		}); ok {
			t.selectedTrack = selectedTarck
			// TODO: we need some kind of callback here because we might need to update stuff on xtouch depending on mode
			// TODO: need to update select button LEDs; turn off old selected track if it changes, turn on new one
		}
	}); err != nil {
		panic(err)
	}
}

func NewTrackManager(x *xtouchlib.XTouchDefault, r *reaper.Reaper) *TrackManager {
	t := &TrackManager{
		x:             x,
		r:             r,
		logicalTracks: make([]*TrackData, 0),
		trackMappings: make(map[int64]int64),
	}
	for i := int64(0); i < NUM_CHANNELS; i++ {
		t.AddHardwareTrack(i)
	}
	t.listenForNewTracks()
	return t
}

func (m *TrackManager) TransitionMix() (errs error) {
	for _, track := range m.logicalTracks {
		errs = errors.Join(errs, track.TransitionMix())
	}
	return errs
}

func normFloatToInt(norm float64) uint16 {
	return uint16((norm) * float64(0x4000))
}

func intToNormFloat(val uint16) float64 {
	return float64(val) / float64(0x4000)
}

type TrackData struct {
	x          *xtouchlib.XTouchDefault
	r          *reaper.Reaper
	m          *TrackManager
	surfaceIdx int64
	reaperIdx  int64
	name       string
	volume     float64
	pan        float64
	mute       bool
	solo       bool
	rec        bool
	sends      map[int64]*trackSendData
	rcvs       map[int64]*trackSendData
}

func NewTrackData(m *TrackManager, reaperIdx, surfaceIdx int64) *TrackData {
	t := &TrackData{
		x:          m.x,
		r:          m.r,
		reaperIdx:  reaperIdx,
		surfaceIdx: surfaceIdx,
		sends:      make(map[int64]*trackSendData),
		rcvs:       make(map[int64]*trackSendData),
	}
	// Select
	t.r.Track(t.reaperIdx).Select.Bind(func(v bool) (errs error) {
		switch CurrMode() {
		case MIX:
			// Turn off select button for the previously selected track
			errs = errors.Join(errs, m.x.Channels[m.selectedTrack.surfaceIdx].Select.LED.Set(!v))
			m.selectedTrack = t
			// Turn on select button for the newly selected track
			errs = errors.Join(errs, t.x.Channels[t.surfaceIdx].Select.LED.Set(v))
		}
		return errs
	})
	// REC
	t.r.Track(t.reaperIdx).Recarm.Bind(func(v bool) error {
		t.rec = v
		switch CurrMode() {
		case MIX:
			return t.x.Channels[t.surfaceIdx].Rec.LED.Set(v)
		}
		return nil
	})
	// SOLO
	t.r.Track(t.reaperIdx).Solo.Bind(func(v bool) error {
		t.solo = v
		switch CurrMode() {
		case MIX:
			return t.x.Channels[t.surfaceIdx].Solo.LED.Set(v)
		}
		return nil
	})
	// MUTE
	t.r.Track(t.reaperIdx).Mute.Bind(func(v bool) error {
		t.mute = v
		switch CurrMode() {
		case MIX:
			return t.x.Channels[t.surfaceIdx].Mute.LED.Set(v)
		}
		return nil
	})
	// Fader
	t.r.Track(t.reaperIdx).Volume.Bind(func(v float64) error {
		t.volume = v
		switch CurrMode() {
		case MIX:
			return t.x.Channels[t.surfaceIdx].Fader.Set(normFloatToInt(v))
		}
		return nil
	})
	// Pan
	t.r.Track(t.reaperIdx).Pan.Bind(func(v float64) error {
		t.pan = v
		switch CurrMode() {
		case MIX:
			return t.x.Channels[t.surfaceIdx].Encoder.Ring.Set(v) // TODO: verify
		}
		return nil
	})
	OnTransition(MIX, t.TransitionMix)
	return t
}

func (t *TrackData) TransitionMix() (errs error) {
	xt := t.x.Channels[t.surfaceIdx]
	return errors.Join(errs,
		xt.Fader.Set(normFloatToInt(t.volume)),
		xt.Encoder.Ring.Set(t.pan),
		xt.Mute.LED.Set(t.mute),
		xt.Solo.LED.Set(t.solo),
		xt.Rec.LED.Set(t.rec),
	)
}

type trackSendData struct {
	*TrackData
	sendIdx int64
	rcvIdx  int64
	vol     float64
	pan     float64
}

func NewTrackSendData(parent *TrackData, sendIdx, rcvIdx int64) *trackSendData {
	s := &trackSendData{
		TrackData: parent,
		sendIdx:   sendIdx,
		rcvIdx:    rcvIdx,
	}

	// Volume
	s.r.Track(s.reaperIdx).Send(s.sendIdx).Volume.Bind(func(v float64) error {
		s.vol = v
		return nil
	})
	// TODO: implement send to xtouch

	// Pan
	s.r.Track(s.reaperIdx).Send(s.sendIdx).Pan.Bind(func(v float64) error {
		s.pan = v
		return nil
	})
	// TODO: implement send to xtouch

	return s
}

func (s *trackSendData) OnTransition() (errs error) {
	// TODO: set send/rcv vol on device using s.vol
	// TODO: set send/rcv pan on device using s.pan
	return
}
