package waf

import (
	"context"

	pb "github.com/wallarm/gotestwaf/internal/payload/encoder/grpc"
)

type grpcServer struct {
	pb.UnimplementedServiceFooBarServer
}

func (s *grpcServer) Foo(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	return &pb.Response{Value: "OK"}, nil
}
