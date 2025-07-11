package layers

import (
	"errors"
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

func link[T any](b bindable[T], s setable[T]) {
	b.Bind(func(v T) error { return s.Set(v) })
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
	tracks []*TrackData
}

func (t *TrackManager) listenForNewTracks(reaper *reaper.Reaper) {
	// Find and populate our collection of track states
	if err := reaper.OscDispatcher().AddMsgHandler("/track/@/*", func(m *osc.Message) {
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
				idx,
			)
			link(reaper.Track(idx).Volume.Db, newTrack.Volume)
			link(reaper.Track(idx).Pan, newTrack.Pan)
			link(reaper.Track(idx).Mute, newTrack.Mute)
			link(reaper.Track(idx).Solo, newTrack.Solo)
			link(reaper.Track(idx).Recarm, newTrack.Rec)
			t.tracks = append(t.tracks, newTrack)
		}
	}); err != nil {
		panic(err)
	}
}

func NewTrackManager(x *xtouchlib.XTouchDefault, reaper *reaper.Reaper) *TrackManager {
	t := &TrackManager{
		x:      x,
		tracks: make([]*TrackData, 0),
	}
	t.listenForNewTracks(reaper)
	return t
}

func (t *TrackManager) TransitionMix() error {
	var err error
	for _, track := range t.tracks {
		err = errors.Join(err, track.TransitionMix())
	}
	return err
}

func normFloatToInt(norm float64) int16 {
	return int16((norm - 0.5) * float64(math.MaxInt16))
}

type TrackData struct {
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

	Volume *trackDataVolume
	Pan    *trackDataPan
	Mute   *trackDataMute
	Solo   *trackDataSolo
	Rec    *trackDataRec
}

func NewTrackData(x *xtouchlib.XTouchDefault, reaperIdx int64) *TrackData {
	t := &TrackData{
		x:         x,
		reaperIdx: reaperIdx,
		sends:     make(map[int]*trackSendData),
		rcvs:      make(map[int]*trackSendData),
	}
	t.Volume = &trackDataVolume{TrackData: t}
	t.Pan = &trackDataPan{TrackData: t}
	t.Mute = &trackDataMute{TrackData: t}
	t.Solo = &trackDataSolo{TrackData: t}
	t.Rec = &trackDataRec{TrackData: t}
	return t
}

type trackDataVolume struct{ *TrackData }

func (v *trackDataVolume) Set(val float64) error {
	v.volume = val
	v.x.Channels[v.surfaceIdx].Fader.Set(normFloatToInt(val))
	return nil
}

type trackDataPan struct{ *TrackData }

func (p *trackDataPan) Set(val float64) error {
	p.pan = val
	p.x.Channels[p.surfaceIdx].Encoder.Ring.Set(val)
	return nil
}

type trackDataMute struct{ *TrackData }

func (m *trackDataMute) Set(val bool) error {
	m.mute = val
	m.x.Channels[m.surfaceIdx].Mute.SetLED(val)
	return nil
}

type trackDataSolo struct{ *TrackData }

func (s *trackDataSolo) Set(val bool) error {
	s.solo = val
	s.x.Channels[s.surfaceIdx].Solo.SetLED(val)
	return nil
}

type trackDataRec struct{ *TrackData }

func (r *trackDataRec) Set(val bool) error {
	r.rec = val
	r.x.Channels[r.surfaceIdx].Rec.SetLED(val)
	return nil
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
