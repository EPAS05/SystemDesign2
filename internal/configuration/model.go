package configuration

import "errors"

var ErrNotFound = errors.New("not found")

type Bom struct {
	ID        string
	ProductID string
	IsDefault bool
	Name      *string
	Rows      []*BomRow
}

type BomRow struct {
	ID          string
	BomID       string
	ComponentID string
	Quantity    string
}
