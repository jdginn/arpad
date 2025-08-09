package layers

import (
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"

	reaper "github.com/jdginn/arpad/devices/reaper"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
	"github.com/jdginn/arpad/logging"
)

var appLog *slog.Logger

func init() {
	appLog = logging.Get(logging.APP)
}

const (
	FADER_EPSILON float64 = 0.001
	NUM_CHANNELS  int64   = 8
)

type GUID = string

type trackMapping struct {
	reaperIdx  int64
	surfaceIdx int64
}

type mapper struct {
	mux                *sync.Mutex
	guidToSurfaceIndex map[GUID]int64
	surfaceIndexToGuid map[int64]GUID
}

func NewMapper() *mapper {
	return &mapper{
		mux:                &sync.Mutex{},
		guidToSurfaceIndex: make(map[GUID]int64),
		surfaceIndexToGuid: make(map[int64]GUID),
	}
}

func (m *mapper) AddGuid(guid GUID) *mappingGuid {
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.guidToSurfaceIndex[guid]; !exists {
		appLog.Info("Adding GUID to mapper", slog.String("guid", guid))
		idx := int64(len(m.guidToSurfaceIndex))
		m.guidToSurfaceIndex[guid] = idx
		m.surfaceIndexToGuid[idx] = guid
	}
	return &mappingGuid{m, guid}
}

func (m *mapper) ByGuid(guid GUID) *mappingGuid {
	return &mappingGuid{m, guid}
}

func (m *mapper) BySurfIdx(idx int64) *mappingSurfaceIdx {
	return &mappingSurfaceIdx{m, idx}
}

type mappingGuid struct {
	*mapper
	guid GUID
}

func (m *mappingGuid) MaybeSurfIdx() (int64, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if surfaceIdx, ok := m.guidToSurfaceIndex[m.guid]; ok {
		return surfaceIdx, true
	}
	return 0, false
}

func (m *mappingGuid) SurfIdx() int64 {
	if surfaceIdx, ok := m.guidToSurfaceIndex[m.guid]; ok {
		return surfaceIdx
	}
	panic("mappingGuid: no surface index found for guid " + m.guid)
}

func (m *mappingGuid) SetSurfIdx(idx int64) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.surfaceIndexToGuid, idx)
	m.guidToSurfaceIndex[m.guid] = idx
	m.surfaceIndexToGuid[idx] = m.guid
}

type mappingSurfaceIdx struct {
	*mapper
	idx int64
}

func (m *mappingSurfaceIdx) MaybeGuid() (GUID, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if guid, ok := m.surfaceIndexToGuid[m.idx]; ok {
		return guid, true
	}
	return "", false
}

func (m *mappingSurfaceIdx) Guid() GUID {
	if guid, ok := m.surfaceIndexToGuid[m.idx]; ok {
		return guid
	}
	panic("mappingSurfaceIdx: no guid found for surface index " + string(m.idx))
}

func (m *mappingSurfaceIdx) SetGuid(guid GUID) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.surfaceIndexToGuid[m.idx] = guid
	m.guidToSurfaceIndex[guid] = m.idx
}

type TrackManager struct {
	*Manager
	*mapper
	tracks        map[GUID]*TrackData
	selectedTrack *TrackData
}

func (m *TrackManager) getTrackAtIdx(idx int64) (*TrackData, bool) {
	guid, ok := m.BySurfIdx(idx).MaybeGuid()
	if !ok {
		return nil, false
	}
	m.mux.Lock()
	defer m.mux.Unlock()
	track, ok := m.tracks[guid]
	return track, ok
}

