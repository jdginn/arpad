package xtouch

import (
	"fmt"
	"math"
	"sync"
	"time"

	dev "github.com/jdginn/arpad/devices"

	midi "gitlab.com/gomidi/midi/v2"
)

const (
	// Handshake message sent by X-Touch every 2 seconds
	handshakePingMessage = "\xF0\x00\x20\x32\x58\x54\x00\xF7"
	// Expected response message (should be received every 6-8 seconds)
	handshakeResponseMessage = "\xF0\x00\x00\x66\x14\x00\xF7"

	// Timing constants
	pingInterval    = 2 * time.Second
	responseTimeout = 4 * time.Second
)

// Fader represents a motorized fader on an xtouch controller.
//
// Faders send MIDI PitchBend data on their specified channel.
// Faders can be remotely moved at will using SetFader*.
type Fader struct {
	d *dev.MidiDevice

	ChannelNo uint8
}

func (f *Fader) Bind(callback func(int16) error) {
	f.d.PitchBend(uint8(f.ChannelNo)).Bind(callback)
}

func (f *Fader) Set(val int16) error {
	return f.d.PitchBend(uint8(f.ChannelNo)).Set(val)
}

type ScribbleColor int

const (
	Off       ScribbleColor = 0x00
	Red       ScribbleColor = 0x01
	Green     ScribbleColor = 0x02
	Yellow    ScribbleColor = 0x03
	Blue      ScribbleColor = 0x04
	Pink      ScribbleColor = 0x05
	Cyan      ScribbleColor = 0x06
	White     ScribbleColor = 0x07
	RedInv    ScribbleColor = 0x41
	GreenInv  ScribbleColor = 0x42
	YellowInv ScribbleColor = 0x43
	BlueInv   ScribbleColor = 0x44
	PinkInv   ScribbleColor = 0x45
	CyanInv   ScribbleColor = 0x46
	WhiteInv  ScribbleColor = 0x47
)

type SysExHeader []byte

var HeaderScribble SysExHeader = []byte{0x00, 0x00, 0x66, 0x58}

type Scribble struct {
	d *dev.MidiDevice

	channel uint8
}

// TODO: consider making this take strings instead of []byte?
func (s *Scribble) SendScribble(color ScribbleColor, msgTop, msgBottom []byte) error {
	// TODO: check msg for length, support best-effort truncation?
	b := make([]byte, 0, 20)
	b = append(HeaderScribble, byte(s.channel))
	b = append(b, byte(color))
	b = append(b, msgTop...)
	b = append(b, msgBottom...)
	return s.d.SysEx.Set(midi.SysEx(b)) // TODO: check this
}

type Meter struct {
	d *dev.MidiDevice

	channel uint8
}

func (m *Meter) Send(val float64) error {
	if val > 1.0 {
		return fmt.Errorf("Invalid val: val must be between 0 and 1.0")
	}
	offset := m.channel*16 + uint8(math.Round(8*val))
	return m.d.Aftertouch(0).Set(offset) // TODO: check this
}

type XTouch struct {
	base *dev.MidiDevice

	// Handshake management
	handshakeActive   bool
	lastResponse      time.Time
	handshakeMutex    sync.RWMutex
	handshakeStopChan chan struct{}
}

// startHandshake begins the handshake protocol
func (x *XTouch) startHandshake() error {
	x.handshakeMutex.Lock()
	defer x.handshakeMutex.Unlock()

	if x.handshakeActive {
		return nil
	}

	x.handshakeStopChan = make(chan struct{})
	x.handshakeActive = true
	x.lastResponse = time.Now()

	// Start sending ping messages
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := x.base.SysEx.Set(midi.SysEx([]byte(handshakePingMessage))); err != nil {
					fmt.Printf("Error sending handshake ping: %v\n", err)
				}

				// Check for response timeout
				x.handshakeMutex.RLock()
				if time.Since(x.lastResponse) > responseTimeout {
					fmt.Println("Warning: No handshake response received within timeout period; giving up")
					return
				}
				x.handshakeMutex.RUnlock()

			case <-x.handshakeStopChan:
				return
			}
		}
	}()

	// Set up handler for response messages
	x.base.SysEx.Match([]byte(handshakeResponseMessage)).Bind(func(msg []byte) error {
		x.handshakeMutex.Lock()
		x.lastResponse = time.Now()
		x.handshakeMutex.Unlock()
		return nil
	})

	return nil
}

// stopHandshake stops the handshake protocol
func (x *XTouch) stopHandshake() {
	x.handshakeMutex.Lock()
	defer x.handshakeMutex.Unlock()

	if !x.handshakeActive {
		return
	}

	close(x.handshakeStopChan)
	x.handshakeActive = false
}

func (x *XTouch) Run() {
	if err := x.startHandshake(); err != nil {
		fmt.Printf("Failed to start handshake: %v\n", err)
	}
	x.base.Run()
}

