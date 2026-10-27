package configuration

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

const effectiveRouteRowsSQL = `
WITH own AS (
    SELECT r.id, r.bom_id, r.seq, r.operation_class_id, r.equipment_id, r.tool_id,
           r.product_id, r.quantity, r.labor_norm, r.is_base, r.is_excluded,
           r.reference_bom_id, r.reference_revision
    FROM bom_route_rows r
    WHERE r.bom_id = $1
)
SELECT r.id, r.bom_id, r.seq, r.operation_class_id, oc.name,
       r.equipment_id, e.name, r.tool_id, t.name, r.product_id,
       r.quantity, r.labor_norm, r.is_base, r.is_excluded,
       r.reference_bom_id, r.reference_revision, false AS inherited
FROM own r
JOIN operation_classes oc ON oc.id = r.operation_class_id
LEFT JOIN equipment e ON e.id = r.equipment_id
LEFT JOIN tools t ON t.id = r.tool_id
WHERE NOT r.is_excluded
UNION ALL
SELECT r.id, r.bom_id, r.seq, r.operation_class_id, oc.name,
       r.equipment_id, e.name, r.tool_id, t.name, r.product_id,
       r.quantity, r.labor_norm, r.is_base, r.is_excluded,
       r.reference_bom_id, r.reference_revision, true AS inherited
FROM bom_route_rows r
JOIN boms b ON b.id = r.bom_id
JOIN boms cur ON cur.id = $1
JOIN operation_classes oc ON oc.id = r.operation_class_id
LEFT JOIN equipment e ON e.id = r.equipment_id
LEFT JOIN tools t ON t.id = r.tool_id
WHERE b.is_default = true
  AND b.product_id = cur.product_id
  AND r.is_base = true
  AND NOT r.is_excluded
  AND NOT EXISTS (SELECT 1 FROM own o WHERE o.seq = r.seq)
ORDER BY seq
`

func scanRouteRow(row pgx.Row) (*RouteRow, error) {
	r := &RouteRow{}
	err := row.Scan(
		&r.ID, &r.BomID, &r.Seq, &r.OperationClassID, &r.OperationClassName,
		&r.EquipmentID, &r.EquipmentName, &r.ToolID, &r.ToolName,
		&r.ProductID, &r.Quantity, &r.LaborNorm, &r.IsBase, &r.IsExcluded,
		&r.ReferenceBomID, &r.ReferenceRevision, &r.Inherited,
	)
	if err != nil {
		return nil, err
	}
	r.SourceBomID = r.BomID
	return r, nil
}

