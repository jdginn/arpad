package devices

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/hypebeast/go-osc/osc"

	"github.com/jdginn/arpad/logging"
)

var oscInLog, oscOutLog *slog.Logger

func init() {
	oscInLog = logging.Get(logging.OSC_IN)
	oscOutLog = logging.Get(logging.OSC_OUT)
}

type Dispatcher interface {
	osc.Dispatcher
	AddMsgHandler(string, func(*osc.Message)) func()
}

type OscDevice struct {
	Client     *osc.Client
	Server     *osc.Server
	Dispatcher Dispatcher

	clientIP   string
	clientPort int
	serverIP   string
	serverPort int
}

func NewOscDevice(clientIp string, clientPort int, serverIp string, serverPort int, dispatcher Dispatcher) *OscDevice {
	return &OscDevice{
		Client:     osc.NewClient(clientIp, clientPort),
		Dispatcher: dispatcher,
		clientIP:   clientIp,
		clientPort: clientPort,
		serverIP:   serverIp,
		serverPort: serverPort,
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
	o.Server = &osc.Server{
		Addr:       fmt.Sprintf("%s:%d", o.serverIP, o.serverPort),
		Dispatcher: o.Dispatcher,
	}
	return o.Server.ListenAndServe()
}

func (o *OscDevice) SetInt(key string, val int64) error {
	oscOutLog.Debug("Sending OSC message", slog.String("address", key), slog.Any("arguments", val))
	return o.Client.Send(osc.NewMessage(key, int32(val)))
}

func (o *OscDevice) SetFloat(key string, val float64) error {
	oscOutLog.Debug("Sending OSC message", slog.String("address", key), slog.Any("arguments", val))
	return o.Client.Send(osc.NewMessage(key, float32(val)))
}

func (o *OscDevice) SetString(key string, val string) error {
	oscOutLog.Debug("Sending OSC message", slog.String("address", key), slog.Any("arguments", val))
	return o.Client.Send(osc.NewMessage(key, val))
}

func (o *OscDevice) SetBool(key string, val bool) error {
	oscOutLog.Debug("Sending OSC message", slog.String("address", key), slog.Any("arguments", val))
	return o.Client.Send(osc.NewMessage(key, val))
}

// BindInt binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as an int.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as an int.
func (o *OscDevice) BindInt(addr string, effect func(int64) error) func() {
	return o.Dispatcher.AddMsgHandler(addr, func(msg *osc.Message) {
		var val any
		if len(msg.Arguments) == 0 {
			val = 0
		} else {
			val = msg.Arguments[0]
			if val == nil {
				val = 0
			}
		}
		switch val := val.(type) {
		case int:
			if err := effect(int64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int32:
			if err := effect(int64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int64:
			if err := effect(int64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float64:
			if err := effect(int64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float32:
			if err := effect(int64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case string:
			asInt, err := strconv.Atoi(val)
			if err != nil {
				oscInLog.Error("Error converting string to int when handling OSC route", slog.String("route", msg.Address), slog.Any("value", val), slog.Any("err", err))
				return
			}
			if err := effect(int64(asInt)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		default:
			panic(fmt.Sprintf("Unsupported message type %T", val))
		}
	})
}

// BindFloat binds a callback to run whenever a message is received for the given OSC address.
//
// The given address MUST return a float or be convertable to float.
// WARNING: Conversions are best-effort and could panic.
func (o *OscDevice) BindFloat(key string, effect func(float64) error) func() {
	return o.Dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
		// oscInLog.Debug("Received OSC message", slog.String("address", msg.Address), slog.Any("arguments", msg.Arguments))
		var val any
		if len(msg.Arguments) == 0 {
			val = 0.0
		} else {
			val = msg.Arguments[0]
			if val == nil {
				val = 0.0
			}
		}
		switch val := val.(type) {
		case float64:
			if err := effect(val); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float32:
			if err := effect(float64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int:
			if err := effect(float64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int32:
			if err := effect(float64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int64:
			if err := effect(float64(val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case string:
			asNum, err := strconv.Atoi(val)
			if err != nil {
				oscInLog.Error("Error converting string to int when handling OSC route", slog.String("route", msg.Address), slog.Any("value", val), slog.Any("err", err))
				return
			}
			if err := effect(float64(asNum)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		default:
			panic(fmt.Sprintf("Unsupported message type %T", val))
		}
	})
}

// BindString binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a string.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a string.
func (o *OscDevice) BindString(key string, effect func(string) error) func() {
	return o.Dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
		oscInLog.Debug("Received OSC message", slog.String("address", msg.Address), slog.Any("arguments", msg.Arguments))
		var val any
		if len(msg.Arguments) == 0 {
			val = ""
		} else {
			val = msg.Arguments[0]
			if val == nil {
				val = ""
			}
		}
		switch val := val.(type) {
		case float64:
			if err := effect(fmt.Sprintf("%f", val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float32:
			if err := effect(fmt.Sprintf("%f", val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int:
			if err := effect(fmt.Sprintf("%d", val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int32:
			if err := effect(fmt.Sprintf("%d", val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int64:
			if err := effect(fmt.Sprintf("%d", val)); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case string:
			if err := effect(val); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		default:
			panic(fmt.Sprintf("Unsupported message type %T", val))
		}
	})
}

// BindBool binds a callback to run whenever a message is received for the given OSC address.
//
// The given address should return a value that can be interpreted as a boolean.
//
// WARNING: Conversions are best-effort and could panic if the value cannot be interpreted as a boolean.
func (o *OscDevice) BindBool(key string, effect func(bool) error) func() {
	return o.Dispatcher.AddMsgHandler(key, func(msg *osc.Message) {
		// oscInLog.Debug("Received OSC message", slog.String("address", msg.Address), slog.Any("arguments", msg.Arguments))
		var val any
		if len(msg.Arguments) == 0 {
			val = false
		} else {
			val = msg.Arguments[0]
			if val == nil {
				val = false
			}
		}
		switch val := val.(type) {
		case bool:
			if err := effect(val); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float64:
			if err := effect(val > 0); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case float32:
			if err := effect(val > 0); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int:
			if err := effect(val > 0); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int32:
			if err := effect(val > 0); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case int64:
			if err := effect(val > 0); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		case string:
			if err := effect(val == "true"); err != nil {
				oscInLog.Error("Error in function bound to osc route", slog.String("route", msg.Address), slog.Any("err", err))
				return
			}
		default:
			panic(fmt.Sprintf("Unsupported message type %T", val))
		}
	})
}
