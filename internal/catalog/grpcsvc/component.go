package grpcsvc

import (
	"context"
	"errors"

	catalogv1 "github.com/EPAS05/catalog-service/gen/go/catalog/v1"
	"github.com/EPAS05/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type componentService struct {
	catalogv1.UnimplementedComponentServiceServer
	svc *catalog.Service
}

func NewComponentService(svc *catalog.Service) *componentService {
	return &componentService{svc: svc}
}

func (s *componentService) CreateComponent(ctx context.Context, req *catalogv1.CreateComponentRequest) (*catalogv1.CreateComponentResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}
	c, err := s.svc.CreateComponent(ctx, req.NodeId, req.Name)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.CreateComponentResponse{Component: toComponentProto(c)}, nil
}

func (s *componentService) GetComponent(ctx context.Context, req *catalogv1.GetComponentRequest) (*catalogv1.GetComponentResponse, error) {
	c, err := s.svc.GetComponent(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "component not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.GetComponentResponse{Component: toComponentProto(c)}, nil
}

func (s *componentService) ListComponents(ctx context.Context, req *catalogv1.ListComponentsRequest) (*catalogv1.ListComponentsResponse, error) {
	components, err := s.svc.ListComponentsByNode(ctx, req.NodeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*catalogv1.Component, 0, len(components))
	for _, c := range components {
		out = append(out, toComponentProto(c))
	}
	return &catalogv1.ListComponentsResponse{Components: out}, nil
}

func (s *componentService) UpdateComponent(ctx context.Context, req *catalogv1.UpdateComponentRequest) (*catalogv1.UpdateComponentResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	c, err := s.svc.UpdateComponent(ctx, req.Id, req.Name)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "component not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.UpdateComponentResponse{Component: toComponentProto(c)}, nil
}

func (s *componentService) DeleteComponent(ctx context.Context, req *catalogv1.DeleteComponentRequest) (*catalogv1.DeleteComponentResponse, error) {
	err := s.svc.DeleteComponent(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "component not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.DeleteComponentResponse{}, nil
}

func toComponentProto(c *catalog.Component) *catalogv1.Component {
	return &catalogv1.Component{
		Id:     c.ID,
		NodeId: c.NodeID,
		Name:   c.Name,
	}
}
