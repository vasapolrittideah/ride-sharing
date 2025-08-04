package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcHandler struct {
	service   domain.TripService
	publisher *events.TripEventPublisher

	pb.UnimplementedTripServiceServer
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *grpcHandler {
	handler := &grpcHandler{
		service:   service,
		publisher: publisher,
	}

	pb.RegisterTripServiceServer(server, handler)

	return handler
}

func (h *grpcHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create trip: %v", err)
	}

	if err := h.publisher.PublishTripCreated(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish trip created event: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
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

	route, err := h.service.GetRoute(ctx, pickup, destination)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)
	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, req.GetUserID(), route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate ride fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}
