package olivepb

import (
	"crypto/md5"
	"encoding/binary"
	"strconv"
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

func (m *ProcessInstance) ID() uint64 {
	id, _ := strconv.ParseUint(m.Id, 10, 64)
	return id
}

func (m *Runner) Clone() *Runner {
	out := new(Runner)
	*out = *m
	return out
}
