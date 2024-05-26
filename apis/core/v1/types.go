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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

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
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

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
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

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
