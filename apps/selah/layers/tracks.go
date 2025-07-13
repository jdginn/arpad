package layers

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/hypebeast/go-osc/osc"

	reaper "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

type bindable[A any] interface {
	Bind(func(A) error)
}

type setable[T any] interface {
	Set(T) error
}

type bindableSetable[T any] interface {
	bindable[T]
	setable[T]
}

func link[T any](b bindable[T], s setable[T]) {
	b.Bind(func(v T) error { return s.Set(v) })
}

func link2[T any](x, y bindableSetable[T]) {
	x.Bind(func(v T) error { return y.Set(v) })
	y.Bind(func(v T) error { return x.Set(v) })
}

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
	x      *xtouchlib.XTouchDefault
	r      *reaper.Reaper
	tracks []*TrackData
}

func (t *TrackManager) listenForNewTracks() {
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
			newTrack := NewTrackData(
				t.x,
				t.r,
				idx,
			)

			// REC
			t.r.Track(idx).Recarm.Bind(func(v bool) error {
				newTrack.rec = v
				return t.x.Channels[newTrack.surfaceIdx].Solo.SetLED(v)
			})
			t.x.Channels[newTrack.surfaceIdx].Rec.On.Bind(func(v uint8) error {
				newTrack.rec = !newTrack.rec
				return t.r.Track(idx).Recarm.Set(newTrack.rec)
			})
			// SOLO
			t.r.Track(idx).Solo.Bind(func(v bool) error {
				newTrack.solo = v
				return t.x.Channels[newTrack.surfaceIdx].Solo.SetLED(v)
			})
			t.x.Channels[newTrack.surfaceIdx].Solo.On.Bind(func(v uint8) error {
				newTrack.solo = !newTrack.solo
				return t.r.Track(idx).Solo.Set(newTrack.solo)
			})
			// MUTE
			t.r.Track(idx).Mute.Bind(func(v bool) error {
				newTrack.mute = v
				return t.x.Channels[newTrack.surfaceIdx].Mute.SetLED(v)
			})
			t.x.Channels[newTrack.surfaceIdx].Mute.On.Bind(func(v uint8) error {
				newTrack.mute = !newTrack.mute
				return t.r.Track(idx).Mute.Set(newTrack.rec)
			})
			// Fader
			t.r.Track(idx).Volume.Bind(func(v float64) error {
				newTrack.volume = v
				return t.x.Channels[newTrack.surfaceIdx].Fader.Set(normFloatToInt(v))
			})
			t.x.Channels[newTrack.surfaceIdx].Fader.Bind(func(v int16) error {
				newTrack.volume = int16ToNormFloat(v)
				fmt.Printf("fader %d -> volume %f\n", v, newTrack.volume)
				return t.r.Track(idx).Volume.Set(newTrack.volume)
			})
			// Pan
			t.r.Track(idx).Pan.Bind(func(v float64) error {
				newTrack.pan = v
				return t.x.Channels[newTrack.surfaceIdx].Encoder.Ring.Set(v) // TODO: verify
			})
			t.x.Channels[newTrack.surfaceIdx].Encoder.Ring.Bind(func(v uint8) error {
				newTrack.pan = float64(v) / float64(math.MaxUint8)
				return t.r.Track(idx).Pan.Set(newTrack.pan)
			})

			t.tracks = append(t.tracks, newTrack)
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

func (t *TrackManager) TransitionMix() error {
	var err error
	for _, track := range t.tracks {
		err = errors.Join(err, track.TransitionMix())
	}
	return err
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
	sends      map[int]*trackSendData
	rcvs       map[int]*trackSendData
}

func NewTrackData(x *xtouchlib.XTouchDefault, r *reaper.Reaper, reaperIdx int64) *TrackData {
	t := &TrackData{
		x:         x,
		r:         r,
		reaperIdx: reaperIdx,
		sends:     make(map[int]*trackSendData),
		rcvs:      make(map[int]*trackSendData),
	}
	return t
}

func (t *TrackData) TransitionMix() (errs error) {
	xt := t.x.Channels[t.surfaceIdx]
	return errors.Join(errs,
		xt.Fader.Set(normFloatToInt(t.volume)),
		xt.Encoder.Ring.Set(t.pan),
		xt.Mute.SetLED(t.mute),
		xt.Solo.SetLED(t.solo),
		xt.Rec.SetLED(t.rec),
	)
}

type trackSendData struct {
	*TrackData
	sendIdx uint64
	rcvIdx  uint64
	vol     float64
	pan     float64

	Vol *trackSendDataVol
	Pan *trackSendDataPan
}

func NewTrackSendData(parent *TrackData, sendIdx, rcvIdx uint64) *trackSendData {
	s := &trackSendData{
		TrackData: parent,
		sendIdx:   sendIdx,
		rcvIdx:    rcvIdx,
	}
	s.Vol = &trackSendDataVol{trackSendData: s}
	s.Pan = &trackSendDataPan{trackSendData: s}
	return s
}

type trackSendDataVol struct{ *trackSendData }

func (v *trackSendDataVol) Set(val float64) error {
	v.vol = val
	// TODO: implement vol setting logic for send/rcv
	return nil
}

type trackSendDataPan struct{ *trackSendData }

func (p *trackSendDataPan) Set(val float64) error {
	p.pan = val
	// TODO: implement pan setting logic for send/rcv
	return nil
}

func (s *trackSendData) OnTransition() (errs error) {
	// TODO: set send/rcv vol on device using s.vol
	// TODO: set send/rcv pan on device using s.pan
	return
}
