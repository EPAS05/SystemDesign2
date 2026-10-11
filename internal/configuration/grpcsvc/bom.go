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
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (s *bomService) GetBom(ctx context.Context, req *configurationv1.GetBomRequest) (*configurationv1.GetBomResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	bom, err := s.svc.GetBom(ctx, req.Id)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.GetBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) ListBoms(ctx context.Context, req *configurationv1.ListBomsRequest) (*configurationv1.ListBomsResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	boms, err := s.svc.ListBoms(ctx, req.ProductId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*configurationv1.Bom, 0, len(boms))
	for _, b := range boms {
		out = append(out, toBomProto(b))
	}
	return &configurationv1.ListBomsResponse{Boms: out}, nil
}

func (s *bomService) CreateBom(ctx context.Context, req *configurationv1.CreateBomRequest) (*configurationv1.CreateBomResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if !req.IsDefault && req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for non-default bom")
	}
	bom, err := s.svc.CreateBom(ctx, req.ProductId, req.Name, req.IsDefault)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.CreateBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) DeleteBom(ctx context.Context, req *configurationv1.DeleteBomRequest) (*configurationv1.DeleteBomResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	err := s.svc.DeleteBom(ctx, req.Id)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.DeleteBomResponse{}, nil
}

func (s *bomService) ReplaceBomRows(ctx context.Context, req *configurationv1.ReplaceBomRowsRequest) (*configurationv1.ReplaceBomRowsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
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
	bom, err := s.svc.ReplaceBomRows(ctx, req.BomId, inputs)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &configurationv1.ReplaceBomRowsResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) AddBomRow(ctx context.Context, req *configurationv1.AddBomRowRequest) (*configurationv1.AddBomRowResponse, error) {
	if req.BomId == "" || req.ComponentId == "" || !positiveQuantity(req.Quantity) {
		return nil, status.Error(codes.InvalidArgument, "bom_id, component_id and quantity > 0 required")
	}
	row, err := s.svc.AddRow(ctx, req.BomId, req.ComponentId, req.Quantity)
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

func (s *bomService) ListBomRevisions(ctx context.Context, req *configurationv1.ListBomRevisionsRequest) (*configurationv1.ListBomRevisionsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	revs, err := s.svc.ListBomRevisions(ctx, req.BomId)
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "bom not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*configurationv1.BomRevision, 0, len(revs))
	for _, rv := range revs {
		out = append(out, &configurationv1.BomRevision{
			BomId:     rv.BomID,
			Revision:  int32(rv.Revision),
			ChangedAt: timestamppb.New(rv.ChangedAt),
		})
	}
	return &configurationv1.ListBomRevisionsResponse{Revisions: out}, nil
}

func (s *bomService) GetBomRevision(ctx context.Context, req *configurationv1.GetBomRevisionRequest) (*configurationv1.GetBomRevisionResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	if req.Revision < 1 {
		return nil, status.Error(codes.InvalidArgument, "revision is required")
	}
	rows, changedAt, err := s.svc.GetBomRevision(ctx, req.BomId, int(req.Revision))
	if errors.Is(err, configuration.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "revision not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*configurationv1.BomRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRowProto(r))
	}
	return &configurationv1.GetBomRevisionResponse{
		Revision:  req.Revision,
		ChangedAt: timestamppb.New(changedAt),
		Rows:      out,
	}, nil
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
