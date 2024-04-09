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
// source: github.com/olive-io/olive/api/gatewaypb/rpc.proto

package gatewaypb

import (
	reflect "reflect"
	sync "sync"

	discoverypb "github.com/olive-io/olive/api/discoverypb"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PingRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *PingRequest) Reset() {
	*x = PingRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingRequest) ProtoMessage() {}

func (x *PingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingRequest.ProtoReflect.Descriptor instead.
func (*PingRequest) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{0}
}

type PingResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Reply string `protobuf:"bytes,1,opt,name=reply,proto3" json:"reply,omitempty"`
}

func (x *PingResponse) Reset() {
	*x = PingResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PingResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingResponse) ProtoMessage() {}

func (x *PingResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingResponse.ProtoReflect.Descriptor instead.
func (*PingResponse) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{1}
}

func (x *PingResponse) GetReply() string {
	if x != nil {
		return x.Reply
	}
	return ""
}

type TransmitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Activity    *discoverypb.Activity       `protobuf:"bytes,1,opt,name=activity,proto3" json:"activity,omitempty"`
	Headers     map[string]string           `protobuf:"bytes,2,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Properties  map[string]*discoverypb.Box `protobuf:"bytes,3,rep,name=properties,proto3" json:"properties,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	DataObjects map[string]*discoverypb.Box `protobuf:"bytes,4,rep,name=dataObjects,proto3" json:"dataObjects,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TransmitRequest) Reset() {
	*x = TransmitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransmitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransmitRequest) ProtoMessage() {}

func (x *TransmitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransmitRequest.ProtoReflect.Descriptor instead.
func (*TransmitRequest) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{2}
}

func (x *TransmitRequest) GetActivity() *discoverypb.Activity {
	if x != nil {
		return x.Activity
	}
	return nil
}

func (x *TransmitRequest) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *TransmitRequest) GetProperties() map[string]*discoverypb.Box {
	if x != nil {
		return x.Properties
	}
	return nil
}

func (x *TransmitRequest) GetDataObjects() map[string]*discoverypb.Box {
	if x != nil {
		return x.DataObjects
	}
	return nil
}

// Response save the data from gateway Handle function
type TransmitResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Properties  map[string]*discoverypb.Box `protobuf:"bytes,1,rep,name=properties,proto3" json:"properties,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	DataObjects map[string]*discoverypb.Box `protobuf:"bytes,2,rep,name=dataObjects,proto3" json:"dataObjects,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TransmitResponse) Reset() {
	*x = TransmitResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransmitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransmitResponse) ProtoMessage() {}

func (x *TransmitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransmitResponse.ProtoReflect.Descriptor instead.
func (*TransmitResponse) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{3}
}

func (x *TransmitResponse) GetProperties() map[string]*discoverypb.Box {
	if x != nil {
		return x.Properties
	}
	return nil
}

func (x *TransmitResponse) GetDataObjects() map[string]*discoverypb.Box {
	if x != nil {
		return x.DataObjects
	}
	return nil
}

type HelloRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *HelloRequest) Reset() {
	*x = HelloRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HelloRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HelloRequest) ProtoMessage() {}

func (x *HelloRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HelloRequest.ProtoReflect.Descriptor instead.
func (*HelloRequest) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{4}
}

type HelloResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Reply string `protobuf:"bytes,1,opt,name=reply,proto3" json:"reply,omitempty"`
}

func (x *HelloResponse) Reset() {
	*x = HelloResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HelloResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HelloResponse) ProtoMessage() {}

func (x *HelloResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HelloResponse.ProtoReflect.Descriptor instead.
func (*HelloResponse) Descriptor() ([]byte, []int) {
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP(), []int{5}
}

func (x *HelloResponse) GetReply() string {
	if x != nil {
		return x.Reply
	}
	return ""
}

var File_github_com_olive_io_olive_api_gatewaypb_rpc_proto protoreflect.FileDescriptor

var file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDesc = []byte{
	0x0a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x6c, 0x69,
	0x76, 0x65, 0x2d, 0x69, 0x6f, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x2f, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x09, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x1a, 0x39,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65,
	0x2d, 0x69, 0x6f, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x64, 0x69,
	0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x70, 0x62, 0x2f, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x0d, 0x0a, 0x0b, 0x50, 0x69, 0x6e, 0x67, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x24, 0x0a, 0x0c, 0x50, 0x69, 0x6e, 0x67, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x81, 0x04, 0x0a,
	0x0f, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x31, 0x0a, 0x08, 0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x15, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x70, 0x62,
	0x2e, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x08, 0x61, 0x63, 0x74, 0x69, 0x76,
	0x69, 0x74, 0x79, 0x12, 0x41, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62,
	0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x12, 0x4a, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72,
	0x74, 0x69, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x67, 0x61, 0x74,
	0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69,
	0x65, 0x73, 0x12, 0x4d, 0x0a, 0x0b, 0x64, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61,
	0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x64, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x4f, 0x0a,
	0x0f, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x26, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x10, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x70, 0x62, 0x2e,
	0x42, 0x6f, 0x78, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x50,
	0x0a, 0x10, 0x44, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x26, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x70,
	0x62, 0x2e, 0x42, 0x6f, 0x78, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0xd2, 0x02, 0x0a, 0x10, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4b, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74,
	0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x67, 0x61, 0x74, 0x65,
	0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69,
	0x65, 0x73, 0x12, 0x4e, 0x0a, 0x0b, 0x64, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61,
	0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x64, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x73, 0x1a, 0x4f, 0x0a, 0x0f, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x26, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65,
	0x72, 0x79, 0x70, 0x62, 0x2e, 0x42, 0x6f, 0x78, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x1a, 0x50, 0x0a, 0x10, 0x44, 0x61, 0x74, 0x61, 0x4f, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x26, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f,
	0x76, 0x65, 0x72, 0x79, 0x70, 0x62, 0x2e, 0x42, 0x6f, 0x78, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x0e, 0x0a, 0x0c, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x25, 0x0a, 0x0d, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x32, 0xce, 0x01, 0x0a,
	0x07, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x12, 0x57, 0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67,
	0x12, 0x16, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x50, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x70, 0x62, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x76, 0x31, 0x2f, 0x6f,
	0x6c, 0x69, 0x76, 0x65, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x70, 0x69, 0x6e,
	0x67, 0x12, 0x6a, 0x0a, 0x08, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x12, 0x1a, 0x2e,
	0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d,
	0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x67, 0x61, 0x74, 0x65,
	0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x25, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1f, 0x22, 0x1a,
	0x2f, 0x76, 0x31, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61,
	0x79, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x3a, 0x01, 0x2a, 0x32, 0x4b, 0x0a,
	0x0b, 0x54, 0x65, 0x73, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3c, 0x0a, 0x05,
	0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x12, 0x17, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70,
	0x62, 0x2e, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18,
	0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x2e, 0x48, 0x65, 0x6c, 0x6c, 0x6f,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x33, 0x5a, 0x31, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2d, 0x69,
	0x6f, 0x2f, 0x6f, 0x6c, 0x69, 0x76, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x61, 0x74, 0x65,
	0x77, 0x61, 0x79, 0x70, 0x62, 0x3b, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescOnce sync.Once
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescData = file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDesc
)

func file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescGZIP() []byte {
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescOnce.Do(func() {
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescData)
	})
	return file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDescData
}

