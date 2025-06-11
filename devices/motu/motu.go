package motu

import (
	"fmt"
)

type bindable[P, A any] interface {
	Bind(P, func(A) error)
}

// MOTU represents a connection to a MOTU AVB device
type MOTU struct {
	d      *HTTPDatastore
	Global *GlobalBindings
	AVB    *AVBBindings
	// Router *RouterBindings
	Mixer *MixerBindings
}

// NewMOTU creates a new MOTU connection with all bindings initialized
func NewMOTU(d *HTTPDatastore) *MOTU {
	m := &MOTU{d: d}
	m.Global = newGlobalBindings(m)
	m.AVB = newAVBBindings(m)
	// m.Router = newRouterBindings(m)
	m.Mixer = newMixerBindings(m)
	return m
}

// Global section
type GlobalBindings struct {
	m   *MOTU
	UID *UIDEndpoint
}

func newGlobalBindings(m *MOTU) *GlobalBindings {
	return &GlobalBindings{
		m:   m,
		UID: &UIDEndpoint{m: m},
	}
}

type UIDEndpoint struct {
	m *MOTU
}

func (e *UIDEndpoint) Bind(_ struct{}, callback func(string) error) {
	e.m.d.BindString("uid", callback)
}

// AVB section
type AVBBindings struct {
	m          *MOTU
	EntityName *EntityNameEndpoint
	ModelName  *ModelNameEndpoint
	Devices    *DevicesEndpoint
	Hostname   *HostnameEndpoint
	VendorName *VendorNameEndpoint
}

func newAVBBindings(m *MOTU) *AVBBindings {
	return &AVBBindings{
		m:          m,
		EntityName: &EntityNameEndpoint{m: m},
		ModelName:  &ModelNameEndpoint{m: m},
		Devices:    &DevicesEndpoint{m: m},
		Hostname:   &HostnameEndpoint{m: m},
		VendorName: &VendorNameEndpoint{m: m},
	}
}

type EntityNameEndpoint struct {
	m *MOTU
}

func (e *EntityNameEndpoint) Bind(deviceUID string, callback func(string) error) {
	path := fmt.Sprintf("avb/%s/entity_name", deviceUID)
	e.m.d.BindString(path, callback)
}

func (e *EntityNameEndpoint) Set(deviceUID string, val string) error {
	path := fmt.Sprintf("avb/%s/entity_name", deviceUID)
	return e.m.d.SetString(path, val)
}

type ModelNameEndpoint struct {
	m *MOTU
}

func (e *ModelNameEndpoint) Bind(deviceUID string, callback func(string) error) {
	path := fmt.Sprintf("avb/%s/model_name", deviceUID)
	e.m.d.BindString(path, callback)
}

type DevicesEndpoint struct {
	m *MOTU
}

// func (e *DevicesEndpoint) Bind(_ struct{}, callback func([]string) error) {
// 	e.m.d.BindStringList("avb/devs", callback)
// }

type HostnameEndpoint struct {
	m *MOTU
}

func (e *HostnameEndpoint) Bind(deviceUID string, callback func(string) error) {
	path := fmt.Sprintf("avb/%s/hostname", deviceUID)
	e.m.d.BindString(path, callback)
}

type VendorNameEndpoint struct {
	m *MOTU
}

func (e *VendorNameEndpoint) Bind(deviceUID string, callback func(string) error) {
	path := fmt.Sprintf("avb/%s/vendor_name", deviceUID)
	e.m.d.BindString(path, callback)
}

// Mixer section
type MixerBindings struct {
	m      *MOTU
	Chan   *MixerChannelBindings
	Main   *MixerMainBindings
	Aux    *MixerAuxBindings
	Group  *MixerGroupBindings
	Reverb *MixerReverbBindings
}

func newMixerBindings(m *MOTU) *MixerBindings {
	return &MixerBindings{
		m:      m,
		Chan:   newMixerChannelBindings(m),
		Main:   newMixerMainBindings(m),
		Aux:    newMixerAuxBindings(m),
		Group:  newMixerGroupBindings(m),
		Reverb: newMixerReverbBindings(m),
	}
}

// Channel controls
type MixerChannelBindings struct {
	m      *MOTU
	Matrix *ChannelMatrixBindings
	EQ     *ChannelEQBindings
	Gate   *ChannelGateBindings
	Comp   *ChannelCompBindings
}

