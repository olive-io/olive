//
//Copyright 2023 The olive Authors
//
//This program is offered under a commercial and under the AGPL license.
//For AGPL licensing, see below.
//
//AGPL licensing:
//This program is free software: you can redistribute it and/or modify
//it under the terms of the GNU Affero General Public License as published by
//the Free Software Foundation, either version 3 of the License, or
//(at your option) any later version.
//
//This program is distributed in the hope that it will be useful,
//but WITHOUT ANY WARRANTY; without even the implied warranty of
//MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//GNU Affero General Public License for more details.
//
//You should have received a copy of the GNU Affero General Public License
//along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.25.3
// source: github.com/olive-io/olive/apis/pb/auth/auth.proto

package authpbv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PType int32

const (
	PType_UNKNOWN PType = 0
	PType_POLICY  PType = 1 // p
	PType_ROLE    PType = 2 // g
)

// Enum value maps for PType.
var (
	PType_name = map[int32]string{
		0: "UNKNOWN",
		1: "POLICY",
		2: "ROLE",
	}
	PType_value = map[string]int32{
		"UNKNOWN": 0,
		"POLICY":  1,
		"ROLE":    2,
	}
)

func (x PType) Enum() *PType {
	p := new(PType)
	*p = x
	return p
}

func (x PType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[0].Descriptor()
}

func (PType) Type() protoreflect.EnumType {
	return &file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[0]
}

func (x PType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PType.Descriptor instead.
func (PType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{0}
}

type Resource int32

const (
	Resource_UNKNOWN_Resource Resource = 0
	Resource_MetaMember       Resource = 1
	Resource_Runner           Resource = 11
	Resource_Region           Resource = 12
	Resource_AuthRole         Resource = 21
	Resource_AuthUser         Resource = 22
	Resource_Authentication   Resource = 23 // login, rbac policy
	Resource_BpmnDefinition   Resource = 31
	Resource_BpmnProcess      Resource = 32
)

// Enum value maps for Resource.
var (
	Resource_name = map[int32]string{
		0:  "UNKNOWN_Resource",
		1:  "MetaMember",
		11: "Runner",
		12: "Region",
		21: "AuthRole",
		22: "AuthUser",
		23: "Authentication",
		31: "BpmnDefinition",
		32: "BpmnProcess",
	}
	Resource_value = map[string]int32{
		"UNKNOWN_Resource": 0,
		"MetaMember":       1,
		"Runner":           11,
		"Region":           12,
		"AuthRole":         21,
		"AuthUser":         22,
		"Authentication":   23,
		"BpmnDefinition":   31,
		"BpmnProcess":      32,
	}
)

func (x Resource) Enum() *Resource {
	p := new(Resource)
	*p = x
	return p
}

func (x Resource) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Resource) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[1].Descriptor()
}

func (Resource) Type() protoreflect.EnumType {
	return &file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[1]
}

func (x Resource) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Resource.Descriptor instead.
func (Resource) EnumDescriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{1}
}

type Action int32

const (
	Action_UNKNOWN_Action Action = 0
	Action_Read           Action = 1
	Action_Write          Action = 2
)

// Enum value maps for Action.
var (
	Action_name = map[int32]string{
		0: "UNKNOWN_Action",
		1: "Read",
		2: "Write",
	}
	Action_value = map[string]int32{
		"UNKNOWN_Action": 0,
		"Read":           1,
		"Write":          2,
	}
)

func (x Action) Enum() *Action {
	p := new(Action)
	*p = x
	return p
}

func (x Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Action) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[2].Descriptor()
}

func (Action) Type() protoreflect.EnumType {
	return &file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes[2]
}

func (x Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Action.Descriptor instead.
func (Action) EnumDescriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{2}
}

type Role struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name              string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Desc              string            `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	Metadata          map[string]string `protobuf:"bytes,3,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Namespace         string            `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
	CreationTimestamp int64             `protobuf:"varint,11,opt,name=creationTimestamp,proto3" json:"creationTimestamp,omitempty"`
	UpdateTimestamp   int64             `protobuf:"varint,12,opt,name=updateTimestamp,proto3" json:"updateTimestamp,omitempty"`
}

func (x *Role) Reset() {
	*x = Role{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Role) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role) ProtoMessage() {}

func (x *Role) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role.ProtoReflect.Descriptor instead.
func (*Role) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{0}
}

func (x *Role) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Role) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *Role) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Role) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *Role) GetCreationTimestamp() int64 {
	if x != nil {
		return x.CreationTimestamp
	}
	return 0
}

func (x *Role) GetUpdateTimestamp() int64 {
	if x != nil {
		return x.UpdateTimestamp
	}
	return 0
}

