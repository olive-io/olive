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
	"github.com/olive-io/olive/api"
)

// +gogo:genproto=true
// +gogo:deepcopy=true
type TypeMeta struct {
	Kind       string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind,proto3"`
	ApiVersion string `json:"apiVersion,omitempty" protobuf:"bytes,2,opt,name=apiVersion,proto3"`
}

func (m *TypeMeta) SetGroupVersionKind(kind api.GroupVersionKind) {
	m.ApiVersion = kind.GroupVersion().String()
	m.Kind = kind.Kind
}

func (m *TypeMeta) GroupVersionKind() api.GroupVersionKind {
	gv, _ := api.ParseGroupVersion(m.ApiVersion)
	return gv.WithKind(m.Kind)
}

func (m *TypeMeta) GetObjectKind() api.ObjectKind {
	return m
}

func (m *TypeMeta) SetKind(kind string) {
	m.Kind = kind
}

// +gogo:genproto=true
// +gogo:deepcopy=true
type ObjectMeta struct {
	Name              string            `json:"name,omitempty" protobuf:"bytes,1,opt,name=name,proto3"`
	UID               string            `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid,proto3"`
	Description       string            `json:"description,omitempty" protobuf:"bytes,3,opt,name=description,proto3"`
	Namespace         string            `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace,proto3"`
	CreationTimestamp int64             `json:"creationTimestamp,omitempty" protobuf:"varint,5,opt,name=creationTimestamp,proto3"`
	UpdateTimestamp   int64             `json:"updateTimestamp,omitempty" protobuf:"varint,6,opt,name=updateTimestamp,proto3"`
	DeletionTimestamp int64             `json:"deletionTimestamp,omitempty" protobuf:"varint,7,opt,name=deletionTimestamp,proto3"`
	Version           int64             `json:"version,omitempty" protobuf:"varint,8,opt,name=version,proto3"`
	Labels            map[string]string `json:"labels,omitempty" protobuf:"bytes,9,rep,name=labels,proto3"`
	Annotations       map[string]string `json:"annotations,omitempty" protobuf:"bytes,10,rep,name=annotations,proto3"`
}

func (m *ObjectMeta) SetName(name string) {
	m.Name = name
}

func (m *ObjectMeta) GetName() string {
	return m.Name
}

func (m *ObjectMeta) SetUID(uid string) {
	m.UID = uid
}

func (m *ObjectMeta) GetUID() string {
	return m.UID
}

func (m *ObjectMeta) SetDescription(description string) {
	m.Description = description
}

func (m *ObjectMeta) GetDescription() string {
	return m.Description
}

func (m *ObjectMeta) SetNamespace(namespace string) {
	m.Namespace = namespace
}

func (m *ObjectMeta) GetNamespace() string {
	return m.Namespace
}

func (m *ObjectMeta) SetCreationTimestamp(timestamp int64) {
	m.CreationTimestamp = timestamp
}

func (m *ObjectMeta) GetCreationTimestamp() int64 {
	return m.CreationTimestamp
}

func (m *ObjectMeta) SetUpdateTimestamp(timestamp int64) {
	m.UpdateTimestamp = timestamp
}

func (m *ObjectMeta) GetUpdateTimestamp() int64 {
	return m.UpdateTimestamp
}

func (m *ObjectMeta) SetDeletionTimestamp(timestamp int64) {
	m.DeletionTimestamp = timestamp
}

func (m *ObjectMeta) GetDeletionTimestamp() int64 {
	return m.DeletionTimestamp
}

func (m *ObjectMeta) GetResourceVersion() int64 {
	return m.Version
}

func (m *ObjectMeta) SetResourceVersion(rev int64) {
	m.Version = rev
}

func (m *ObjectMeta) SetLabels(labels map[string]string) {
	m.Labels = labels
}

func (m *ObjectMeta) GetLabels() map[string]string {
	return m.Labels
}

func (m *ObjectMeta) SetAnnotations(annotations map[string]string) {
	m.Annotations = annotations
}

func (m *ObjectMeta) GetAnnotations() map[string]string {
	return m.Annotations
}

// +gogo:genproto=true
type ListOptions struct {
	GVK       string            `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	Limit     int64             `json:"limit,omitempty" protobuf:"varint,2,opt,name=limit,proto3"`
	Namespace string            `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace,proto3"`
	Labels    map[string]string `json:"labels,omitempty" protobuf:"bytes,4,rep,name=labels,proto3"`
}

// +gogo:genproto=true
type GetOptions struct {
	GVK       string `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	UID       string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name,proto3"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace,proto3"`
}

// +gogo:genproto=true
type PostOptions struct {
	GVK       string `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	UID       string `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid,proto3"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace,proto3"`
	Data      []byte `json:"data,omitempty" protobuf:"bytes,4,opt,name=data,proto3"`
}

// +gogo:genproto=true
type PatchOptions struct {
	GVK       string `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	UID       string `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid,proto3"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace,proto3"`
	Data      []byte `json:"data,omitempty" protobuf:"bytes,4,opt,name=data,proto3"`
}

// +gogo:genproto=true
type DeleteOptions struct {
	GVK       string `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace,proto3"`
	UID       string `json:"uid,omitempty" protobuf:"bytes,3,opt,name=uid,proto3"`
}

// +gogo:genproto=true
type Result struct {
	GVK   string `json:"gvk,omitempty" protobuf:"bytes,1,opt,name=gvk,proto3"`
	List  bool   `json:"list,omitempty" protobuf:"varint,2,opt,name=list,proto3"`
	Total int64  `json:"total,omitempty" protobuf:"varint,3,opt,name=total,proto3"`
	Data  []byte `json:"data,omitempty" protobuf:"bytes,4,opt,name=data,proto3"`
}
