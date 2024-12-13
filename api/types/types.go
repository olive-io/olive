package types

import (
	"github.com/olive-io/olive/api/meta"
)

type State string

const (
	StateReady    State = "Ready"
	StateNotReady State = "NotReady"
	StateAlarm    State = "Alarm"
)

type ProcessStatus string

const (
	ProcessWaiting ProcessStatus = "Waiting"
	ProcessPrepare ProcessStatus = "Prepare"
	ProcessRunning ProcessStatus = "Running"
	ProcessOk      ProcessStatus = "Ok"
	ProcessFail    ProcessStatus = "Fail"
)

type FlowNodeType string

const (
	FlowNodeUnknown        FlowNodeType = "Unknown"
	StartEvent             FlowNodeType = "StartEvent"
	EndEvent               FlowNodeType = "EndEvent"
	BoundaryEvent          FlowNodeType = "BoundaryEvent"
	IntermediateCatchEvent FlowNodeType = "IntermediateCatchEvent"

	Task             FlowNodeType = "Task"
	UserTask         FlowNodeType = "UserTask"
	ServiceTask      FlowNodeType = "ServiceTask"
	ScriptTask       FlowNodeType = "ScriptTask"
	SendTask         FlowNodeType = "SendTask"
	ReceiveTask      FlowNodeType = "ReceiveTask"
	ManualTask       FlowNodeType = "ManualTask"
	CallActivity     FlowNodeType = "CallActivity"
	BusinessRuleTask FlowNodeType = "BusinessRuleTask"
	SubProcess       FlowNodeType = "SubProcess"

	EventBasedGateway FlowNodeType = "EventBasedGateway"
	ParallelGateway   FlowNodeType = "ParallelGateway"
	InclusiveGateway  FlowNodeType = "InclusiveGateway"
	ExclusiveGateway  FlowNodeType = "ExclusiveGateway"
)

