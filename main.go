package main

import (
	"fmt"
	"math"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"

	"github.com/jdginn/arpad/devices"
	"github.com/jdginn/arpad/devices/motu"
	"github.com/jdginn/arpad/devices/reaper"
	"github.com/jdginn/arpad/devices/xtouch"
)

type Mode int

const (
	RECORD Mode = iota
	MIX
)

type element[T devices.BaseTypes] struct {
	value   T
	actions []devices.Effect[T]
}

type layer struct {
	ints    map[string]element[int64]
	floats  map[string]element[float64]
	strings map[string]element[string]
	bools   map[string]element[bool]
}

func newLayer() layer {
	return layer{
		ints:    map[string]element[int64]{},
		floats:  map[string]element[float64]{},
		strings: map[string]element[string]{},
		bools:   map[string]element[bool]{},
	}
}

type Context struct {
	currMode Mode
	modes    map[Mode]layer
}

func (c *Context) SetMode(mode Mode) {
	c.currMode = mode
	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
	for _, e := range c.modes[c.currMode].ints {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].floats {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].strings {
		for _, a := range e.actions {
			a(e.value)
		}
	}
	for _, e := range c.modes[c.currMode].bools {
		for _, a := range e.actions {
			a(e.value)
		}
	}
}

func (c *Context) RegisterInt(mode Mode, key string, r func(string, devices.Effect[int64]), e devices.Effect[int64]) devices.Effect[int64] {
	r(key, e)

	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
	layer := c.modes[mode]
	if _, ok := layer.ints[key]; !ok {
		layer.ints[key] = element[int64]{
			actions: []devices.Effect[int64]{},
		}
	}
	elem := layer.ints[key]
	elem.actions = append(elem.actions, e)

	return func(v int64) error {
		elem.value = v
		if c.currMode == mode {
			return e(v)
		}
		return nil
	}
}

func (c *Context) RegisterFloat(mode Mode, key string, r func(string, devices.Effect[float64]), e devices.Effect[float64]) {
	if _, ok := c.modes[mode]; !ok {
		c.modes[mode] = newLayer()
	}
	layer := c.modes[mode]
	if _, ok := layer.floats[key]; !ok {
		layer.floats[key] = element[float64]{
			actions: []devices.Effect[float64]{},
		}
	}
	elem := layer.floats[key]
	elem.actions = append(elem.actions, e)

	r(key, func(v float64) error {
		elem.value = v
		if c.currMode == mode {
			return e(v)
		}
		return nil
	})
}

func main() {
	defer midi.CloseDriver()
	fmt.Printf("outports:\n" + midi.GetOutPorts().String() + "\n")

	in, err := midi.FindInPort("IAC Driver Bus 1")
	if err != nil {
		panic(err)
	}
	fmt.Println(in)
	out, err := midi.FindOutPort("IAC Driver Bus 1")
	if err != nil {
		panic(err)
	}
	fmt.Println(out)

	x := xtouch.New(devices.NewMidiDevice(in, out))

	m := motu.NewHTTPDatastore("http://localhost:8888")

	r := reaper.OscServer{}

	c := Context{
		currMode: MIX,
		modes:    map[Mode]layer{},
	}

	for i := 0; i < 8; i++ {
		x.Channels[i].Fader.Register(func(rel int16, abs uint16) error {
			normalized := float64(abs) / 4 / float64(math.MaxUint16)
			switch c.currMode {
			case RECORD:
				return m.SetFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i), normalized)
			default:
				return r.SetFloat(fmt.Sprintf("channels/%d/fader", i), normalized) // TODO:
			}
		})
		x.Channels[i].Mute.Register(func(b bool) error {
			switch c.currMode {
			case RECORD:
				return m.SetBool(fmt.Sprintf("mix/main/%d/matrix/mute", i), b)
			default:
				return r.SetBool(fmt.Sprintf("channels/%d/mute", i), b)
			}
		})
		x.Channels[i].Solo.Register(func(b bool) error {
			switch c.currMode {
			case RECORD:
				return m.SetBool(fmt.Sprintf("mix/main/%d/matrix/solo", i), b)
			default:
				return r.SetBool(fmt.Sprintf("channels/%d/solo", i), b)
			}
		})

		m.RegisterFloat(fmt.Sprintf("mix/main/%d/matrix/fader", i),
			func(v float64) error {
				x.Channels[i].Fader.SetFaderAbsolute(int16(v / 4 * float64(math.MaxUint16)))
				return nil
			})
		// TODO: is there a better way to provide levels to meters?
		c.RegisterFloat(RECORD, "ext/ibank/0/ch/%d/vlLimit", m.RegisterFloat, func(v float64) error {
			x.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		c.RegisterFloat(MIX, "channels/%d/meter", r.RegisterFloat, func(v float64) error { // TODO: path
			x.Channels[i].Meter.SendRelative(0.9)
			return nil
		})
		c.RegisterFloat(RECORD, "ext/ibank/0/ch/%d/vlClip", m.RegisterFloat, func(v float64) error {
			x.Channels[i].Rec.SetLED(xtouch.FLASHING)
			return nil
		})
		c.RegisterFloat(MIX, "channels/%d/clip", r.RegisterFloat, func(v float64) error { // TODO: path
			x.Channels[i].Rec.SetLED(xtouch.FLASHING)
			return nil
		})
		// TODO: trim on encoders
	}

	x.Function[0].Register(func(b bool) error {
		switch c.currMode {
		case MIX:
			c.SetMode(RECORD)
		case RECORD:
			c.SetMode(MIX)
		}
		return nil
	})
}

//
// m.RegisterFloat("mix/main/1/fader", c.Register(RECORD, func(v float64) error {}))