// NewFader returns a new fader on the given channel.
//
// NewFader accepts an optional, variadic list of callbacks to run when the fader is moved.
func (x *XTouch) NewFader(channelNo uint8) *Fader {
	return &Fader{
		d:         x.base,
		ChannelNo: channelNo,
	}
}

func (x *XTouch) NewEncoder(channelNo uint8, id uint8) *Encoder {
	// id should be 0-7
	// encoderCC := 16 + (id % 8) // Maps to CC 16-23
	ledLowCC := 48 + (id % 8)  // Maps to CC 48-55
	ledHighCC := 56 + (id % 8) // Maps to CC 56-63
	enc := &Encoder{
		d:           x.base,
		channel:     channelNo,
		ledRingLow:  ledLowCC,
		ledRingHigh: ledHighCC,
	}
	enc.Ring = ring{
		AllSegments:      ringSetAllSegments{enc},
		ClearAllSegments: ringClearAllSegments{enc},
	}
	return enc
}

func (x *XTouch) NewScribble(channel uint8) *Scribble {
	return &Scribble{
		x.base,
		channel,
	}
}

func (x *XTouch) NewMeter(channel uint8) *Meter {
	return &Meter{
		x.base,
		channel,
	}
}

// channelStrip is a convenience struct that organizes all the components that are replicated
// for each channel strip under control.
type channelStrip struct {
	// TODO: Encoder
	Encoder       *Encoder
	EncoderButton *Button
	Scribble      *Scribble
	Rec           *Button
	Solo          *Button
	Mute          *Button
	Select        *Button
	Meter         *Meter
	Fader         *Fader
	// TODO: 7Seg
	// TODO: JogWheel
}

// NewChannelStrip returns a new channelStrip corresponding to the given index into a
// bank of channelStrips. For typical devices, id will be between 0 and 7.
func (x *XTouch) NewChannelStrip(id uint8) *channelStrip {
	fmt.Printf("New channel strip with itd %d\n", id)
	return &channelStrip{
		Encoder:       x.NewEncoder(0, id+32),
		EncoderButton: x.NewButton(0, id+16),
		Scribble:      x.NewScribble(id + 20),
		Rec:           x.NewButton(0, id),
		Solo:          x.NewButton(0, id+8),
		Mute:          x.NewButton(0, id+16),
		Select:        x.NewButton(0, id+24),
		Meter:         x.NewMeter(id),
		Fader:         x.NewFader(id),
	}
}

type EncoderAssign struct {
	TRACK        *Button
	PAN_SURROUND *Button
	EQ           *Button
	SEND         *Button
	PLUGIN       *Button
	INST         *Button
}

func (x *XTouch) NewEncoderAssign() *EncoderAssign {
	return &EncoderAssign{
		TRACK:        x.NewButton(0, 40),
		PAN_SURROUND: x.NewButton(0, 42),
		EQ:           x.NewButton(0, 44),
		SEND:         x.NewButton(0, 41),
		PLUGIN:       x.NewButton(0, 43),
		INST:         x.NewButton(0, 45),
	}
}

type View struct {
	GLOBAL       *Button
	MIDI         *Button
	INPUTS       *Button
	AUDIO_TRACKS *Button
	AUDIO_INST   *Button
	AUX          *Button
	BUSES        *Button
	OUTPUTS      *Button
	USER         *Button
}

func (x *XTouch) NewView() *View {
	return &View{
		GLOBAL:       x.NewButton(0, 51),
		MIDI:         x.NewButton(0, 62),
		INPUTS:       x.NewButton(0, 63),
		AUDIO_TRACKS: x.NewButton(0, 64),
		AUDIO_INST:   x.NewButton(0, 65),
		AUX:          x.NewButton(0, 66),
		BUSES:        x.NewButton(0, 67),
		OUTPUTS:      x.NewButton(0, 68),
		USER:         x.NewButton(0, 69),
	}
}

type Function struct {
	F1 *Button
	F2 *Button
	F3 *Button
	F4 *Button
	F5 *Button
	F6 *Button
	F7 *Button
	F8 *Button
}

func (x *XTouch) NewFunction() *Function {
	return &Function{
		F1: x.NewButton(0, 54),
		F2: x.NewButton(0, 55),
		F3: x.NewButton(0, 56),
		F4: x.NewButton(0, 57),
		F5: x.NewButton(0, 58),
		F6: x.NewButton(0, 59),
		F7: x.NewButton(0, 60),
		F8: x.NewButton(0, 61),
	}
}

type Modify struct {
	SHIFT   *Button
	OPTION  *Button
	CONTROL *Button
	ALT     *Button
}

func (x *XTouch) NewModify() *Modify {
	return &Modify{
		SHIFT:   x.NewButton(0, 70),
		OPTION:  x.NewButton(0, 71),
		CONTROL: x.NewButton(0, 72),
		ALT:     x.NewButton(0, 73),
	}
}

