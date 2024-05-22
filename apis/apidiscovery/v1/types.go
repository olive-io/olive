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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BoxType string

const (
	StringType  = BoxType("string")
	IntegerType = BoxType("integer")
	FloatType   = BoxType("float")
	BooleanType = BoxType("boolean")
	ArrayType   = BoxType("array")
	ObjectType  = BoxType("object")
	MapType     = BoxType("map")
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Box is an opaque value for a request or response
type Box struct {
	metav1.TypeMeta `json:",inline"`

	Type BoxType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=BoxType"`
	// Box Value by json.Marshal
	Data string `json:"data" protobuf:"bytes,2,opt,name=data"`
	// the reference, points to OpenAPI Component when type is Object
	Ref        string          `json:"ref" protobuf:"bytes,3,opt,name=ref"`
	Parameters map[string]*Box `json:"parameters" protobuf:"bytes,4,rep,name=parameters"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Endpoint is an endpoint provided by a service
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec EndpointSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type EndpointSpec struct {
	Request  Box `json:"request" protobuf:"bytes,1,opt,name=request"`
	Response Box `json:"response" protobuf:"bytes,2,opt,name=response"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EndpointList is a list of Endpoint objects.
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Endpoint
	Items []Endpoint `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Node represents the node the service is on
type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec NodeSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type NodeSpec struct {
	Address string `json:"address" protobuf:"bytes,1,opt,name=address"`
	Port    int64  `json:"port" protobuf:"varint,2,opt,name=port"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList is a list of Node objects.
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Node
	Items []Node `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Service represents a olive service
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ServiceSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type ServiceSpec struct {
	Version   string     `json:"version" protobuf:"bytes,1,opt,name=version"`
	Endpoints []Endpoint `json:"endpoints" protobuf:"bytes,2,rep,name=endpoints"`
	Nodes     []Node     `json:"nodes" protobuf:"bytes,3,rep,name=nodes"`
	Ttl       int64      `json:"ttl" protobuf:"varint,4,opt,name=ttl"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceList is a list of Service objects.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Service
	Items []Service `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type ActivityTask string

const (
	Task         = ActivityTask("Task")
	ServiceTask  = ActivityTask("ServiceTask")
	ScriptTask   = ActivityTask("ScriptTask")
	UserTask     = ActivityTask("UserTask")
	SendTask     = ActivityTask("SendTask")
	ReceiveTask  = ActivityTask("ReceiveTask")
	ManualTask   = ActivityTask("ManualTask")
	CallActivity = ActivityTask("CallActivity")
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Activity defines bpmn Activity of specification
type Activity struct {
	metav1.TypeMeta `json:",inline"`

	// the type of activity node, etc ServiceTask, ScriptTask
	Type ActivityTask `json:"type" protobuf:"bytes,1,opt,name=type,casttype=ActivityTask"`
	// the id of activity node
	Id string `json:"id" protobuf:"bytes,2,opt,name=id"`
	// the name of activity node
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
	// the type of activity node, defines in activity TaskDefinition
	TaskType string `json:"taskType" protobuf:"bytes,4,opt,name=taskType"`
	// the id of bpmn definitions
	Definitions string `json:"definitions" protobuf:"bytes,5,opt,name=definitions"`
	// the version of bpmn definitions
	DefinitionsVersion uint64 `json:"definitionsVersion" protobuf:"varint,6,opt,name=definitionsVersion"`
	// the id if bpmn process
	Process string `json:"process" protobuf:"bytes,7,opt,name=process"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Consumer represents a gateway External interfaces
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ConsumerSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type ConsumerSpec struct {
	Activity ActivityTask `json:"activity" protobuf:"bytes,1,opt,name=activity,casttype=ActivityTask"`
	Action   string       `json:"action" protobuf:"bytes,2,opt,name=action"`
	Request  Box          `json:"request" protobuf:"bytes,3,opt,name=request"`
	Response Box          `json:"response" protobuf:"bytes,4,opt,name=response"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer objects.
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Consumer
	Items []Consumer `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Yard defines the sets of Consumer
type Yard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec YardSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type YardSpec struct {
	Address   string              `json:"address" protobuf:"bytes,1,opt,name=address"`
	Consumers map[string]Consumer `json:"consumers" protobuf:"bytes,2,rep,name=consumers"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// YardList is a list of Yard objects.
type YardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Yard
	Items []Yard `json:"items" protobuf:"bytes,2,rep,name=items"`
}
