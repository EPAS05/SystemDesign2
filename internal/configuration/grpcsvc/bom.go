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

func validQuantity(q string, excluded bool) bool {
	f, err := strconv.ParseFloat(q, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return false
	}
	if excluded {
		return f == 0
	}
	return f > 0
}

func rowReference(refBomID string, refRev int32, excluded bool) (*string, *int, error) {
	if excluded && (refBomID != "" || refRev != 0) {
		return nil, nil, status.Error(codes.InvalidArgument, "excluded row cannot reference another bom")
	}
	if (refBomID == "") != (refRev == 0) {
		return nil, nil, status.Error(codes.InvalidArgument, "reference_bom_id and reference_revision must be set together")
	}
	if refBomID == "" {
		return nil, nil, nil
	}
	if refRev < 1 {
		return nil, nil, status.Error(codes.InvalidArgument, "reference_revision must be positive")
	}
	id := refBomID
	revision := int(refRev)
	return &id, &revision, nil
}

func argumentError(err error) error {
	switch {
	case errors.Is(err, configuration.ErrSelfReference):
		return status.Error(codes.InvalidArgument, "product cannot contain itself")
	case errors.Is(err, configuration.ErrBomCycle):
		return status.Error(codes.InvalidArgument, "bom would create a cycle")
	case errors.Is(err, configuration.ErrBaseOnlyDefault):
		return status.Error(codes.InvalidArgument, "is_base is allowed only for default bom")
	case errors.Is(err, configuration.ErrExcludedOnlyModification):
		return status.Error(codes.InvalidArgument, "is_excluded is allowed only for modification")
	case errors.Is(err, configuration.ErrReferencePair):
		return status.Error(codes.InvalidArgument, "reference bom and revision must be set together")
	case errors.Is(err, configuration.ErrReferenceRevision):
		return status.Error(codes.InvalidArgument, "reference revision must be positive")
	case errors.Is(err, configuration.ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, "quantity must be positive")
	case errors.Is(err, configuration.ErrInvalidExcludedQuantity):
		return status.Error(codes.InvalidArgument, "excluded row must have zero quantity")
	case errors.Is(err, configuration.ErrInvalidRow):
		return status.Error(codes.InvalidArgument, "invalid bom row")
	case errors.Is(err, configuration.ErrRowExists):
		return status.Error(codes.AlreadyExists, "bom row for product already exists")
	case errors.Is(err, configuration.ErrBomNameExists):
		return status.Error(codes.AlreadyExists, "bom name already exists")
	case errors.Is(err, configuration.ErrBomInUse):
		return status.Error(codes.FailedPrecondition, "bom is in use")
	case errors.Is(err, configuration.ErrReferenceNotFound):
		return status.Error(codes.FailedPrecondition, "referenced bom revision not found")
	case errors.Is(err, configuration.ErrDictionaryCodeExists):
		return status.Error(codes.AlreadyExists, "dictionary code already exists")
	case errors.Is(err, configuration.ErrDictionaryInUse):
		return status.Error(codes.FailedPrecondition, "dictionary entry is in use")
	case errors.Is(err, configuration.ErrRouteSeqExists):
		return status.Error(codes.AlreadyExists, "route row with seq already exists")
	case errors.Is(err, configuration.ErrRouteRowInherited):
		return status.Error(codes.FailedPrecondition, "route row is inherited")
	case errors.Is(err, configuration.ErrInvalidRouteRow):
		return status.Error(codes.InvalidArgument, "invalid route row")
	case errors.Is(err, configuration.ErrRouteCycle):
		return status.Error(codes.FailedPrecondition, "route contains a cycle")
	case errors.Is(err, configuration.ErrManualPriceWithRoutes):
		return status.Error(codes.FailedPrecondition, "manual price is allowed only for bom without route rows")
	case errors.Is(err, configuration.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, "invalid bom price")
	case errors.Is(err, configuration.ErrInvalidDictionary):
		return status.Error(codes.InvalidArgument, "invalid dictionary entry")
	}
	return nil
}

func grpcError(err error, notFound string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, configuration.ErrNotFound) {
		return status.Error(codes.NotFound, notFound)
	}
	if mapped := argumentError(err); mapped != nil {
		return mapped
	}
	return status.Error(codes.Internal, err.Error())
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
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.GetBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) ListBoms(ctx context.Context, req *configurationv1.ListBomsRequest) (*configurationv1.ListBomsResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	boms, err := s.svc.ListBoms(ctx, req.ProductId)
	if err := grpcError(err, "product not found"); err != nil {
		return nil, err
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
	if err := grpcError(err, "product not found"); err != nil {
		return nil, err
	}
	return &configurationv1.CreateBomResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) DeleteBom(ctx context.Context, req *configurationv1.DeleteBomRequest) (*configurationv1.DeleteBomResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	err := s.svc.DeleteBom(ctx, req.Id)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.DeleteBomResponse{}, nil
}

func (s *bomService) ReplaceBomRows(ctx context.Context, req *configurationv1.ReplaceBomRowsRequest) (*configurationv1.ReplaceBomRowsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	inputs := make([]configuration.BomRowInput, 0, len(req.Rows))
	for _, r := range req.Rows {
		if r.ProductId == "" || !validQuantity(r.Quantity, r.IsExcluded) {
			return nil, status.Error(codes.InvalidArgument, "each row requires product_id and a valid quantity")
		}
		refBomID, refRev, err := rowReference(r.ReferenceBomId, r.ReferenceRevision, r.IsExcluded)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, configuration.BomRowInput{
			ProductID:         r.ProductId,
			Quantity:          r.Quantity,
			IsBase:            r.IsBase,
			IsExcluded:        r.IsExcluded,
			ReferenceBomID:    refBomID,
			ReferenceRevision: refRev,
		})
	}
	bom, err := s.svc.ReplaceBomRows(ctx, req.BomId, inputs)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.ReplaceBomRowsResponse{Bom: toBomProto(bom)}, nil
}

