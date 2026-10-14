package configuration

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Bom struct {
	ID        string
	ProductID string
	IsDefault bool
	Name      *string
	Rows      []*BomRow
}

type BomRow struct {
	ID        string
	BomID     string
	ProductID string
	Quantity  string
}

type BomRevision struct {
	BomID     string
	Revision  int
	ChangedAt time.Time
}