func listRouteRowsQ(ctx context.Context, q querier, bomID string) ([]*RouteRow, error) {
	rows, err := q.Query(ctx, effectiveRouteRowsSQL, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*RouteRow{}
	for rows.Next() {
		row, err := scanRouteRow(rows)
		if err != nil {
			return nil, err
		}
		row.BomID = bomID
		out = append(out, row)
	}
	return out, rows.Err()
}

func getRouteRowByIDQ(ctx context.Context, q querier, id string) (*RouteRow, error) {
	const query = `
		SELECT r.id, r.bom_id, r.seq, r.operation_class_id, oc.name,
		       r.equipment_id, e.name, r.tool_id, t.name, r.product_id,
		       r.quantity, r.labor_norm, r.is_base, r.is_excluded,
		       r.reference_bom_id, r.reference_revision, false AS inherited
		FROM bom_route_rows r
		JOIN operation_classes oc ON oc.id = r.operation_class_id
		LEFT JOIN equipment e ON e.id = r.equipment_id
		LEFT JOIN tools t ON t.id = r.tool_id
		WHERE r.id = $1
	`
	row, err := scanRouteRow(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return row, err
}

func (r *Repo) ListRouteRows(ctx context.Context, bomID string) ([]*RouteRow, error) {
	if _, err := findBomQ(ctx, r.pool, bomID); err != nil {
		return nil, err
	}
	return listRouteRowsQ(ctx, r.pool, bomID)
}

func routeBomInfoQ(ctx context.Context, q querier, bomID string) (string, bool, bool, error) {
	const query = `SELECT product_id, is_default, price_manual FROM boms WHERE id = $1`
	var productID string
	var isDefault, priceManual bool
	err := q.QueryRow(ctx, query, bomID).Scan(&productID, &isDefault, &priceManual)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, false, ErrNotFound
	}
	return productID, isDefault, priceManual, err
}

func scanOwnRouteRow(row pgx.Row) (*RouteRow, error) {
	r := &RouteRow{}
	err := row.Scan(
		&r.ID, &r.BomID, &r.Seq, &r.OperationClassID, &r.EquipmentID, &r.ToolID,
		&r.ProductID, &r.Quantity, &r.LaborNorm, &r.IsBase, &r.IsExcluded,
		&r.ReferenceBomID, &r.ReferenceRevision,
	)
	if err != nil {
		return nil, err
	}
	r.SourceBomID = r.BomID
	return r, nil
}

func getOwnRouteRowQ(ctx context.Context, q querier, id string) (*RouteRow, error) {
	const query = `
		SELECT id, bom_id, seq, operation_class_id, equipment_id, tool_id,
		       product_id, quantity, labor_norm, is_base, is_excluded,
		       reference_bom_id, reference_revision
		FROM bom_route_rows
		WHERE id = $1
	`
	row, err := scanOwnRouteRow(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return row, err
}

func nullableID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func nullableRevision(rev int) *int {
	if rev <= 0 {
		return nil
	}
	return &rev
}

func normalizeRouteInput(in RouteRowInput) (quantity, laborNorm string, equipmentID, toolID *string, referenceBomID *string, referenceRevision *int) {
	quantity = in.Quantity
	laborNorm = in.LaborNorm
	equipmentID = nullableID(in.EquipmentID)
	toolID = nullableID(in.ToolID)
	referenceBomID = nullableID(in.ReferenceBomID)
	referenceRevision = nullableRevision(in.ReferenceRevision)
	if in.IsExcluded {
		quantity = "0"
		laborNorm = "0"
		equipmentID = nil
		toolID = nil
		referenceBomID = nil
		referenceRevision = nil
	}
	return
}

func insertRouteRowQ(ctx context.Context, tx pgx.Tx, bomID string, seq int, operationClassID, productID string, in RouteRowInput, isBase bool) (string, error) {
	quantity, laborNorm, equipmentID, toolID, referenceBomID, referenceRevision := normalizeRouteInput(in)
	const query = `
		INSERT INTO bom_route_rows (
			bom_id, seq, operation_class_id, equipment_id, tool_id, product_id,
			quantity, labor_norm, is_base, is_excluded, reference_bom_id, reference_revision
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`
	var id string
	err := tx.QueryRow(ctx, query,
		bomID, seq, operationClassID, equipmentID, toolID, productID,
		quantity, laborNorm, isBase, in.IsExcluded, referenceBomID, referenceRevision,
	).Scan(&id)
	return id, err
}

func (r *Repo) AddRouteRow(ctx context.Context, bomID string, in RouteRowInput) (*RouteRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var createdID string
	apply := func() error {
		_, isDefault, priceManual, err := routeBomInfoQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		if priceManual {
			return ErrManualPriceWithRoutes
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM bom_route_rows WHERE bom_id = $1 AND seq = $2)`, bomID, in.Seq).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrRouteSeqExists
		}
		createdID, err = insertRouteRowQ(ctx, tx, bomID, in.Seq, in.OperationClassID, in.ProductID, in, isDefault)
		return err
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return getRouteRowByIDQ(ctx, r.pool, createdID)
}

func (r *Repo) UpdateRouteRow(ctx context.Context, bomID, rowID string, in RouteRowInput) (*RouteRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var savedID string
	apply := func() error {
		current, err := getOwnRouteRowQ(ctx, tx, rowID)
		if err != nil {
			return err
		}
		_, isDefault, priceManual, err := routeBomInfoQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		if priceManual {
			return ErrManualPriceWithRoutes
		}

		if current.BomID == bomID {
			quantity, laborNorm, equipmentID, toolID, referenceBomID, referenceRevision := normalizeRouteInput(in)
			const query = `
				UPDATE bom_route_rows
				SET seq = $2, operation_class_id = $3, equipment_id = $4, tool_id = $5,
				    product_id = $6, quantity = $7, labor_norm = $8, is_base = $9,
				    is_excluded = $10, reference_bom_id = $11, reference_revision = $12
				WHERE id = $1
				RETURNING id
			`
			err = tx.QueryRow(ctx, query,
				rowID, in.Seq, in.OperationClassID, equipmentID, toolID, in.ProductID,
				quantity, laborNorm, isDefault, in.IsExcluded, referenceBomID, referenceRevision,
			).Scan(&savedID)
			return err
		}

		if isDefault {
			return ErrRouteRowInherited
		}
		if in.Seq != current.Seq {
			return ErrInvalidRouteRow
		}
		effective, err := listRouteRowsQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		found := false
		for _, row := range effective {
			if row.ID == current.ID && row.Inherited {
				found = true
				break
			}
		}
		if !found {
			return ErrNotFound
		}
		savedID, err = insertRouteRowQ(ctx, tx, bomID, current.Seq, in.OperationClassID, in.ProductID, in, false)
		return err
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return getRouteRowByIDQ(ctx, r.pool, savedID)
}

func (r *Repo) DeleteRouteRow(ctx context.Context, bomID, rowID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	apply := func() error {
		current, err := getOwnRouteRowQ(ctx, tx, rowID)
		if err != nil {
			return err
		}
		_, isDefault, priceManual, err := routeBomInfoQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		if priceManual {
			return ErrManualPriceWithRoutes
		}
		if current.BomID == bomID {
			tag, err := tx.Exec(ctx, `DELETE FROM bom_route_rows WHERE id = $1`, rowID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return ErrNotFound
			}
			return nil
		}
		if isDefault {
			return ErrRouteRowInherited
		}
		effective, err := listRouteRowsQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		found := false
		for _, row := range effective {
			if row.ID == current.ID && row.Inherited {
				found = true
				break
			}
		}
		if !found {
			return ErrNotFound
		}
		_, err = insertRouteRowQ(ctx, tx, bomID, current.Seq, current.OperationClassID, current.ProductID, RouteRowInput{
			Seq:              current.Seq,
			OperationClassID: current.OperationClassID,
			ProductID:        current.ProductID,
			IsExcluded:       true,
		}, false)
		return err
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) ReplaceRouteRows(ctx context.Context, bomID string, inputs []RouteRowInput) ([]*RouteRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	apply := func() error {
		_, isDefault, priceManual, err := routeBomInfoQ(ctx, tx, bomID)
		if err != nil {
			return err
		}
		if priceManual && len(inputs) > 0 {
			return ErrManualPriceWithRoutes
		}
		if _, err := tx.Exec(ctx, `DELETE FROM bom_route_rows WHERE bom_id = $1`, bomID); err != nil {
			return err
		}
		for _, in := range inputs {
			if _, err := insertRouteRowQ(ctx, tx, bomID, in.Seq, in.OperationClassID, in.ProductID, in, isDefault); err != nil {
				return err
			}
		}
		return nil
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.ListRouteRows(ctx, bomID)
}

func (r *Repo) UpdateBomPrice(ctx context.Context, bomID, price string, manual bool) (*Bom, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return nil, err
	}
	var lockedID string
	err = tx.QueryRow(ctx, `SELECT id FROM boms WHERE id = $1 FOR UPDATE`, bomID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if manual {
		rows, err := listRouteRowsQ(ctx, tx, bomID)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return nil, ErrManualPriceWithRoutes
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE boms SET price = $2, price_manual = $3 WHERE id = $1`, bomID, price, manual); err != nil {
		return nil, mapPgError(err)
	}
	if err := recalcPricesTx(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetBom(ctx, bomID)
}

func snapshotRouteRowsJSON(ctx context.Context, tx pgx.Tx, bomID string) ([]byte, error) {
	rows, err := listRouteRowsQ(ctx, tx, bomID)
	if err != nil {
		return nil, err
	}
	out := make([]RouteRowSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, RouteRowSnapshot{
			ID:                 row.ID,
			BomID:              row.SourceBomID,
			Seq:                row.Seq,
			OperationClassID:   row.OperationClassID,
			OperationClassName: row.OperationClassName,
			EquipmentID:        row.EquipmentID,
			EquipmentName:      row.EquipmentName,
			ToolID:             row.ToolID,
			ToolName:           row.ToolName,
			ProductID:          row.ProductID,
			Quantity:           row.Quantity,
			LaborNorm:          row.LaborNorm,
			IsBase:             row.IsBase,
			IsExcluded:         row.IsExcluded,
			ReferenceBomID:     row.ReferenceBomID,
			ReferenceRevision:  row.ReferenceRevision,
			Inherited:          row.Inherited,
		})
	}
	return json.Marshal(out)
}

func decodeRouteRowSnapshots(bomID string, raw []byte) ([]*RouteRow, error) {
	var snaps []RouteRowSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return nil, err
	}
	out := make([]*RouteRow, 0, len(snaps))
	for _, s := range snaps {
		if s.IsExcluded {
			continue
		}
		sourceBomID := s.BomID
		if sourceBomID == "" {
			sourceBomID = bomID
		}
		out = append(out, &RouteRow{
			ID:                 s.ID,
			BomID:              bomID,
			SourceBomID:        sourceBomID,
			Seq:                s.Seq,
			OperationClassID:   s.OperationClassID,
			OperationClassName: s.OperationClassName,
			EquipmentID:        s.EquipmentID,
			EquipmentName:      s.EquipmentName,
			ToolID:             s.ToolID,
			ToolName:           s.ToolName,
			ProductID:          s.ProductID,
			Quantity:           s.Quantity,
			LaborNorm:          s.LaborNorm,
			IsBase:             s.IsBase,
			IsExcluded:         s.IsExcluded,
			ReferenceBomID:     s.ReferenceBomID,
			ReferenceRevision:  s.ReferenceRevision,
			Inherited:          s.Inherited,
		})
	}
	return out, nil
}
