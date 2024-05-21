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

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	discovery "github.com/olive-io/olive/apis/pb/discovery"
)

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
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Definition is bpmn definitions
type Definition struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   DefinitionSpec
	Status DefinitionStatus
}

type DefinitionSpec struct {
	Content string
	Version uint64
	// the id of olive region
	Region uint64
}

type DefinitionStatus struct {
	Phase DefPhase
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefinitionList is a list of Definition objects.
type DefinitionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Definition
	Items []Definition
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

// ProcessInstance is bpmn process instance
type ProcessInstance struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   ProcessInstanceSpec
	Status ProcessInstanceStatus
}

type ProcessInstanceSpec struct {
	// the id of Definition
	Definition string
	// the version if Definition
	Version uint64
	// the process id of bpmn Process Element in Definition
	BpmnProcess string
	Headers     map[string]string
	Properties  map[string]discovery.Box
	DataObjects map[string]string
}

type ProcessInstanceStatus struct {
	Phase   ProcessPhase
	Message string
	// the id of olive region
	Region uint64
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessInstanceList is a list of ProcessInstance objects.
type ProcessInstanceList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of ProcessInstance
	Items []ProcessInstance
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessInstanceStat is stat information of ProcessInstance
type ProcessInstanceStat struct {
	metav1.TypeMeta

	// The id of ProcessInstance
	Id                string
	DefinitionContent string

	ProcessState ProcessRunningState

	Attempts  uint64
	FlowNodes map[string]FlowNodeStat
	StartTime int64
	EndTime   int64
}

type ProcessRunningState struct {
	Properties  map[string]string
	DataObjects map[string]string
	Variables   map[string]string
}

type FlowNodeStat struct {
	Id          string
	Name        string
	Headers     map[string]string
	Properties  map[string]string
	DataObjects map[string]string
	StartTime   int64
	EndTime     int64
}
