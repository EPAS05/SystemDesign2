package configuration

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	bom, err := r.findDefaultBom(ctx, productID)
	if err == nil {
		return bom, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return r.createDefaultBom(ctx, productID)
}

func (r *Repo) findDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	const q = `SELECT id, product_id, is_default, name FROM boms WHERE product_id = $1 AND is_default = true`
	b := &Bom{}
	err := r.pool.QueryRow(ctx, q, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.listRows(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	b.Rows = rows
	return b, nil
}

func (r *Repo) createDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	const q = `
		INSERT INTO boms (product_id, is_default)
		VALUES ($1, true)
		ON CONFLICT DO NOTHING
		RETURNING id, product_id, is_default, name
	`
	b := &Bom{}
	err := r.pool.QueryRow(ctx, q, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.findDefaultBom(ctx, productID)
	}
	if err != nil {
		return nil, err
	}
	b.Rows = []*BomRow{}
	return b, nil
}

func (r *Repo) listRows(ctx context.Context, bomID string) ([]*BomRow, error) {
	const q = `SELECT id, bom_id, component_id, quantity FROM bom_rows WHERE bom_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*BomRow
	for rows.Next() {
		row := &BomRow{}
		if err := rows.Scan(&row.ID, &row.BomID, &row.ComponentID, &row.Quantity); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repo) ReplaceDefaultBom(ctx context.Context, productID string, inputs []BomRowInput) (*Bom, error) {
	bom, err := r.GetDefaultBom(ctx, productID)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM bom_rows WHERE bom_id = $1`, bom.ID); err != nil {
		return nil, err
	}

	const ins = `INSERT INTO bom_rows (bom_id, component_id, quantity) VALUES ($1, $2, $3)`
	for _, in := range inputs {
		if _, err := tx.Exec(ctx, ins, bom.ID, in.ComponentID, in.Quantity); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetDefaultBom(ctx, productID)
}

func (r *Repo) AddRow(ctx context.Context, productID, componentID, quantity string) (*BomRow, error) {
	bom, err := r.GetDefaultBom(ctx, productID)
	if err != nil {
		return nil, err
	}
	const q = `
		INSERT INTO bom_rows (bom_id, component_id, quantity)
		VALUES ($1, $2, $3)
		RETURNING id, bom_id, component_id, quantity
	`
	row := &BomRow{}
	err = r.pool.QueryRow(ctx, q, bom.ID, componentID, quantity).Scan(&row.ID, &row.BomID, &row.ComponentID, &row.Quantity)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *Repo) UpdateRow(ctx context.Context, id, componentID, quantity string) (*BomRow, error) {
	const q = `
		UPDATE bom_rows SET component_id = $2, quantity = $3
		WHERE id = $1
		RETURNING id, bom_id, component_id, quantity
	`
	row := &BomRow{}
	err := r.pool.QueryRow(ctx, q, id, componentID, quantity).Scan(&row.ID, &row.BomID, &row.ComponentID, &row.Quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *Repo) DeleteRow(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM bom_rows WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
