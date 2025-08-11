package reaper

import (
	"log/slog"
	"strings"
	"sync"
	stdTime "time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/jdginn/arpad/logging"
)

var oscInLog *slog.Logger

func init() {
	oscInLog = logging.Get(logging.OSC_IN)
}

type namedHandler struct {
	name    string
	handler func(*osc.Message)
}

// Dispatcher is a custom osc.Dispatcher, implementing the osc.Dispatcher interface
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[int]namedHandler
	nextID   int
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[int]namedHandler)}
}

func (s *Dispatcher) AddMsgHandler(addr string, handler func(*osc.Message)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	s.handlers[id] = namedHandler{addr, handler}
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.handlers, id)
	}
}

// matchAddr checks if messageAddr matches the path pattern.
// Any * is treated as a wildcard. Paths are also allowed to end with a "*",
func matchAddr(path, messageAddr string) bool {
	pathSegs := strings.Split(path, "/")
	addrSegs := strings.Split(messageAddr, "/")

	endsWithStar := len(pathSegs) > 0 && pathSegs[len(pathSegs)-1] == "*"
	matchLen := len(pathSegs)
	if endsWithStar {
		// Remove the "*" for matching; allow extra segments in addrSegs
		matchLen--
		if len(addrSegs) < matchLen {
			return false
		}
	} else {
		if len(pathSegs) != len(addrSegs) {
			return false
		}
	}

	for i := 0; i < matchLen; i++ {
		p := pathSegs[i]
		if p == "*" {
			continue
		}
		if p != addrSegs[i] {
			return false
		}
	}
	return true
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (s *Dispatcher) Dispatch(packet osc.Packet) {
	switch p := packet.(type) {
	default:
		return

	case *osc.Message:
		oscInLog.Debug("Received OSC message", slog.String("address", p.Address), slog.Any("arguments", p.Arguments))
		for _, namedHandler := range s.handlers {
			if matchAddr(namedHandler.name, p.Address) {
				s.mu.RLock()
				namedHandler.handler(p)
				s.mu.RUnlock()
			}
		}

	case *osc.Bundle:
		timer := stdTime.NewTimer(p.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range p.Messages {
				for _, namedHandler := range s.handlers {
					if matchAddr(namedHandler.name, message.Address) {
						s.mu.RLock()
						namedHandler.handler(message)
						s.mu.RUnlock()
					}
				}
			}

			// Process all bundles
			for _, b := range p.Bundles {
				s.Dispatch(b)
			}
		}()
	}
}
