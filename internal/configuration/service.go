package configuration

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSelfReference = errors.New("product cannot contain itself")
	ErrBomCycle      = errors.New("bom would create a cycle")
)

type BomRowInput struct {
	ProductID string
	Quantity  string
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
	owner, err := s.repo.BomOwnerProduct(ctx, bomID)
	if err != nil {
		return nil, err
	}
	for _, in := range inputs {
		if err := s.checkRow(ctx, owner, in.ProductID); err != nil {
			return nil, err
		}
	}
	return s.repo.ReplaceBomRows(ctx, bomID, inputs)
}

func (s *Service) AddRow(ctx context.Context, bomID, productID, quantity string) (*BomRow, error) {
	owner, err := s.repo.BomOwnerProduct(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if err := s.checkRow(ctx, owner, productID); err != nil {
		return nil, err
	}
	return s.repo.AddRow(ctx, bomID, productID, quantity)
}

func (s *Service) UpdateRow(ctx context.Context, id, productID, quantity string) (*BomRow, error) {
	owner, err := s.repo.RowOwnerProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkRow(ctx, owner, productID); err != nil {
		return nil, err
	}
	return s.repo.UpdateRow(ctx, id, productID, quantity)
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

func (s *Service) checkRow(ctx context.Context, owner, productID string) error {
	if owner == productID {
		return ErrSelfReference
	}
	reaches, err := s.repo.ProductReaches(ctx, productID, owner)
	if err != nil {
		return err
	}
	if reaches {
		return ErrBomCycle
	}
	return nil
}
