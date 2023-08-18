package flow

import (
	"github.com/oliveio/olive/engine/id"
	"github.com/oliveio/olive/engine/sequence_flow"
)

type Snapshot struct {
	flowId       id.Id
	sequenceFlow *sequence_flow.SequenceFlow
}

func (s *Snapshot) Id() id.Id {
	return s.flowId
}

func (s *Snapshot) SequenceFlow() *sequence_flow.SequenceFlow {
	return s.sequenceFlow
}
