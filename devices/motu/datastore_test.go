package motu

import (
	"testing"
	"time"
	// "github.com/stretchr/testify/assert"
)

func TestPoll(t *testing.T) {
	d := NewHTTPDatastore("http://localhost:8888/datastore")
	go d.poll()
	for i := 0; i < 20; i++ {
		time.Sleep(3 * time.Second)
		d.SetInt("mix/chan/1/hpf/freq", 100)
	}
}
