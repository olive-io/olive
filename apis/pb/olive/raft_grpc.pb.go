// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.3
// source: github.com/olive-io/olive/apis/pb/olive/raft.proto

package olivepb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RunnerRPCClient is the client API for RunnerRPC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RunnerRPCClient interface {
	GetDefinitionArchive(ctx context.Context, in *GetDefinitionArchiveRequest, opts ...grpc.CallOption) (*GetDefinitionArchiveResponse, error)
	GetProcessStat(ctx context.Context, in *GetProcessStatRequest, opts ...grpc.CallOption) (*GetProcessStatResponse, error)
}

type runnerRPCClient struct {
	cc grpc.ClientConnInterface
}

func NewRunnerRPCClient(cc grpc.ClientConnInterface) RunnerRPCClient {
	return &runnerRPCClient{cc}
}

func (c *runnerRPCClient) GetDefinitionArchive(ctx context.Context, in *GetDefinitionArchiveRequest, opts ...grpc.CallOption) (*GetDefinitionArchiveResponse, error) {
	out := new(GetDefinitionArchiveResponse)
	err := c.cc.Invoke(ctx, "/olivepb.RunnerRPC/GetDefinitionArchive", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runnerRPCClient) GetProcessStat(ctx context.Context, in *GetProcessStatRequest, opts ...grpc.CallOption) (*GetProcessStatResponse, error) {
	out := new(GetProcessStatResponse)
	err := c.cc.Invoke(ctx, "/olivepb.RunnerRPC/GetProcessStat", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RunnerRPCServer is the server API for RunnerRPC service.
// All implementations must embed UnimplementedRunnerRPCServer
// for forward compatibility
type RunnerRPCServer interface {
	GetDefinitionArchive(context.Context, *GetDefinitionArchiveRequest) (*GetDefinitionArchiveResponse, error)
	GetProcessStat(context.Context, *GetProcessStatRequest) (*GetProcessStatResponse, error)
	mustEmbedUnimplementedRunnerRPCServer()
}

// UnimplementedRunnerRPCServer must be embedded to have forward compatible implementations.
type UnimplementedRunnerRPCServer struct {
}

func (UnimplementedRunnerRPCServer) GetDefinitionArchive(context.Context, *GetDefinitionArchiveRequest) (*GetDefinitionArchiveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDefinitionArchive not implemented")
}
func (UnimplementedRunnerRPCServer) GetProcessStat(context.Context, *GetProcessStatRequest) (*GetProcessStatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProcessStat not implemented")
}
func (UnimplementedRunnerRPCServer) mustEmbedUnimplementedRunnerRPCServer() {}

// UnsafeRunnerRPCServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RunnerRPCServer will
// result in compilation errors.
type UnsafeRunnerRPCServer interface {
	mustEmbedUnimplementedRunnerRPCServer()
}

func RegisterRunnerRPCServer(s grpc.ServiceRegistrar, srv RunnerRPCServer) {
	s.RegisterService(&RunnerRPC_ServiceDesc, srv)
}

func _RunnerRPC_GetDefinitionArchive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDefinitionArchiveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunnerRPCServer).GetDefinitionArchive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/olivepb.RunnerRPC/GetDefinitionArchive",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunnerRPCServer).GetDefinitionArchive(ctx, req.(*GetDefinitionArchiveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunnerRPC_GetProcessStat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProcessStatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunnerRPCServer).GetProcessStat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/olivepb.RunnerRPC/GetProcessStat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunnerRPCServer).GetProcessStat(ctx, req.(*GetProcessStatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RunnerRPC_ServiceDesc is the grpc.ServiceDesc for RunnerRPC service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RunnerRPC_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "olivepb.RunnerRPC",
	HandlerType: (*RunnerRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetDefinitionArchive",
			Handler:    _RunnerRPC_GetDefinitionArchive_Handler,
		},
		{
			MethodName: "GetProcessStat",
			Handler:    _RunnerRPC_GetProcessStat_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/olive-io/olive/apis/pb/olive/raft.proto",
}