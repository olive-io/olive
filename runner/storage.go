package runner

import sm "github.com/lni/dragonboat/v4/statemachine"

type GenericFactory struct {
	Option
}

func (f *GenericFactory) NewBrainStorage(clusterID, nodeID uint64) sm.IOnDiskStateMachine {
	return &BrainStorage{}
}
