package main

import (
	"context"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcHandler struct {
	Service *Service

	pb.UnimplementedDriverServiceServer
}

func NewGrpcHandler(server *grpc.Server, service *Service) {
	handler := &grpcHandler{
		Service: service,
	}
	pb.RegisterDriverServiceServer(server, handler)
}

func (h *grpcHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driver, err := h.Service.RegisterDriver(req.GetDriverID(), req.GetPackageSlug())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register driver")
	}

	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}

func (h *grpcHandler) UnRegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	h.Service.UnregisterDriver(req.GetDriverID())

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}
