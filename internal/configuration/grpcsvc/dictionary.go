package grpcsvc

import (
	"context"

	configurationv1 "github.com/EPAS05/catalog-service/gen/go/configuration/v1"
	"github.com/EPAS05/catalog-service/internal/configuration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type operationClassService struct {
	configurationv1.UnimplementedOperationClassServiceServer
	svc *configuration.Service
}

func NewOperationClassService(svc *configuration.Service) *operationClassService {
	return &operationClassService{svc: svc}
}

func (s *operationClassService) ListOperationClasses(ctx context.Context, _ *configurationv1.ListOperationClassesRequest) (*configurationv1.ListOperationClassesResponse, error) {
	items, err := s.svc.ListOperationClasses(ctx)
	if err := grpcError(err, "operation classes not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.OperationClass, 0, len(items))
	for _, item := range items {
		out = append(out, toOperationClassProto(item))
	}
	return &configurationv1.ListOperationClassesResponse{OperationClasses: out}, nil
}

func (s *operationClassService) GetOperationClass(ctx context.Context, req *configurationv1.GetOperationClassRequest) (*configurationv1.GetOperationClassResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.GetOperationClass(ctx, req.Id)
	if err := grpcError(err, "operation class not found"); err != nil {
		return nil, err
	}
	return &configurationv1.GetOperationClassResponse{OperationClass: toOperationClassProto(item)}, nil
}

func (s *operationClassService) CreateOperationClass(ctx context.Context, req *configurationv1.CreateOperationClassRequest) (*configurationv1.CreateOperationClassResponse, error) {
	item, err := s.svc.CreateOperationClass(ctx, req.Code, req.Name, req.HourlyRate)
	if err := grpcError(err, "operation class not found"); err != nil {
		return nil, err
	}
	return &configurationv1.CreateOperationClassResponse{OperationClass: toOperationClassProto(item)}, nil
}

func (s *operationClassService) UpdateOperationClass(ctx context.Context, req *configurationv1.UpdateOperationClassRequest) (*configurationv1.UpdateOperationClassResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.UpdateOperationClass(ctx, req.Id, req.Code, req.Name, req.HourlyRate)
	if err := grpcError(err, "operation class not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateOperationClassResponse{OperationClass: toOperationClassProto(item)}, nil
}

func (s *operationClassService) DeleteOperationClass(ctx context.Context, req *configurationv1.DeleteOperationClassRequest) (*configurationv1.DeleteOperationClassResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.svc.DeleteOperationClass(ctx, req.Id); err != nil {
		return nil, grpcError(err, "operation class not found")
	}
	return &configurationv1.DeleteOperationClassResponse{}, nil
}

func toOperationClassProto(item *configuration.OperationClass) *configurationv1.OperationClass {
	return &configurationv1.OperationClass{
		Id:         item.ID,
		Code:       item.Code,
		Name:       item.Name,
		HourlyRate: item.HourlyRate,
	}
}

type equipmentService struct {
	configurationv1.UnimplementedEquipmentServiceServer
	svc *configuration.Service
}

func NewEquipmentService(svc *configuration.Service) *equipmentService {
	return &equipmentService{svc: svc}
}

func (s *equipmentService) ListEquipment(ctx context.Context, _ *configurationv1.ListEquipmentRequest) (*configurationv1.ListEquipmentResponse, error) {
	items, err := s.svc.ListEquipment(ctx)
	if err := grpcError(err, "equipment not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.Equipment, 0, len(items))
	for _, item := range items {
		out = append(out, toEquipmentProto(item))
	}
	return &configurationv1.ListEquipmentResponse{Equipment: out}, nil
}

func (s *equipmentService) GetEquipment(ctx context.Context, req *configurationv1.GetEquipmentRequest) (*configurationv1.GetEquipmentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.GetEquipment(ctx, req.Id)
	if err := grpcError(err, "equipment not found"); err != nil {
		return nil, err
	}
	return &configurationv1.GetEquipmentResponse{Equipment: toEquipmentProto(item)}, nil
}

func (s *equipmentService) CreateEquipment(ctx context.Context, req *configurationv1.CreateEquipmentRequest) (*configurationv1.CreateEquipmentResponse, error) {
	item, err := s.svc.CreateEquipment(ctx, req.Code, req.Name)
	if err := grpcError(err, "equipment not found"); err != nil {
		return nil, err
	}
	return &configurationv1.CreateEquipmentResponse{Equipment: toEquipmentProto(item)}, nil
}

func (s *equipmentService) UpdateEquipment(ctx context.Context, req *configurationv1.UpdateEquipmentRequest) (*configurationv1.UpdateEquipmentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.UpdateEquipment(ctx, req.Id, req.Code, req.Name)
	if err := grpcError(err, "equipment not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateEquipmentResponse{Equipment: toEquipmentProto(item)}, nil
}

func (s *equipmentService) DeleteEquipment(ctx context.Context, req *configurationv1.DeleteEquipmentRequest) (*configurationv1.DeleteEquipmentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.svc.DeleteEquipment(ctx, req.Id); err != nil {
		return nil, grpcError(err, "equipment not found")
	}
	return &configurationv1.DeleteEquipmentResponse{}, nil
}

func toEquipmentProto(item *configuration.Equipment) *configurationv1.Equipment {
	return &configurationv1.Equipment{Id: item.ID, Code: item.Code, Name: item.Name}
}

type toolService struct {
	configurationv1.UnimplementedToolServiceServer
	svc *configuration.Service
}

func NewToolService(svc *configuration.Service) *toolService {
	return &toolService{svc: svc}
}

func (s *toolService) ListTools(ctx context.Context, _ *configurationv1.ListToolsRequest) (*configurationv1.ListToolsResponse, error) {
	items, err := s.svc.ListTools(ctx)
	if err := grpcError(err, "tools not found"); err != nil {
		return nil, err
	}
	out := make([]*configurationv1.Tool, 0, len(items))
	for _, item := range items {
		out = append(out, toToolProto(item))
	}
	return &configurationv1.ListToolsResponse{Tools: out}, nil
}

func (s *toolService) GetTool(ctx context.Context, req *configurationv1.GetToolRequest) (*configurationv1.GetToolResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.GetTool(ctx, req.Id)
	if err := grpcError(err, "tool not found"); err != nil {
		return nil, err
	}
	return &configurationv1.GetToolResponse{Tool: toToolProto(item)}, nil
}

func (s *toolService) CreateTool(ctx context.Context, req *configurationv1.CreateToolRequest) (*configurationv1.CreateToolResponse, error) {
	item, err := s.svc.CreateTool(ctx, req.Code, req.Name)
	if err := grpcError(err, "tool not found"); err != nil {
		return nil, err
	}
	return &configurationv1.CreateToolResponse{Tool: toToolProto(item)}, nil
}

func (s *toolService) UpdateTool(ctx context.Context, req *configurationv1.UpdateToolRequest) (*configurationv1.UpdateToolResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.svc.UpdateTool(ctx, req.Id, req.Code, req.Name)
	if err := grpcError(err, "tool not found"); err != nil {
		return nil, err
	}
	return &configurationv1.UpdateToolResponse{Tool: toToolProto(item)}, nil
}

func (s *toolService) DeleteTool(ctx context.Context, req *configurationv1.DeleteToolRequest) (*configurationv1.DeleteToolResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.svc.DeleteTool(ctx, req.Id); err != nil {
		return nil, grpcError(err, "tool not found")
	}
	return &configurationv1.DeleteToolResponse{}, nil
}

func toToolProto(item *configuration.Tool) *configurationv1.Tool {
	return &configurationv1.Tool{Id: item.ID, Code: item.Code, Name: item.Name}
}
