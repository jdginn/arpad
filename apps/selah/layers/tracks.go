package layers

import (
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"

	"github.com/jdginn/arpad/devices/reaper"
	"github.com/jdginn/arpad/devices/xtouch"
	"github.com/jdginn/arpad/logging"

	"github.com/jdginn/arpad/apps/selah/mapper"
	mode "github.com/jdginn/arpad/apps/selah/modemanager"
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

type Devices struct {
	XTouch *xtouch.XTouchDefault
	Reaper *reaper.Reaper
}

type TrackManager struct {
	*Devices
	*mode.Manager
	*mapper.Mapper

	mux           sync.RWMutex
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

func (t *TrackManager) addTrack(guid GUID) {
	appLog.Info("Adding track", slog.String("guid", guid), slog.String("name", guid))
	t.AddGuid(guid)
	t.mux.Lock()
	defer t.mux.Unlock()
	if _, exists := t.tracks[guid]; !exists {
		t.tracks[guid] = NewTrackData(t, guid)
	}
}

func (m *TrackManager) resetSurfaceChannel(idx int64) {
	xt := m.XTouch.Channels[idx]
	err := xt.Fader.Set(0)
	err = errors.Join(err, xt.Encoder.Ring.Set(0))
	err = errors.Join(err, xt.Mute.LED.Set(false))
	err = errors.Join(err, xt.Solo.LED.Set(false))
	err = errors.Join(err, xt.Rec.LED.Set(false))
	err = errors.Join(err, xt.Select.LED.Set(false))
	err = errors.Join(err, xt.Scribble.ChangeTopMessage("Track").ChangeBottomMessage("Name").ChangeColor(xtouch.Off).Set())
	if err != nil {
		appLog.Error("Error resetting surface", slog.Int64("idx", idx), slog.Any("error", err))
	} else {
		appLog.Debug("Reset surface", slog.Int64("idx", idx))
	}
}

func (t *TrackManager) deleteTrack(guid GUID) {
	t.mux.Lock()
	defer t.mux.Unlock()
	if track, exists := t.tracks[guid]; exists {
		appLog.Info("Deleting track", slog.String("guid", guid), slog.String("name", track.name))
		t.DeleteGuid(guid)
		delete(t.tracks, guid)
	}
	// If after deleting the track, we have fewer than NUM_CHANNELS tracks, reset any channels on the surface that
	// are no longer in use.
	for idx := int64(0); idx < NUM_CHANNELS; idx++ {
		if _, ok := t.BySurfIdx(idx).MaybeGuid(); !ok {
			appLog.Debug("Resetting surface for deleted track", slog.Int64("idx", idx), slog.String("guid", guid))
			t.resetSurfaceChannel(idx)
		}
	}
}

func (t *TrackManager) listenForNewTracks() {
	// Find and populate our collection of track states
	// TODO: we need to do a little custom handling for the master track to make sure it gets mapped to fader 9
	t.Reaper.OscDispatcher().AddMsgHandler("/track/*", func(msg *osc.Message) {
		segments := strings.Split(msg.Address, "/")
		guid := GUID(segments[2])
		if len(segments) == 3 && segments[2] == "delete" {
			t.deleteTrack(guid)
			return
		}
		t.addTrack(guid)
	})
}

func (m *TrackManager) AddHardwareTrack(idx int64) {
	// Select
	m.XTouch.Channels[idx].Select.On.Bind(func() (errs error) {
		if track, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				if m.selectedTrack != nil {
					errs = errors.Join(errs, m.Reaper.Track(m.selectedTrack.guid).Selected.Set(false))
				}
				m.selectedTrack = track
				errs = errors.Join(errs, m.Reaper.Track(m.BySurfIdx(idx).Guid()).Selected.Set(true))
				return errs
			}
		}
		return nil
	})
	// REC
	m.XTouch.Channels[idx].Rec.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				track.rec = !track.rec
				return m.Reaper.Track(m.BySurfIdx(idx).Guid()).Recarm.Set(track.rec)
			}
		}
		return nil
	})
	// SOLO
	m.XTouch.Channels[idx].Solo.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				track.solo = !track.solo
				return m.Reaper.Track(m.BySurfIdx(idx).Guid()).Solo.Set(track.solo)
			}
		}
		return nil
	})
	// MUTE
	m.XTouch.Channels[idx].Mute.On.Bind(func() error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				track.mute = !track.mute
				return m.Reaper.Track(m.BySurfIdx(idx).Guid()).Mute.Set(track.mute)
			case mode.MIX_SELECTED_TRACK_SENDS:
			}
		}
		return nil
	})
	// Fader
	m.XTouch.Channels[idx].Fader.Bind(func(v uint16) error {
		if track, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				newVal := intToNormFloat(v)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-track.volume) < FADER_EPSILON {
					track.volume = newVal
					return nil
				}
				track.volume = newVal
				err := m.Reaper.Track(m.BySurfIdx(idx).Guid()).Volume.Set(track.volume)
				if err != nil {
					panic(err)
				}
				return err
			case mode.MIX_SELECTED_TRACK_SENDS:
				newVal := intToNormFloat(v)
				// Because both feedback and input are implemented on the same physical control for fader,
				// we need some deduplication to avoid jittering the faders or flooding the system with
				// echoing messages.
				if math.Abs(newVal-track.sends[idx].volume) < FADER_EPSILON {
					track.sends[idx].volume = newVal
					return nil
				}
				track.sends[idx].volume = newVal
				return m.Reaper.Track(m.selectedTrack.guid).Send(idx).Volume.Set(track.volume)
			}
		}
		return nil
	})
	// Pan
	m.XTouch.Channels[idx].Encoder.Bind(func(v uint8) error {
		if t, ok := m.getTrackAtIdx(idx); ok {
			switch m.CurrMode() {
			case mode.MIX:
				t.pan = float64(v) / float64(math.MaxUint8)
				return m.Reaper.Track(m.BySurfIdx(idx).Guid()).Pan.Set(t.pan)
			case mode.MIX_SELECTED_TRACK_SENDS:
				t.pan = float64(v) / float64(math.MaxUint8)
				return m.Reaper.Track(m.selectedTrack.guid).Send(idx).Pan.Set(t.volume)
			}
		}
		return nil
	})
}

