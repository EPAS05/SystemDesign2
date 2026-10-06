package grpcsvc

import (
	"context"
	"errors"

	catalogv1 "github.com/EPAS05/catalog-service/gen/go/catalog/v1"
	"github.com/EPAS05/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nodeService struct {
	catalogv1.UnimplementedNodeServiceServer
	svc *catalog.Service
}

func NewNodeService(svc *catalog.Service) *nodeService {
	return &nodeService{svc: svc}
}

func (s *nodeService) CreateNode(ctx context.Context, req *catalogv1.CreateNodeRequest) (*catalogv1.CreateNodeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	var parentID *string
	if req.ParentId != "" {
		parentID = &req.ParentId
	}
	n, err := s.svc.CreateNode(ctx, parentID, req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.CreateNodeResponse{Node: toProto(n)}, nil
}

func (s *nodeService) GetNode(ctx context.Context, req *catalogv1.GetNodeRequest) (*catalogv1.GetNodeResponse, error) {
	n, err := s.svc.GetNode(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.GetNodeResponse{Node: toProto(n)}, nil
}

func (s *nodeService) ListChildren(ctx context.Context, req *catalogv1.ListChildrenRequest) (*catalogv1.ListChildrenResponse, error) {
	nodes, err := s.svc.ListChildren(ctx, req.ParentId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*catalogv1.Node, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, toProto(n))
	}
	return &catalogv1.ListChildrenResponse{Nodes: out}, nil
}

func (s *nodeService) UpdateNode(ctx context.Context, req *catalogv1.UpdateNodeRequest) (*catalogv1.UpdateNodeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	n, err := s.svc.UpdateNode(ctx, req.Id, req.Name)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.UpdateNodeResponse{Node: toProto(n)}, nil
}

func (s *nodeService) DeleteNode(ctx context.Context, req *catalogv1.DeleteNodeRequest) (*catalogv1.DeleteNodeResponse, error) {
	err := s.svc.DeleteNode(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.DeleteNodeResponse{}, nil
}

func toProto(n *catalog.Node) *catalogv1.Node {
	out := &catalogv1.Node{
		Id:   n.ID,
		Name: n.Name,
	}
	if n.ParentID != nil {
		out.ParentId = *n.ParentID
	}
	return out
}
