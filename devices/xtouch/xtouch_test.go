package xtouch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	dev "github.com/jdginn/arpad/devices"
	devtest "github.com/jdginn/arpad/devices/devicestesting"
)

func TestScribbleString(t *testing.T) {
	assert := assert.New(t)

	midiIn := devtest.NewMockMIDIPort()
	midiOut := devtest.NewMockMIDIPort()

	xtouch := New(dev.NewMidiDevice(midiIn, midiOut))
	fmt.Printf("chan: %v\n", xtouch.Channels[0].Scribble.channel)
	xtouch.Channels[0].Scribble.ChangeColor(RedInv).ChangeTopMessage("Ch 1").ChangeBottomMessage("aB3").Set()

	assert.Equal([]byte{0xf0, 0x00, 0x00, 0x66, 0x58, 0x20, 0x41, 0x43, 0x68, 0x20, 0x31, 0x00, 0x00, 0x00, 0x20, 0x20, 0x20, 0x20, 0x61, 0x42, 0x33, 0xf7}, midiOut.GetSentMessages()[0].Bytes())
}