func (s *bomService) AddBomRow(ctx context.Context, req *configurationv1.AddBomRowRequest) (*configurationv1.AddBomRowResponse, error) {
	if req.BomId == "" || req.ProductId == "" || !validQuantity(req.Quantity, req.IsExcluded) {
		return nil, status.Error(codes.InvalidArgument, "bom_id, product_id and a valid quantity are required")
	}
	refBomID, refRev, err := rowReference(req.ReferenceBomId, req.ReferenceRevision, req.IsExcluded)
	if err != nil {
		return nil, err
	}
	row, err := s.svc.AddRow(ctx, req.BomId, req.ProductId, req.Quantity, req.IsBase, refBomID, refRev, req.IsExcluded)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.AddBomRowResponse{Row: toRowProto(row)}, nil
}

func (s *bomService) UpdateBomRow(ctx context.Context, req *configurationv1.UpdateBomRowRequest) (*configurationv1.UpdateBomRowResponse, error) {
	if req.Id == "" || req.ProductId == "" || !validQuantity(req.Quantity, req.IsExcluded) {
		return nil, status.Error(codes.InvalidArgument, "id, product_id and a valid quantity are required")
	}
	refBomID, refRev, err := rowReference(req.ReferenceBomId, req.ReferenceRevision, req.IsExcluded)
	if err != nil {
		return nil, err
	}
	row, err := s.svc.UpdateRow(ctx, req.Id, req.ProductId, req.Quantity, req.IsBase, refBomID, refRev, req.IsExcluded)
	if err := grpcError(err, "bom row not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateBomRowResponse{Row: toRowProto(row)}, nil
}

func (s *bomService) DeleteBomRow(ctx context.Context, req *configurationv1.DeleteBomRowRequest) (*configurationv1.DeleteBomRowResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	err := s.svc.DeleteRow(ctx, req.Id)
	if err := grpcError(err, "bom row not found"); err != nil {
		return nil, err
	}
	return &configurationv1.DeleteBomRowResponse{}, nil
}

func (s *bomService) ListBomRevisions(ctx context.Context, req *configurationv1.ListBomRevisionsRequest) (*configurationv1.ListBomRevisionsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	revs, err := s.svc.ListBomRevisions(ctx, req.BomId)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
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
	if req.BomId == "" || req.Revision < 1 {
		return nil, status.Error(codes.InvalidArgument, "bom_id and revision are required")
	}
	rows, routeRows, changedAt, err := s.svc.GetBomRevision(ctx, req.BomId, int(req.Revision))
	if err := grpcError(err, "revision not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.BomRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRowProto(r))
	}
	routeOut := make([]*configurationv1.RouteRow, 0, len(routeRows))
	for _, r := range routeRows {
		routeOut = append(routeOut, toRouteRowProto(r))
	}
	return &configurationv1.GetBomRevisionResponse{
		Revision:  req.Revision,
		ChangedAt: timestamppb.New(changedAt),
		Rows:      out,
		RouteRows: routeOut,
	}, nil
}

func (s *bomService) ExpandBom(ctx context.Context, req *configurationv1.ExpandBomRequest) (*configurationv1.ExpandBomResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	rows, err := s.svc.ExpandBom(ctx, req.BomId)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.BomRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRowProto(r))
	}
	return &configurationv1.ExpandBomResponse{Rows: out}, nil
}

func (s *bomService) UpdateBomPrice(ctx context.Context, req *configurationv1.UpdateBomPriceRequest) (*configurationv1.UpdateBomPriceResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	bom, err := s.svc.UpdateBomPrice(ctx, req.Id, req.Price, req.PriceManual)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateBomPriceResponse{Bom: toBomProto(bom)}, nil
}

func toBomProto(b *configuration.Bom) *configurationv1.Bom {
	out := &configurationv1.Bom{
		Id:          b.ID,
		ProductId:   b.ProductID,
		IsDefault:   b.IsDefault,
		Price:       b.Price,
		PriceManual: b.PriceManual,
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
	out := &configurationv1.BomRow{
		Id:         r.ID,
		ProductId:  r.ProductID,
		Quantity:   r.Quantity,
		IsBase:     r.IsBase,
		IsExcluded: r.IsExcluded,
	}
	if r.ReferenceBomID != nil {
		out.ReferenceBomId = *r.ReferenceBomID
	}
	if r.ReferenceRevision != nil {
		out.ReferenceRevision = int32(*r.ReferenceRevision)
	}
	return out
}
