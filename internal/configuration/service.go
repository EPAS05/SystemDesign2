package configuration

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrSelfReference            = errors.New("product cannot contain itself")
	ErrBomCycle                 = errors.New("bom would create a cycle")
	ErrBaseOnlyDefault          = errors.New("is_base is allowed only for default bom")
	ErrExcludedOnlyModification = errors.New("is_excluded is allowed only for modification")
	ErrReferencePair            = errors.New("reference bom and revision must be set together")
	ErrReferenceRevision        = errors.New("reference revision must be positive")
	ErrInvalidQuantity          = errors.New("quantity must be positive")
	ErrInvalidExcludedQuantity  = errors.New("excluded row must have zero quantity")
)

type BomRowInput struct {
	ProductID         string
	Quantity          string
	IsBase            *bool
	IsExcluded        bool
	ReferenceBomID    *string
	ReferenceRevision *int
}

type resolvedRow struct {
	ProductID         string
	Quantity          string
	IsBase            bool
	IsExcluded        bool
	ReferenceBomID    *string
	ReferenceRevision *int
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
	owner, isDefault, err := s.repo.BomInfo(ctx, bomID)
	if err != nil {
		return nil, err
	}
	resolved := make([]resolvedRow, 0, len(inputs))
	seen := make(map[string]bool, len(inputs))
	checked := make(map[string]bool, len(inputs))
	for _, in := range inputs {
		row, err := s.resolveRow(bomID, owner, isDefault, in)
		if err != nil {
			return nil, err
		}
		if seen[row.ProductID] {
			return nil, ErrRowExists
		}
		seen[row.ProductID] = true
		if !row.IsExcluded && row.ReferenceBomID == nil && !checked[row.ProductID] {
			checked[row.ProductID] = true
			reaches, err := s.repo.ProductReaches(ctx, row.ProductID, owner)
			if err != nil {
				return nil, err
			}
			if reaches {
				return nil, ErrBomCycle
			}
		}
		resolved = append(resolved, row)
	}
	return s.repo.ReplaceBomRows(ctx, bomID, resolved)
}

func (s *Service) AddRow(ctx context.Context, bomID, productID, quantity string, isBase *bool, referenceBomID *string, referenceRevision *int, isExcluded bool) (*BomRow, error) {
	owner, isDefault, err := s.repo.BomInfo(ctx, bomID)
	if err != nil {
		return nil, err
	}
	row, err := s.resolveRow(bomID, owner, isDefault, BomRowInput{
		ProductID:         productID,
		Quantity:          quantity,
		IsBase:            isBase,
		IsExcluded:        isExcluded,
		ReferenceBomID:    referenceBomID,
		ReferenceRevision: referenceRevision,
	})
	if err != nil {
		return nil, err
	}
	if !row.IsExcluded && row.ReferenceBomID == nil {
		reaches, err := s.repo.ProductReaches(ctx, row.ProductID, owner)
		if err != nil {
			return nil, err
		}
		if reaches {
			return nil, ErrBomCycle
		}
	}
	return s.repo.AddRow(ctx, bomID, row.ProductID, row.Quantity, row.IsBase, row.IsExcluded, row.ReferenceBomID, row.ReferenceRevision)
}

func (s *Service) UpdateRow(ctx context.Context, id, productID, quantity string, isBase *bool, referenceBomID *string, referenceRevision *int, isExcluded bool) (*BomRow, error) {
	bomID, owner, isDefault, existingIsBase, err := s.repo.RowBomInfo(ctx, id)
	if err != nil {
		return nil, err
	}
	if isDefault && isBase == nil {
		isBase = &existingIsBase
	}
	row, err := s.resolveRow(bomID, owner, isDefault, BomRowInput{
		ProductID:         productID,
		Quantity:          quantity,
		IsBase:            isBase,
		IsExcluded:        isExcluded,
		ReferenceBomID:    referenceBomID,
		ReferenceRevision: referenceRevision,
	})
	if err != nil {
		return nil, err
	}
	if !row.IsExcluded && row.ReferenceBomID == nil {
		reaches, err := s.repo.ProductReaches(ctx, row.ProductID, owner)
		if err != nil {
			return nil, err
		}
		if reaches {
			return nil, ErrBomCycle
		}
	}
	return s.repo.UpdateRow(ctx, id, row.ProductID, row.Quantity, row.IsBase, row.IsExcluded, row.ReferenceBomID, row.ReferenceRevision)
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

func (s *Service) GetBomRevision(ctx context.Context, bomID string, revision int) ([]*BomRow, []*RouteRow, time.Time, error) {
	if bomID == "" || revision < 1 {
		return nil, nil, time.Time{}, ErrNotFound
	}
	return s.repo.GetRevision(ctx, bomID, revision)
}

func (s *Service) ExpandBom(ctx context.Context, bomID string) ([]*BomRow, error) {
	if bomID == "" {
		return nil, ErrNotFound
	}
	return s.repo.ExpandBom(ctx, bomID)
}

func (s *Service) resolveRow(bomID, owner string, isDefault bool, in BomRowInput) (resolvedRow, error) {
	if in.ProductID == "" {
		return resolvedRow{}, ErrInvalidRow
	}
	if (in.ReferenceBomID == nil) != (in.ReferenceRevision == nil) {
		return resolvedRow{}, ErrReferencePair
	}
	if in.ReferenceBomID != nil {
		if *in.ReferenceBomID == "" {
			return resolvedRow{}, ErrReferencePair
		}
		if *in.ReferenceBomID == bomID {
			return resolvedRow{}, ErrSelfReference
		}
	}
	if in.ReferenceRevision != nil && *in.ReferenceRevision < 1 {
		return resolvedRow{}, ErrReferenceRevision
	}
	base, err := resolveIsBase(isDefault, in.IsBase)
	if err != nil {
		return resolvedRow{}, err
	}
	if isDefault && in.IsExcluded {
		return resolvedRow{}, ErrExcludedOnlyModification
	}
	if in.IsExcluded && in.ReferenceBomID != nil {
		return resolvedRow{}, ErrInvalidRow
	}
	if in.IsExcluded {
		q, ok := parseQuantity(in.Quantity)
		if !ok || q != 0 {
			return resolvedRow{}, ErrInvalidExcludedQuantity
		}
	} else {
		q, ok := parseQuantity(in.Quantity)
		if !ok || q <= 0 {
			return resolvedRow{}, ErrInvalidQuantity
		}
		if in.ReferenceBomID == nil && in.ProductID == owner {
			return resolvedRow{}, ErrSelfReference
		}
	}
	return resolvedRow{
		ProductID:         in.ProductID,
		Quantity:          in.Quantity,
		IsBase:            base,
		IsExcluded:        in.IsExcluded,
		ReferenceBomID:    in.ReferenceBomID,
		ReferenceRevision: in.ReferenceRevision,
	}, nil
}

func resolveIsBase(isDefault bool, in *bool) (bool, error) {
	if isDefault {
		if in == nil {
			return true, nil
		}
		return *in, nil
	}
	if in != nil && *in {
		return false, ErrBaseOnlyDefault
	}
	return false, nil
}

func parseQuantity(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, false
	}
	return f, true
}
