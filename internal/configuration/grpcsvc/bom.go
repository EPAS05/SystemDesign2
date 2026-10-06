package grpcsvc

import (
	"context"
	"errors"
	"math"
	"strconv"

	configurationv1 "github.com/EPAS05/catalog-service/gen/go/configuration/v1"
	"github.com/EPAS05/catalog-service/internal/configuration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func positiveQuantity(q string) bool {
	f, err := strconv.ParseFloat(q, 64)
	return err == nil && f > 0 && !math.IsInf(f, 0)
}

type bomService struct {
	configurationv1.UnimplementedBomServiceServer
	svc *configuration.Service
}

func NewBomService(svc *configuration.Service) *bomService {
	return &bomService{svc: svc}
}

func (s *bomService) GetDefaultBom(ctx context.Context, req *configurationv1.GetDefaultBomRequest) (*configurationv1.GetDefaultBomResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	bom, err := s.svc.GetDefaultBom(ctx, req.ProductId)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.GetDefaultBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) ReplaceDefaultBom(ctx context.Context, req *configurationv1.ReplaceDefaultBomRequest) (*configurationv1.ReplaceDefaultBomResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	inputs := make([]configuration.BomRowInput, 0, len(req.Rows))
	for _, r := range req.Rows {
		if r.ComponentId == "" || !positiveQuantity(r.Quantity) {
			return nil, status.Error(codes.InvalidArgument, "each row requires component_id and quantity > 0")
		}
		inputs = append(inputs, configuration.BomRowInput{
			ComponentID: r.ComponentId,
			Quantity:    r.Quantity,
		})
	}
	bom, err := s.svc.ReplaceDefaultBom(ctx, req.ProductId, inputs)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.ReplaceDefaultBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) AddBomRow(ctx context.Context, req *configurationv1.AddBomRowRequest) (*configurationv1.AddBomRowResponse, error) {
	if req.ProductId == "" || req.ComponentId == "" || !positiveQuantity(req.Quantity) {
		return nil, status.Error(codes.InvalidArgument, "product_id, component_id and quantity > 0 required")
	}
	row, err := s.svc.AddRow(ctx, req.ProductId, req.ComponentId, req.Quantity)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.AddBomRowResponse{Row: toRowProto(row)}, nil
}

func (s *bomService) UpdateBomRow(ctx context.Context, req *configurationv1.UpdateBomRowRequest) (*configurationv1.UpdateBomRowResponse, error) {
	if req.Id == "" || req.ComponentId == "" || !positiveQuantity(req.Quantity) {
		return nil, status.Error(codes.InvalidArgument, "id, component_id and quantity > 0 required")
	}
	row, err := s.svc.UpdateRow(ctx, req.Id, req.ComponentId, req.Quantity)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom row not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.UpdateBomRowResponse{Row: toRowProto(row)}, nil
}

func (s *bomService) DeleteBomRow(ctx context.Context, req *configurationv1.DeleteBomRowRequest) (*configurationv1.DeleteBomRowResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	err := s.svc.DeleteRow(ctx, req.Id)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom row not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.DeleteBomRowResponse{}, nil
}

func toBomProto(b *configuration.Bom) *configurationv1.Bom {
	out := &configurationv1.Bom{
		Id:        b.ID,
		ProductId: b.ProductID,
		IsDefault: b.IsDefault,
	}
	if b.Name != nil {
		out.Name = *b.Name
	}
	out.Rows = make([]*configurationv1.BomRow, 0, len(b.Rows))
	for _, r := range b.Rows {
		out.Rows = append(out.Rows, toRowProto(r))
	}
	return out
}

func toRowProto(r *configuration.BomRow) *configurationv1.BomRow {
	return &configurationv1.BomRow{
		Id:          r.ID,
		ComponentId: r.ComponentID,
		Quantity:    r.Quantity,
	}
}
