package configuration

import (
	"context"
	"math"
	"strconv"
)

type BomRowInput struct {
	ComponentID string
	Quantity    string
}

func validQuantity(q string) bool {
	f, err := strconv.ParseFloat(q, 64)
	return err == nil && f > 0 && !math.IsInf(f, 0)
}

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	if productID == "" {
		return nil, ErrInvalid
	}
	return s.repo.GetDefaultBom(ctx, productID)
}

func (s *Service) ReplaceDefaultBom(ctx context.Context, productID string, inputs []BomRowInput) (*Bom, error) {
	if productID == "" {
		return nil, ErrInvalid
	}
	for _, in := range inputs {
		if in.ComponentID == "" || !validQuantity(in.Quantity) {
			return nil, ErrInvalid
		}
	}
	return s.repo.ReplaceDefaultBom(ctx, productID, inputs)
}

func (s *Service) AddRow(ctx context.Context, productID, componentID, quantity string) (*BomRow, error) {
	if productID == "" || componentID == "" || !validQuantity(quantity) {
		return nil, ErrInvalid
	}
	return s.repo.AddRow(ctx, productID, componentID, quantity)
}

func (s *Service) UpdateRow(ctx context.Context, id, componentID, quantity string) (*BomRow, error) {
	if id == "" || componentID == "" || !validQuantity(quantity) {
		return nil, ErrInvalid
	}
	return s.repo.UpdateRow(ctx, id, componentID, quantity)
}

func (s *Service) DeleteRow(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalid
	}
	return s.repo.DeleteRow(ctx, id)
}