func NewTrackManager(d Devices, m *mode.Manager) *TrackManager {
	t := &TrackManager{
		Devices:       &d,
		Manager:       m,
		Mapper:        mapper.NewMapper(),
		mux:           sync.RWMutex{},
		tracks:        make(map[GUID]*TrackData),
		selectedTrack: &TrackData{},
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
	x      *xtouch.XTouchDefault
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
		x:     m.XTouch,
		r:     m.Reaper,
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
		case mode.MIX:
			return m.XTouch.Channels[m.ByGuid(guid).SurfIdx()].Scribble.
				ChangeTopMessage(t.name).
				Set()
		}
		return nil
	})
	t.r.Track(guid).Color.Bind(func(v int64) error {
		appLog.Debug("Track color changed", slog.String("guid", guid), slog.Int64("color", v))
		switch m.CurrMode() {
		case mode.MIX:
			return m.XTouch.Channels[m.ByGuid(guid).SurfIdx()].Scribble.ChangeColor(xtouch.Red).Set() // TODO: need to get colors from v
		}
		return nil
	})

	// Select
	t.r.Track(guid).Selected.Bind(func(v bool) (errs error) {
		appLog.Debug("Track selected changed", slog.String("guid", guid), slog.Bool("selected", v))
		switch m.CurrMode() {
		case mode.MIX:
			// Turn off select button for the previously selected track
			errs = errors.Join(errs, m.XTouch.Channels[m.ByGuid(m.selectedTrack.guid).SurfIdx()].
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
		case mode.MIX:
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
		case mode.MIX:
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
		case mode.MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].
				Mute.LED.Set(v)
		}
		return nil
	})
	// Fader
	t.r.Track(guid).Volume.Bind(func(v float64) error {
		t.volume = v
		switch m.CurrMode() {
		case mode.MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].Fader.Set(normFloatToInt(v))
		}
		return nil
	})
	// Pan
	t.r.Track(guid).Pan.Bind(func(v float64) error {
		t.pan = v
		switch m.CurrMode() {
		case mode.MIX:
			return t.x.Channels[m.ByGuid(guid).SurfIdx()].Encoder.Ring.Set(v) // TODO: verify
		}
		return nil
	})
	m.OnTransition(mode.MIX, t.TransitionMix)
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
