package grpcsvc

import (
	"context"

	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
)

type healthService struct {
	commonv1.UnimplementedHealthCheckServiceServer
}

func (healthService) Ping(_ context.Context, _ *commonv1.PingRequest) (*commonv1.PingResponse, error) {
	return &commonv1.PingResponse{Message: "pong"}, nil
}