type RolePatcher struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Desc      string            `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty"`
	Metadata  map[string]string `protobuf:"bytes,2,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Namespace string            `protobuf:"bytes,3,opt,name=namespace,proto3" json:"namespace,omitempty"`
}

func (x *RolePatcher) Reset() {
	*x = RolePatcher{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RolePatcher) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RolePatcher) ProtoMessage() {}

func (x *RolePatcher) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RolePatcher.ProtoReflect.Descriptor instead.
func (*RolePatcher) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{1}
}

func (x *RolePatcher) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *RolePatcher) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *RolePatcher) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

type User struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name              string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Desc              string            `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	Metadata          map[string]string `protobuf:"bytes,3,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Role              string            `protobuf:"bytes,4,opt,name=role,proto3" json:"role,omitempty"`
	Namespace         string            `protobuf:"bytes,5,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Password          string            `protobuf:"bytes,6,opt,name=password,proto3" json:"password,omitempty"`
	CreationTimestamp int64             `protobuf:"varint,11,opt,name=creationTimestamp,proto3" json:"creationTimestamp,omitempty"`
	UpdateTimestamp   int64             `protobuf:"varint,12,opt,name=updateTimestamp,proto3" json:"updateTimestamp,omitempty"`
}

func (x *User) Reset() {
	*x = User{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{2}
}

func (x *User) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *User) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *User) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *User) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

func (x *User) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *User) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *User) GetCreationTimestamp() int64 {
	if x != nil {
		return x.CreationTimestamp
	}
	return 0
}

func (x *User) GetUpdateTimestamp() int64 {
	if x != nil {
		return x.UpdateTimestamp
	}
	return 0
}

