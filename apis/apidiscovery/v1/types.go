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

type BoxType string

const (
	StringType  BoxType = "string"
	IntegerType BoxType = "integer"
	FloatType   BoxType = "float"
	BooleanType BoxType = "boolean"
	ArrayType   BoxType = "array"
	ObjectType  BoxType = "object"
	MapType     BoxType = "map"
)

// +k8s:deepcopy-gen:true

// Box is an opaque value for a request or response
type Box struct {
	Type BoxType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=BoxType"`
	// Box Value by json format
	Data string `json:"data" protobuf:"bytes,2,opt,name=data"`
	// Example Value by json format
	Example string `json:"example" protobuf:"bytes,3,opt,name=example"`
	// Default Value by json format
	Default string `json:"default" protobuf:"bytes,4,opt,name=default"`
	// the reference, points to OpenAPI Component when type is Object
	Ref        string         `json:"ref" protobuf:"bytes,5,opt,name=ref"`
	Parameters map[string]Box `json:"parameters" protobuf:"bytes,6,rep,name=parameters"`
}

type ActivityType string

const (
	Task         ActivityType = "Task"
	ServiceTask  ActivityType = "ServiceTask"
	ScriptTask   ActivityType = "ScriptTask"
	UserTask     ActivityType = "UserTask"
	SendTask     ActivityType = "SendTask"
	ReceiveTask  ActivityType = "ReceiveTask"
	ManualTask   ActivityType = "ManualTask"
	CallActivity ActivityType = "CallActivity"
)

// +k8s:deepcopy-gen:true

// Activity defines bpmn Activity of specification
type Activity struct {
	metav1.TypeMeta `json:",inline"`

	// the type of activity node, etc ServiceTask, ScriptTask
	Type ActivityType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=ActivityType"`
	// the id of activity node
	Id string `json:"id" protobuf:"bytes,2,opt,name=id"`
	// the name of activity node
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
	// the type of activity node, defines in activity TaskDefinition
	TaskType string `json:"taskType" protobuf:"bytes,4,opt,name=taskType"`
	// the id of bpmn definition
	Definition string `json:"definition" protobuf:"bytes,5,opt,name=definition"`
	// the version of bpmn definitions
	DefinitionsVersion uint64 `json:"definitionsVersion" protobuf:"varint,6,opt,name=definitionsVersion"`
	// the id if bpmn process
	Process string `json:"process" protobuf:"bytes,7,opt,name=process"`
}

type EndpointPhase string

const (
	EndpointActive EndpointPhase = "Active"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Endpoint is an endpoint provided by a service
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EndpointSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status EndpointStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type EndpointSpec struct {
	Http *EndpointHttp `json:"http,omitempty" protobuf:"bytes,1,opt,name=http"`
}

type EndpointHttp struct {
	URL      string `json:"url" protobuf:"bytes,1,opt,name=url"`
	Method   string `json:"method" protobuf:"bytes,2,opt,name=method"`
	Request  Box    `json:"request" protobuf:"bytes,3,opt,name=request"`
	Response Box    `json:"response" protobuf:"bytes,4,opt,name=response"`
}

type EndpointStatus struct {
	Phase EndpointPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=EndpointPhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EndpointList is a list of Endpoint objects.
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Endpoint
	Items []Endpoint `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type PluginServicePhase string

const (
	ServiceActive PluginServicePhase = "Active"
)

// +k8s:deepcopy-gen:true

// Node represents the node the service is on
type Node struct {
	metav1.TypeMeta `json:",inline"`

	Id       string            `json:"id" protobuf:"bytes,1,opt,name=id"`
	Metadata map[string]string `json:"metadata" protobuf:"bytes,2,rep,name=metadata"`
	Address  string            `json:"address" protobuf:"bytes,3,opt,name=address"`
	Port     int64             `json:"port" protobuf:"varint,4,opt,name=port"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PluginService represents the pluggable service
type PluginService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PluginServiceSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status PluginServiceStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type PluginServiceSpec struct {
	Runner    string   `json:"runner" protobuf:"bytes,1,opt,name=runner"`
	Version   string   `json:"version" protobuf:"bytes,2,opt,name=version"`
	Endpoints []string `json:"endpoints" protobuf:"bytes,3,rep,name=endpoints"`
	Nodes     []Node   `json:"nodes" protobuf:"bytes,4,rep,name=nodes"`
}

type PluginServiceStatus struct {
	Phase PluginServicePhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=ServicePhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PluginServiceList is a list of PluginService objects.
type PluginServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of PluginServicePhase
	Items []PluginService `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type EdgePhase string

const (
	EdgeActive EdgePhase = "Active"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Edge defines the sets of Endpoint
type Edge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EdgeSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status EdgeStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type EdgeSpec struct {
	Address string `json:"address" protobuf:"bytes,1,opt,name=address"`

	Endpoints []string `json:"endpoints" protobuf:"bytes,2,rep,name=endpoints"`
}

type EdgeStatus struct {
	Phase EdgePhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=EdgePhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeList is a list of Edge objects.
type EdgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Edge
	Items []Edge `json:"items" protobuf:"bytes,2,rep,name=items"`
}
