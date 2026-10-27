package grpcsvc

import (
	"context"

	configurationv1 "github.com/EPAS05/catalog-service/gen/go/configuration/v1"
	"github.com/EPAS05/catalog-service/internal/configuration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type routeService struct {
	configurationv1.UnimplementedRouteServiceServer
	svc *configuration.Service
}

func NewRouteService(svc *configuration.Service) *routeService {
	return &routeService{svc: svc}
}

func toRouteRowInput(in *configurationv1.RouteRowInput) configuration.RouteRowInput {
	if in == nil {
		return configuration.RouteRowInput{}
	}
	return configuration.RouteRowInput{
		Seq:               int(in.Seq),
		OperationClassID:  in.OperationClassId,
		EquipmentID:       in.EquipmentId,
		ToolID:            in.ToolId,
		ProductID:         in.ProductId,
		Quantity:          in.Quantity,
		LaborNorm:         in.LaborNorm,
		IsExcluded:        in.IsExcluded,
		ReferenceBomID:    in.ReferenceBomId,
		ReferenceRevision: int(in.ReferenceRevision),
	}
}

func (s *routeService) ListRouteRows(ctx context.Context, req *configurationv1.ListRouteRowsRequest) (*configurationv1.ListRouteRowsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	rows, err := s.svc.ListRouteRows(ctx, req.BomId)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.RouteRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRouteRowProto(row))
	}
	return &configurationv1.ListRouteRowsResponse{Rows: out}, nil
}

func (s *routeService) AddRouteRow(ctx context.Context, req *configurationv1.AddRouteRowRequest) (*configurationv1.AddRouteRowResponse, error) {
	if req.BomId == "" || req.Row == nil {
		return nil, status.Error(codes.InvalidArgument, "bom_id and row are required")
	}
	row, err := s.svc.AddRouteRow(ctx, req.BomId, toRouteRowInput(req.Row))
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	return &configurationv1.AddRouteRowResponse{Row: toRouteRowProto(row)}, nil
}

func (s *routeService) UpdateRouteRow(ctx context.Context, req *configurationv1.UpdateRouteRowRequest) (*configurationv1.UpdateRouteRowResponse, error) {
	if req.BomId == "" || req.RowId == "" || req.Row == nil {
		return nil, status.Error(codes.InvalidArgument, "bom_id, row_id and row are required")
	}
	row, err := s.svc.UpdateRouteRow(ctx, req.BomId, req.RowId, toRouteRowInput(req.Row))
	if err := grpcError(err, "route row not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateRouteRowResponse{Row: toRouteRowProto(row)}, nil
}

func (s *routeService) DeleteRouteRow(ctx context.Context, req *configurationv1.DeleteRouteRowRequest) (*configurationv1.DeleteRouteRowResponse, error) {
	if req.BomId == "" || req.RowId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id and row_id are required")
	}
	if err := s.svc.DeleteRouteRow(ctx, req.BomId, req.RowId); err != nil {
		return nil, grpcError(err, "route row not found")
	}
	return &configurationv1.DeleteRouteRowResponse{}, nil
}

func (s *routeService) ReplaceRouteRows(ctx context.Context, req *configurationv1.ReplaceRouteRowsRequest) (*configurationv1.ReplaceRouteRowsResponse, error) {
	if req.BomId == "" {
		return nil, status.Error(codes.InvalidArgument, "bom_id is required")
	}
	inputs := make([]configuration.RouteRowInput, 0, len(req.Rows))
	for _, row := range req.Rows {
		inputs = append(inputs, toRouteRowInput(row))
	}
	rows, err := s.svc.ReplaceRouteRows(ctx, req.BomId, inputs)
	if err := grpcError(err, "bom not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.RouteRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRouteRowProto(row))
	}
	return &configurationv1.ReplaceRouteRowsResponse{Rows: out}, nil
}

func toRouteRowProto(row *configuration.RouteRow) *configurationv1.RouteRow {
	out := &configurationv1.RouteRow{
		Id:                 row.ID,
		BomId:              row.BomID,
		Seq:                int32(row.Seq),
		OperationClassId:   row.OperationClassID,
		OperationClassName: row.OperationClassName,
		ProductId:          row.ProductID,
		Quantity:           row.Quantity,
		LaborNorm:          row.LaborNorm,
		IsBase:             row.IsBase,
		IsExcluded:         row.IsExcluded,
		Inherited:          row.Inherited,
		SourceBomId:        row.SourceBomID,
	}
	if row.EquipmentID != nil {
		out.EquipmentId = *row.EquipmentID
	}
	if row.EquipmentName != nil {
		out.EquipmentName = *row.EquipmentName
	}
	if row.ToolID != nil {
		out.ToolId = *row.ToolID
	}
	if row.ToolName != nil {
		out.ToolName = *row.ToolName
	}
	if row.ReferenceBomID != nil {
		out.ReferenceBomId = *row.ReferenceBomID
	}
	if row.ReferenceRevision != nil {
		out.ReferenceRevision = int32(*row.ReferenceRevision)
	}
	return out
}
