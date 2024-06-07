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

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
)

// RunnerPhase defines the phase in which a runner is in
type RunnerPhase string

// These are the valid phases of node.
const (
	// RunnerPending means the node has been created/added by the system, but not configured.
	RunnerPending RunnerPhase = "Pending"
	// RunnerActive means the olive-runner has been configured and has Olive components active.
	RunnerActive RunnerPhase = "Running"
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
	Hostname string `json:"hostname" protobuf:"bytes,2,opt,name=hostname"`
	// peerURL is the URL the member exposes to the cluster for communication.
	PeerURL string `json:"peerURL" protobuf:"bytes,3,opt,name=peerURL"`
	// clientURL is the URL the member exposes to clients for communication. If the member is not started, clientURLs will be empty.
	ClientURL  string `json:"clientURL" protobuf:"bytes,4,opt,name=clientURL"`
	VersionRef string `json:"versionRef" protobuf:"bytes,5,opt,name=versionRef"`
	// isLearner indicates if the member is raft learner.
	IsLearner *bool `json:"isLearner" protobuf:"varint,6,opt,name=isLearner"`
}

type RunnerStatus struct {
	Phase   RunnerPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RunnerPhase"`
	Message string      `json:"message" protobuf:"bytes,2,opt,name=message"`

	CpuTotal    float64  `json:"cpuTotal" protobuf:"fixed64,3,opt,name=cpuTotal"`
	MemoryTotal float64  `json:"memoryTotal" protobuf:"fixed64,4,opt,name=memoryTotal"`
	Regions     []int64  `json:"regions" protobuf:"varint,5,rep,name=regions"`
	Leaders     []string `json:"leaders" protobuf:"bytes,6,rep,name=leaders"`
	Definitions int64    `json:"definitions" protobuf:"varint,7,opt,name=definitions"`

	Stat *RunnerStat `json:"stat" protobuf:"bytes,8,opt,name=stat"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerStat is the stat information of Runner
type RunnerStat struct {
	metav1.TypeMeta `json:",inline"`

	CpuUsed    float64 `json:"cpuUsed,omitempty" protobuf:"fixed64,1,opt,name=cpuUsed"`
	MemoryUsed float64 `json:"memoryUsed,omitempty" protobuf:"fixed64,2,opt,name=memoryUsed"`

	Bpmn    *BpmnStat `json:"bpmn,omitempty" protobuf:"bytes,3,opt,name=bpmn"`
	Timeout int64     `json:"timeout,omitempty" protobuf:"varint,4,opt,name=timeout"`
}

type BpmnStat struct {
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

// RegionPhase defines the phase in which a region is in
type RegionPhase string

// These are the valid phases of node.
const (
	// RegionPending means the node has been created/added by the system, but not configured.
	RegionPending RegionPhase = "Pending"
	// RegionTerminated means the region has been removed from the cluster.
	RegionTerminated RegionPhase = "Terminated"
	// RegionActive means the region is a normal raft machine, it runs definition.
	RegionActive = "Active"
	// RegionFailed means the region in exception mode, we need to fix it.
	RegionFailed = "Failed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Region the olive node
type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RegionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RegionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RegionSpec is the specification of a Region.
type RegionSpec struct {
	Id               int64           `json:"id" protobuf:"varint,1,opt,name=id"`
	DeploymentId     int64           `json:"deploymentId" protobuf:"varint,2,opt,name=deploymentId"`
	Replicas         []RegionReplica `json:"replicas" protobuf:"bytes,3,rep,name=replicas"`
	ElectionRTT      int64           `json:"electionRTT" protobuf:"varint,4,opt,name=electionRTT"`
	HeartbeatRTT     int64           `json:"heartbeatRTT" protobuf:"varint,5,opt,name=heartbeatRTT"`
	Leader           int64           `json:"leader" protobuf:"varint,6,opt,name=leader"`
	DefinitionsLimit int64           `json:"definitionsLimit" protobuf:"varint,7,opt,name=definitionsLimit"`
}

type RegionReplica struct {
	Id          int64  `json:"id" protobuf:"varint,1,opt,name=id"`
	Runner      string `json:"runner" protobuf:"bytes,2,opt,name=runner"`
	Region      int64  `json:"region" protobuf:"varint,3,opt,name=region"`
	RaftAddress string `json:"raftAddress" protobuf:"bytes,4,opt,name=raftAddress"`
	IsNonVoting *bool  `json:"isNonVoting" protobuf:"varint,5,opt,name=isNonVoting"`
	IsWitness   *bool  `json:"isWitness" protobuf:"varint,6,opt,name=isWitness"`
	IsJoin      bool   `json:"isJoin" protobuf:"varint,7,opt,name=isJoin"`
}

type RegionStatus struct {
	Phase   RegionPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RegionPhase"`
	Message string      `json:"message" protobuf:"bytes,2,opt,name=message"`

	Leader      int64 `json:"leader" protobuf:"varint,3,opt,name=leader"`
	Term        int64 `json:"term" protobuf:"varint,4,opt,name=term"`
	Replicas    int32 `json:"replicas" protobuf:"varint,5,opt,name=replicas"`
	Definitions int64 `json:"definitions" protobuf:"varint,6,opt,name=definitions"`

	Stat *RegionStat `json:"stat" protobuf:"bytes,7,opt,name=stat"`
}

