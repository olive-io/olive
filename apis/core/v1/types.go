/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunnerPhase defines the phase in which a runner is in
type RunnerPhase string

// These are the valid phases of node.
const (
	// RunnerPending means the node has been created/added by the system, but not configured.
	RunnerPending RunnerPhase = "Pending"
	// RunnerActive means the olive-runner has been configured and has Olive components active.
	RunnerActive RunnerPhase = "Active"
	// RunnerDown means the olive-runner doesn't interval update status
	RunnerDown RunnerPhase = "Down"
	// RunnerTerminated means the node has been removed from the cluster.
	RunnerTerminated RunnerPhase = "Terminated"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner the olive node
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RunnerSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RunnerStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RunnerSpec is the specification of a Runner.
type RunnerSpec struct {
	// ID is the member ID for this member.
	ID int64 `json:"id" protobuf:"varint,1,opt,name=id"`
	// hostname is the host name get by os.Hostname().
	Hostname    string `json:"hostname" protobuf:"bytes,2,opt,name=hostname"`
	HeartbeatMs int64  `json:"heartbeatMs" protobuf:"varint,3,opt,name=heartbeatMs"`
	// peerURL is the URL the member exposes to the cluster for communication.
	ListenURL string `json:"listenURL" protobuf:"bytes,4,opt,name=listenURL"`
	Version   string `json:"version" protobuf:"bytes,5,opt,name=version"`

	Features map[string]string `json:"features" protobuf:"bytes,6,rep,name=features"`
}

type RunnerStatus struct {
	Phase   RunnerPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RunnerPhase"`
	Message string      `json:"message" protobuf:"bytes,2,opt,name=message"`

	CpuTotal    float64 `json:"cpuTotal" protobuf:"fixed64,3,opt,name=cpuTotal"`
	MemoryTotal float64 `json:"memoryTotal" protobuf:"fixed64,4,opt,name=memoryTotal"`
	DiskSize    int64   `json:"diskSize" protobuf:"varint,5,opt,name=diskSize"`

	Stat RunnerStatistics `json:"stat" protobuf:"bytes,6,opt,name=stat"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerStatistics is the stat information of Runner
type RunnerStatistics struct {
	metav1.TypeMeta `json:",inline"`

	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	CpuUsed    float64 `json:"cpuUsed,omitempty" protobuf:"fixed64,2,opt,name=cpuUsed"`
	MemoryUsed float64 `json:"memoryUsed,omitempty" protobuf:"fixed64,3,opt,name=memoryUsed"`

	BpmnStat *BpmnStatistics `json:"bpmnStat,omitempty" protobuf:"bytes,4,opt,name=bpmnStat"`

	Timestamp int64 `json:"timestamp,omitempty" protobuf:"varint,5,opt,name=timestamp"`
}

type BpmnStatistics struct {
	Definitions int64 `json:"definitions" protobuf:"varint,1,opt,name=definitions"`
	Processes   int64 `json:"processes" protobuf:"varint,2,opt,name=processes"`
	Events      int64 `json:"events" protobuf:"varint,3,opt,name=events"`
	Tasks       int64 `json:"tasks" protobuf:"varint,4,opt,name=tasks"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerList is a list of Runner objects.
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Runner
	Items []Runner `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// The below types are used by olive_client and api_server.

// ConditionStatus defines conditions of resources
type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition;
// "ConditionFalse" means a resource is not in the condition; "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// NamespaceSpec describes the attributes on a Namespace
type NamespaceSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []FinalizerName `json:"finalizers" protobuf:"bytes,1,rep,name=finalizers,casttype=FinalizerName"`
}

// FinalizerName is the name identifying a finalizer during namespace lifecycle.
type FinalizerName string

// FinalizerOlive there are internal finalizer values to Olive, must be qualified name unless defined here or
// in metav1.
const (
	FinalizerOlive FinalizerName = "Olive"
)

// NamespaceStatus is information about the current status of a Namespace.
type NamespaceStatus struct {
	// Phase is the current lifecycle phase of the namespace.
	// +optional
	Phase NamespacePhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=NamespacePhase"`
	// +optional
	Conditions []NamespaceCondition `json:"conditions" protobuf:"bytes,2,rep,name=conditions"`
}

// NamespacePhase defines the phase in which the namespace is
type NamespacePhase string

// These are the valid phases of a namespace.
const (
	// NamespaceActive means the namespace is available for use in the system
	NamespaceActive NamespacePhase = "Active"
	// NamespaceTerminating means the namespace is undergoing graceful termination
	NamespaceTerminating NamespacePhase = "Terminating"
)

// NamespaceConditionType defines constants reporting on status during namespace lifetime and deletion progress
type NamespaceConditionType string

// These are valid conditions of a namespace.
const (
	NamespaceDeletionDiscoveryFailure NamespaceConditionType = "NamespaceDeletionDiscoveryFailure"
	NamespaceDeletionContentFailure   NamespaceConditionType = "NamespaceDeletionContentFailure"
	NamespaceDeletionGVParsingFailure NamespaceConditionType = "NamespaceDeletionGroupVersionParsingFailure"
)

// NamespaceCondition contains details about state of namespace.
type NamespaceCondition struct {
	// Type of namespace controller condition.
	Type NamespaceConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NamespaceConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// +optional
	Reason string `json:"reason" protobuf:"bytes,4,opt,name=reason"`
	// +optional
	Message string `json:"message" protobuf:"bytes,5,opt,name=message"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Namespace provides a scope for Names.
// Use of multiple namespaces is optional
type Namespace struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the Namespace.
	// +optional
	Spec NamespaceSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status describes the current status of a Namespace
	// +optional
	Status NamespaceStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceList is a list of Namespaces.
type NamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Items []Namespace `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// DefPhase defines the phase in which a definition is in
type DefPhase string

// These are the valid phases of node.
const (
	// DefPending means the node has been created/added by the system, but not configured.
	DefPending DefPhase = "Pending"
	DefBinding DefPhase = "Binding"
	DefActive  DefPhase = "Active"
	// DefTerminated means the node has been removed from the cluster.
	DefTerminated DefPhase = "Terminated"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Definition is bpmn definitions
type Definition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   DefinitionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status DefinitionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type DefinitionSpec struct {
	Content string `json:"content,omitempty" protobuf:"bytes,1,opt,name=content"`
	Version int64  `json:"version,omitempty" protobuf:"varint,2,opt,name=version"`
}

type DefinitionStatus struct {
	Phase DefPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=DefPhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefinitionList is a list of Definition objects.
type DefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Definition
	Items []Definition `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type State string

const (
	StateReady    State = "Ready"
	StateNotReady State = "NotReady"
	StateAlarm    State = "Alarm"
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

// ProcessPhase defines the phase in which a bpmn process instance is in
type ProcessPhase string

// These are the valid phases of node.
const (
	// ProcessPending means the node has been created/added by the system, but not configured.
	ProcessPending ProcessPhase = "Pending"
	// ProcessTerminated means the node has been removed from the cluster.
	ProcessTerminated ProcessPhase = "Terminated"
	ProcessBinding    ProcessPhase = "Binding"
	ProcessPrepare    ProcessPhase = "Prepare"
	ProcessRunning    ProcessPhase = "Running"
	ProcessSuccess    ProcessPhase = "Success"
	ProcessFailed     ProcessPhase = "Failed"
)

type BpmnArgs struct {
	Headers     map[string]string `json:"headers,omitempty" protobuf:"bytes,1,rep,name=headers"`
	Properties  map[string][]byte `json:"properties,omitempty" protobuf:"bytes,2,rep,name=properties"`
	DataObjects map[string][]byte `json:"dataObjects,omitempty" protobuf:"bytes,3,rep,name=dataObjects"`
}

type ProcessContext struct {
	Variables   map[string][]byte `json:"variables,omitempty" protobuf:"bytes,1,rep,name=variables"`
	DataObjects map[string][]byte `json:"dataObjects,omitempty" protobuf:"bytes,2,rep,name=dataObjects"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Process struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ProcessSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status ProcessStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type ProcessSpec struct {
	Args BpmnArgs `json:"args,omitempty" protobuf:"bytes,1,opt,name=args"`

	DefinitionsName    string `json:"definitionsName,omitempty" protobuf:"bytes,2,opt,name=definitionsName"`
	DefinitionsVersion int64  `json:"definitionsVersion,omitempty" protobuf:"varint,3,opt,name=definitionsVersion"`
	DefinitionsProcess string `json:"definitionsProcess,omitempty" protobuf:"bytes,4,opt,name=definitionsProcess"`
	DefinitionsContent string `json:"definitionsContent,omitempty" protobuf:"bytes,5,opt,name=definitionsContent"`
}

type ProcessStatus struct {
	Phase   ProcessPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=ProcessPhase"`
	Message string       `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	Context ProcessContext `json:"context,omitempty" protobuf:"bytes,3,opt,name=context"`

	FlowNodes       []FlowNode              `json:"flowNodes,omitempty" protobuf:"bytes,4,rep,name=flowNodes"`
	FlowNodeStatMap map[string]FlowNodeStat `json:"flowNodeStatMap,omitempty" protobuf:"bytes,5,rep,name=flowNodeStatMap"`
	Attempts        int32                   `json:"attempts,omitempty" protobuf:"varint,6,opt,name=attempts"`

	StartTimestamp int64 `json:"creationTimestamp,omitempty" protobuf:"varint,7,opt,name=creationTimestamp"`
	EndTimestamp   int64 `json:"endTimestamp,omitempty" protobuf:"varint,8,opt,name=endTimestamp"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Process `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type FlowNode struct {
	Type FlowNodeType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=FlowNodeType"`
	Id   string       `json:"id" protobuf:"bytes,2,opt,name=id"`
}

type FlowNodeStat struct {
	Id      string         `json:"id" protobuf:"bytes,1,opt,name=id"`
	Name    string         `json:"name" protobuf:"bytes,2,opt,name=name"`
	Context ProcessContext `json:"context" protobuf:"bytes,3,opt,name=context"`
	Retries int32          `json:"retries" protobuf:"varint,4,opt,name=retries"`
	Message string         `json:"message" protobuf:"bytes,5,opt,name=message"`

	StartTime int64 `json:"startTime" protobuf:"varint,6,opt,name=startTime"`
	EndTime   int64 `json:"endTime" protobuf:"varint,7,opt,name=endTime"`
}
