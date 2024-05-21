/*
   Copyright 2024 The olive Authors

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

package olive

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

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner the olive node
type Runner struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   RunnerSpec
	Status RunnerStatus
}

// RunnerSpec is the specification of a Runner.
type RunnerSpec struct {
	// ID is the member ID for this member.
	ID uint64
	// name is the human-readable name of the member. If the member is not started, the name will be an empty string.
	Name string
	// peerURLs is the list of URLs the member exposes to the cluster for communication.
	PeerURLs []string
	// clientURLs is the list of URLs the member exposes to clients for communication. If the member is not started, clientURLs will be empty.
	ClientURLs []string
	// isLearner indicates if the member is raft learner.
	IsLearner *bool
}

type RunnerStatus struct {
	Phase RunnerPhase
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerList is a list of Runner objects.
type RunnerList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Runner
	Items []Runner
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerStat is the stat information of Runner
type RunnerStat struct {
	metav1.TypeMeta

	Id            uint64
	CpuPer        float64
	MemoryPer     float64
	Regions       []uint64
	Leaders       []string
	Definitions   uint64
	BpmnProcesses uint64
	BpmnEvents    uint64
	BpmnTasks     uint64
	Message       string
	Timestamp     int64
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

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Region the olive node
type Region struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   RegionSpec
	Status RegionStatus
}

// RegionSpec is the specification of a Region.
type RegionSpec struct {
	Id               uint64
	DeploymentId     uint64
	Replicas         []RegionReplica
	ElectionRTT      uint64
	HeartbeatRTT     uint64
	Leader           uint64
	Definitions      uint64
	DefinitionsLimit uint64
}

type RegionReplica struct {
	Id          uint64
	Runner      uint64
	Region      uint64
	RaftAddress string
	IsNonVoting bool
	IsWitness   bool
	IsJoin      bool
}

type RegionStatus struct {
	Phase RegionPhase
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionList is a list of Region objects.
type RegionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Region
	Items []Region
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionStat is the stat information of Region
type RegionStat struct {
	metav1.TypeMeta

	Id                 uint64
	Leader             uint64
	Term               uint64
	Replicas           int32
	Definitions        uint64
	RunningDefinitions uint64
	BpmnProcesses      uint64
	BpmnEvents         uint64
	BpmnTasks          uint64
	Message            string
	Timestamp          int64
}