func (m *TrackManager) AddHardwareTrack(idx int64) {
	// Select
	m.x.Channels[idx].Select.On.Bind(func() (errs error) {
		if track, ok := m.getTrackAtIdx(idx); ok {
			if _, ok := m.BySurfIdx(idx).MaybeGuid(); !ok {
				return nil
			}
			switch m.CurrMode() {
			case MIX:
				if m.selectedTrack != nil {
					errs = errors.Join(errs, m.r.Track(m.selectedTrack.guid).Selected.Set(false))
				}
				m.selectedTrack = track
				errs = errors.Join(errs, m.r.Track(m.BySurfIdx(idx).Guid()).Selected.Set(true))
				return errs
			}
		}
		return nil
	})
	// REC
	m.x.Channels[idx].Rec.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			if _, ok := m.BySurfIdx(idx).MaybeGuid(); !ok {
				return nil
			}
			switch m.CurrMode() {
			case MIX:
				track.rec = !track.rec
				return m.r.Track(m.BySurfIdx(idx).Guid()).Recarm.Set(track.rec)
			}
		}
		return nil
	})
	// SOLO
	m.x.Channels[idx].Solo.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			if _, ok := m.BySurfIdx(idx).MaybeGuid(); !ok {
				return nil
			}
			switch m.CurrMode() {
			case MIX:
				track.solo = !track.solo
				return m.r.Track(m.BySurfIdx(idx).Guid()).Solo.Set(track.solo)
			}
		}
		return nil
	})
	// MUTE
	m.x.Channels[idx].Mute.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			if _, ok := m.BySurfIdx(idx).MaybeGuid(); !ok {
				return nil
			}
			switch m.CurrMode() {
			case MIX:
				track.mute = !track.mute
				return m.r.Track(m.BySurfIdx(idx).Guid()).Mute.Set(track.mute)
			case MIX_SELECTED_TRACK_SENDS:
			}
		}
		return nil
	})
	// Fader
	m.x.Channels[idx].Fader.Bind(func(v uint16) error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			if _, ok := m.BySurfIdx(idx).MaybeGuid(); !ok {
				return nil
			}
			switch m.CurrMode() {
			case MIX:
				newVal := intToNormFloat(v)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-track.volume) < FADER_EPSILON {
					track.volume = newVal
					return nil
				}
				track.volume = newVal
				err := m.r.Track(m.BySurfIdx(idx).Guid()).Volume.Set(track.volume)
				if err != nil {
					panic(err)
				}
				return err
			case MIX_SELECTED_TRACK_SENDS:
				newVal := intToNormFloat(v)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-track.sends[idx].volume) < FADER_EPSILON {
					track.sends[idx].volume = newVal
					return nil
				}
				track.sends[idx].volume = newVal
				return m.r.Track(m.selectedTrack.guid).Send(idx).Volume.Set(track.volume)
			}
		}
		return nil
	})
	// Pan
	m.x.Channels[idx].Encoder.Bind(func(v uint8) error {
		if t, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case MIX:
				t.pan = float64(v) / float64(math.MaxUint8)
				return m.r.Track(m.BySurfIdx(idx).Guid()).Pan.Set(t.pan)
			case MIX_SELECTED_TRACK_SENDS:
				t.pan = float64(v) / float64(math.MaxUint8)
				return m.r.Track(m.selectedTrack.guid).Send(idx).Pan.Set(t.volume)
			}
		}
		return nil
	})
}

func (t *TrackManager) listenForNewTracks() {
	// Find and populate our collection of track states
	// TODO: we need to do a little custom handling for the master track to make sure it gets mapped to fader 9
	if err := t.r.OscDispatcher().AddMsgHandler("/track/*", func(msg *osc.Message) {
		segments := strings.Split(msg.Address, "/")
		guid := GUID(segments[2])
		t.mapper.AddGuid(guid)
		if _, exists := t.tracks[guid]; !exists {
			t.tracks[guid] = NewTrackData(t, guid)
		}
	}); err != nil {
		appLog.Error(err.Error())
	}
}

func NewTrackManager(m *Manager) *TrackManager {
	t := &TrackManager{
		Manager: m,
		mapper:  NewMapper(),
		tracks:  make(map[GUID]*TrackData),
	}
	t.listenForNewTracks()
	return t
}

func (m *TrackManager) TransitionMix() (errs error) {
	// for _, track := range m.logicalTracks {
	// 	errs = errors.Join(errs, track.TransitionMix())
	// }
	return errs
}

func normFloatToInt(norm float64) uint16 {
	return uint16((norm) * float64(0x4000))
}

func intToNormFloat(val uint16) float64 {
	return float64(val) / float64(0x4000)
}

type TrackData struct {
	x      *xtouchlib.XTouchDefault
	r      *reaper.Reaper
	m      *TrackManager
	guid   GUID
	name   string
	volume float64
	pan    float64
	mute   bool
	solo   bool
	rec    bool
	sends  map[int64]*trackSendData
	rcvs   map[int64]*trackSendData
}

