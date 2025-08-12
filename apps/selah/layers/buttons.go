package layers

import (
	"errors"

	xtouchlib "github.com/jdginn/arpad/devices/xtouch"

	mode "github.com/jdginn/arpad/apps/selah/modemanager"
)

type EncoderAssign struct {
	*Devices
	*mode.Manager
	currentlyIlluminated *xtouchlib.Button
}

func NewEncoderAssign(m *mode.Manager) *EncoderAssign {
	e := &EncoderAssign{
		Manager: m,
	}

	e.XTouch.EncoderAssign.TRACK.On.Bind(func() error {
		return e.SetMode(mode.MIX)
	})
	e.OnTransition(mode.MIX, func() (errs error) {
		if e.currentlyIlluminated == e.XTouch.EncoderAssign.TRACK {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.XTouch.EncoderAssign.TRACK.LED.On.Set())
		return errs
	})

	e.XTouch.EncoderAssign.PAN_SURROUND.On.Bind(func() error {
		return e.SetMode(mode.MIX_SELECTED_TRACK_SENDS)
	})
	e.OnTransition(mode.MIX_SELECTED_TRACK_SENDS, func() (errs error) {
		if e.currentlyIlluminated == e.XTouch.EncoderAssign.PAN_SURROUND {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.XTouch.EncoderAssign.PAN_SURROUND.LED.On.Set())
		return errs
	})

	return e
}