type UserPatcher struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Desc      string            `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty"`
	Metadata  map[string]string `protobuf:"bytes,2,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Role      string            `protobuf:"bytes,3,opt,name=role,proto3" json:"role,omitempty"`
	Namespace string            `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Password  string            `protobuf:"bytes,5,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *UserPatcher) Reset() {
	*x = UserPatcher{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserPatcher) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserPatcher) ProtoMessage() {}

func (x *UserPatcher) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserPatcher.ProtoReflect.Descriptor instead.
func (*UserPatcher) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{3}
}

func (x *UserPatcher) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *UserPatcher) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *UserPatcher) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

func (x *UserPatcher) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *UserPatcher) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type Token struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TokenText      string `protobuf:"bytes,1,opt,name=tokenText,proto3" json:"tokenText,omitempty"`
	Role           string `protobuf:"bytes,2,opt,name=role,proto3" json:"role,omitempty"`
	User           string `protobuf:"bytes,3,opt,name=user,proto3" json:"user,omitempty"`
	StartTimestamp int64  `protobuf:"varint,11,opt,name=startTimestamp,proto3" json:"startTimestamp,omitempty"`
	EndTimestamp   int64  `protobuf:"varint,12,opt,name=endTimestamp,proto3" json:"endTimestamp,omitempty"`
}

func (x *Token) Reset() {
	*x = Token{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Token) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token) ProtoMessage() {}

func (x *Token) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token.ProtoReflect.Descriptor instead.
func (*Token) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{4}
}

func (x *Token) GetTokenText() string {
	if x != nil {
		return x.TokenText
	}
	return ""
}

func (x *Token) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

func (x *Token) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *Token) GetStartTimestamp() int64 {
	if x != nil {
		return x.StartTimestamp
	}
	return 0
}

func (x *Token) GetEndTimestamp() int64 {
	if x != nil {
		return x.EndTimestamp
	}
	return 0
}

type Policy struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ptype  PType    `protobuf:"varint,1,opt,name=ptype,proto3,enum=authpbv1.PType" json:"ptype,omitempty"`
	Sub    string   `protobuf:"bytes,2,opt,name=sub,proto3" json:"sub,omitempty"`
	Role   string   `protobuf:"bytes,11,opt,name=role,proto3" json:"role,omitempty"`
	Domain string   `protobuf:"bytes,12,opt,name=domain,proto3" json:"domain,omitempty"`
	Data   Resource `protobuf:"varint,21,opt,name=data,proto3,enum=authpbv1.Resource" json:"data,omitempty"`
	Action Action   `protobuf:"varint,22,opt,name=action,proto3,enum=authpbv1.Action" json:"action,omitempty"`
}

func (x *Policy) Reset() {
	*x = Policy{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Policy) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Policy) ProtoMessage() {}

func (x *Policy) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Policy.ProtoReflect.Descriptor instead.
func (*Policy) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{5}
}

func (x *Policy) GetPtype() PType {
	if x != nil {
		return x.Ptype
	}
	return PType_UNKNOWN
}

func (x *Policy) GetSub() string {
	if x != nil {
		return x.Sub
	}
	return ""
}

func (x *Policy) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

func (x *Policy) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *Policy) GetData() Resource {
	if x != nil {
		return x.Data
	}
	return Resource_UNKNOWN_Resource
}

func (x *Policy) GetAction() Action {
	if x != nil {
		return x.Action
	}
	return Action_UNKNOWN_Action
}

type Scope struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Resource Resource `protobuf:"varint,1,opt,name=resource,proto3,enum=authpbv1.Resource" json:"resource,omitempty"`
	Action   Action   `protobuf:"varint,2,opt,name=action,proto3,enum=authpbv1.Action" json:"action,omitempty"`
}

func (x *Scope) Reset() {
	*x = Scope{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Scope) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Scope) ProtoMessage() {}

func (x *Scope) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Scope.ProtoReflect.Descriptor instead.
func (*Scope) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP(), []int{6}
}

func (x *Scope) GetResource() Resource {
	if x != nil {
		return x.Resource
	}
	return Resource_UNKNOWN_Resource
}

func (x *Scope) GetAction() Action {
	if x != nil {
		return x.Action
	}
	return Action_UNKNOWN_Action
}

var File_github_com_olive_io_olive_api_pb_auth_auth_proto protoreflect.FileDescriptor

var file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDesc = []byte{
	0x0a, 0x30, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x6c, 0x69,
	0x76, 0x65, 0x2d, 0x69, 0x6f, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x70, 0x62, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x08, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x22, 0x9b, 0x02, 0x0a,
	0x04, 0x52, 0x6f, 0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73,
	0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x38, 0x0a,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x52, 0x6f, 0x6c, 0x65, 0x2e,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x2c, 0x0a, 0x11, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x11, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x12, 0x28, 0x0a, 0x0f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0f, 0x75, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x1a, 0x3b, 0x0a,
	0x0d, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xbd, 0x01, 0x0a, 0x0b, 0x52,
	0x6f, 0x6c, 0x65, 0x50, 0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65,
	0x73, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x3f,
	0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x23, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x52, 0x6f, 0x6c, 0x65,
	0x50, 0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12,
	0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x1a, 0x3b, 0x0a,
	0x0d, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xcb, 0x02, 0x0a, 0x04, 0x55,
	0x73, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x38, 0x0a, 0x08, 0x6d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x2e, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6c, 0x65, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6c, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d,
	0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77,
	0x6f, 0x72, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77,
	0x6f, 0x72, 0x64, 0x12, 0x2c, 0x0a, 0x11, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x03, 0x52, 0x11,
	0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x12, 0x28, 0x0a, 0x0f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0f, 0x75, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x1a, 0x3b, 0x0a, 0x0d, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xed, 0x01, 0x0a, 0x0b, 0x55, 0x73, 0x65,
	0x72, 0x50, 0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x3f, 0x0a, 0x08,
	0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23,
	0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61,
	0x74, 0x63, 0x68, 0x65, 0x72, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a,
	0x04, 0x72, 0x6f, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6c,
	0x65, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12,
	0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x1a, 0x3b, 0x0a, 0x0d, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x99, 0x01, 0x0a, 0x05, 0x54, 0x6f, 0x6b,
	0x65, 0x6e, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x54, 0x65, 0x78, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x54, 0x65, 0x78, 0x74,
	0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x72, 0x6f, 0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x26, 0x0a, 0x0e, 0x73, 0x74, 0x61, 0x72,
	0x74, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x0e, 0x73, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x12, 0x22, 0x0a, 0x0c, 0x65, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x18, 0x0c, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x65, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x22, 0xbf, 0x01, 0x0a, 0x06, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x12,
	0x25, 0x0a, 0x05, 0x70, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f,
	0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x50, 0x54, 0x79, 0x70, 0x65, 0x52,
	0x05, 0x70, 0x74, 0x79, 0x70, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x75, 0x62, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x73, 0x75, 0x62, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6c, 0x65,
	0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6c, 0x65, 0x12, 0x16, 0x0a, 0x06,
	0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x6f,
	0x6d, 0x61, 0x69, 0x6e, 0x12, 0x26, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x15, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x12, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x28, 0x0a, 0x06,
	0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x16, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x10, 0x2e, 0x61,
	0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06,
	0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x61, 0x0a, 0x05, 0x53, 0x63, 0x6f, 0x70, 0x65, 0x12,
	0x2e, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x12, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12,
	0x28, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x10, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2a, 0x2a, 0x0a, 0x05, 0x50, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12,
	0x0a, 0x0a, 0x06, 0x50, 0x4f, 0x4c, 0x49, 0x43, 0x59, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x52,
	0x4f, 0x4c, 0x45, 0x10, 0x02, 0x2a, 0x9d, 0x01, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x12, 0x14, 0x0a, 0x10, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x5f, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x4d, 0x65, 0x74, 0x61,
	0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x75, 0x6e, 0x6e,
	0x65, 0x72, 0x10, 0x0b, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x10, 0x0c,
	0x12, 0x0c, 0x0a, 0x08, 0x41, 0x75, 0x74, 0x68, 0x52, 0x6f, 0x6c, 0x65, 0x10, 0x15, 0x12, 0x0c,
	0x0a, 0x08, 0x41, 0x75, 0x74, 0x68, 0x55, 0x73, 0x65, 0x72, 0x10, 0x16, 0x12, 0x12, 0x0a, 0x0e,
	0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x10, 0x17,
	0x12, 0x12, 0x0a, 0x0e, 0x42, 0x70, 0x6d, 0x6e, 0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x10, 0x1f, 0x12, 0x0f, 0x0a, 0x0b, 0x42, 0x70, 0x6d, 0x6e, 0x50, 0x72, 0x6f, 0x63,
	0x65, 0x73, 0x73, 0x10, 0x20, 0x2a, 0x31, 0x0a, 0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x12, 0x0a, 0x0e, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x5f, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x52, 0x65, 0x61, 0x64, 0x10, 0x01, 0x12, 0x09, 0x0a,
	0x05, 0x57, 0x72, 0x69, 0x74, 0x65, 0x10, 0x02, 0x42, 0x30, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2d, 0x69, 0x6f, 0x2f,
	0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x62, 0x2f, 0x61, 0x75, 0x74,
	0x68, 0x3b, 0x61, 0x75, 0x74, 0x68, 0x70, 0x62, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescOnce sync.Once
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescData = file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDesc
)

func file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescGZIP() []byte {
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescOnce.Do(func() {
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescData)
	})
	return file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDescData
}

var file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_github_com_olive_io_olive_api_pb_auth_auth_proto_goTypes = []interface{}{
	(PType)(0),          // 0: authpbv1.PType
	(Resource)(0),       // 1: authpbv1.Resource
	(Action)(0),         // 2: authpbv1.Action
	(*Role)(nil),        // 3: authpbv1.Role
	(*RolePatcher)(nil), // 4: authpbv1.RolePatcher
	(*User)(nil),        // 5: authpbv1.User
	(*UserPatcher)(nil), // 6: authpbv1.UserPatcher
	(*Token)(nil),       // 7: authpbv1.Token
	(*Policy)(nil),      // 8: authpbv1.Policy
	(*Scope)(nil),       // 9: authpbv1.Scope
	nil,                 // 10: authpbv1.Role.MetadataEntry
	nil,                 // 11: authpbv1.RolePatcher.MetadataEntry
	nil,                 // 12: authpbv1.User.MetadataEntry
	nil,                 // 13: authpbv1.UserPatcher.MetadataEntry
}
var file_github_com_olive_io_olive_api_pb_auth_auth_proto_depIdxs = []int32{
	10, // 0: authpbv1.Role.metadata:type_name -> authpbv1.Role.MetadataEntry
	11, // 1: authpbv1.RolePatcher.metadata:type_name -> authpbv1.RolePatcher.MetadataEntry
	12, // 2: authpbv1.User.metadata:type_name -> authpbv1.User.MetadataEntry
	13, // 3: authpbv1.UserPatcher.metadata:type_name -> authpbv1.UserPatcher.MetadataEntry
	0,  // 4: authpbv1.Policy.ptype:type_name -> authpbv1.PType
	1,  // 5: authpbv1.Policy.data:type_name -> authpbv1.Resource
	2,  // 6: authpbv1.Policy.action:type_name -> authpbv1.Action
	1,  // 7: authpbv1.Scope.resource:type_name -> authpbv1.Resource
	2,  // 8: authpbv1.Scope.action:type_name -> authpbv1.Action
	9,  // [9:9] is the sub-list for method output_type
	9,  // [9:9] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_github_com_olive_io_olive_api_pb_auth_auth_proto_init() }
func file_github_com_olive_io_olive_api_pb_auth_auth_proto_init() {
	if File_github_com_olive_io_olive_api_pb_auth_auth_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Role); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RolePatcher); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*User); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserPatcher); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Token); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Policy); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Scope); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_olive_io_olive_api_pb_auth_auth_proto_goTypes,
		DependencyIndexes: file_github_com_olive_io_olive_api_pb_auth_auth_proto_depIdxs,
		EnumInfos:         file_github_com_olive_io_olive_api_pb_auth_auth_proto_enumTypes,
		MessageInfos:      file_github_com_olive_io_olive_api_pb_auth_auth_proto_msgTypes,
	}.Build()
	File_github_com_olive_io_olive_api_pb_auth_auth_proto = out.File
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_rawDesc = nil
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_goTypes = nil
	file_github_com_olive_io_olive_api_pb_auth_auth_proto_depIdxs = nil
}