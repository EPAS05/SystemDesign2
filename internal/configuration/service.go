package configuration

import "context"

type BomRowInput struct {
	ComponentID string
	Quantity    string
}

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	return s.repo.GetDefaultBom(ctx, productID)
}

func (s *Service) ReplaceDefaultBom(ctx context.Context, productID string, inputs []BomRowInput) (*Bom, error) {
	return s.repo.ReplaceDefaultBom(ctx, productID, inputs)
}

func (s *Service) AddRow(ctx context.Context, productID, componentID, quantity string) (*BomRow, error) {
	return s.repo.AddRow(ctx, productID, componentID, quantity)
}

func (s *Service) UpdateRow(ctx context.Context, id, componentID, quantity string) (*BomRow, error) {
	return s.repo.UpdateRow(ctx, id, componentID, quantity)
}

func (s *Service) DeleteRow(ctx context.Context, id string) error {
	return s.repo.DeleteRow(ctx, id)
}
