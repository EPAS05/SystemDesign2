package configuration

import (
	"context"
	"time"
)

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

func (s *Service) GetBom(ctx context.Context, id string) (*Bom, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	return s.repo.GetBom(ctx, id)
}

func (s *Service) ListBoms(ctx context.Context, productID string) ([]*Bom, error) {
	if productID == "" {
		return nil, ErrNotFound
	}
	return s.repo.ListBoms(ctx, productID)
}

func (s *Service) CreateBom(ctx context.Context, productID, name string, isDefault bool) (*Bom, error) {
	if productID == "" {
		return nil, ErrNotFound
	}
	if !isDefault && name == "" {
		return nil, ErrNotFound
	}
	return s.repo.CreateBom(ctx, productID, name, isDefault)
}

func (s *Service) DeleteBom(ctx context.Context, id string) error {
	if id == "" {
		return ErrNotFound
	}
	return s.repo.DeleteBom(ctx, id)
}

func (s *Service) ReplaceBomRows(ctx context.Context, bomID string, inputs []BomRowInput) (*Bom, error) {
	if bomID == "" {
		return nil, ErrNotFound
	}
	return s.repo.ReplaceBomRows(ctx, bomID, inputs)
}

func (s *Service) AddRow(ctx context.Context, bomID, componentID, quantity string) (*BomRow, error) {
	return s.repo.AddRow(ctx, bomID, componentID, quantity)
}

func (s *Service) UpdateRow(ctx context.Context, id, componentID, quantity string) (*BomRow, error) {
	return s.repo.UpdateRow(ctx, id, componentID, quantity)
}

func (s *Service) DeleteRow(ctx context.Context, id string) error {
	return s.repo.DeleteRow(ctx, id)
}

func (s *Service) ListBomRevisions(ctx context.Context, bomID string) ([]*BomRevision, error) {
	if bomID == "" {
		return nil, ErrNotFound
	}
	return s.repo.ListRevisions(ctx, bomID)
}

func (s *Service) GetBomRevision(ctx context.Context, bomID string, revision int) ([]*BomRow, time.Time, error) {
	if bomID == "" || revision < 1 {
		return nil, time.Time{}, ErrNotFound
	}
	return s.repo.GetRevision(ctx, bomID, revision)
}
