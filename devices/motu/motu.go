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

// Mid1 EQ bindings for Main
type MainEQMid1Bindings struct {
	m      *MOTU
	Enable *MainEQMid1EnableEndpoint
	Freq   *MainEQMid1FreqEndpoint
	Gain   *MainEQMid1GainEndpoint
	BW     *MainEQMid1BWEndpoint
}

func newMainEQMid1Bindings(m *MOTU) *MainEQMid1Bindings {
	return &MainEQMid1Bindings{
		m:      m,
		Enable: &MainEQMid1EnableEndpoint{m: m},
		Freq:   &MainEQMid1FreqEndpoint{m: m},
		Gain:   &MainEQMid1GainEndpoint{m: m},
		BW:     &MainEQMid1BWEndpoint{m: m},
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
	m      *MOTU
	Enable *MainEQMid2EnableEndpoint
	Freq   *MainEQMid2FreqEndpoint
	Gain   *MainEQMid2GainEndpoint
	BW     *MainEQMid2BWEndpoint
}

func newMainEQMid2Bindings(m *MOTU) *MainEQMid2Bindings {
	return &MainEQMid2Bindings{
		m:      m,
		Enable: &MainEQMid2EnableEndpoint{m: m},
		Freq:   &MainEQMid2FreqEndpoint{m: m},
		Gain:   &MainEQMid2GainEndpoint{m: m},
		BW:     &MainEQMid2BWEndpoint{m: m},
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
	m      *MOTU
	Enable *MainEQLowShelfEnableEndpoint
	Freq   *MainEQLowShelfFreqEndpoint
	Gain   *MainEQLowShelfGainEndpoint
	BW     *MainEQLowShelfBWEndpoint
	Mode   *MainEQLowShelfModeEndpoint
}

func newMainEQLowShelfBindings(m *MOTU) *MainEQLowShelfBindings {
	return &MainEQLowShelfBindings{
		m:      m,
		Enable: &MainEQLowShelfEnableEndpoint{m: m},
		Freq:   &MainEQLowShelfFreqEndpoint{m: m},
		Gain:   &MainEQLowShelfGainEndpoint{m: m},
		BW:     &MainEQLowShelfBWEndpoint{m: m},
		Mode:   &MainEQLowShelfModeEndpoint{m: m},
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
	m         *MOTU
	Enable    *MainLevelerEnableEndpoint
	Makeup    *MainLevelerMakeupEndpoint
	Reduction *MainLevelerReductionEndpoint
	Limit     *MainLevelerLimitEndpoint
}

func newMainLevelerBindings(m *MOTU) *MainLevelerBindings {
	return &MainLevelerBindings{
		m:         m,
		Enable:    &MainLevelerEnableEndpoint{m: m},
		Makeup:    &MainLevelerMakeupEndpoint{m: m},
		Reduction: &MainLevelerReductionEndpoint{m: m},
		Limit:     &MainLevelerLimitEndpoint{m: m},
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

// Aux section bindings
type MixerAuxBindings struct {
	m      *MOTU
	Matrix *AuxMatrixBindings
	EQ     *AuxEQBindings
}

func newMixerAuxBindings(m *MOTU) *MixerAuxBindings {
	return &MixerAuxBindings{
		m:      m,
		Matrix: newAuxMatrixBindings(m),
		EQ:     newAuxEQBindings(m),
	}
}

// Aux Matrix bindings
type AuxMatrixBindings struct {
	m        *MOTU
	Enable   *AuxMatrixEnableEndpoint
	PreFader *AuxMatrixPreFaderEndpoint
	Panner   *AuxMatrixPannerEndpoint
	Mute     *AuxMatrixMuteEndpoint
	Fader    *AuxMatrixFaderEndpoint
}

func newAuxMatrixBindings(m *MOTU) *AuxMatrixBindings {
	return &AuxMatrixBindings{
		m:        m,
		Enable:   &AuxMatrixEnableEndpoint{m: m},
		PreFader: &AuxMatrixPreFaderEndpoint{m: m},
		Panner:   &AuxMatrixPannerEndpoint{m: m},
		Mute:     &AuxMatrixMuteEndpoint{m: m},
		Fader:    &AuxMatrixFaderEndpoint{m: m},
	}
}

type AuxMatrixEnableEndpoint struct {
	m *MOTU
}

func (e *AuxMatrixEnableEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/matrix/enable", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxMatrixEnableEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/matrix/enable", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxMatrixPreFaderEndpoint struct {
	m *MOTU
}

func (e *AuxMatrixPreFaderEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/matrix/prefader", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxMatrixPreFaderEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/matrix/prefader", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxMatrixPannerEndpoint struct {
	m *MOTU
}

func (e *AuxMatrixPannerEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/matrix/panner", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxMatrixPannerEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/matrix/panner", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxMatrixMuteEndpoint struct {
	m *MOTU
}

func (e *AuxMatrixMuteEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/matrix/mute", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxMatrixMuteEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/matrix/mute", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxMatrixFaderEndpoint struct {
	m *MOTU
}

func (e *AuxMatrixFaderEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/matrix/fader", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxMatrixFaderEndpoint) Set(auxIndex int64, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("fader value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/aux/%d/matrix/fader", auxIndex)
	return e.m.d.SetFloat(path, val)
}

// Aux EQ bindings
type AuxEQBindings struct {
	m         *MOTU
	HighShelf *AuxEQHighShelfBindings
	Mid1      *AuxEQMid1Bindings
	Mid2      *AuxEQMid2Bindings
	LowShelf  *AuxEQLowShelfBindings
}

func newAuxEQBindings(m *MOTU) *AuxEQBindings {
	return &AuxEQBindings{
		m:         m,
		HighShelf: newAuxEQHighShelfBindings(m),
		Mid1:      newAuxEQMid1Bindings(m),
		Mid2:      newAuxEQMid2Bindings(m),
		LowShelf:  newAuxEQLowShelfBindings(m),
	}
}

type AuxEQHighShelfBindings struct {
	m      *MOTU
	Enable *AuxEQHighShelfEnableEndpoint
	Freq   *AuxEQHighShelfFreqEndpoint
	Gain   *AuxEQHighShelfGainEndpoint
	BW     *AuxEQHighShelfBWEndpoint
	Mode   *AuxEQHighShelfModeEndpoint
}

func newAuxEQHighShelfBindings(m *MOTU) *AuxEQHighShelfBindings {
	return &AuxEQHighShelfBindings{
		m:      m,
		Enable: &AuxEQHighShelfEnableEndpoint{m: m},
		Freq:   &AuxEQHighShelfFreqEndpoint{m: m},
		Gain:   &AuxEQHighShelfGainEndpoint{m: m},
		BW:     &AuxEQHighShelfBWEndpoint{m: m},
		Mode:   &AuxEQHighShelfModeEndpoint{m: m},
	}
}

type AuxEQHighShelfEnableEndpoint struct {
	m *MOTU
}

func (e *AuxEQHighShelfEnableEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/enable", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxEQHighShelfEnableEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/enable", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxEQHighShelfFreqEndpoint struct {
	m *MOTU
}

func (e *AuxEQHighShelfFreqEndpoint) Bind(auxIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/freq", auxIndex)
	e.m.d.BindInt(path, callback)
}

func (e *AuxEQHighShelfFreqEndpoint) Set(auxIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/freq", auxIndex)
	return e.m.d.SetInt(path, val)
}

type AuxEQHighShelfGainEndpoint struct {
	m *MOTU
}

func (e *AuxEQHighShelfGainEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/gain", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQHighShelfGainEndpoint) Set(auxIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/gain", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQHighShelfBWEndpoint struct {
	m *MOTU
}

func (e *AuxEQHighShelfBWEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/bw", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQHighShelfBWEndpoint) Set(auxIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/bw", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQHighShelfModeEndpoint struct {
	m *MOTU
}

func (e *AuxEQHighShelfModeEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/mode", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQHighShelfModeEndpoint) Set(auxIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/highshelf/mode", auxIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid1 EQ bindings for Aux
type AuxEQMid1Bindings struct {
	m      *MOTU
	Enable *AuxEQMid1EnableEndpoint
	Freq   *AuxEQMid1FreqEndpoint
	Gain   *AuxEQMid1GainEndpoint
	BW     *AuxEQMid1BWEndpoint
}

func newAuxEQMid1Bindings(m *MOTU) *AuxEQMid1Bindings {
	return &AuxEQMid1Bindings{
		m:      m,
		Enable: &AuxEQMid1EnableEndpoint{m: m},
		Freq:   &AuxEQMid1FreqEndpoint{m: m},
		Gain:   &AuxEQMid1GainEndpoint{m: m},
		BW:     &AuxEQMid1BWEndpoint{m: m},
	}
}

type AuxEQMid1EnableEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid1EnableEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/enable", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxEQMid1EnableEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/enable", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxEQMid1FreqEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid1FreqEndpoint) Bind(auxIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/freq", auxIndex)
	e.m.d.BindInt(path, callback)
}

func (e *AuxEQMid1FreqEndpoint) Set(auxIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/freq", auxIndex)
	return e.m.d.SetInt(path, val)
}

type AuxEQMid1GainEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid1GainEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/gain", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQMid1GainEndpoint) Set(auxIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/gain", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQMid1BWEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid1BWEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/bw", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQMid1BWEndpoint) Set(auxIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid1/bw", auxIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid2 EQ bindings for Aux
type AuxEQMid2Bindings struct {
	m      *MOTU
	Enable *AuxEQMid2EnableEndpoint
	Freq   *AuxEQMid2FreqEndpoint
	Gain   *AuxEQMid2GainEndpoint
	BW     *AuxEQMid2BWEndpoint
}

func newAuxEQMid2Bindings(m *MOTU) *AuxEQMid2Bindings {
	return &AuxEQMid2Bindings{
		m:      m,
		Enable: &AuxEQMid2EnableEndpoint{m: m},
		Freq:   &AuxEQMid2FreqEndpoint{m: m},
		Gain:   &AuxEQMid2GainEndpoint{m: m},
		BW:     &AuxEQMid2BWEndpoint{m: m},
	}
}

type AuxEQMid2EnableEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid2EnableEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/enable", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxEQMid2EnableEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/enable", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxEQMid2FreqEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid2FreqEndpoint) Bind(auxIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/freq", auxIndex)
	e.m.d.BindInt(path, callback)
}

func (e *AuxEQMid2FreqEndpoint) Set(auxIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/freq", auxIndex)
	return e.m.d.SetInt(path, val)
}

type AuxEQMid2GainEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid2GainEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/gain", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQMid2GainEndpoint) Set(auxIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/gain", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQMid2BWEndpoint struct {
	m *MOTU
}

func (e *AuxEQMid2BWEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/bw", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQMid2BWEndpoint) Set(auxIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/mid2/bw", auxIndex)
	return e.m.d.SetFloat(path, val)
}

// LowShelf EQ bindings for Aux
type AuxEQLowShelfBindings struct {
	m      *MOTU
	Enable *AuxEQLowShelfEnableEndpoint
	Freq   *AuxEQLowShelfFreqEndpoint
	Gain   *AuxEQLowShelfGainEndpoint
	BW     *AuxEQLowShelfBWEndpoint
	Mode   *AuxEQLowShelfModeEndpoint
}

func newAuxEQLowShelfBindings(m *MOTU) *AuxEQLowShelfBindings {
	return &AuxEQLowShelfBindings{
		m:      m,
		Enable: &AuxEQLowShelfEnableEndpoint{m: m},
		Freq:   &AuxEQLowShelfFreqEndpoint{m: m},
		Gain:   &AuxEQLowShelfGainEndpoint{m: m},
		BW:     &AuxEQLowShelfBWEndpoint{m: m},
		Mode:   &AuxEQLowShelfModeEndpoint{m: m},
	}
}

type AuxEQLowShelfEnableEndpoint struct {
	m *MOTU
}

func (e *AuxEQLowShelfEnableEndpoint) Bind(auxIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/enable", auxIndex)
	e.m.d.BindBool(path, callback)
}

func (e *AuxEQLowShelfEnableEndpoint) Set(auxIndex int64, val bool) error {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/enable", auxIndex)
	return e.m.d.SetBool(path, val)
}

type AuxEQLowShelfFreqEndpoint struct {
	m *MOTU
}

func (e *AuxEQLowShelfFreqEndpoint) Bind(auxIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/freq", auxIndex)
	e.m.d.BindInt(path, callback)
}

func (e *AuxEQLowShelfFreqEndpoint) Set(auxIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/freq", auxIndex)
	return e.m.d.SetInt(path, val)
}

type AuxEQLowShelfGainEndpoint struct {
	m *MOTU
}

func (e *AuxEQLowShelfGainEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/gain", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQLowShelfGainEndpoint) Set(auxIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/gain", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQLowShelfBWEndpoint struct {
	m *MOTU
}

func (e *AuxEQLowShelfBWEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/bw", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQLowShelfBWEndpoint) Set(auxIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/bw", auxIndex)
	return e.m.d.SetFloat(path, val)
}

type AuxEQLowShelfModeEndpoint struct {
	m *MOTU
}

func (e *AuxEQLowShelfModeEndpoint) Bind(auxIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/mode", auxIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *AuxEQLowShelfModeEndpoint) Set(auxIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/aux/%d/eq/lowshelf/mode", auxIndex)
	return e.m.d.SetFloat(path, val)
}

// Group section bindings
type MixerGroupBindings struct {
	m      *MOTU
	Matrix *GroupMatrixBindings
	EQ     *GroupEQBindings
	Comp   *GroupCompBindings
}

func newMixerGroupBindings(m *MOTU) *MixerGroupBindings {
	return &MixerGroupBindings{
		m:      m,
		Matrix: newGroupMatrixBindings(m),
		EQ:     newGroupEQBindings(m),
		Comp:   newGroupCompBindings(m),
	}
}

// Group Matrix bindings
type GroupMatrixBindings struct {
	m        *MOTU
	Enable   *GroupMatrixEnableEndpoint
	Solo     *GroupMatrixSoloEndpoint
	Mute     *GroupMatrixMuteEndpoint
	Pan      *GroupMatrixPanEndpoint
	Fader    *GroupMatrixFaderEndpoint
	MainSend *GroupMatrixMainSendEndpoint
}

func newGroupMatrixBindings(m *MOTU) *GroupMatrixBindings {
	return &GroupMatrixBindings{
		m:        m,
		Enable:   &GroupMatrixEnableEndpoint{m: m},
		Solo:     &GroupMatrixSoloEndpoint{m: m},
		Mute:     &GroupMatrixMuteEndpoint{m: m},
		Pan:      &GroupMatrixPanEndpoint{m: m},
		Fader:    &GroupMatrixFaderEndpoint{m: m},
		MainSend: &GroupMatrixMainSendEndpoint{m: m},
	}
}

type GroupMatrixEnableEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixEnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupMatrixEnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/matrix/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupMatrixSoloEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixSoloEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/solo", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupMatrixSoloEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/matrix/solo", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupMatrixMuteEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixMuteEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/mute", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupMatrixMuteEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/matrix/mute", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupMatrixPanEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixPanEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/pan", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupMatrixPanEndpoint) Set(groupIndex int64, val float64) error {
	if val < -1 || val > 1 {
		return fmt.Errorf("pan value must be between -1 (left) and 1 (right)")
	}
	path := fmt.Sprintf("mix/group/%d/matrix/pan", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupMatrixFaderEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixFaderEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/fader", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupMatrixFaderEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("fader value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/group/%d/matrix/fader", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupMatrixMainSendEndpoint struct {
	m *MOTU
}

func (e *GroupMatrixMainSendEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/matrix/main/send", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupMatrixMainSendEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("main send value must be between 0 and 4")
	}
	path := fmt.Sprintf("mix/group/%d/matrix/main/send", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// Group EQ bindings
type GroupEQBindings struct {
	m         *MOTU
	HighShelf *GroupEQHighShelfBindings
	Mid1      *GroupEQMid1Bindings
	Mid2      *GroupEQMid2Bindings
	LowShelf  *GroupEQLowShelfBindings
}

func newGroupEQBindings(m *MOTU) *GroupEQBindings {
	return &GroupEQBindings{
		m:         m,
		HighShelf: newGroupEQHighShelfBindings(m),
		Mid1:      newGroupEQMid1Bindings(m),
		Mid2:      newGroupEQMid2Bindings(m),
		LowShelf:  newGroupEQLowShelfBindings(m),
	}
}

type GroupEQHighShelfBindings struct {
	m      *MOTU
	Enable *GroupEQHighShelfEnableEndpoint
	Freq   *GroupEQHighShelfFreqEndpoint
	Gain   *GroupEQHighShelfGainEndpoint
	BW     *GroupEQHighShelfBWEndpoint
	Mode   *GroupEQHighShelfModeEndpoint
}

func newGroupEQHighShelfBindings(m *MOTU) *GroupEQHighShelfBindings {
	return &GroupEQHighShelfBindings{
		m:      m,
		Enable: &GroupEQHighShelfEnableEndpoint{m: m},
		Freq:   &GroupEQHighShelfFreqEndpoint{m: m},
		Gain:   &GroupEQHighShelfGainEndpoint{m: m},
		BW:     &GroupEQHighShelfBWEndpoint{m: m},
		Mode:   &GroupEQHighShelfModeEndpoint{m: m},
	}
}

type GroupEQHighShelfEnableEndpoint struct {
	m *MOTU
}

func (e *GroupEQHighShelfEnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupEQHighShelfEnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupEQHighShelfFreqEndpoint struct {
	m *MOTU
}

func (e *GroupEQHighShelfFreqEndpoint) Bind(groupIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/freq", groupIndex)
	e.m.d.BindInt(path, callback)
}

func (e *GroupEQHighShelfFreqEndpoint) Set(groupIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/freq", groupIndex)
	return e.m.d.SetInt(path, val)
}

type GroupEQHighShelfGainEndpoint struct {
	m *MOTU
}

func (e *GroupEQHighShelfGainEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/gain", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQHighShelfGainEndpoint) Set(groupIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/gain", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQHighShelfBWEndpoint struct {
	m *MOTU
}

func (e *GroupEQHighShelfBWEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/bw", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQHighShelfBWEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/bw", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQHighShelfModeEndpoint struct {
	m *MOTU
}

func (e *GroupEQHighShelfModeEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/mode", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQHighShelfModeEndpoint) Set(groupIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/group/%d/eq/highshelf/mode", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid1 EQ bindings for Group
type GroupEQMid1Bindings struct {
	m      *MOTU
	Enable *GroupEQMid1EnableEndpoint
	Freq   *GroupEQMid1FreqEndpoint
	Gain   *GroupEQMid1GainEndpoint
	BW     *GroupEQMid1BWEndpoint
}

func newGroupEQMid1Bindings(m *MOTU) *GroupEQMid1Bindings {
	return &GroupEQMid1Bindings{
		m:      m,
		Enable: &GroupEQMid1EnableEndpoint{m: m},
		Freq:   &GroupEQMid1FreqEndpoint{m: m},
		Gain:   &GroupEQMid1GainEndpoint{m: m},
		BW:     &GroupEQMid1BWEndpoint{m: m},
	}
}

type GroupEQMid1EnableEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid1EnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid1/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupEQMid1EnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/eq/mid1/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupEQMid1FreqEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid1FreqEndpoint) Bind(groupIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid1/freq", groupIndex)
	e.m.d.BindInt(path, callback)
}

func (e *GroupEQMid1FreqEndpoint) Set(groupIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid1/freq", groupIndex)
	return e.m.d.SetInt(path, val)
}

type GroupEQMid1GainEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid1GainEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid1/gain", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQMid1GainEndpoint) Set(groupIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid1/gain", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQMid1BWEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid1BWEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid1/bw", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQMid1BWEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid1/bw", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// Mid2 EQ bindings for Group
type GroupEQMid2Bindings struct {
	m      *MOTU
	Enable *GroupEQMid2EnableEndpoint
	Freq   *GroupEQMid2FreqEndpoint
	Gain   *GroupEQMid2GainEndpoint
	BW     *GroupEQMid2BWEndpoint
}

func newGroupEQMid2Bindings(m *MOTU) *GroupEQMid2Bindings {
	return &GroupEQMid2Bindings{
		m:      m,
		Enable: &GroupEQMid2EnableEndpoint{m: m},
		Freq:   &GroupEQMid2FreqEndpoint{m: m},
		Gain:   &GroupEQMid2GainEndpoint{m: m},
		BW:     &GroupEQMid2BWEndpoint{m: m},
	}
}

type GroupEQMid2EnableEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid2EnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid2/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupEQMid2EnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/eq/mid2/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupEQMid2FreqEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid2FreqEndpoint) Bind(groupIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid2/freq", groupIndex)
	e.m.d.BindInt(path, callback)
}

func (e *GroupEQMid2FreqEndpoint) Set(groupIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid2/freq", groupIndex)
	return e.m.d.SetInt(path, val)
}

type GroupEQMid2GainEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid2GainEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid2/gain", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQMid2GainEndpoint) Set(groupIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid2/gain", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQMid2BWEndpoint struct {
	m *MOTU
}

func (e *GroupEQMid2BWEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/mid2/bw", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQMid2BWEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/group/%d/eq/mid2/bw", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// LowShelf EQ bindings for Group
type GroupEQLowShelfBindings struct {
	m      *MOTU
	Enable *GroupEQLowShelfEnableEndpoint
	Freq   *GroupEQLowShelfFreqEndpoint
	Gain   *GroupEQLowShelfGainEndpoint
	BW     *GroupEQLowShelfBWEndpoint
	Mode   *GroupEQLowShelfModeEndpoint
}

func newGroupEQLowShelfBindings(m *MOTU) *GroupEQLowShelfBindings {
	return &GroupEQLowShelfBindings{
		m:      m,
		Enable: &GroupEQLowShelfEnableEndpoint{m: m},
		Freq:   &GroupEQLowShelfFreqEndpoint{m: m},
		Gain:   &GroupEQLowShelfGainEndpoint{m: m},
		BW:     &GroupEQLowShelfBWEndpoint{m: m},
		Mode:   &GroupEQLowShelfModeEndpoint{m: m},
	}
}

type GroupEQLowShelfEnableEndpoint struct {
	m *MOTU
}

func (e *GroupEQLowShelfEnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupEQLowShelfEnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupEQLowShelfFreqEndpoint struct {
	m *MOTU
}

func (e *GroupEQLowShelfFreqEndpoint) Bind(groupIndex int64, callback func(int64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/freq", groupIndex)
	e.m.d.BindInt(path, callback)
}

func (e *GroupEQLowShelfFreqEndpoint) Set(groupIndex int64, val int64) error {
	if val < 20 || val > 20000 {
		return fmt.Errorf("frequency must be between 20 and 20000 Hz")
	}
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/freq", groupIndex)
	return e.m.d.SetInt(path, val)
}

type GroupEQLowShelfGainEndpoint struct {
	m *MOTU
}

func (e *GroupEQLowShelfGainEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/gain", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQLowShelfGainEndpoint) Set(groupIndex int64, val float64) error {
	if val < -20 || val > 20 {
		return fmt.Errorf("gain must be between -20 and 20 dB")
	}
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/gain", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQLowShelfBWEndpoint struct {
	m *MOTU
}

func (e *GroupEQLowShelfBWEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/bw", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQLowShelfBWEndpoint) Set(groupIndex int64, val float64) error {
	if val < 0.01 || val > 3 {
		return fmt.Errorf("bandwidth must be between 0.01 and 3 octaves")
	}
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/bw", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupEQLowShelfModeEndpoint struct {
	m *MOTU
}

func (e *GroupEQLowShelfModeEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/mode", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupEQLowShelfModeEndpoint) Set(groupIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("mode must be 0 (Shelf) or 1 (Para)")
	}
	path := fmt.Sprintf("mix/group/%d/eq/lowshelf/mode", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// Group Compressor bindings
type GroupCompBindings struct {
	m         *MOTU
	Enable    *GroupCompEnableEndpoint
	Release   *GroupCompReleaseEndpoint
	Threshold *GroupCompThresholdEndpoint
	Ratio     *GroupCompRatioEndpoint
	Attack    *GroupCompAttackEndpoint
	Trim      *GroupCompTrimEndpoint
	Peak      *GroupCompPeakEndpoint
}

func newGroupCompBindings(m *MOTU) *GroupCompBindings {
	return &GroupCompBindings{
		m:         m,
		Enable:    &GroupCompEnableEndpoint{m: m},
		Release:   &GroupCompReleaseEndpoint{m: m},
		Threshold: &GroupCompThresholdEndpoint{m: m},
		Ratio:     &GroupCompRatioEndpoint{m: m},
		Attack:    &GroupCompAttackEndpoint{m: m},
		Trim:      &GroupCompTrimEndpoint{m: m},
		Peak:      &GroupCompPeakEndpoint{m: m},
	}
}

type GroupCompEnableEndpoint struct {
	m *MOTU
}

func (e *GroupCompEnableEndpoint) Bind(groupIndex int64, callback func(bool) error) {
	path := fmt.Sprintf("mix/group/%d/comp/enable", groupIndex)
	e.m.d.BindBool(path, callback)
}

func (e *GroupCompEnableEndpoint) Set(groupIndex int64, val bool) error {
	path := fmt.Sprintf("mix/group/%d/comp/enable", groupIndex)
	return e.m.d.SetBool(path, val)
}

type GroupCompReleaseEndpoint struct {
	m *MOTU
}

func (e *GroupCompReleaseEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/release", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompReleaseEndpoint) Set(groupIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/group/%d/comp/release", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupCompThresholdEndpoint struct {
	m *MOTU
}

func (e *GroupCompThresholdEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/threshold", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompThresholdEndpoint) Set(groupIndex int64, val float64) error {
	// Expressed in dB per API spec
	path := fmt.Sprintf("mix/group/%d/comp/threshold", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupCompRatioEndpoint struct {
	m *MOTU
}

func (e *GroupCompRatioEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/ratio", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompRatioEndpoint) Set(groupIndex int64, val float64) error {
	path := fmt.Sprintf("mix/group/%d/comp/ratio", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupCompAttackEndpoint struct {
	m *MOTU
}

func (e *GroupCompAttackEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/attack", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompAttackEndpoint) Set(groupIndex int64, val float64) error {
	// Expressed in milliseconds per API spec
	path := fmt.Sprintf("mix/group/%d/comp/attack", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupCompTrimEndpoint struct {
	m *MOTU
}

func (e *GroupCompTrimEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/trim", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompTrimEndpoint) Set(groupIndex int64, val float64) error {
	// Expressed in dB per API spec
	path := fmt.Sprintf("mix/group/%d/comp/trim", groupIndex)
	return e.m.d.SetFloat(path, val)
}

type GroupCompPeakEndpoint struct {
	m *MOTU
}

func (e *GroupCompPeakEndpoint) Bind(groupIndex int64, callback func(float64) error) {
	path := fmt.Sprintf("mix/group/%d/comp/peak", groupIndex)
	e.m.d.BindFloat(path, callback)
}

func (e *GroupCompPeakEndpoint) Set(groupIndex int64, val float64) error {
	if val != 0 && val != 1 {
		return fmt.Errorf("peak mode must be 0 (RMS) or 1 (Peak)")
	}
	path := fmt.Sprintf("mix/group/%d/comp/peak", groupIndex)
	return e.m.d.SetFloat(path, val)
}

// Reverb bindings
type MixerReverbBindings struct {
	m         *MOTU
	Enable    *ReverbEnableEndpoint
	Type      *ReverbTypeEndpoint
	Time      *ReverbTimeEndpoint
	PreDelay  *ReverbPreDelayEndpoint
	Mix       *ReverbMixEndpoint
	Damping   *ReverbDampingEndpoint
	Width     *ReverbWidthEndpoint
	Diffusion *ReverbDiffusionEndpoint
}

func newMixerReverbBindings(m *MOTU) *MixerReverbBindings {
	return &MixerReverbBindings{
		m:         m,
		Enable:    &ReverbEnableEndpoint{m: m},
		Type:      &ReverbTypeEndpoint{m: m},
		Time:      &ReverbTimeEndpoint{m: m},
		PreDelay:  &ReverbPreDelayEndpoint{m: m},
		Mix:       &ReverbMixEndpoint{m: m},
		Damping:   &ReverbDampingEndpoint{m: m},
		Width:     &ReverbWidthEndpoint{m: m},
		Diffusion: &ReverbDiffusionEndpoint{m: m},
	}
}

type ReverbEnableEndpoint struct {
	m *MOTU
}

func (e *ReverbEnableEndpoint) Bind(callback func(bool) error) {
	path := "mix/reverb/enable"
	e.m.d.BindBool(path, callback)
}

func (e *ReverbEnableEndpoint) Set(val bool) error {
	path := "mix/reverb/enable"
	return e.m.d.SetBool(path, val)
}

type ReverbTypeEndpoint struct {
	m *MOTU
}

func (e *ReverbTypeEndpoint) Bind(callback func(int64) error) {
	path := "mix/reverb/type"
	e.m.d.BindInt(path, callback)
}

func (e *ReverbTypeEndpoint) Set(val int64) error {
	if val < 0 || val > 4 {
		return fmt.Errorf("reverb type must be between 0 and 4")
	}
	path := "mix/reverb/type"
	return e.m.d.SetInt(path, val)
}

type ReverbTimeEndpoint struct {
	m *MOTU
}

func (e *ReverbTimeEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/time"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbTimeEndpoint) Set(val float64) error {
	if val < 0.1 || val > 60.0 {
		return fmt.Errorf("reverb time must be between 0.1 and 60.0 seconds")
	}
	path := "mix/reverb/time"
	return e.m.d.SetFloat(path, val)
}

type ReverbPreDelayEndpoint struct {
	m *MOTU
}

func (e *ReverbPreDelayEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/predelay"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbPreDelayEndpoint) Set(val float64) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("reverb pre-delay must be between 0 and 100 ms")
	}
	path := "mix/reverb/predelay"
	return e.m.d.SetFloat(path, val)
}

type ReverbMixEndpoint struct {
	m *MOTU
}

func (e *ReverbMixEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/mix"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbMixEndpoint) Set(val float64) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("reverb mix must be between 0 and 100%%")
	}
	path := "mix/reverb/mix"
	return e.m.d.SetFloat(path, val)
}

type ReverbDampingEndpoint struct {
	m *MOTU
}

func (e *ReverbDampingEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/damping"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbDampingEndpoint) Set(val float64) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("reverb damping must be between 0 and 100%%")
	}
	path := "mix/reverb/damping"
	return e.m.d.SetFloat(path, val)
}

type ReverbWidthEndpoint struct {
	m *MOTU
}

func (e *ReverbWidthEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/width"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbWidthEndpoint) Set(val float64) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("reverb width must be between 0 and 100%%")
	}
	path := "mix/reverb/width"
	return e.m.d.SetFloat(path, val)
}

type ReverbDiffusionEndpoint struct {
	m *MOTU
}

func (e *ReverbDiffusionEndpoint) Bind(callback func(float64) error) {
	path := "mix/reverb/diffusion"
	e.m.d.BindFloat(path, callback)
}

func (e *ReverbDiffusionEndpoint) Set(val float64) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("reverb diffusion must be between 0 and 100%%")
	}
	path := "mix/reverb/diffusion"
	return e.m.d.SetFloat(path, val)
}
