package xtouch

import (
	"fmt"
	"math"

	dev "github.com/jdginn/arpad/devices"

	midi "gitlab.com/gomidi/midi/v2"
)

// Fader represents a motorized fader on an xtouch controller.
//
// Faders send MIDI PitchBend data on their specified channel.
// Faders can be remotely moved at will using SetFader*.
type Fader struct {
	d dev.MidiDevice

	ChannelNo uint8
}

// Bind specifies the callback to run when this fader is moved.
func (f *Fader) Bind(effect dev.CallbackPitchBend) {
	f.d.BindPitchBend(uint8(1+f.ChannelNo), effect)
}

// SetFaderAbsolute moves this fader to a value between 0 and max(int16).
func (f *Fader) SetFaderAbsolute(val int16) error {
	return f.d.Send(midi.Pitchbend(uint8(1+f.ChannelNo), val))
}

type LEDState uint8

const (
	OFF LEDState = iota
	ON
	FLASHING
)

// Button represents a button on an xtouch controller.
//
// Buttons send MIDI notes on a specified channel and key.
// Buttons incorporate LEDs, which can either be off, on, or flashing.
//
// TODO: need toggle vs. momentary functionality
type Button struct {
	d dev.MidiDevice

	channel uint8
	key     uint8

	isToggled bool
	isPressed bool
}

// Bind specifies the callback to run when this button is pressed.
func (b *Button) Bind(effect dev.CallbackNote) {
	b.d.BindNote(b.channel, b.key, func(v bool) error {
		b.isPressed = v
		if v {
			b.isToggled = !b.isToggled
		}
		return effect(v)
	})
}

func (b *Button) IsToggled() bool {
	return b.isToggled
}

func (b *Button) IsPressed() bool {
	return b.isPressed
}

func (f *Button) SetLED(state LEDState) error {
	switch state {
	case OFF:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 0))
	case ON:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 1))
	case FLASHING:
		return f.d.Send(midi.NoteOn(f.channel, f.key, 127))
	default:
		return fmt.Errorf("Unrecognized LED state")
	}
}

type Encoder struct {
	d dev.MidiDevice

	channel    uint8
	controller uint8
}

func (e *Encoder) Bind(effect dev.CallbackCC) {
	e.d.BindCC(e.channel, e.controller, effect)
}

// TODO: does this need any special wrapping?
func (e *Encoder) SetLEDRing(value uint8) error {
	return e.d.Send(midi.ControlChange(e.channel, e.controller, value))
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
	d dev.MidiDevice

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
	return s.d.Send(midi.SysEx(b))
}

type Meter struct {
	d dev.MidiDevice

	channel uint8
}

func (m *Meter) SendRelative(val float64) error {
	if val > 1.0 {
		return fmt.Errorf("Invalid val: val must be between 0 and 1.0")
	}
	offset := m.channel*16 + uint8(math.Round(8*val))
	return m.d.Send(midi.AfterTouch(0, offset))
}

type XTouch struct {
	base dev.MidiDevice
}

// NewFader returns a new fader on the gien channel.
//
// NewFader accepts an optional, variadic list of callbacks to run when the fader is moved.
func (x *XTouch) NewFader(channelNo uint8, callbacks ...dev.CallbackPitchBend) Fader {
	for _, e := range callbacks {
		x.base.BindPitchBend(uint8(1+channelNo), e)
	}
	return Fader{
		d:         x.base,
		ChannelNo: channelNo,
	}
}

func (x *XTouch) NewEncoder(channelNo uint8, control uint8, callbacks ...dev.CallbackCC) Encoder {
	for _, e := range callbacks {
		x.base.BindCC(channelNo, control, e)
	}
	return Encoder{
		d:          x.base,
		channel:    channelNo,
		controller: control,
	}
}

// NewButton returns a new button corresponding to the given channel and MIDI key.
//
// NewButton accepts an optional, variadic list of callbacks to run when the button is pressed.
func (x *XTouch) NewButton(channel, key uint8, callbacks ...dev.CallbackNote) Button {
	for _, e := range callbacks {
		x.base.BindNote(channel, key, e)
	}
	return Button{
		d:       x.base,
		channel: channel,
		key:     key,
	}
}

func (x *XTouch) NewScribble(channel uint8) Scribble {
	return Scribble{
		x.base,
		channel,
	}
}

func (x *XTouch) NewMeter(channel uint8) Meter {
	return Meter{
		x.base,
		channel,
	}
}

// channelStrip is a convenience struct that organizes all the components that are replicated
// for each channel strip under control.
type channelStrip struct {
	// TODO: Encoder
	Encoder       Encoder
	EncoderButton Button
	Scribble      Scribble
	Rec           Button
	Solo          Button
	Mute          Button
	Select        Button
	Meter         Meter
	Fader         Fader
	// TODO: 7Seg
	// TODO: JogWheel
}

// NewChannelStrip returns a new channelStrip corresponding to the given index into a
// bank of channelStrips. For typical devices, id will be between 0 and 7.
func (x *XTouch) NewChannelStrip(id uint8) channelStrip {
	return channelStrip{
		Encoder:       x.NewEncoder(0, id+32),
		EncoderButton: x.NewButton(0, id+16),
		Scribble:      x.NewScribble(id + 20),
		Rec:           x.NewButton(0, id),
		Solo:          x.NewButton(0, id+8),
		Mute:          x.NewButton(0, id+16),
		Select:        x.NewButton(0, id+24),
		Meter:         x.NewMeter(id),
		Fader:         x.NewFader(id + 1),
	}
}

