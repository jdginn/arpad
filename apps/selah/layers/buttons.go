package layers

import (
	"errors"

	. "github.com/jdginn/arpad/apps/selah/layers/mode"
	"github.com/jdginn/arpad/devices/xtouch"
	xtouchlib "github.com/jdginn/arpad/devices/xtouch"
)

type EncoderAssign struct {
	x                    *xtouchlib.XTouchDefault
	currentlyIlluminated *xtouchlib.Button
}

func NewEncoderAssign(x *xtouch.XTouchDefault) *EncoderAssign {
	e := &EncoderAssign{
		x: x,
	}

	e.x.EncoderAssign.TRACK.On.Bind(func() error {
		return SetMode(MIX)
	})
	OnTransition(MIX, func() (errs error) {
		if e.currentlyIlluminated == e.x.EncoderAssign.TRACK {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.x.EncoderAssign.TRACK.LED.On.Set())
		return errs
	})

	e.x.EncoderAssign.PAN_SURROUND.On.Bind(func() error {
		return SetMode(MIX_SELECTED_TRACK_SENDS)
	})
	OnTransition(MIX_SELECTED_TRACK_SENDS, func() (errs error) {
		if e.currentlyIlluminated == e.x.EncoderAssign.PAN_SURROUND {
			return
		}
		errs = errors.Join(errs, e.currentlyIlluminated.LED.Off.Set())
		errs = errors.Join(errs, e.x.EncoderAssign.PAN_SURROUND.LED.On.Set())
		return errs
	})

	return e
}