func newMixerChannelBindings(m *MOTU) *MixerChannelBindings {
	return &MixerChannelBindings{
		m:      m,
		Matrix: newChannelMatrixBindings(m),
		EQ:     newChannelEQBindings(m),
		Gate:   newChannelGateBindings(m),
		Comp:   newChannelCompBindings(m),
	}
}

// Channel Matrix endpoints
type ChannelMatrixBindings struct {
	m         *MOTU
	Enable    *ChannelMatrixEnableEndpoint
	Solo      *ChannelMatrixSoloEndpoint
	Mute      *ChannelMatrixMuteEndpoint
	Pan       *ChannelMatrixPanEndpoint
	Fader     *ChannelMatrixFaderEndpoint
	AuxSend   *ChannelMatrixAuxSendEndpoint
	GroupSend *ChannelMatrixGroupSendEndpoint
}

func newChannelMatrixBindings(m *MOTU) *ChannelMatrixBindings {
	return &ChannelMatrixBindings{
		m:         m,
		Enable:    &ChannelMatrixEnableEndpoint{m: m},
		Solo:      &ChannelMatrixSoloEndpoint{m: m},
		Mute:      &ChannelMatrixMuteEndpoint{m: m},
		Pan:       &ChannelMatrixPanEndpoint{m: m},
		Fader:     &ChannelMatrixFaderEndpoint{m: m},
		AuxSend:   &ChannelMatrixAuxSendEndpoint{m: m},
		GroupSend: &ChannelMatrixGroupSendEndpoint{m: m},
	}
}

type PathChannelAuxSend struct {
	ChannelIndex int64
	AuxIndex     int64
}

type PathChannelGroupSend struct {
	ChannelIndex int64
	GroupIndex   int64
}

type ChannelMatrixEnableEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixEnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelMatrixEnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/matrix/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelMatrixSoloEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixSoloEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/solo", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelMatrixSoloEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/matrix/solo", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelMatrixMuteEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixMuteEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/mute", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelMatrixMuteEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/matrix/mute", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelMatrixPanEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixPanEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/pan", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelMatrixPanEndpoint) Set(channelIndex int64, val float64) error {
	path := fmt.Sprintf("mix/chan/%d/matrix/pan", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelMatrixFaderEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixFaderEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/fader", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelMatrixFaderEndpoint) Set(channelIndex int64, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("fader value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/chan/%d/matrix/fader", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelMatrixAuxSendEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixAuxSendEndpoint) Bind(p PathChannelAuxSend, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/aux/%d/send", p.ChannelIndex, p.AuxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelMatrixAuxSendEndpoint) Set(p PathChannelAuxSend, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("aux send value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/chan/%d/matrix/aux/%d/send", p.ChannelIndex, p.AuxIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelMatrixGroupSendEndpoint struct {
	m *MOTU
}

func (e *ChannelMatrixGroupSendEndpoint) Bind(p PathChannelGroupSend, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/matrix/group/%d/send", p.ChannelIndex, p.GroupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelMatrixGroupSendEndpoint) Set(p PathChannelGroupSend, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("group send value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/chan/%d/matrix/group/%d/send", p.ChannelIndex, p.GroupIndex)
	return e.m.d.SetFloat(path, val)
}

// Channel EQ endpoints
type ChannelEQBindings struct {
	m         *MOTU
	HighShelf *ChannelEQHighShelfBindings
	Mid1      *ChannelEQMid1Bindings
	Mid2      *ChannelEQMid2Bindings
	LowShelf  *ChannelEQLowShelfBindings
}

func newChannelEQBindings(m *MOTU) *ChannelEQBindings {
	return &ChannelEQBindings{
		m:         m,
		HighShelf: newChannelEQHighShelfBindings(m),
		Mid1:      newChannelEQMid1Bindings(m),
		Mid2:      newChannelEQMid2Bindings(m),
		LowShelf:  newChannelEQLowShelfBindings(m),
	}
}

type ChannelEQHighShelfBindings struct {
	m      *MOTU
	Enable *ChannelEQHighShelfEnableEndpoint
	Freq   *ChannelEQHighShelfFreqEndpoint
	Gain   *ChannelEQHighShelfGainEndpoint
	BW     *ChannelEQHighShelfBWEndpoint
	Mode   *ChannelEQHighShelfModeEndpoint
}

func newChannelEQHighShelfBindings(m *MOTU) *ChannelEQHighShelfBindings {
	return &ChannelEQHighShelfBindings{
		m:      m,
		Enable: &ChannelEQHighShelfEnableEndpoint{m: m},
		Freq:   &ChannelEQHighShelfFreqEndpoint{m: m},
		Gain:   &ChannelEQHighShelfGainEndpoint{m: m},
		BW:     &ChannelEQHighShelfBWEndpoint{m: m},
		Mode:   &ChannelEQHighShelfModeEndpoint{m: m},
	}
}

type ChannelEQHighShelfEnableEndpoint struct {
	m *MOTU
}

func (e *ChannelEQHighShelfEnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelEQHighShelfEnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelEQHighShelfFreqEndpoint struct {
	m *MOTU
}

func (e *ChannelEQHighShelfFreqEndpoint) Bind(channelIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/freq", channelIndex)
	e.m.d.BindInt(path, callback)
}

func (e *ChannelEQHighShelfFreqEndpoint) Set(channelIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/freq", channelIndex)
	return e.m.d.SetInt(path, val)
}

type ChannelEQHighShelfGainEndpoint struct {
	m *MOTU
}

func (e *ChannelEQHighShelfGainEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/gain", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQHighShelfGainEndpoint) Set(channelIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/gain", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQHighShelfBWEndpoint struct {
	m *MOTU
}

func (e *ChannelEQHighShelfBWEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/bw", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQHighShelfBWEndpoint) Set(channelIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/bw", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQHighShelfModeEndpoint struct {
	m *MOTU
}

func (e *ChannelEQHighShelfModeEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/mode", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQHighShelfModeEndpoint) Set(channelIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/highshelf/mode", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid1 EQ Bindings
type ChannelEQMid1Bindings struct {
	m      *MOTU
	Enable *ChannelEQMid1EnableEndpoint
	Freq   *ChannelEQMid1FreqEndpoint
	Gain   *ChannelEQMid1GainEndpoint
	BW     *ChannelEQMid1BWEndpoint
}

func newChannelEQMid1Bindings(m *MOTU) *ChannelEQMid1Bindings {
	return &ChannelEQMid1Bindings{
		m:      m,
		Enable: &ChannelEQMid1EnableEndpoint{m: m},
		Freq:   &ChannelEQMid1FreqEndpoint{m: m},
		Gain:   &ChannelEQMid1GainEndpoint{m: m},
		BW:     &ChannelEQMid1BWEndpoint{m: m},
	}
}

type ChannelEQMid1EnableEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid1EnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelEQMid1EnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelEQMid1FreqEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid1FreqEndpoint) Bind(channelIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/freq", channelIndex)
	e.m.d.BindInt(path, callback)
}

func (e *ChannelEQMid1FreqEndpoint) Set(channelIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/freq", channelIndex)
	return e.m.d.SetInt(path, val)
}

type ChannelEQMid1GainEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid1GainEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/gain", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQMid1GainEndpoint) Set(channelIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/gain", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQMid1BWEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid1BWEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/bw", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQMid1BWEndpoint) Set(channelIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid1/bw", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid2 EQ Bindings
type ChannelEQMid2Bindings struct {
	m      *MOTU
	Enable *ChannelEQMid2EnableEndpoint
	Freq   *ChannelEQMid2FreqEndpoint
	Gain   *ChannelEQMid2GainEndpoint
	BW     *ChannelEQMid2BWEndpoint
}

func newChannelEQMid2Bindings(m *MOTU) *ChannelEQMid2Bindings {
	return &ChannelEQMid2Bindings{
		m:      m,
		Enable: &ChannelEQMid2EnableEndpoint{m: m},
		Freq:   &ChannelEQMid2FreqEndpoint{m: m},
		Gain:   &ChannelEQMid2GainEndpoint{m: m},
		BW:     &ChannelEQMid2BWEndpoint{m: m},
	}
}

type ChannelEQMid2EnableEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid2EnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelEQMid2EnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelEQMid2FreqEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid2FreqEndpoint) Bind(channelIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/freq", channelIndex)
	e.m.d.BindInt(path, callback)
}

func (e *ChannelEQMid2FreqEndpoint) Set(channelIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/freq", channelIndex)
	return e.m.d.SetInt(path, val)
}

type ChannelEQMid2GainEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid2GainEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/gain", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQMid2GainEndpoint) Set(channelIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/gain", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQMid2BWEndpoint struct {
	m *MOTU
}

func (e *ChannelEQMid2BWEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/bw", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQMid2BWEndpoint) Set(channelIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/mid2/bw", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// LowShelf EQ Bindings
type ChannelEQLowShelfBindings struct {
	m      *MOTU
	Enable *ChannelEQLowShelfEnableEndpoint
	Freq   *ChannelEQLowShelfFreqEndpoint
	Gain   *ChannelEQLowShelfGainEndpoint
	BW     *ChannelEQLowShelfBWEndpoint
	Mode   *ChannelEQLowShelfModeEndpoint
}

func newChannelEQLowShelfBindings(m *MOTU) *ChannelEQLowShelfBindings {
	return &ChannelEQLowShelfBindings{
		m:      m,
		Enable: &ChannelEQLowShelfEnableEndpoint{m: m},
		Freq:   &ChannelEQLowShelfFreqEndpoint{m: m},
		Gain:   &ChannelEQLowShelfGainEndpoint{m: m},
		BW:     &ChannelEQLowShelfBWEndpoint{m: m},
		Mode:   &ChannelEQLowShelfModeEndpoint{m: m},
	}
}

type ChannelEQLowShelfEnableEndpoint struct {
	m *MOTU
}

func (e *ChannelEQLowShelfEnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelEQLowShelfEnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelEQLowShelfFreqEndpoint struct {
	m *MOTU
}

func (e *ChannelEQLowShelfFreqEndpoint) Bind(channelIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/freq", channelIndex)
	e.m.d.BindInt(path, callback)
}

func (e *ChannelEQLowShelfFreqEndpoint) Set(channelIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/freq", channelIndex)
	return e.m.d.SetInt(path, val)
}

type ChannelEQLowShelfGainEndpoint struct {
	m *MOTU
}

func (e *ChannelEQLowShelfGainEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/gain", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQLowShelfGainEndpoint) Set(channelIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/gain", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQLowShelfBWEndpoint struct {
	m *MOTU
}

func (e *ChannelEQLowShelfBWEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/bw", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQLowShelfBWEndpoint) Set(channelIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/bw", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelEQLowShelfModeEndpoint struct {
	m *MOTU
}

func (e *ChannelEQLowShelfModeEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/mode", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelEQLowShelfModeEndpoint) Set(channelIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/chan/%d/eq/lowshelf/mode", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// Channel Gate endpoints
type ChannelGateBindings struct {
	m         *MOTU
	Enable    *ChannelGateEnableEndpoint
	Release   *ChannelGateReleaseEndpoint
	Threshold *ChannelGateThresholdEndpoint
	Attack    *ChannelGateAttackEndpoint
}

func newChannelGateBindings(m *MOTU) *ChannelGateBindings {
	return &ChannelGateBindings{
		m:         m,
		Enable:    &ChannelGateEnableEndpoint{m: m},
		Release:   &ChannelGateReleaseEndpoint{m: m},
		Threshold: &ChannelGateThresholdEndpoint{m: m},
		Attack:    &ChannelGateAttackEndpoint{m: m},
	}
}

type ChannelGateEnableEndpoint struct {
	m *MOTU
}

func (e *ChannelGateEnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/gate/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelGateEnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/gate/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelGateReleaseEndpoint struct {
	m *MOTU
}

func (e *ChannelGateReleaseEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/gate/release", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelGateReleaseEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/chan/%d/gate/release", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelGateThresholdEndpoint struct {
	m *MOTU
}

func (e *ChannelGateThresholdEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/gate/threshold", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelGateThresholdEndpoint) Set(channelIndex int64, val float64) error {
	// Linear units per API spec
	path := fmt.Sprintf("mix/chan/%d/gate/threshold", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelGateAttackEndpoint struct {
	m *MOTU
}

func (e *ChannelGateAttackEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/gate/attack", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelGateAttackEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/chan/%d/gate/attack", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// Channel Compressor endpoints
type ChannelCompBindings struct {
	m         *MOTU
	Enable    *ChannelCompEnableEndpoint
	Release   *ChannelCompReleaseEndpoint
	Threshold *ChannelCompThresholdEndpoint
	Ratio     *ChannelCompRatioEndpoint
	Attack    *ChannelCompAttackEndpoint
	Trim      *ChannelCompTrimEndpoint
	Peak      *ChannelCompPeakEndpoint
}

func newChannelCompBindings(m *MOTU) *ChannelCompBindings {
	return &ChannelCompBindings{
		m:         m,
		Enable:    &ChannelCompEnableEndpoint{m: m},
		Release:   &ChannelCompReleaseEndpoint{m: m},
		Threshold: &ChannelCompThresholdEndpoint{m: m},
		Ratio:     &ChannelCompRatioEndpoint{m: m},
		Attack:    &ChannelCompAttackEndpoint{m: m},
		Trim:      &ChannelCompTrimEndpoint{m: m},
		Peak:      &ChannelCompPeakEndpoint{m: m},
	}
}

type ChannelCompEnableEndpoint struct {
	m *MOTU
}

func (e *ChannelCompEnableEndpoint) Bind(channelIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/enable", channelIndex)
	e.m.d.BindBool(path, callback)
}

func (e *ChannelCompEnableEndpoint) Set(channelIndex int64, val bool) error {
	path := fmt.Sprintf("mix/chan/%d/comp/enable", channelIndex)
	return e.m.d.SetBool(path, val)
}

type ChannelCompReleaseEndpoint struct {
	m *MOTU
}

func (e *ChannelCompReleaseEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/release", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompReleaseEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/chan/%d/comp/release", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelCompThresholdEndpoint struct {
	m *MOTU
}

func (e *ChannelCompThresholdEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/threshold", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompThresholdEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in dB per API spec
	path := fmt.Sprintf("mix/chan/%d/comp/threshold", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelCompRatioEndpoint struct {
	m *MOTU
}

func (e *ChannelCompRatioEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/ratio", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompRatioEndpoint) Set(channelIndex int64, val float64) error {
	path := fmt.Sprintf("mix/chan/%d/comp/ratio", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelCompAttackEndpoint struct {
	m *MOTU
}

func (e *ChannelCompAttackEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/attack", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompAttackEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/chan/%d/comp/attack", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelCompTrimEndpoint struct {
	m *MOTU
}

func (e *ChannelCompTrimEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/trim", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompTrimEndpoint) Set(channelIndex int64, val float64) error {
	// Expressed in dB per API spec
	path := fmt.Sprintf("mix/chan/%d/comp/trim", channelIndex)
	return e.m.d.SetFloat(path, val)
}

type ChannelCompPeakEndpoint struct {
	m *MOTU
}

func (e *ChannelCompPeakEndpoint) Bind(channelIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/chan/%d/comp/peak", channelIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *ChannelCompPeakEndpoint) Set(channelIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("peak mode must be 0 (RMS) or 1 (Peak)")
	}
	path := fmt.Sprintf("mix/chan/%d/comp/peak", channelIndex)
	return e.m.d.SetFloat(path, val)
}

// Main section bindings
type MixerMainBindings struct {
	m       *MOTU
	Matrix  *MainMatrixBindings
	EQ      *MainEQBindings
	Leveler *MainLevelerBindings
}

func newMixerMainBindings(m *MOTU) *MixerMainBindings {
	return &MixerMainBindings{
		m:       m,
		Matrix:  newMainMatrixBindings(m),
		EQ:      newMainEQBindings(m),
		Leveler: newMainLevelerBindings(m),
	}
}

// Main Matrix bindings
type MainMatrixBindings struct {
	m      *MOTU
	Enable *MainMatrixEnableEndpoint
	Mute   *MainMatrixMuteEndpoint
	Fader  *MainMatrixFaderEndpoint
}

func newMainMatrixBindings(m *MOTU) *MainMatrixBindings {
	return &MainMatrixBindings{
		m:      m,
		Enable: &MainMatrixEnableEndpoint{m: m},
		Mute:   &MainMatrixMuteEndpoint{m: m},
		Fader:  &MainMatrixFaderEndpoint{m: m},
	}
}

type MainMatrixEnableEndpoint struct {
	m *MOTU
}

func (e *MainMatrixEnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/main/%d/matrix/enable", mainIndex)
	e.m.d.BindBool(path, callback)
}

func (e *MainMatrixEnableEndpoint) Set(mainIndex int64, val bool) error {
	path := fmt.Sprintf("mix/main/%d/matrix/enable", mainIndex)
	return e.m.d.SetBool(path, val)
}

type MainMatrixMuteEndpoint struct {
	m *MOTU
}

func (e *MainMatrixMuteEndpoint) Bind(mainIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/main/%d/matrix/mute", mainIndex)
	e.m.d.BindBool(path, callback)
}

func (e *MainMatrixMuteEndpoint) Set(mainIndex int64, val bool) error {
	path := fmt.Sprintf("mix/main/%d/matrix/mute", mainIndex)
	return e.m.d.SetBool(path, val)
}

type MainMatrixFaderEndpoint struct {
	m *MOTU
}

func (e *MainMatrixFaderEndpoint) Bind(mainIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/main/%d/matrix/fader", mainIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *MainMatrixFaderEndpoint) Set(mainIndex int64, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("fader value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/main/%d/matrix/fader", mainIndex)
	return e.m.d.SetFloat(path, val)
}

// Main EQ bindings
type MainEQBindings struct {
	m         *MOTU
	HighShelf *MainEQHighShelfBindings
	Mid1      *MainEQMid1Bindings
	Mid2      *MainEQMid2Bindings
	LowShelf  *MainEQLowShelfBindings
}

func newMainEQBindings(m *MOTU) *MainEQBindings {
	return &MainEQBindings{
		m:         m,
		HighShelf: newMainEQHighShelfBindings(m),
		Mid1:      newMainEQMid1Bindings(m),
		Mid2:      newMainEQMid2Bindings(m),
		LowShelf:  newMainEQLowShelfBindings(m),
	}
}

type MainEQHighShelfBindings struct {
	m      *MOTU
	Enable *MainEQHighShelfEnableEndpoint
	Freq   *MainEQHighShelfFreqEndpoint
	Gain   *MainEQHighShelfGainEndpoint
	BW     *MainEQHighShelfBWEndpoint
	Mode   *MainEQHighShelfModeEndpoint
}

func newMainEQHighShelfBindings(m *MOTU) *MainEQHighShelfBindings {
	return &MainEQHighShelfBindings{
		m:      m,
		Enable: &MainEQHighShelfEnableEndpoint{m: m},
		Freq:   &MainEQHighShelfFreqEndpoint{m: m},
		Gain:   &MainEQHighShelfGainEndpoint{m: m},
		BW:     &MainEQHighShelfBWEndpoint{m: m},
		Mode:   &MainEQHighShelfModeEndpoint{m: m},
	}
}

type MainEQHighShelfEnableEndpoint struct {
	m *MOTU
}

func (e *MainEQHighShelfEnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/enable", mainIndex)
	e.m.d.BindBool(path, callback)
}

func (e *MainEQHighShelfEnableEndpoint) Set(mainIndex int64, val bool) error {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/enable", mainIndex)
	return e.m.d.SetBool(path, val)
}

type MainEQHighShelfFreqEndpoint struct {
	m *MOTU
}

func (e *MainEQHighShelfFreqEndpoint) Bind(mainIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/freq", mainIndex)
	e.m.d.BindInt(path, callback)
}

func (e *MainEQHighShelfFreqEndpoint) Set(mainIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/freq", mainIndex)
	return e.m.d.SetInt(path, val)
}

type MainEQHighShelfGainEndpoint struct {
	m *MOTU
}

func (e *MainEQHighShelfGainEndpoint) Bind(mainIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/gain", mainIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *MainEQHighShelfGainEndpoint) Set(mainIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/gain", mainIndex)
	return e.m.d.SetFloat(path, val)
}

type MainEQHighShelfBWEndpoint struct {
	m *MOTU
}

func (e *MainEQHighShelfBWEndpoint) Bind(mainIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/bw", mainIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *MainEQHighShelfBWEndpoint) Set(mainIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/bw", mainIndex)
	return e.m.d.SetFloat(path, val)
}

type MainEQHighShelfModeEndpoint struct {
	m *MOTU
}

func (e *MainEQHighShelfModeEndpoint) Bind(mainIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/mode", mainIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *MainEQHighShelfModeEndpoint) Set(mainIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/main/%d/eq/highshelf/mode", mainIndex)
	return e.m.d.SetFloat(path, val)
}

i// Mid1 EQ bindings for Main
type MainEQMid1Bindings struct {
    m *MOTU
    Enable *MainEQMid1EnableEndpoint
    Freq   *MainEQMid1FreqEndpoint
    Gain   *MainEQMid1GainEndpoint
    BW     *MainEQMid1BWEndpoint
}

func newMainEQMid1Bindings(m *MOTU) *MainEQMid1Bindings {
    return &MainEQMid1Bindings{
        m: m,
        Enable: &MainEQMid1EnableEndpoint{m: m},
        Freq: &MainEQMid1FreqEndpoint{m: m},
        Gain: &MainEQMid1GainEndpoint{m: m},
        BW: &MainEQMid1BWEndpoint{m: m},
    }
}

type MainEQMid1EnableEndpoint struct {
    m *MOTU
}

func (e *MainEQMid1EnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid1/enable", mainIndex)
    e.m.d.BindBool(path, callback)
}

func (e *MainEQMid1EnableEndpoint) Set(mainIndex int64, val bool) error {
    path := fmt.Sprintf("mix/main/%d/eq/mid1/enable", mainIndex)
    return e.m.d.SetBool(path, val)
}

type MainEQMid1FreqEndpoint struct {
    m *MOTU
}

func (e *MainEQMid1FreqEndpoint) Bind(mainIndex int64, callback func(int64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid1/freq", mainIndex)
    e.m.d.BindInt(path, callback)
}

func (e *MainEQMid1FreqEndpoint) Set(mainIndex int64, val int64) error {
    if val < 20 || val > 20000 {
        return fmt.Errorf("frequency must be between 20 and 20000 Hz")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid1/freq", mainIndex)
    return e.m.d.SetInt(path, val)
}

type MainEQMid1GainEndpoint struct {
    m *MOTU
}

func (e *MainEQMid1GainEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid1/gain", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQMid1GainEndpoint) Set(mainIndex int64, val float64) error {
    if val < -20 || val > 20 {
        return fmt.Errorf("gain must be between -20 and 20 dB")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid1/gain", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainEQMid1BWEndpoint struct {
    m *MOTU
}

func (e *MainEQMid1BWEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid1/bw", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQMid1BWEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0.01 || val > 3 {
        return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid1/bw", mainIndex)
    return e.m.d.SetFloat(path, val)
}

// Mid2 EQ bindings for Main
type MainEQMid2Bindings struct {
    m *MOTU
    Enable *MainEQMid2EnableEndpoint
    Freq   *MainEQMid2FreqEndpoint
    Gain   *MainEQMid2GainEndpoint
    BW     *MainEQMid2BWEndpoint
}

func newMainEQMid2Bindings(m *MOTU) *MainEQMid2Bindings {
    return &MainEQMid2Bindings{
        m: m,
        Enable: &MainEQMid2EnableEndpoint{m: m},
        Freq: &MainEQMid2FreqEndpoint{m: m},
        Gain: &MainEQMid2GainEndpoint{m: m},
        BW: &MainEQMid2BWEndpoint{m: m},
    }
}

type MainEQMid2EnableEndpoint struct {
    m *MOTU
}

func (e *MainEQMid2EnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid2/enable", mainIndex)
    e.m.d.BindBool(path, callback)
}

func (e *MainEQMid2EnableEndpoint) Set(mainIndex int64, val bool) error {
    path := fmt.Sprintf("mix/main/%d/eq/mid2/enable", mainIndex)
    return e.m.d.SetBool(path, val)
}

type MainEQMid2FreqEndpoint struct {
    m *MOTU
}

func (e *MainEQMid2FreqEndpoint) Bind(mainIndex int64, callback func(int64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid2/freq", mainIndex)
    e.m.d.BindInt(path, callback)
}

func (e *MainEQMid2FreqEndpoint) Set(mainIndex int64, val int64) error {
    if val < 20 || val > 20000 {
        return fmt.Errorf("frequency must be between 20 and 20000 Hz")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid2/freq", mainIndex)
    return e.m.d.SetInt(path, val)
}

type MainEQMid2GainEndpoint struct {
    m *MOTU
}

func (e *MainEQMid2GainEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid2/gain", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQMid2GainEndpoint) Set(mainIndex int64, val float64) error {
    if val < -20 || val > 20 {
        return fmt.Errorf("gain must be between -20 and 20 dB")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid2/gain", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainEQMid2BWEndpoint struct {
    m *MOTU
}

func (e *MainEQMid2BWEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/mid2/bw", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQMid2BWEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0.01 || val > 3 {
        return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
    }
    path := fmt.Sprintf("mix/main/%d/eq/mid2/bw", mainIndex)
    return e.m.d.SetFloat(path, val)
}

// LowShelf EQ bindings for Main
type MainEQLowShelfBindings struct {
    m *MOTU
    Enable *MainEQLowShelfEnableEndpoint
    Freq   *MainEQLowShelfFreqEndpoint
    Gain   *MainEQLowShelfGainEndpoint
    BW     *MainEQLowShelfBWEndpoint
    Mode   *MainEQLowShelfModeEndpoint
}

func newMainEQLowShelfBindings(m *MOTU) *MainEQLowShelfBindings {
    return &MainEQLowShelfBindings{
        m: m,
        Enable: &MainEQLowShelfEnableEndpoint{m: m},
        Freq: &MainEQLowShelfFreqEndpoint{m: m},
        Gain: &MainEQLowShelfGainEndpoint{m: m},
        BW: &MainEQLowShelfBWEndpoint{m: m},
        Mode: &MainEQLowShelfModeEndpoint{m: m},
    }
}

type MainEQLowShelfEnableEndpoint struct {
    m *MOTU
}

func (e *MainEQLowShelfEnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/enable", mainIndex)
    e.m.d.BindBool(path, callback)
}

func (e *MainEQLowShelfEnableEndpoint) Set(mainIndex int64, val bool) error {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/enable", mainIndex)
    return e.m.d.SetBool(path, val)
}

type MainEQLowShelfFreqEndpoint struct {
    m *MOTU
}

func (e *MainEQLowShelfFreqEndpoint) Bind(mainIndex int64, callback func(int64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/freq", mainIndex)
    e.m.d.BindInt(path, callback)
}

func (e *MainEQLowShelfFreqEndpoint) Set(mainIndex int64, val int64) error {
    if val < 20 || val > 20000 {
        return fmt.Errorf("frequency must be between 20 and 20000 Hz")
    }
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/freq", mainIndex)
    return e.m.d.SetInt(path, val)
}

type MainEQLowShelfGainEndpoint struct {
    m *MOTU
}

func (e *MainEQLowShelfGainEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/gain", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQLowShelfGainEndpoint) Set(mainIndex int64, val float64) error {
    if val < -20 || val > 20 {
        return fmt.Errorf("gain must be between -20 and 20 dB")
    }
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/gain", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainEQLowShelfBWEndpoint struct {
    m *MOTU
}

func (e *MainEQLowShelfBWEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/bw", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQLowShelfBWEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0.01 || val > 3 {
        return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
    }
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/bw", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainEQLowShelfModeEndpoint struct {
    m *MOTU
}

func (e *MainEQLowShelfModeEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/mode", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainEQLowShelfModeEndpoint) Set(mainIndex int64, val float64) error {
    if val != 0 && val != 1 {
        return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
    }
    path := fmt.Sprintf("mix/main/%d/eq/lowshelf/mode", mainIndex)
    return e.m.d.SetFloat(path, val)
}

// Main Leveler bindings
type MainLevelerBindings struct {
    m *MOTU
    Enable    *MainLevelerEnableEndpoint
    Makeup    *MainLevelerMakeupEndpoint
    Reduction *MainLevelerReductionEndpoint
    Limit     *MainLevelerLimitEndpoint
}

func newMainLevelerBindings(m *MOTU) *MainLevelerBindings {
    return &MainLevelerBindings{
        m: m,
        Enable: &MainLevelerEnableEndpoint{m: m},
        Makeup: &MainLevelerMakeupEndpoint{m: m},
        Reduction: &MainLevelerReductionEndpoint{m: m},
        Limit: &MainLevelerLimitEndpoint{m: m},
    }
}

type MainLevelerEnableEndpoint struct {
    m *MOTU
}

func (e *MainLevelerEnableEndpoint) Bind(mainIndex int64, callback func(bool) error) {
    path := fmt.Sprintf("mix/main/%d/leveler/enable", mainIndex)
    e.m.d.BindBool(path, callback)
}

func (e *MainLevelerEnableEndpoint) Set(mainIndex int64, val bool) error {
    path := fmt.Sprintf("mix/main/%d/leveler/enable", mainIndex)
    return e.m.d.SetBool(path, val)
}

type MainLevelerMakeupEndpoint struct {
    m *MOTU
}

func (e *MainLevelerMakeupEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/leveler/makeup", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainLevelerMakeupEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0 || val > 100 {
        return fmt.Errorf("makeup must be between 0 and 100%%")
    }
    path := fmt.Sprintf("mix/main/%d/leveler/makeup", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainLevelerReductionEndpoint struct {
    m *MOTU
}

func (e *MainLevelerReductionEndpoint) Bind(mainIndex int64, callback func(float64) error) {
    path := fmt.Sprintf("mix/main/%d/leveler/reduction", mainIndex)
    e.m.d.BindFloat(path, callback)
}

func (e *MainLevelerReductionEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0 || val > 100 {
        return fmt.Errorf("reduction must be between 0 and 100%%")
    }
    path := fmt.Sprintf("mix/main/%d/leveler/reduction", mainIndex)
    return e.m.d.SetFloat(path, val)
}

type MainLevelerLimitEndpoint struct {
    m *MOTU
}

func (e *MainLevelerLimitEndpoint) Bind(mainIndex int64, callback func(bool) error) {
    path := fmt.Sprintf("mix/main/%d/leveler/limit", mainIndex)
    e.m.d.BindBool(path, callback)
}

func (e *MainLevelerLimitEndpoint) Set(mainIndex int64, val bool) error {
    path := fmt.Sprintf("mix/main/%d/leveler/limit", mainIndex)
    return e.m.d.SetBool(path, val)
}


