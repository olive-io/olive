package olivepb

import (
	"google.golang.org/protobuf/proto"
)

func (m *ProcessInstance) ID() uint64 {
	return m.Id
}

func (m *Runner) ID() string {
	return m.Name
}

func (m *Runner) Clone() *Runner {
	out := proto.Clone(m)
	return out.(*Runner)
}

func (m *RunnerStatistics) ID() string {
	return m.Id
}
