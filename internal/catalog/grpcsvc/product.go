package grpcsvc

import (
	"context"
	"errors"

	catalogv1 "github.com/EPAS05/catalog-service/gen/go/catalog/v1"
	"github.com/EPAS05/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productService struct {
	catalogv1.UnimplementedProductServiceServer
	svc *catalog.Service
}

func NewProductService(svc *catalog.Service) *productService {
	return &productService{svc: svc}
}

func (s *productService) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}
	p, err := s.svc.CreateProduct(ctx, req.NodeId, req.Name)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.CreateProductResponse{Product: toProductProto(p)}, nil
}

func (s *productService) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	p, err := s.svc.GetProduct(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.GetProductResponse{Product: toProductProto(p)}, nil
}

func (s *productService) ListProducts(ctx context.Context, req *catalogv1.ListProductsRequest) (*catalogv1.ListProductsResponse, error) {
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}
	products, err := s.svc.ListProductsByNode(ctx, req.NodeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*catalogv1.Product, 0, len(products))
	for _, p := range products {
		out = append(out, toProductProto(p))
	}
	return &catalogv1.ListProductsResponse{Products: out}, nil
}

func (s *productService) UpdateProduct(ctx context.Context, req *catalogv1.UpdateProductRequest) (*catalogv1.UpdateProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	p, err := s.svc.UpdateProduct(ctx, req.Id, req.Name)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.UpdateProductResponse{Product: toProductProto(p)}, nil
}

func (s *productService) DeleteProduct(ctx context.Context, req *catalogv1.DeleteProductRequest) (*catalogv1.DeleteProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	err := s.svc.DeleteProduct(ctx, req.Id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &catalogv1.DeleteProductResponse{}, nil
}

func toProductProto(p *catalog.Product) *catalogv1.Product {
	return &catalogv1.Product{
		Id:     p.ID,
		NodeId: p.NodeID,
		Name:   p.Name,
	}
}
