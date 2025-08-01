package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"
)

type RideFareModel struct {
	ID                primitive.ObjectID
	UserID            string
	PackageSlug       string
	TotalPriceInCents float64
	Route             *tripTypes.OsrmApiResponse
}

func (r *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
	}
}

func ToRideFaresProto(fares []*RideFareModel) []*pb.RideFare {
	rideFares := make([]*pb.RideFare, len(fares))
	for i, fare := range fares {
		rideFares[i] = fare.ToProto()
	}

	return rideFares
}