type EncoderAssign struct {
	TRACK        Button
	PAN_SURROUND Button
	EQ           Button
	SEND         Button
	PLUGIN       Button
	INST         Button
}

func (x *XTouch) NewEncoderAssign() EncoderAssign {
	return EncoderAssign{
		TRACK:        x.NewButton(0, 40),
		PAN_SURROUND: x.NewButton(0, 42),
		EQ:           x.NewButton(0, 44),
		SEND:         x.NewButton(0, 41),
		PLUGIN:       x.NewButton(0, 43),
		INST:         x.NewButton(0, 45),
	}
}

type View struct {
	GLOBAL       Button
	MIDI         Button
	INPUTS       Button
	AUDIO_TRACKS Button
	AUDIO_INST   Button
	AUX          Button
	BUSES        Button
	OUTPUTS      Button
	USER         Button
}

func (x XTouch) NewView() View {
	return View{
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
	F1 Button
	F2 Button
	F3 Button
	F4 Button
	F5 Button
	F6 Button
	F7 Button
	F8 Button
}

func (x XTouch) NewFunction() Function {
	return Function{
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
	SHIFT   Button
	OPTION  Button
	CONTROL Button
	ALT     Button
}

func (x XTouch) NewModify() Modify {
	return Modify{
		SHIFT:   x.NewButton(0, 70),
		OPTION:  x.NewButton(0, 71),
		CONTROL: x.NewButton(0, 72),
		ALT:     x.NewButton(0, 73),
	}
}

type Automation struct {
	READ_OFF Button
	WRITE    Button
	TRIM     Button
	TOUCH    Button
	LATCH    Button
	GROUP    Button
}

func (x XTouch) NewAutomation() Automation {
	return Automation{
		READ_OFF: x.NewButton(0, 74),
		WRITE:    x.NewButton(0, 75),
		TRIM:     x.NewButton(0, 76),
		TOUCH:    x.NewButton(0, 77),
		LATCH:    x.NewButton(0, 78),
		GROUP:    x.NewButton(0, 79),
	}
}

type Utility struct {
	SAVE   Button
	UNDO   Button
	CANCEL Button
	ENTER  Button
}

func (x XTouch) NewUtility() Utility {
	return Utility{
		SAVE:   x.NewButton(0, 80),
		UNDO:   x.NewButton(0, 81),
		CANCEL: x.NewButton(0, 82),
		ENTER:  x.NewButton(0, 83),
	}
}

type Transport struct {
	Marker  Button
	Nudge   Button
	Cycle   Button
	Drop    Button
	Replace Button
	Click   Button
	Solo    Button
	REW     Button
	FF      Button
	STOP    Button
	PLAY    Button
	RECORD  Button
}

func (x *XTouch) NewTransport() Transport {
	return Transport{
		Marker:  x.NewButton(0, 84),
		Nudge:   x.NewButton(0, 85),
		Cycle:   x.NewButton(0, 86),
		Drop:    x.NewButton(0, 87),
		Replace: x.NewButton(0, 88),
		Click:   x.NewButton(0, 89),
		Solo:    x.NewButton(0, 90),
		REW:     x.NewButton(0, 91),
		FF:      x.NewButton(0, 92),
		STOP:    x.NewButton(0, 93),
		PLAY:    x.NewButton(0, 94),
		RECORD:  x.NewButton(0, 95),
	}
}

type Page struct {
	BANK_L    Button
	BANK_R    Button
	CHANNEL_L Button
	CHANNEL_R Button
}

func (x *XTouch) NewPage() Page {
	return Page{
		BANK_L:    x.NewButton(0, 46),
		BANK_R:    x.NewButton(0, 47),
		CHANNEL_L: x.NewButton(0, 48),
		CHANNEL_R: x.NewButton(0, 49),
	}
}

type Navigation struct {
	UP    Button
	DOWN  Button
	LEFT  Button
	RIGHT Button
	ZOOM  Button
	SCRUB Button
}

func (x *XTouch) NewNavigation() Navigation {
	return Navigation{
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
	XTouch

	Channels      []channelStrip
	EncoderAssign EncoderAssign
	View          View
	Function      Function
	Modify        Modify
	Automation    Automation
	Utility       Utility
	Transport     Transport
	Page          Page
	Navigation    Navigation
}

// New returns a properly initialized XTouchDefault struct.
func New(d dev.MidiDevice) XTouchDefault {
	x := XTouchDefault{
		XTouch: XTouch{d},
	}
	for i := 0; i < 8; i++ {
		x.Channels[i] = x.NewChannelStrip(uint8(i))
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
	XTouch

	Channels []channelStrip
}

func NewExtender(d dev.MidiDevice) XTouchExtender {
	x := XTouchExtender{
		XTouch: XTouch{d},
	}
	for i := 0; i < 8; i++ {
		x.Channels[i] = x.NewChannelStrip(uint8(i))
	}

	return x
}
