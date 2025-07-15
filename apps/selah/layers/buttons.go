package layers

import (
	"errors"

	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

type EncoderAssign struct {
	*Manager
	currentlyIlluminated *xtouchlib.Button
}

func NewEncoderAssign(m *Manager) *EncoderAssign {
	e := &EncoderAssign{
		Manager: m,
	}

	e.x.EncoderAssign.TRACK.On.Bind(func() error {
		return e.SetMode(MIX)
	})
	e.OnTransition(MIX, func() (errs error) {
		if e.currentlyIlluminated == e.x.EncoderAssign.TRACK {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.x.EncoderAssign.TRACK.LED.On.Set())
		return errs
	})

	e.x.EncoderAssign.PAN_SURROUND.On.Bind(func() error {
		return e.SetMode(MIX_SELECTED_TRACK_SENDS)
	})
	e.OnTransition(MIX_SELECTED_TRACK_SENDS, func() (errs error) {
		if e.currentlyIlluminated == e.x.EncoderAssign.PAN_SURROUND {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.x.EncoderAssign.PAN_SURROUND.LED.On.Set())
		return errs
	})

	return e
}
