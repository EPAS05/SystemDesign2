package configuration

import (
	"context"
)

func (s *Service) ListRouteRows(ctx context.Context, bomID string) ([]*RouteRow, error) {
	if bomID == "" {
		return nil, ErrNotFound
	}
	return s.repo.ListRouteRows(ctx, bomID)
}

func (s *Service) AddRouteRow(ctx context.Context, bomID string, in RouteRowInput) (*RouteRow, error) {
	owner, isDefault, err := s.repo.BomInfo(ctx, bomID)
	if err != nil {
		return nil, err
	}
	in, err = s.resolveRouteRow(bomID, owner, isDefault, in)
	if err != nil {
		return nil, err
	}
	return s.repo.AddRouteRow(ctx, bomID, in)
}

func (s *Service) UpdateRouteRow(ctx context.Context, bomID, rowID string, in RouteRowInput) (*RouteRow, error) {
	owner, isDefault, err := s.repo.BomInfo(ctx, bomID)
	if err != nil {
		return nil, err
	}
	in, err = s.resolveRouteRow(bomID, owner, isDefault, in)
	if err != nil {
		return nil, err
	}
	return s.repo.UpdateRouteRow(ctx, bomID, rowID, in)
}

func (s *Service) DeleteRouteRow(ctx context.Context, bomID, rowID string) error {
	if bomID == "" || rowID == "" {
		return ErrNotFound
	}
	return s.repo.DeleteRouteRow(ctx, bomID, rowID)
}

func (s *Service) ReplaceRouteRows(ctx context.Context, bomID string, inputs []RouteRowInput) ([]*RouteRow, error) {
	owner, isDefault, err := s.repo.BomInfo(ctx, bomID)
	if err != nil {
		return nil, err
	}
	resolved := make([]RouteRowInput, 0, len(inputs))
	seen := make(map[int]bool, len(inputs))
	for _, in := range inputs {
		row, err := s.resolveRouteRow(bomID, owner, isDefault, in)
		if err != nil {
			return nil, err
		}
		if seen[row.Seq] {
			return nil, ErrRouteSeqExists
		}
		seen[row.Seq] = true
		resolved = append(resolved, row)
	}
	return s.repo.ReplaceRouteRows(ctx, bomID, resolved)
}

func (s *Service) UpdateBomPrice(ctx context.Context, bomID, price string, manual bool) (*Bom, error) {
	if bomID == "" {
		return nil, ErrNotFound
	}
	if price == "" {
		price = "0"
	}
	value, ok := parseQuantity(price)
	if !ok || value < 0 {
		return nil, ErrInvalidPrice
	}
	if !manual {
		price = "0"
	}
	return s.repo.UpdateBomPrice(ctx, bomID, price, manual)
}

func (s *Service) resolveRouteRow(bomID, owner string, isDefault bool, in RouteRowInput) (RouteRowInput, error) {
	if in.Seq <= 0 || in.OperationClassID == "" || in.ProductID == "" {
		return RouteRowInput{}, ErrInvalidRouteRow
	}
	if in.ProductID == owner {
		return RouteRowInput{}, ErrSelfReference
	}
	if (in.ReferenceBomID == "") != (in.ReferenceRevision <= 0) {
		return RouteRowInput{}, ErrReferencePair
	}
	if in.ReferenceBomID != "" {
		if in.ReferenceBomID == bomID {
			return RouteRowInput{}, ErrSelfReference
		}
		if in.ReferenceRevision <= 0 {
			return RouteRowInput{}, ErrReferenceRevision
		}
	}
	if isDefault && in.IsExcluded {
		return RouteRowInput{}, ErrExcludedOnlyModification
	}
	if in.IsExcluded {
		in.Quantity = "0"
		in.LaborNorm = "0"
		in.EquipmentID = ""
		in.ToolID = ""
		in.ReferenceBomID = ""
		in.ReferenceRevision = 0
		return in, nil
	}
	quantity, ok := parseQuantity(in.Quantity)
	if !ok || quantity <= 0 {
		return RouteRowInput{}, ErrInvalidQuantity
	}
	laborNorm, ok := parseQuantity(in.LaborNorm)
	if !ok || laborNorm < 0 {
		return RouteRowInput{}, ErrInvalidRouteRow
	}
	return in, nil
}
