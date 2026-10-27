package configuration

import (
	"errors"
	"time"
)

var (
	ErrNotFound              = errors.New("not found")
	ErrRowExists             = errors.New("bom row for product already exists")
	ErrBomNameExists         = errors.New("bom name already exists")
	ErrBomInUse              = errors.New("bom is in use")
	ErrReferenceNotFound     = errors.New("referenced bom revision not found")
	ErrInvalidRow            = errors.New("invalid bom row")
	ErrDictionaryCodeExists  = errors.New("dictionary code already exists")
	ErrDictionaryInUse       = errors.New("dictionary entry is in use")
	ErrRouteSeqExists        = errors.New("route row with seq already exists")
	ErrRouteRowInherited     = errors.New("route row is inherited")
	ErrInvalidRouteRow       = errors.New("invalid route row")
	ErrRouteCycle            = errors.New("route contains a cycle")
	ErrManualPriceWithRoutes = errors.New("manual price is allowed only for bom without route rows")
	ErrInvalidPrice          = errors.New("invalid bom price")
	ErrInvalidDictionary     = errors.New("invalid dictionary entry")
)

type Bom struct {
	ID          string
	ProductID   string
	IsDefault   bool
	Name        *string
	Rows        []*BomRow
	Price       string
	PriceManual bool
}

type BomRow struct {
	ID                string
	BomID             string
	ProductID         string
	Quantity          string
	IsBase            bool
	IsExcluded        bool
	ReferenceBomID    *string
	ReferenceRevision *int
}

type BomRevision struct {
	BomID     string
	Revision  int
	ChangedAt time.Time
}

type OperationClass struct {
	ID         string
	Code       string
	Name       string
	HourlyRate string
}

type Equipment struct {
	ID   string
	Code string
	Name string
}

type Tool struct {
	ID   string
	Code string
	Name string
}

type RouteRow struct {
	ID                 string
	BomID              string
	Seq                int
	OperationClassID   string
	OperationClassName string
	EquipmentID        *string
	EquipmentName      *string
	ToolID             *string
	ToolName           *string
	ProductID          string
	Quantity           string
	LaborNorm          string
	IsBase             bool
	IsExcluded         bool
	ReferenceBomID     *string
	ReferenceRevision  *int
	Inherited          bool
	SourceBomID        string
}

type RouteRowInput struct {
	Seq               int
	OperationClassID  string
	EquipmentID       string
	ToolID            string
	ProductID         string
	Quantity          string
	LaborNorm         string
	IsExcluded        bool
	ReferenceBomID    string
	ReferenceRevision int
}

type RouteRowSnapshot struct {
	ID                 string  `json:"id"`
	BomID              string  `json:"bom_id"`
	Seq                int     `json:"seq"`
	OperationClassID   string  `json:"operation_class_id"`
	OperationClassName string  `json:"operation_class_name"`
	EquipmentID        *string `json:"equipment_id,omitempty"`
	EquipmentName      *string `json:"equipment_name,omitempty"`
	ToolID             *string `json:"tool_id,omitempty"`
	ToolName           *string `json:"tool_name,omitempty"`
	ProductID          string  `json:"product_id"`
	Quantity           string  `json:"quantity"`
	LaborNorm          string  `json:"labor_norm"`
	IsBase             bool    `json:"is_base"`
	IsExcluded         bool    `json:"is_excluded"`
	ReferenceBomID     *string `json:"reference_bom_id,omitempty"`
	ReferenceRevision  *int    `json:"reference_revision,omitempty"`
	Inherited          bool    `json:"inherited,omitempty"`
}
