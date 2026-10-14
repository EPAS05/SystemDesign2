package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetBom(ctx context.Context, id string) (*Bom, error) {
	b, err := r.findBom(ctx, id)
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

func (r *Repo) findBom(ctx context.Context, id string) (*Bom, error) {
	const q = `SELECT id, product_id, is_default, name FROM boms WHERE id = $1`
	b := &Bom{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Repo) BomOwnerProduct(ctx context.Context, bomID string) (string, error) {
	const q = `SELECT product_id FROM boms WHERE id = $1`
	var productID string
	err := r.pool.QueryRow(ctx, q, bomID).Scan(&productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return productID, nil
}

func (r *Repo) RowOwnerProduct(ctx context.Context, rowID string) (string, error) {
	const q = `
		SELECT b.product_id
		FROM bom_rows r
		JOIN boms b ON b.id = r.bom_id
		WHERE r.id = $1
	`
	var productID string
	err := r.pool.QueryRow(ctx, q, rowID).Scan(&productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return productID, nil
}

func (r *Repo) ProductReaches(ctx context.Context, from, to string) (bool, error) {
	const q = `
		WITH RECURSIVE reach(pid) AS (
			SELECT r.product_id
			FROM bom_rows r
			JOIN boms b ON b.id = r.bom_id
			WHERE b.product_id = $1
			UNION
			SELECT r2.product_id
			FROM reach
			JOIN boms b2 ON b2.product_id = reach.pid
			JOIN bom_rows r2 ON r2.bom_id = b2.id
		)
		SELECT EXISTS (SELECT 1 FROM reach WHERE pid = $2)
	`
	var ok bool
	err := r.pool.QueryRow(ctx, q, from, to).Scan(&ok)
	return ok, err
}

func (r *Repo) ListBoms(ctx context.Context, productID string) ([]*Bom, error) {
	const q = `SELECT id, product_id, is_default, name FROM boms WHERE product_id = $1 ORDER BY is_default DESC, name`
	rows, err := r.pool.Query(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Bom
	for rows.Next() {
		b := &Bom{}
		if err := rows.Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, b := range out {
		lines, err := r.listRows(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		b.Rows = lines
	}
	return out, nil
}

func (r *Repo) listRows(ctx context.Context, bomID string) ([]*BomRow, error) {
	const q = `SELECT id, bom_id, product_id, quantity FROM bom_rows WHERE bom_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*BomRow
	for rows.Next() {
		row := &BomRow{}
		if err := rows.Scan(&row.ID, &row.BomID, &row.ProductID, &row.Quantity); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type bomRowSnapshot struct {
	ProductID string `json:"product_id"`
	Quantity  string `json:"quantity"`
}

func snapshotRowsJSON(ctx context.Context, tx pgx.Tx, bomID string) ([]byte, error) {
	const q = `SELECT product_id, quantity FROM bom_rows WHERE bom_id = $1 ORDER BY id`
	rows, err := tx.Query(ctx, q, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []bomRowSnapshot
	for rows.Next() {
		var s bomRowSnapshot
		if err := rows.Scan(&s.ProductID, &s.Quantity); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

func (r *Repo) withRevision(ctx context.Context, tx pgx.Tx, bomID string, apply func() error) error {
	if _, err := tx.Exec(ctx, `SELECT id FROM boms WHERE id = $1 FOR UPDATE`, bomID); err != nil {
		return err
	}
	if err := apply(); err != nil {
		return err
	}
	rowsJSON, err := snapshotRowsJSON(ctx, tx, bomID)
	if err != nil {
		return err
	}
	const ins = `
		INSERT INTO bom_revisions (bom_id, revision, rows)
		SELECT $1, COALESCE(MAX(revision), 0) + 1, $2
		FROM bom_revisions
		WHERE bom_id = $1
	`
	_, err = tx.Exec(ctx, ins, bomID, rowsJSON)
	return err
}

func (r *Repo) CreateBom(ctx context.Context, productID, name string, isDefault bool) (*Bom, error) {
	if isDefault {
		return r.getOrCreateDefaultBom(ctx, productID)
	}

	defaultBom, err := r.getOrCreateDefaultBom(ctx, productID)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const insBom = `
		INSERT INTO boms (product_id, is_default, name)
		VALUES ($1, false, $2)
		RETURNING id, product_id, is_default, name
	`
	b := &Bom{}
	if err := tx.QueryRow(ctx, insBom, productID, name).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name); err != nil {
		return nil, err
	}

	const insRow = `INSERT INTO bom_rows (bom_id, product_id, quantity) VALUES ($1, $2, $3)`
	copyRows := func() error {
		for _, row := range defaultBom.Rows {
			if _, err := tx.Exec(ctx, insRow, b.ID, row.ProductID, row.Quantity); err != nil {
				return err
			}
		}
		return nil
	}
	if err := r.withRevision(ctx, tx, b.ID, copyRows); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetBom(ctx, b.ID)
}

func (r *Repo) getOrCreateDefaultBom(ctx context.Context, productID string) (*Bom, error) {
	const find = `SELECT id, product_id, is_default, name FROM boms WHERE product_id = $1 AND is_default = true`
	b := &Bom{}
	err := r.pool.QueryRow(ctx, find, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name)
	if err == nil {
		rows, err := r.listRows(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		b.Rows = rows
		return b, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const create = `
		INSERT INTO boms (product_id, is_default)
		VALUES ($1, true)
		ON CONFLICT DO NOTHING
		RETURNING id, product_id, is_default, name
	`
	b = &Bom{}
	err = tx.QueryRow(ctx, create, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.getOrCreateDefaultBom(ctx, productID)
	}
	if err != nil {
		return nil, err
	}

	if err := r.withRevision(ctx, tx, b.ID, func() error { return nil }); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	b.Rows = []*BomRow{}
	return b, nil
}

func (r *Repo) DeleteBom(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM boms WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ReplaceBomRows(ctx context.Context, bomID string, inputs []BomRowInput) (*Bom, error) {
	if _, err := r.findBom(ctx, bomID); err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	apply := func() error {
		if _, err := tx.Exec(ctx, `DELETE FROM bom_rows WHERE bom_id = $1`, bomID); err != nil {
			return err
		}
		const ins = `INSERT INTO bom_rows (bom_id, product_id, quantity) VALUES ($1, $2, $3)`
		for _, in := range inputs {
			if _, err := tx.Exec(ctx, ins, bomID, in.ProductID, in.Quantity); err != nil {
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

	return r.GetBom(ctx, bomID)
}

func (r *Repo) AddRow(ctx context.Context, bomID, productID, quantity string) (*BomRow, error) {
	if _, err := r.findBom(ctx, bomID); err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := &BomRow{}
	apply := func() error {
		const q = `
			INSERT INTO bom_rows (bom_id, product_id, quantity)
			VALUES ($1, $2, $3)
			RETURNING id, bom_id, product_id, quantity
		`
		return tx.QueryRow(ctx, q, bomID, productID, quantity).Scan(&row.ID, &row.BomID, &row.ProductID, &row.Quantity)
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return row, nil
}

func (r *Repo) UpdateRow(ctx context.Context, id, productID, quantity string) (*BomRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var bomID string
	err = tx.QueryRow(ctx, `SELECT bom_id FROM bom_rows WHERE id = $1`, id).Scan(&bomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	row := &BomRow{}
	apply := func() error {
		const q = `
			UPDATE bom_rows SET product_id = $2, quantity = $3
			WHERE id = $1
			RETURNING id, bom_id, product_id, quantity
		`
		return tx.QueryRow(ctx, q, id, productID, quantity).Scan(&row.ID, &row.BomID, &row.ProductID, &row.Quantity)
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return row, nil
}

func (r *Repo) DeleteRow(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var bomID string
	err = tx.QueryRow(ctx, `SELECT bom_id FROM bom_rows WHERE id = $1`, id).Scan(&bomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	apply := func() error {
		tag, err := tx.Exec(ctx, `DELETE FROM bom_rows WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repo) ListRevisions(ctx context.Context, bomID string) ([]*BomRevision, error) {
	const q = `SELECT bom_id, revision, changed_at FROM bom_revisions WHERE bom_id = $1 ORDER BY revision`
	rows, err := r.pool.Query(ctx, q, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*BomRevision
	for rows.Next() {
		rv := &BomRevision{}
		if err := rows.Scan(&rv.BomID, &rv.Revision, &rv.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func (r *Repo) GetRevision(ctx context.Context, bomID string, revision int) ([]*BomRow, time.Time, error) {
	const q = `SELECT rows, changed_at FROM bom_revisions WHERE bom_id = $1 AND revision = $2`
	var raw []byte
	var changedAt time.Time
	err := r.pool.QueryRow(ctx, q, bomID, revision).Scan(&raw, &changedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, ErrNotFound
	}
	if err != nil {
		return nil, time.Time{}, err
	}

	var snaps []bomRowSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return nil, time.Time{}, err
	}
	out := make([]*BomRow, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, &BomRow{ProductID: s.ProductID, Quantity: s.Quantity})
	}
	return out, changedAt, nil
}