// +gogo:genproto=true
// +gogo:deepcopy=true
// +gogo:deepcopy:interfaces=github.com/olive-io/olive/api.Object
type Runner struct {
	meta.TypeMeta   `json:",inline" protobuf:"bytes,1,opt,name=typeMeta,proto3"`
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata,proto3"`

	// listenURL is the URL the runner is listening on.
	ListenURL   string `json:"listenURL" protobuf:"bytes,3,opt,name=listenURL,proto3"`
	Version     string `json:"version" protobuf:"bytes,4,opt,name=version,proto3"`
	HeartbeatMs int64  `json:"heartbeatMs" protobuf:"varint,5,opt,name=heartbeatMs,proto3"`
	Hostname    string `json:"hostname" protobuf:"bytes,6,opt,name=hostname,proto3"`

	Features map[string]string `json:"features" protobuf:"bytes,7,rep,name=features,proto3"`

	CPU      int32 `json:"cpu" protobuf:"varint,8,opt,name=cpu,proto3"`
	Memory   int64 `json:"memory" protobuf:"varint,9,opt,name=memory,proto3"`
	DiskSize int64 `json:"diskSize" protobuf:"varint,10,opt,name=diskSize,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type RunnerStatistics struct {
	Name       string  `json:"name" protobuf:"bytes,1,opt,name=name,proto3"`
	CPUUsed    float64 `json:"cpuUsed" protobuf:"fixed64,2,opt,name=cpuUsed,proto3"`
	MemoryUsed float64 `json:"memoryUsed" protobuf:"fixed64,3,opt,name=memoryUsed,proto3"`

	BpmnProcesses uint64 `json:"bpmnProcesses" protobuf:"varint,4,opt,name=bpmnProcesses,proto3"`
	BpmnEvents    uint64 `json:"bpmnEvents" protobuf:"varint,5,opt,name=bpmnEvents,proto3"`
	BpmnTasks     uint64 `json:"bpmnTasks" protobuf:"varint,6,opt,name=bpmnTasks,proto3"`

	State     State  `json:"state" protobuf:"bytes,7,opt,name=state,proto3,casttype=State"`
	Error     string `json:"error" protobuf:"bytes,8,opt,name=error,proto3"`
	Timestamp int64  `json:"timestamp" protobuf:"varint,9,opt,name=timestamp,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
// +gogo:deepcopy:interfaces=github.com/olive-io/olive/api.Object
type Definition struct {
	meta.TypeMeta   `json:",inline" protobuf:"bytes,1,opt,name=typeMeta,proto3"`
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata,proto3"`

	Content string `json:"content,omitempty" protobuf:"bytes,3,opt,name=content,proto3"`
	Version int64  `json:"version,omitempty" protobuf:"varint,4,opt,name=version,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type BpmnArgs struct {
	Headers     map[string]string `json:"headers,omitempty" protobuf:"bytes,1,rep,name=headers,proto3"`
	Properties  map[string][]byte `json:"properties,omitempty" protobuf:"bytes,2,rep,name=properties,proto3"`
	DataObjects map[string][]byte `json:"dataObjects,omitempty" protobuf:"bytes,3,rep,name=dataObjects,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type ProcessContext struct {
	Variables   map[string][]byte `json:"variables,omitempty" protobuf:"bytes,1,rep,name=variables,proto3"`
	DataObjects map[string][]byte `json:"dataObjects,omitempty" protobuf:"bytes,2,rep,name=dataObjects,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
// +gogo:deepcopy:interfaces=github.com/olive-io/olive/api.Object
type ProcessInstance struct {
	meta.TypeMeta   `json:",inline" protobuf:"bytes,1,opt,name=typeMeta,proto3"`
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata,proto3"`

	Args BpmnArgs `json:"args,omitempty" protobuf:"bytes,3,opt,name=args,proto3"`

	DefinitionsId      string `json:"definitionsId,omitempty" protobuf:"bytes,4,opt,name=definitionsId,proto3"`
	DefinitionsVersion int64  `json:"definitionsVersion,omitempty" protobuf:"varint,5,opt,name=definitionsVersion,proto3"`
	DefinitionsProcess string `json:"definitionsProcess,omitempty" protobuf:"bytes,6,opt,name=definitionsProcess,proto3"`
	DefinitionsContent string `json:"definitionsContent,omitempty" protobuf:"bytes,7,opt,name=definitionsContent,proto3"`

	Context ProcessContext `json:"context,omitempty" protobuf:"bytes,8,opt,name=context,proto3"`

	FlowNodes       []FlowNode              `json:"flowNodes,omitempty" protobuf:"bytes,9,rep,name=flowNodes,proto3"`
	FlowNodeStatMap map[string]FlowNodeStat `json:"flowNodeStatMap,omitempty" protobuf:"bytes,10,rep,name=flowNodeStatMap,proto3"`
	Attempts        int32                   `json:"attempts,omitempty" protobuf:"varint,11,opt,name=attempts,proto3"`

	StartTimestamp int64 `json:"creationTimestamp,omitempty" protobuf:"varint,12,opt,name=creationTimestamp,proto3"`
	EndTimestamp   int64 `json:"endTimestamp,omitempty" protobuf:"varint,13,opt,name=endTimestamp,proto3"`

	Status  ProcessStatus `json:"status,omitempty" protobuf:"bytes,14,opt,name=status,proto3,casttype=ProcessStatus"`
	Message string        `json:"message,omitempty" protobuf:"bytes,15,opt,name=message,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type FlowNode struct {
	Type FlowNodeType `json:"type" protobuf:"bytes,1,opt,name=type,proto3,casttype=FlowNodeType"`
	Id   string       `json:"id" protobuf:"bytes,2,opt,name=id,proto3"`
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type FlowNodeStat struct {
	Id      string         `json:"id" protobuf:"bytes,1,opt,name=id,proto3"`
	Name    string         `json:"name" protobuf:"bytes,2,opt,name=name,proto3"`
	Context ProcessContext `json:"context" protobuf:"bytes,3,opt,name=context,proto3"`
	Retries int32          `json:"retries" protobuf:"varint,4,opt,name=retries,proto3"`
	Message string         `json:"message" protobuf:"bytes,5,opt,name=message,proto3"`

	StartTime int64 `json:"startTime" protobuf:"varint,6,opt,name=startTime,proto3"`
	EndTime   int64 `json:"endTime" protobuf:"varint,7,opt,name=endTime,proto3"`
}
