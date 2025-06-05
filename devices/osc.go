package devices

import (
	"fmt"
	"strconv"

	"github.com/hypebeast/go-osc/osc"
)

type OscDevice struct {
	client     OscClient
	server     OscServer
	dispatcher OscDispatcher
}

// Interfaces for better testability
type OscClient interface {
	Send(*osc.Message) error
}

type OscServer interface {
	ListenAndServe() error
}

type OscDispatcher interface {
	AddMsgHandler(string, func(*osc.Message))
}

func NewOscDevice(client OscClient, server OscServer, dispatcher OscDispatcher) *OscDevice {
	return &OscDevice{
		client:     client,
		server:     server,
		dispatcher: dispatcher,
	}
}

// // For convenience, create real OSC implementations
// func NewRealOscDevice(sendAddr, listenAddr string) (*OscDevice, error) {
// 	client := osc.NewClient(sendAddr)
// 	server := osc.NewServer(listenAddr)
// 	dispatcher := server.Dispatcher()
//
// 	return NewOscDevice(client, server, dispatcher), nil
// }

func (o *OscDevice) Run() error {
	// Now Run() just starts the server
	return o.server.ListenAndServe()
}

func (o *OscDevice) SetInt(key string, val int64) error {
	return o.client.Send(osc.NewMessage(key, val))
}

func (o *OscDevice) SetFloat(key string, val float64) error {
	return o.client.Send(osc.NewMessage(key, float32(val)))
}

func (o *OscDevice) SetString(key string, val string) error {
	return o.client.Send(osc.NewMessage(key, val))
}

func (o *OscDevice) SetBool(key string, val bool) error {
	return o.client.Send(osc.NewMessage(key, val))
}

// BindInt binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as an int.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as an int.
func (o *OscDevice) BindInt(addr string, effect func(int64) error) {
	o.dispatcher.AddMsgHandler(addr, func(msg *osc.Message) {
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
}

// BindFloat binds a callback to run whenever a message is received for the given OSC address.
//
// The given address MUST return a float or be convertable to float.
// WARNING: Conversions are best-effort and could panic.
func (o *OscDevice) BindFloat(key string, effect func(float64) error) {
	o.dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
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
}

// BindString binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a string.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a string.
func (o *OscDevice) BindString(key string, effect func(string) error) {
	o.dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
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
}

// BindBool binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a boolean.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a boolean.
func (o *OscDevice) BindBool(key string, effect func(bool) error) {
	o.dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
		val := msg.Arguments[len(msg.Arguments)-1]
		switch val := val.(type) {
		case bool:
			effect(val)
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
}