func (m *Region) InitialURL() map[int64]string {
	initial := map[int64]string{}
	for _, replica := range m.Spec.Replicas {
		initial[replica.Id] = replica.RaftAddress
	}
	return initial
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionStat is the stat information of Region
type RegionStat struct {
	metav1.TypeMeta `json:",inline"`

	RunningDefinitions int64 `json:"runningDefinitions" protobuf:"varint,1,opt,name=runningDefinitions"`

	Bpmn    *BpmnStat `json:"bpmn,omitempty" protobuf:"bytes,2,opt,name=bpmn"`
	Timeout int64     `json:"timeout,omitempty" protobuf:"varint,3,opt,name=timeout"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionList is a list of Region objects.
type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Region
	Items []Region `json:"items" protobuf:"bytes,2,rep,name=items"`
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
	// DefTerminated means the node has been removed from the cluster.
	DefTerminated DefPhase = "Terminated"
)

// +genclient
// +k8s:protobuf-gen=true
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
	// the id of olive region
	Region int64 `json:"region,omitempty" protobuf:"varint,3,opt,name=region"`
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

// ProcessPhase defines the phase in which a bpmn process instance is in
type ProcessPhase string

// These are the valid phases of node.
const (
	// ProcessPending means the node has been created/added by the system, but not configured.
	ProcessPending ProcessPhase = "Pending"
	// ProcessTerminated means the node has been removed from the cluster.
	ProcessTerminated ProcessPhase = "Terminated"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Process is bpmn process instance
type Process struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ProcessSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status ProcessStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type ProcessSpec struct {
	// the id of Definition
	Definition string `json:"definition" protobuf:"bytes,1,opt,name=definition"`
	// the version if Definition
	Version int64 `json:"version" protobuf:"varint,2,opt,name=version"`
	// the process id of bpmn Process Element in Definition
	BpmnProcess string                        `json:"bpmnProcess" protobuf:"bytes,3,opt,name=bpmnProcess"`
	Headers     map[string]string             `json:"headers" protobuf:"bytes,4,rep,name=headers"`
	Properties  map[string]apidiscoveryv1.Box `json:"properties" protobuf:"bytes,5,rep,name=properties"`
	DataObjects map[string]string             `json:"dataObjects" protobuf:"bytes,6,rep,name=dataObjects"`
}

type ProcessStatus struct {
	Phase   ProcessPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=ProcessPhase"`
	Message string       `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// the id of olive region
	Region int64 `json:"region" protobuf:"varint,3,opt,name=region"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessList is a list of Process objects.
type ProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Process
	Items []Process `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessStat is stat information of Process
type ProcessStat struct {
	metav1.TypeMeta `json:",inline"`

	// The id of Process
	Id                string `json:"id" protobuf:"bytes,1,opt,name=id"`
	DefinitionContent string `json:"definitionContent" protobuf:"bytes,2,opt,name=definitionContent"`

	State ProcessRunningState `json:"processState" protobuf:"bytes,3,opt,name=processState"`

	Attempts  int64                   `json:"attempts" protobuf:"varint,4,opt,name=attempts"`
	FlowNodes map[string]FlowNodeStat `json:"flowNodes" protobuf:"bytes,5,rep,name=flowNodes"`
	StartTime int64                   `json:"startTime" protobuf:"varint,6,opt,name=startTime"`
	EndTime   int64                   `json:"endTime" protobuf:"varint,7,opt,name=endTime"`
}

type ProcessRunningState struct {
	Properties  map[string]string `json:"properties" protobuf:"bytes,1,rep,name=properties"`
	DataObjects map[string]string `json:"dataObjects" protobuf:"bytes,2,rep,name=dataObjects"`
	Variables   map[string]string `json:"variables" protobuf:"bytes,3,rep,name=variables"`
}

type FlowNodeStat struct {
	Id          string            `json:"id" protobuf:"bytes,1,opt,name=id"`
	Name        string            `json:"name" protobuf:"bytes,2,opt,name=name"`
	Headers     map[string]string `json:"headers" protobuf:"bytes,3,rep,name=headers"`
	Properties  map[string]string `json:"properties" protobuf:"bytes,4,rep,name=properties"`
	DataObjects map[string]string `json:"dataObjects" protobuf:"bytes,5,rep,name=dataObjects"`
	StartTime   int64             `json:"startTime" protobuf:"varint,6,opt,name=startTime"`
	EndTime     int64             `json:"endTime" protobuf:"varint,7,opt,name=endTime"`
}
