package motu

import (
	"testing"
	"time"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoll(t *testing.T) {
	require := require.New(t)

	d := NewHTTPDatastore("http://localhost:8888/datastore")
	go d.poll()
	time.Sleep(1 * time.Second)
	d.SetInt("mix/chan/1/hpf/freq", 100)
	v, err := d.GetInt("mix/chan/1/hpf/freq")
	require.NoError(err)
	require.EqualValues(100, v)

	d.SetInt("mix/chan/1/hpf/freq", 200)
	v, err = d.GetInt("mix/chan/1/hpf/freq")
	require.NoError(err)
	require.EqualValues(200, v)

	d.SetInt("mix/chan/1/hpf/freq", 300)
	v, err = d.GetInt("mix/chan/1/hpf/freq")
	require.NoError(err)
	require.EqualValues(300, v)
}