var file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_goTypes = []interface{}{
	(*PingRequest)(nil),          // 0: gatewaypb.PingRequest
	(*PingResponse)(nil),         // 1: gatewaypb.PingResponse
	(*TransmitRequest)(nil),      // 2: gatewaypb.TransmitRequest
	(*TransmitResponse)(nil),     // 3: gatewaypb.TransmitResponse
	(*HelloRequest)(nil),         // 4: gatewaypb.HelloRequest
	(*HelloResponse)(nil),        // 5: gatewaypb.HelloResponse
	nil,                          // 6: gatewaypb.TransmitRequest.HeadersEntry
	nil,                          // 7: gatewaypb.TransmitRequest.PropertiesEntry
	nil,                          // 8: gatewaypb.TransmitRequest.DataObjectsEntry
	nil,                          // 9: gatewaypb.TransmitResponse.PropertiesEntry
	nil,                          // 10: gatewaypb.TransmitResponse.DataObjectsEntry
	(*discoverypb.Activity)(nil), // 11: discoverypb.Activity
	(*discoverypb.Box)(nil),      // 12: discoverypb.Box
}
var file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_depIdxs = []int32{
	11, // 0: gatewaypb.TransmitRequest.activity:type_name -> discoverypb.Activity
	6,  // 1: gatewaypb.TransmitRequest.headers:type_name -> gatewaypb.TransmitRequest.HeadersEntry
	7,  // 2: gatewaypb.TransmitRequest.properties:type_name -> gatewaypb.TransmitRequest.PropertiesEntry
	8,  // 3: gatewaypb.TransmitRequest.dataObjects:type_name -> gatewaypb.TransmitRequest.DataObjectsEntry
	9,  // 4: gatewaypb.TransmitResponse.properties:type_name -> gatewaypb.TransmitResponse.PropertiesEntry
	10, // 5: gatewaypb.TransmitResponse.dataObjects:type_name -> gatewaypb.TransmitResponse.DataObjectsEntry
	12, // 6: gatewaypb.TransmitRequest.PropertiesEntry.value:type_name -> discoverypb.Box
	12, // 7: gatewaypb.TransmitRequest.DataObjectsEntry.value:type_name -> discoverypb.Box
	12, // 8: gatewaypb.TransmitResponse.PropertiesEntry.value:type_name -> discoverypb.Box
	12, // 9: gatewaypb.TransmitResponse.DataObjectsEntry.value:type_name -> discoverypb.Box
	0,  // 10: gatewaypb.Gateway.Ping:input_type -> gatewaypb.PingRequest
	2,  // 11: gatewaypb.Gateway.Transmit:input_type -> gatewaypb.TransmitRequest
	4,  // 12: gatewaypb.TestService.Hello:input_type -> gatewaypb.HelloRequest
	1,  // 13: gatewaypb.Gateway.Ping:output_type -> gatewaypb.PingResponse
	3,  // 14: gatewaypb.Gateway.Transmit:output_type -> gatewaypb.TransmitResponse
	5,  // 15: gatewaypb.TestService.Hello:output_type -> gatewaypb.HelloResponse
	13, // [13:16] is the sub-list for method output_type
	10, // [10:13] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_init() }
func file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_init() {
	if File_github_com_olive_io_olive_api_gatewaypb_rpc_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PingRequest); i {
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
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PingResponse); i {
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
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransmitRequest); i {
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
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransmitResponse); i {
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
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HelloRequest); i {
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
		file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HelloResponse); i {
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
			RawDescriptor: file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_goTypes,
		DependencyIndexes: file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_depIdxs,
		MessageInfos:      file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_msgTypes,
	}.Build()
	File_github_com_olive_io_olive_api_gatewaypb_rpc_proto = out.File
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_rawDesc = nil
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_goTypes = nil
	file_github_com_olive_io_olive_api_gatewaypb_rpc_proto_depIdxs = nil
}
