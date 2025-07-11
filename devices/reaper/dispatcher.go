package reaper

import (
	"fmt"
	"strings"
	stdTime "time"

	"github.com/hypebeast/go-osc/osc"
)

type namedHandler struct {
	name    string
	handler func(*osc.Message)
}

// Dispatcher is a custom osc.Dispatcher, implementing the osc.Dispatcher interface
type Dispatcher struct {
	handlers []namedHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: []namedHandler{}}
}

func (s *Dispatcher) AddMsgHandler(addr string, handler func(*osc.Message)) error {
	s.handlers = append(s.handlers, namedHandler{addr, handler})
	return nil
}

// matchAddr checks if messageAddr matches the path pattern.
// Each "@" in path acts as a wildcard for a segment, and captured segments are returned.
// If path ends with "*", any additional segments in messageAddr are ignored.
// "*" does not capture anything.
func matchAddr(path, messageAddr string) (bool, []string) {
	pathSegs := strings.Split(path, "/")
	addrSegs := strings.Split(messageAddr, "/")

	endsWithStar := len(pathSegs) > 0 && pathSegs[len(pathSegs)-1] == "*"
	matchLen := len(pathSegs)
	if endsWithStar {
		// Remove the "*" for matching; allow extra segments in addrSegs
		matchLen--
		if len(addrSegs) < matchLen {
			return false, nil
		}
	} else {
		if len(pathSegs) != len(addrSegs) {
			return false, nil
		}
	}

	var captures []string
	for i := 0; i < matchLen; i++ {
		p := pathSegs[i]
		if p == "@" {
			captures = append(captures, addrSegs[i])
		} else if p != addrSegs[i] {
			return false, nil
		}
	}

	// If endsWithStar, allow any suffix
	return true, captures
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (s *Dispatcher) Dispatch(packet osc.Packet) {
	switch p := packet.(type) {
	default:
		return

	case *osc.Message:
		fmt.Printf("Osc message: %s\n", p.Address)
		for _, namedHandler := range s.handlers {
			if match, args := matchAddr(namedHandler.name, p.Address); match {
				fmt.Printf("args: %v\n", args)
				for _, arg := range args {
					p.Arguments = append(p.Arguments, arg)
				}
				namedHandler.handler(p)
			}
		}

	case *osc.Bundle:
		timer := stdTime.NewTimer(p.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range p.Messages {
				fmt.Printf("Osc message: %s\n", message.Address)
				for _, namedHandler := range s.handlers {
					if match, args := matchAddr(namedHandler.name, message.Address); match {
						fmt.Printf("args: %v\n", args)
						for _, arg := range args {
							message.Arguments = append(message.Arguments, arg)
						}
						namedHandler.handler(message)
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