type Automation struct {
	READ_OFF *Button
	WRITE    *Button
	TRIM     *Button
	TOUCH    *Button
	LATCH    *Button
	GROUP    *Button
}

func (x *XTouch) NewAutomation() *Automation {
	return &Automation{
		READ_OFF: x.NewButton(0, 74),
		WRITE:    x.NewButton(0, 75),
		TRIM:     x.NewButton(0, 76),
		TOUCH:    x.NewButton(0, 77),
		LATCH:    x.NewButton(0, 78),
		GROUP:    x.NewButton(0, 79),
	}
}

type Utility struct {
	SAVE   *Button
	UNDO   *Button
	CANCEL *Button
	ENTER  *Button
}

func (x *XTouch) NewUtility() *Utility {
	return &Utility{
		SAVE:   x.NewButton(0, 80),
		UNDO:   x.NewButton(0, 81),
		CANCEL: x.NewButton(0, 82),
		ENTER:  x.NewButton(0, 83),
	}
}

type Transport struct {
	Marker  *Button
	Nudge   *Button
	Cycle   *ToggleButton
	Drop    *Button
	Replace *Button
	Click   *ToggleButton
	Solo    *ToggleButton
	REW     *Button
	FF      *Button
	STOP    *Button
	PLAY    *Button
	RECORD  *Button
}

func (x *XTouch) NewTransport() *Transport {
	return &Transport{
		Marker:  x.NewButton(0, 84),
		Nudge:   x.NewButton(0, 85),
		Cycle:   x.NewToggleButton(0, 86),
		Drop:    x.NewButton(0, 87),
		Replace: x.NewButton(0, 88),
		Click:   x.NewToggleButton(0, 89),
		Solo:    x.NewToggleButton(0, 90),
		REW:     x.NewButton(0, 91),
		FF:      x.NewButton(0, 92),
		STOP:    x.NewButton(0, 93),
		PLAY:    x.NewButton(0, 94),
		RECORD:  x.NewButton(0, 95),
	}
}

type Page struct {
	BANK_L    *Button
	BANK_R    *Button
	CHANNEL_L *Button
	CHANNEL_R *Button
}

func (x *XTouch) NewPage() *Page {
	return &Page{
		BANK_L:    x.NewButton(0, 46),
		BANK_R:    x.NewButton(0, 47),
		CHANNEL_L: x.NewButton(0, 48),
		CHANNEL_R: x.NewButton(0, 49),
	}
}

type Navigation struct {
	UP    *Button
	DOWN  *Button
	LEFT  *Button
	RIGHT *Button
	ZOOM  *Button
	SCRUB *Button
}

func (x *XTouch) NewNavigation() *Navigation {
	return &Navigation{
		UP:    x.NewButton(0, 96),
		DOWN:  x.NewButton(0, 97),
		LEFT:  x.NewButton(0, 98),
		RIGHT: x.NewButton(0, 99),
		ZOOM:  x.NewButton(0, 100),
		SCRUB: x.NewButton(0, 101),
	}
}

// XTouchDefault represents a Behringer XTouch DAW control surface.
type XTouchDefault struct {
	*XTouch

	Channels      []*channelStrip
	EncoderAssign *EncoderAssign
	View          *View
	Function      *Function
	Modify        *Modify
	Automation    *Automation
	Utility       *Utility
	Transport     *Transport
	Page          *Page
	Navigation    *Navigation
}

// New returns a properly initialized XTouchDefault struct.
func New(d *dev.MidiDevice) *XTouchDefault {
	x := &XTouchDefault{
		XTouch: &XTouch{
			base:            d,
			handshakeActive: false,
			lastResponse:    time.Time{},
		},
	}
	for i := 0; i < 8; i++ {
		x.Channels = append(x.Channels, x.NewChannelStrip(uint8(i)))
	}
	x.EncoderAssign = x.NewEncoderAssign()
	x.View = x.NewView()
	x.Function = x.NewFunction()
	x.Modify = x.NewModify()
	x.Automation = x.NewAutomation()
	x.Utility = x.NewUtility()
	x.Transport = x.NewTransport()
	x.Page = x.NewPage()
	x.Navigation = x.NewNavigation()

	return x
}

// XTouchExtender represents a Behringer XTouchExtender DAW control surface.
type XTouchExtender struct {
	*XTouch

	Channels []*channelStrip
}

func NewExtender(d *dev.MidiDevice) *XTouchExtender {
	x := &XTouchExtender{
		XTouch: &XTouch{
			base:            d,
			handshakeActive: false,
			lastResponse:    time.Time{},
		},
	}
	for i := 0; i < 8; i++ {
		x.Channels[i] = x.NewChannelStrip(uint8(i))
	}

	return x
}
