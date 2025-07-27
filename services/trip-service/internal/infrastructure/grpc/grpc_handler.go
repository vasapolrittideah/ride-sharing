package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcHandler struct {
	service domain.TripService
	pb.UnimplementedTripServiceServer
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService) *grpcHandler {
	handler := &grpcHandler{
		service: service,
	}

	pb.RegisterTripServiceServer(server, handler)

	return handler
}

func (h *grpcHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickupProto := req.GetStartLocation()
	destinationProto := req.GetEndLocation()

	pickup := &types.Coordinate{
		Latitude:  pickupProto.Latitude,
		Longitude: pickupProto.Longitude,
	}

	destination := &types.Coordinate{
		Latitude:  destinationProto.Latitude,
		Longitude: destinationProto.Longitude,
	}

	trip, err := h.service.GetRoute(ctx, pickup, destination)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     trip.ToProto(),
		RideFares: []*pb.RideFare{},
	}, nil
}
