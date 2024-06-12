// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: webchela.proto

package webchela

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

const (
	Server_GetLoad_FullMethodName = "/webchela.Server/GetLoad"
	Server_RunTask_FullMethodName = "/webchela.Server/RunTask"
)

// ServerClient is the client API for Server service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServerClient interface {
	GetLoad(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Load, error)
	RunTask(ctx context.Context, in *Task, opts ...grpc.CallOption) (Server_RunTaskClient, error)
}

type serverClient struct {
	cc grpc.ClientConnInterface
}

func NewServerClient(cc grpc.ClientConnInterface) ServerClient {
	return &serverClient{cc}
}

func (c *serverClient) GetLoad(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Load, error) {
	out := new(Load)
	err := c.cc.Invoke(ctx, Server_GetLoad_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) RunTask(ctx context.Context, in *Task, opts ...grpc.CallOption) (Server_RunTaskClient, error) {
	stream, err := c.cc.NewStream(ctx, &Server_ServiceDesc.Streams[0], Server_RunTask_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &serverRunTaskClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Server_RunTaskClient interface {
	Recv() (*Chunk, error)
	grpc.ClientStream
}

type serverRunTaskClient struct {
	grpc.ClientStream
}

func (x *serverRunTaskClient) Recv() (*Chunk, error) {
	m := new(Chunk)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ServerServer is the server API for Server service.
// All implementations must embed UnimplementedServerServer
// for forward compatibility
type ServerServer interface {
	GetLoad(context.Context, *Empty) (*Load, error)
	RunTask(*Task, Server_RunTaskServer) error
	mustEmbedUnimplementedServerServer()
}

// UnimplementedServerServer must be embedded to have forward compatible implementations.
type UnimplementedServerServer struct {
}

func (UnimplementedServerServer) GetLoad(context.Context, *Empty) (*Load, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLoad not implemented")
}
func (UnimplementedServerServer) RunTask(*Task, Server_RunTaskServer) error {
	return status.Errorf(codes.Unimplemented, "method RunTask not implemented")
}
func (UnimplementedServerServer) mustEmbedUnimplementedServerServer() {}

// UnsafeServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServerServer will
// result in compilation errors.
type UnsafeServerServer interface {
	mustEmbedUnimplementedServerServer()
}

func RegisterServerServer(s grpc.ServiceRegistrar, srv ServerServer) {
	s.RegisterService(&Server_ServiceDesc, srv)
}

func _Server_GetLoad_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).GetLoad(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_GetLoad_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).GetLoad(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_RunTask_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Task)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ServerServer).RunTask(m, &serverRunTaskServer{stream})
}

type Server_RunTaskServer interface {
	Send(*Chunk) error
	grpc.ServerStream
}

type serverRunTaskServer struct {
	grpc.ServerStream
}

func (x *serverRunTaskServer) Send(m *Chunk) error {
	return x.ServerStream.SendMsg(m)
}

// Server_ServiceDesc is the grpc.ServiceDesc for Server service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Server_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "webchela.Server",
	HandlerType: (*ServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLoad",
			Handler:    _Server_GetLoad_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RunTask",
			Handler:       _Server_RunTask_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "webchela.proto",
}