func NewTrackData(m *TrackManager, guid GUID) *TrackData {
	t := &TrackData{
		x:     m.x,
		r:     m.r,
		guid:  guid,
		sends: make(map[int64]*trackSendData),
		rcvs:  make(map[int64]*trackSendData),
	}
	t.r.Track(guid).Index.Bind(func(idx int64) error {
		m.ByGuid(guid).SetSurfIdx(idx - 1)
		return nil
	})
	// Track name to scribble strip
	//
	// TODO: how do we get color? What do we put on the bottom line?
	// TODO: how do we truncate names?
	t.r.Track(guid).Name.Bind(func(v string) error {
		appLog.Debug("Track name changed", slog.String("guid", guid), slog.String("name", v))
		t.name = v
		switch m.CurrMode() {
		case MIX:
			return m.x.Channels[m.ByGuid(guid).SurfIdx()].Scribble.
				WithColor(xtouchlib.White).
				WithTopMessage(t.name).
				WithBottomMessage("").
				Set()
		}
		return nil
	})
	// Select
	t.r.Track(guid).Selected.Bind(func(v bool) (errs error) {
		appLog.Debug("Track selected changed", slog.String("guid", guid), slog.Bool("selected", v))
		switch m.CurrMode() {
		case MIX:
			// Turn off select button for the previously selected track
			errs = errors.Join(errs, m.x.Channels[m.ByGuid(m.selectedTrack.guid).SurfIdx()].
				Select.LED.Set(!v))
			m.selectedTrack = t
			// Turn on select button for the newly selected track
			errs = errors.Join(errs, t.x.Channels[m.ByGuid(guid).SurfIdx()].
				Select.LED.Set(v))
		}
		return errs
	})
	// REC
	t.r.Track(guid).Recarm.Bind(func(v bool) error {
		appLog.Debug("Track recarm changed", slog.String("guid", guid), slog.Bool("recarm", v))
		t.rec = v
		switch m.CurrMode() {
		case MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].
				Rec.LED.Set(v)
		}
		return nil
	})
	// SOLO
	t.r.Track(guid).Solo.Bind(func(v bool) error {
		appLog.Debug("Track solo changed", slog.String("guid", guid), slog.Bool("solo", v))
		t.solo = v
		switch m.CurrMode() {
		case MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].
				Solo.LED.Set(v)
		}
		return nil
	})
	// MUTE
	t.r.Track(guid).Mute.Bind(func(v bool) error {
		appLog.Debug("Track mute changed", slog.String("guid", guid), slog.Bool("mute", v))
		t.mute = v
		switch m.CurrMode() {
		case MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].
				Mute.LED.Set(v)
		}
		return nil
	})
	// Fader
	t.r.Track(guid).Volume.Bind(func(v float64) error {
		t.volume = v
		switch m.CurrMode() {
		case MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].Fader.Set(normFloatToInt(v))
		}
		return nil
	})
	// Pan
	t.r.Track(guid).Pan.Bind(func(v float64) error {
		t.pan = v
		switch m.CurrMode() {
		case MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].Encoder.Ring.Set(v) // TODO: verify
		}
		return nil
	})
	m.OnTransition(MIX, t.TransitionMix)
	return t
}

func (t *TrackData) TransitionMix() (errs error) {
	xt := t.x.Channels[t.m.ByGuid(t.guid).SurfIdx()]
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
	sendIdx  int64 // index of this send into this track's sends
	sendGuid GUID  // GUID of the track to which we are sending
	vol      float64
	pan      float64
}

func NewTrackSendData(parent *TrackData, sendIdx, rcvIdx int64) *trackSendData {
	s := &trackSendData{
		TrackData: parent,
		sendIdx:   sendIdx,
	}

	s.r.Track(s.guid).Send(s.sendIdx).Guid.Bind(func(guid GUID) error {
		s.sendGuid = guid
		return nil
	})

	s.r.Track(s.guid).Send(s.sendIdx).Volume.Bind(func(v float64) error {
		s.vol = v
		return nil
	})
	// TODO: implement send to xtouch

	// Pan
	s.r.Track(s.guid).Send(s.sendIdx).Pan.Bind(func(v float64) error {
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
