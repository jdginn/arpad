package devices

import (
	"fmt"
	"strconv"

	"github.com/hypebeast/go-osc/osc"
)

type Osc struct {
	c osc.Client
	s osc.Server

	d osc.StandardDispatcher
}

func (o *Osc) SetInt(key string, val int64) error {
	return o.c.Send(osc.NewMessage(key, val))
}

func (o *Osc) SetFloat(key string, val float64) error {
	return o.c.Send(osc.NewMessage(key, val))
}

func (o *Osc) SetString(key string, val string) error {
	return o.c.Send(osc.NewMessage(key, val))
}

func (o *Osc) SetBool(key string, val bool) error {
	return o.c.Send(osc.NewMessage(key, val))
}

// BindInt binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as an int.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as an int.
func (o *Osc) BindInt(addr string, effect func(int64) error) error {
	o.d.AddMsgHandler(addr, func(msg *osc.Message) {
		val := msg.Arguments[len(msg.Arguments)-1]
		switch val := val.(type) {
		case int:
			effect(int64(val))
		case float64:
			effect(int64(val))
		case string:
			asint, err := strconv.Atoi(val)
			if err != nil {
				panic("bad")
			}
			effect(int64(asint))
		default:
			panic("bad")
		}
	})
	return nil
}

// BindFloat binds a callback to run whenever a message is received for the given OSC address.
//
// The given address MUST return a float or be convertable to float.
// WARNING: Conversions are best-effort and could panic.
func (o *Osc) BindFloat(key string, effect func(float64) error) error {
	o.d.AddMsgHandler(key, func(msg *osc.Message) {
		val := msg.Arguments[len(msg.Arguments)-1]
		switch val := val.(type) {
		case float64:
			effect(val)
		case float32:
			effect(float64(val))
		case int:
			effect(float64(val))
		case int64:
			effect(float64(val))
		case string:
			asNum, err := strconv.Atoi(val)
			if err != nil {
				panic("bad")
			}
			effect(float64(asNum))
		default:
			panic("bad")
		}
	})
	return nil
}

// BindString binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a string.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a string.
func (o *Osc) BindString(key string, effect func(string) error) error {
	o.d.AddMsgHandler(key, func(msg *osc.Message) {
		val := msg.Arguments[len(msg.Arguments)-1]
		switch val := val.(type) {
		case float64:
			effect(fmt.Sprintf("%f", val))
		case float32:
			effect(fmt.Sprintf("%f", val))
		case int:
			effect(fmt.Sprintf("%d", val))
		case int64:
			effect(fmt.Sprintf("%d", val))
		case string:
			effect(val)
		default:
			panic("bad")
		}
	})
	return nil
}

// BindBool binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a boolean.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a boolean.
func (o *Osc) BindBool(key string, effect func(bool) error) error {
	o.d.AddMsgHandler(key, func(msg *osc.Message) {
		val := msg.Arguments[len(msg.Arguments)-1]
		switch val := val.(type) {
		case float64:
			effect(val > 0)
		case float32:
			effect(val > 0)
		case int:
			effect(val > 0)
		case int64:
			effect(val > 0)
		case string:
			effect(val == "true")
		default:
			panic("bad")
		}
	})
	return nil
}
