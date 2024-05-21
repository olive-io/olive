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

package discovery

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
	metav1.TypeMeta

	Type BoxType
	// Box Value by json.Marshal
	Data string
	// the reference, points to OpenAPI Component when type is Object
	Ref        string
	Parameters map[string]*Box
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Endpoint is an endpoint provided by a service
type Endpoint struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec EndpointSpec
}

type EndpointSpec struct {
	Request  Box
	Response Box
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EndpointList is a list of Endpoint objects.
type EndpointList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Endpoint
	Items []Endpoint
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Node represents the node the service is on
type Node struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec NodeSpec
}

type NodeSpec struct {
	Address string
	Port    int64
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList is a list of Node objects.
type NodeList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Node
	Items []Node
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Service represents a olive service
type Service struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec ServiceSpec
}

type ServiceSpec struct {
	Version   string
	Endpoints []Endpoint
	Nodes     []Node
	Ttl       int64
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceList is a list of Service objects.
type ServiceList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Service
	Items []Service
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
	metav1.TypeMeta

	// the type of activity node, etc ServiceTask, ScriptTask
	Type ActivityTask `json:"type"`
	// the id of activity node
	Id string `json:"id"`
	// the name of activity node
	Name string `json:"name"`
	// the type of activity node, defines in activity TaskDefinition
	TaskType string `json:"task-type"`
	// the id of bpmn definitions
	Definitions string `json:"definitions"`
	// the version of bpmn definitions
	DefinitionsVersion uint64 `json:"definitions-version"`
	// the id if bpmn process
	Process string `json:"process"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Consumer represents a gateway External interfaces
type Consumer struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec ConsumerSpec
}

type ConsumerSpec struct {
	Activity ActivityTask
	Action   string
	Request  Box
	Response Box
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsumerList is a list of Consumer objects.
type ConsumerList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Consumer
	Items []Consumer
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Yard defines the sets of Consumer
type Yard struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec YardSpec
}

type YardSpec struct {
	Address   string
	Consumers map[string]Consumer
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// YardList is a list of Yard objects.
type YardList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items is a list of Yard
	Items []Yard
}
