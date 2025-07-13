package layers

import (
	"errors"
	"math"
	"strconv"

	"github.com/hypebeast/go-osc/osc"

	. "github.com/jdginn/arpad/apps/selah/layers/mode"
	reaper "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

const (
	FADER_EPSILON float64 = 0.001
	NUM_CHANNELS  int     = 8
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

type TrackManager struct {
	x             *xtouchlib.XTouchDefault
	r             *reaper.Reaper
	tracks        []*TrackData
	selectedTrack *TrackData
}

func (t *TrackManager) listenForNewTracks() {
	// TODO: don't panic here
	// Find and populate our collection of track states
	if err := t.r.OscDispatcher().AddMsgHandler("/track/@/*", func(m *osc.Message) {
		// TODO: this seems a bit brittle
		idx, err := strconv.ParseInt(m.Arguments[1].(string), 10, 64)
		if err != nil {
			return
		}
		if _, exists := get(t.tracks, func(track *TrackData) bool {
			return track.reaperIdx == idx
		}); !exists {
			t.tracks = append(t.tracks, NewTrackData(
				t.x,
				t.r,
				idx,
				idx,
			))
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
		if _, exists := get(t.tracks, func(track *TrackData) bool {
			return track.reaperIdx == trackIdx
		}); !exists {
			t.tracks = append(t.tracks, NewTrackData(
				t.x,
				t.r,
				trackIdx,
				trackIdx,
			))
		}
		track, _ := get(t.tracks, func(track *TrackData) bool {
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
		if selectedTarck, ok := get(t.tracks, func(track *TrackData) bool {
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
		x:      x,
		r:      r,
		tracks: make([]*TrackData, 0),
	}
	t.listenForNewTracks()
	return t
}

func (m *TrackManager) TransitionMix() (errs error) {
	for _, track := range m.tracks {
		errs = errors.Join(errs, track.TransitionMix())
	}
	return errs
}

// TODO: verify this
func normFloatToInt(norm float64) int16 {
	return int16((norm)*float64(0x8000)) - 0x4000
}

func int16ToNormFloat(val int16) float64 {
	return float64(val) / float64(0x4000)
}

type TrackData struct {
	x          *xtouchlib.XTouchDefault
	r          *reaper.Reaper
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

func NewTrackData(x *xtouchlib.XTouchDefault, r *reaper.Reaper, reaperIdx, surfaceIdx int64) *TrackData {
	t := &TrackData{
		x:          x,
		r:          r,
		reaperIdx:  reaperIdx,
		surfaceIdx: surfaceIdx,
		sends:      make(map[int64]*trackSendData),
		rcvs:       make(map[int64]*trackSendData),
	}
	// REC
	t.r.Track(t.reaperIdx).Recarm.Bind(func(v bool) error {
		t.rec = v
		return t.x.Channels[t.surfaceIdx].Solo.LED.Set(v)
	})
	FilterBind(MIX, t.x.Channels[t.surfaceIdx].Rec.On).Bind(func(v uint8) error {
		t.rec = !t.rec
		return t.r.Track(t.reaperIdx).Recarm.Set(t.rec)
	})
	// SOLO
	t.r.Track(t.reaperIdx).Solo.Bind(func(v bool) error {
		t.solo = v
		return t.x.Channels[t.surfaceIdx].Solo.LED.Set(v)
	})
	FilterBind(MIX, t.x.Channels[t.surfaceIdx].Solo.On).Bind(func(v uint8) error {
		t.solo = !t.solo
		return t.r.Track(t.reaperIdx).Solo.Set(t.solo)
	})
	// MUTE
	t.r.Track(t.reaperIdx).Mute.Bind(func(v bool) error {
		t.mute = v
		return t.x.Channels[t.surfaceIdx].Mute.LED.Set(v)
	})
	FilterBind(MIX, t.x.Channels[t.surfaceIdx].Mute.On).Bind(func(v uint8) error {
		t.mute = !t.mute
		return t.r.Track(t.reaperIdx).Mute.Set(t.rec)
	})
	// Fader
	t.r.Track(t.reaperIdx).Volume.Bind(func(v float64) error {
		t.volume = v
		return t.x.Channels[t.surfaceIdx].Fader.Set(normFloatToInt(v))
	})
	FilterBind(MIX, t.x.Channels[t.surfaceIdx].Fader).Bind(func(v int16) error {
		newVal := int16ToNormFloat(v)
		// Because both feedback and input are implemented on the same physical control for fader,
		// we need some deduplication to avoid jittering the faders or flooding the system with
		// echoing messages.
		if math.Abs(newVal-t.volume) < FADER_EPSILON {
			t.volume = newVal
			return nil
		}
		t.volume = newVal
		return t.r.Track(t.reaperIdx).Volume.Set(t.volume)
	})
	// Pan
	t.r.Track(t.reaperIdx).Pan.Bind(func(v float64) error {
		t.pan = v
		return t.x.Channels[t.surfaceIdx].Encoder.Ring.Set(v) // TODO: verify
	})
	FilterBind(MIX, t.x.Channels[t.surfaceIdx].Encoder.Ring).Bind(func(v uint8) error {
		t.pan = float64(v) / float64(math.MaxUint8)
		return t.r.Track(t.reaperIdx).Pan.Set(t.pan)
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
