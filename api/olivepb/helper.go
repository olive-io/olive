package olivepb

import (
	"crypto/md5"
	"encoding/binary"
)

func (m *RunnerStat) ID() uint64 {
	return m.Id
}

func (m *RegionStat) ID() uint64 {
	return m.Id
}

func (m *DefinitionMeta) ID() uint64 {
	h := md5.New()
	data := h.Sum([]byte(m.Id))
	return binary.LittleEndian.Uint64(data)
}
