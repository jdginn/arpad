package mapper

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/jdginn/arpad/logging"
)

var appLog *slog.Logger

func init() {
	appLog = logging.Get(logging.APP)
}

type GUID = string

type Mapper struct {
	mux                *sync.Mutex
	guidToSurfaceIndex map[GUID]int64
	surfaceIndexToGuid map[int64]GUID
}

func NewMapper() *Mapper {
	return &Mapper{
		mux:                &sync.Mutex{},
		guidToSurfaceIndex: make(map[GUID]int64),
		surfaceIndexToGuid: make(map[int64]GUID),
	}
}

func (m *Mapper) AddGuid(guid GUID) *mappingGuid {
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.guidToSurfaceIndex[guid]; !exists {
		appLog.Info("Adding GUID to mapper", slog.String("guid", guid))
		idx := int64(len(m.guidToSurfaceIndex))
		m.guidToSurfaceIndex[guid] = idx
		m.surfaceIndexToGuid[idx] = guid
	}
	return &mappingGuid{m, guid}
}

func (m *Mapper) DeleteGuid(guid GUID) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if idx, exists := m.guidToSurfaceIndex[guid]; exists {
		appLog.Info("Deleting GUID from mapper", slog.String("guid", guid))
		delete(m.guidToSurfaceIndex, guid)
		delete(m.surfaceIndexToGuid, idx)
	}
}

func (m *Mapper) ByGuid(guid GUID) *mappingGuid {
	return &mappingGuid{m, guid}
}

func (m *Mapper) BySurfIdx(idx int64) *mappingSurfaceIdx {
	return &mappingSurfaceIdx{m, idx}
}

type mappingGuid struct {
	*Mapper
	guid GUID
}

func (m *mappingGuid) MaybeSurfIdx() (int64, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if surfaceIdx, ok := m.guidToSurfaceIndex[m.guid]; ok {
		return surfaceIdx, true
	}
	return 0, false
}

func (m *mappingGuid) SurfIdx() int64 {
	if surfaceIdx, ok := m.guidToSurfaceIndex[m.guid]; ok {
		return surfaceIdx
	}
	panic("mappingGuid: no surface index found for guid " + m.guid)
}

func (m *mappingGuid) SetSurfIdx(idx int64) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.surfaceIndexToGuid, idx)
	m.guidToSurfaceIndex[m.guid] = idx
	m.surfaceIndexToGuid[idx] = m.guid
}

type mappingSurfaceIdx struct {
	*Mapper
	idx int64
}

func (m *mappingSurfaceIdx) MaybeGuid() (GUID, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if guid, ok := m.surfaceIndexToGuid[m.idx]; ok {
		return guid, true
	}
	return "", false
}

func (m *mappingSurfaceIdx) Guid() GUID {
	if guid, ok := m.surfaceIndexToGuid[m.idx]; ok {
		return guid
	}
	panic(fmt.Sprintf("mappingSurfaceIdx: no guid found for surface index %d", m.idx))
}

func (m *mappingSurfaceIdx) SetGuid(guid GUID) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.surfaceIndexToGuid[m.idx] = guid
	m.guidToSurfaceIndex[guid] = m.idx
}
