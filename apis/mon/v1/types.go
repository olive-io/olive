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
	// RunnerRunning means the node has been configured and has Olive components running.
	RunnerRunning RunnerPhase = "Running"
	// RunnerTerminated means the node has been removed from the cluster.
	RunnerTerminated RunnerPhase = "Terminated"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner the olive node
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RunnerSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RunnerStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RunnerSpec is the specification of a Runner.
type RunnerSpec struct {
	// ID is the member ID for this member.
	ID uint64 `json:"id" protobuf:"varint,1,opt,name=id"`
	// name is the human-readable name of the member. If the member is not started, the name will be an empty string.
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// peerURLs is the list of URLs the member exposes to the cluster for communication.
	PeerURLs []string `json:"peerURLs" protobuf:"bytes,3,rep,name=peerURLs"`
	// clientURLs is the list of URLs the member exposes to clients for communication. If the member is not started, clientURLs will be empty.
	ClientURLs []string `json:"clientURLs" protobuf:"bytes,4,rep,name=clientURLs"`
	// isLearner indicates if the member is raft learner.
	IsLearner *bool `json:"isLearner" protobuf:"varint,5,opt,name=isLearner"`
}

type RunnerStatus struct {
	Phase RunnerPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RunnerPhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerList is a list of Runner objects.
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Runner
	Items []Runner `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerStat is the stat information of Runner
type RunnerStat struct {
	metav1.TypeMeta

	Id            uint64   `json:"id" protobuf:"varint,1,opt,name=id"`
	CpuPer        float64  `json:"cpuPer" protobuf:"fixed64,2,opt,name=cpuPer"`
	MemoryPer     float64  `json:"memoryPer" protobuf:"fixed64,3,opt,name=memoryPer"`
	Regions       []uint64 `json:"regions" protobuf:"varint,4,rep,name=regions"`
	Leaders       []string `json:"leaders" protobuf:"bytes,5,rep,name=leaders"`
	Definitions   uint64   `json:"definitions" protobuf:"varint,6,opt,name=definitions"`
	BpmnProcesses uint64   `json:"bpmnProcesses" protobuf:"varint,7,opt,name=bpmnProcesses"`
	BpmnEvents    uint64   `json:"bpmnEvents" protobuf:"varint,8,opt,name=bpmnEvents"`
	BpmnTasks     uint64   `json:"bpmnTasks" protobuf:"varint,9,opt,name=bpmnTasks"`
	Message       string   `json:"message" protobuf:"bytes,10,opt,name=message"`
	Timestamp     int64    `json:"timestamp" protobuf:"varint,11,opt,name=timestamp"`
}

// RegionPhase defines the phase in which a region is in
type RegionPhase string

// These are the valid phases of node.
const (
	// RegionPending means the node has been created/added by the system, but not configured.
	RegionPending RegionPhase = "Pending"
	// RegionTerminated means the region has been removed from the cluster.
	RegionTerminated RegionPhase = "Terminated"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Region the olive node
type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RegionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RegionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RegionSpec is the specification of a Region.
type RegionSpec struct {
	Id               uint64          `json:"id" protobuf:"varint,1,opt,name=id"`
	DeploymentId     uint64          `json:"deploymentId" protobuf:"varint,2,opt,name=deploymentId"`
	Replicas         []RegionReplica `json:"replicas" protobuf:"bytes,3,rep,name=replicas"`
	ElectionRTT      uint64          `json:"electionRTT" protobuf:"varint,4,opt,name=electionRTT"`
	HeartbeatRTT     uint64          `json:"heartbeatRTT" protobuf:"varint,5,opt,name=heartbeatRTT"`
	Leader           uint64          `json:"leader" protobuf:"varint,6,opt,name=leader"`
	Definitions      uint64          `json:"definitions" protobuf:"varint,7,opt,name=definitions"`
	DefinitionsLimit uint64          `json:"definitionsLimit" protobuf:"varint,8,opt,name=definitionsLimit"`
}

type RegionReplica struct {
	Id          uint64 `json:"id" protobuf:"varint,1,opt,name=id"`
	Runner      uint64 `json:"runner" protobuf:"varint,2,opt,name=runner"`
	Region      uint64 `json:"region" protobuf:"varint,3,opt,name=region"`
	RaftAddress string `json:"raftAddress" protobuf:"bytes,4,opt,name=raftAddress"`
	IsNonVoting bool   `json:"isNonVoting" protobuf:"varint,5,opt,name=isNonVoting"`
	IsWitness   bool   `json:"isWitness" protobuf:"varint,6,opt,name=isWitness"`
	IsJoin      bool   `json:"isJoin" protobuf:"varint,7,opt,name=isJoin"`
}

type RegionStatus struct {
	Phase RegionPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RegionPhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionList is a list of Region objects.
type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Region
	Items []Region `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionStat is the stat information of Region
type RegionStat struct {
	metav1.TypeMeta

	Id                 uint64 `json:"id" protobuf:"varint,1,opt,name=id"`
	Leader             uint64 `json:"leader" protobuf:"varint,2,opt,name=leader"`
	Term               uint64 `json:"term" protobuf:"varint,3,opt,name=term"`
	Replicas           int32  `json:"replicas" protobuf:"varint,4,opt,name=replicas"`
	Definitions        uint64 `json:"definitions" protobuf:"varint,5,opt,name=definitions"`
	RunningDefinitions uint64 `json:"runningDefinitions" protobuf:"varint,6,opt,name=runningDefinitions"`
	BpmnProcesses      uint64 `json:"bpmnProcesses" protobuf:"varint,7,opt,name=bpmnProcesses"`
	BpmnEvents         uint64 `json:"bpmnEvents" protobuf:"varint,8,opt,name=bpmnEvents"`
	BpmnTasks          uint64 `json:"bpmnTasks" protobuf:"varint,9,opt,name=bpmnTasks"`
	Message            string `json:"message" protobuf:"bytes,10,opt,name=message"`
	Timestamp          int64  `json:"timestamp" protobuf:"varint,11,opt,name=timestamp"`
}