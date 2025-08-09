package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"
)

type LogCategory string

const (
	META     LogCategory = "meta" // For logs about logging
	MIDI_IN  LogCategory = "midi_in"
	MIDI_OUT LogCategory = "midi_out"
	OSC_IN   LogCategory = "osc_in"
	OSC_OUT  LogCategory = "osc_out"
	APP      LogCategory = "app" // For application-specific logs (i.e. business logic)
)

func strToLogCategory(s string) (LogCategory, bool) {
	switch s {
	case "meta":
		return META, true
	case "midi_in":
		return MIDI_IN, true
	case "midi_out":
		return MIDI_OUT, true
	case "osc_in":
		return OSC_IN, true
	case "osc_out":
		return OSC_OUT, true
	case "app":
		return APP, true
	default:
		return "", false
	}
}

const (
	// LOGGER_OSC_SEND_IP = "0.0.0.0"
	// LOGGER_OSC_SEND_PORT = 9090
	LOGGER_OSC_LISTEN_IP   = "0.0.0.0"
	LOGGER_OSC_LISTEN_PORT = 9085
)

type namedHandler struct {
	name    string
	handler func(*osc.Message)
}

// Dispatcher is a custom osc.Dispatcher, implementing the osc.Dispatcher interface
type Dispatcher struct{}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (s *Dispatcher) Dispatch(packet osc.Packet) {
	switch p := packet.(type) {
	default:
		return

	case *osc.Message:
		HandleOSCSetCategoryLevel(p)
	}
}

type OscRouter struct {
	// Client *osc.Client
	Server     *osc.Server
	Dispatcher osc.Dispatcher

	// clientIP   string
	// clientPort int
	serverIP   string
	serverPort int
}

func (o *OscRouter) Run() error {
	// Now Run() just starts the server
	fmt.Println("Starting OSC server at", fmt.Sprintf("%s:%d", o.serverIP, o.serverPort))
	o.Server = &osc.Server{
		Addr:       fmt.Sprintf("%s:%d", o.serverIP, o.serverPort),
		Dispatcher: o.Dispatcher,
	}
	return o.Server.ListenAndServe()
}

// Internal state for loggers per category
var (
	rootLogger       *slog.Logger
	mu               *sync.RWMutex
	loggers          = map[LogCategory]*slog.Logger{}
	categoryLvls     map[LogCategory]*slog.LevelVar
	defaultLogLevels map[LogCategory]slog.Level
	oscRouter        *OscRouter
)

func init() {
	mu = new(sync.RWMutex)
	defaultLogLevels = map[LogCategory]slog.Level{
		META:     slog.LevelInfo,
		MIDI_IN:  slog.LevelWarn,
		MIDI_OUT: slog.LevelWarn,
		OSC_IN:   slog.LevelWarn,
		OSC_OUT:  slog.LevelWarn,
		APP:      slog.LevelInfo,
	}
	categoryLvls = make(map[LogCategory]*slog.LevelVar)
	// Default to text output, can be customized
	rootLogger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	dispatcher := NewDispatcher()
	oscRouter = &OscRouter{
		Dispatcher: dispatcher,
		serverIP:   LOGGER_OSC_LISTEN_IP,
		serverPort: LOGGER_OSC_LISTEN_PORT,
	}
	go func() {
		if err := oscRouter.Run(); err != nil {
			panic("Failed to start OSC server: " + err.Error())
		}
	}()
}

// Get returns a slog.Logger that always has the "category" attribute set.
// Each category gets its own logger instance.
func Get(category LogCategory) *slog.Logger {
	mu.RLock()
	l, ok := loggers[category]
	mu.RUnlock()
	if ok {
		return l
	}
	mu.Lock()
	defer mu.Unlock()
	// Double-check after locking
	if l, ok := loggers[category]; ok {
		return l
	}
	// Create a new LevelVar for this category if it doesn't exist
	lvlVar, ok := categoryLvls[category]
	if !ok {
		lvlVar = new(slog.LevelVar)
		lvlVar.Set(defaultLogLevels[category])
		categoryLvls[category] = lvlVar
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvlVar,
	})
	catLogger := slog.New(handler).With("category", category)
	loggers[category] = catLogger
	return catLogger
}

func SetCategoryLevel(category LogCategory, level slog.Level) {
	mu.Lock()
	defer mu.Unlock()
	lvlVar, ok := categoryLvls[category]
	if !ok {
		panic("categoryLvls map is not initialized for category: " + string(category))
	}
	lvlVar.Set(level)
}

func splitOscPath(path string) []string {
	return strings.Split(path, "/")[1:]
}

// OSC handler for runtime config
//
// Routes:
// /meta/logger/{category}/level as int where -4 is Debug, 0 is Info, 4 is Warn, 8 is Error
func HandleOSCSetCategoryLevel(msg *osc.Message) {
	pathSegs := splitOscPath(msg.Address)

	if (pathSegs[0] != "meta") || (pathSegs[1] != "logging") {
		return
	}
	if len(pathSegs) == 4 && pathSegs[3] == "level" {
		cat, ok := strToLogCategory(pathSegs[2])
		if !ok {
			slog.Info("Unrecognized log category in OSC message", "category", pathSegs[2])
			return
		}
		level, ok := msg.Arguments[0].(int32)
		if !ok {
			slog.Error("Invalid level type in OSC message", "expected", "int32", "got", fmt.Sprintf("%T", msg.Arguments[0]))
			return
		}
		if categoryLvls == nil {
			panic("categoryLvls map is not initialized")
		}
		Get(META).Info("Setting category level via OSC",
			"category", cat,
			"level", level)
		if handle, ok := categoryLvls[cat]; ok {
			handle.Set(slog.Level(level))
		}
	}
}
