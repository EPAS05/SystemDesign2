package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetBom(ctx context.Context, id string) (*Bom, error) {
	b, err := findBomQ(ctx, r.pool, id)
	if err != nil {
		return nil, err
	}
	rows, err := listRowsQ(ctx, r.pool, b.ID)
	if err != nil {
		return nil, err
	}
	b.Rows = rows
	return b, nil
}

func findBomQ(ctx context.Context, q querier, id string) (*Bom, error) {
	const query = `SELECT id, product_id, is_default, name, price, price_manual FROM boms WHERE id = $1`
	b := &Bom{}
	err := q.QueryRow(ctx, query, id).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name, &b.Price, &b.PriceManual)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Repo) BomInfo(ctx context.Context, bomID string) (string, bool, error) {
	const q = `SELECT product_id, is_default FROM boms WHERE id = $1`
	var productID string
	var isDefault bool
	err := r.pool.QueryRow(ctx, q, bomID).Scan(&productID, &isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrNotFound
	}
	if err != nil {
		return "", false, err
	}
	return productID, isDefault, nil
}

func (r *Repo) RowBomInfo(ctx context.Context, rowID string) (string, string, bool, bool, error) {
	const q = `
		SELECT b.id, b.product_id, b.is_default, r.is_base
		FROM bom_rows r
		JOIN boms b ON b.id = r.bom_id
		WHERE r.id = $1
	`
	var bomID, productID string
	var isDefault, isBase bool
	err := r.pool.QueryRow(ctx, q, rowID).Scan(&bomID, &productID, &isDefault, &isBase)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, false, ErrNotFound
	}
	if err != nil {
		return "", "", false, false, err
	}
	return bomID, productID, isDefault, isBase, nil
}

func (r *Repo) ProductReaches(ctx context.Context, from, to string) (bool, error) {
	const q = `
		WITH RECURSIVE reach(pid) AS (
			SELECT r.product_id
			FROM bom_rows r
			JOIN boms b ON b.id = r.bom_id
			WHERE b.product_id = $1
			  AND b.is_default = true
			  AND r.reference_bom_id IS NULL
			  AND r.is_excluded = false
			UNION
			SELECT r2.product_id
			FROM reach
			JOIN boms b2 ON b2.product_id = reach.pid AND b2.is_default = true
			JOIN bom_rows r2 ON r2.bom_id = b2.id
			WHERE r2.reference_bom_id IS NULL
			  AND r2.is_excluded = false
		)
		SELECT EXISTS (SELECT 1 FROM reach WHERE pid = $2)
	`
	var ok bool
	err := r.pool.QueryRow(ctx, q, from, to).Scan(&ok)
	return ok, err
}

func (r *Repo) ListBoms(ctx context.Context, productID string) ([]*Bom, error) {
	const q = `SELECT id, product_id, is_default, name, price, price_manual FROM boms WHERE product_id = $1 ORDER BY is_default DESC, name`
	rows, err := r.pool.Query(ctx, q, productID)
	if err != nil {
		return nil, err
	}

	out := []*Bom{}
	byID := map[string]*Bom{}
	for rows.Next() {
		b := &Bom{Rows: []*BomRow{}}
		if err := rows.Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name, &b.Price, &b.PriceManual); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, b)
		byID[b.ID] = b
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	const rq = `
		SELECT r.id, r.bom_id, r.product_id, r.quantity, r.is_base, r.is_excluded, r.reference_bom_id, r.reference_revision
		FROM bom_rows r
		JOIN boms b ON b.id = r.bom_id
		WHERE b.product_id = $1
		ORDER BY r.bom_id, r.id
	`
	rrows, err := r.pool.Query(ctx, rq, productID)
	if err != nil {
		return nil, err
	}
	defer rrows.Close()
	for rrows.Next() {
		row, err := scanBomRow(rrows)
		if err != nil {
			return nil, err
		}
		if b, ok := byID[row.BomID]; ok {
			b.Rows = append(b.Rows, row)
		}
	}
	if err := rrows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func listRowsQ(ctx context.Context, q querier, bomID string) ([]*BomRow, error) {
	const query = `SELECT id, bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision FROM bom_rows WHERE bom_id = $1 ORDER BY id`
	rows, err := q.Query(ctx, query, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*BomRow{}
	for rows.Next() {
		row, err := scanBomRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanBomRow(row pgx.Row) (*BomRow, error) {
	b := &BomRow{}
	err := row.Scan(&b.ID, &b.BomID, &b.ProductID, &b.Quantity, &b.IsBase, &b.IsExcluded, &b.ReferenceBomID, &b.ReferenceRevision)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type bomRowSnapshot struct {
	ProductID         string `json:"product_id"`
	Quantity          string `json:"quantity"`
	IsBase            bool   `json:"is_base"`
	IsExcluded        bool   `json:"is_excluded,omitempty"`
	ReferenceBomID    string `json:"reference_bom_id,omitempty"`
	ReferenceRevision int    `json:"reference_revision,omitempty"`
}

func effectiveRowsQ(ctx context.Context, q querier, bomID string) ([]*BomRow, error) {
	const query = `
		WITH own AS (
			SELECT id, bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision
			FROM bom_rows
			WHERE bom_id = $1
		)
		SELECT id, bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision
		FROM own
		UNION ALL
		SELECT r.id, r.bom_id, r.product_id, r.quantity, r.is_base, r.is_excluded, r.reference_bom_id, r.reference_revision
		FROM bom_rows r
		JOIN boms b ON b.id = r.bom_id
		WHERE b.is_default = true
		  AND b.product_id = (SELECT product_id FROM boms WHERE id = $1)
		  AND r.is_base = true
		  AND r.is_excluded = false
		  AND NOT EXISTS (SELECT 1 FROM own o WHERE o.product_id = r.product_id)
		ORDER BY id
	`
	rows, err := q.Query(ctx, query, bomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*BomRow{}
	for rows.Next() {
		row, err := scanBomRow(rows)
		if err != nil {
			return nil, err
		}
		if row.IsExcluded {
			continue
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func snapshotRowsJSON(ctx context.Context, tx pgx.Tx, bomID string) ([]byte, error) {
	rows, err := effectiveRowsQ(ctx, tx, bomID)
	if err != nil {
		return nil, err
	}
	out := make([]bomRowSnapshot, 0, len(rows))
	for _, row := range rows {
		s := bomRowSnapshot{
			ProductID:  row.ProductID,
			Quantity:   row.Quantity,
			IsBase:     row.IsBase,
			IsExcluded: row.IsExcluded,
		}
		if row.ReferenceBomID != nil {
			s.ReferenceBomID = *row.ReferenceBomID
		}
		if row.ReferenceRevision != nil {
			s.ReferenceRevision = *row.ReferenceRevision
		}
		out = append(out, s)
	}
	return json.Marshal(out)
}

func (r *Repo) withRevision(ctx context.Context, tx pgx.Tx, bomID string, apply func() error) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return err
	}
	var lockedID, productID string
	err := tx.QueryRow(ctx, `SELECT id, product_id FROM boms WHERE id = $1 FOR UPDATE`, bomID).Scan(&lockedID, &productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := apply(); err != nil {
		return mapPgError(err)
	}
	rowsJSON, err := snapshotRowsJSON(ctx, tx, bomID)
	if err != nil {
		return err
	}
	routesJSON, err := snapshotRouteRowsJSON(ctx, tx, bomID)
	if err != nil {
		return err
	}
	const ins = `
		INSERT INTO bom_revisions (bom_id, revision, product_id, rows, routes)
		SELECT $1, COALESCE(MAX(revision), 0) + 1, $2, $3, $4
		FROM bom_revisions
		WHERE bom_id = $1
	`
	if _, err := tx.Exec(ctx, ins, bomID, productID, rowsJSON, routesJSON); err != nil {
		return mapPgError(err)
	}
	if err := recalcPricesTx(ctx, tx); err != nil {
		return err
	}
	return nil
}

func (r *Repo) CreateBom(ctx context.Context, productID, name string, isDefault bool) (*Bom, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return nil, err
	}
	defaultBom, err := r.getOrCreateDefaultBomTx(ctx, tx, productID)
	if err != nil {
		return nil, err
	}
	if isDefault {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return r.GetBom(ctx, defaultBom.ID)
	}

	const insBom = `
		INSERT INTO boms (product_id, is_default, name)
		VALUES ($1, false, $2)
		RETURNING id, product_id, is_default, name, price, price_manual
	`
	b := &Bom{}
	if err := tx.QueryRow(ctx, insBom, productID, name).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name, &b.Price, &b.PriceManual); err != nil {
		return nil, mapPgError(err)
	}

	const copyRows = `
		INSERT INTO bom_rows (bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision)
		SELECT $1, product_id, quantity, false, false, reference_bom_id, reference_revision
		FROM bom_rows
		WHERE bom_id = $2 AND NOT is_base AND NOT is_excluded
	`
	apply := func() error {
		_, err := tx.Exec(ctx, copyRows, b.ID, defaultBom.ID)
		return err
	}
	if err := r.withRevision(ctx, tx, b.ID, apply); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetBom(ctx, b.ID)
}

func (r *Repo) getOrCreateDefaultBomTx(ctx context.Context, tx pgx.Tx, productID string) (*Bom, error) {
	const find = `
		SELECT id, product_id, is_default, name, price, price_manual
		FROM boms
		WHERE product_id = $1 AND is_default = true
		FOR SHARE
	`
	const create = `
		INSERT INTO boms (product_id, is_default)
		VALUES ($1, true)
		ON CONFLICT DO NOTHING
		RETURNING id, product_id, is_default, name, price, price_manual
	`
	for attempt := 0; attempt < 3; attempt++ {
		b := &Bom{}
		err := tx.QueryRow(ctx, find, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name, &b.Price, &b.PriceManual)
		if err == nil {
			return b, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		b = &Bom{}
		err = tx.QueryRow(ctx, create, productID).Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Name, &b.Price, &b.PriceManual)
		if err == nil {
			if err := r.withRevision(ctx, tx, b.ID, func() error { return nil }); err != nil {
				return nil, err
			}
			return b, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, mapPgError(err)
		}
	}
	return nil, errors.New("failed to get or create default bom")
}

func (r *Repo) DeleteBom(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return err
	}
	var productID string
	var isDefault bool
	err = tx.QueryRow(ctx, `SELECT product_id, is_default FROM boms WHERE id = $1 FOR UPDATE`, id).Scan(&productID, &isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if isDefault {
		var inUse bool
		const check = `SELECT EXISTS (SELECT 1 FROM boms WHERE product_id = $1 AND is_default = false)`
		if err := tx.QueryRow(ctx, check, productID).Scan(&inUse); err != nil {
			return err
		}
		if inUse {
			return ErrBomInUse
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM boms WHERE id = $1`, id); err != nil {
		mapped := mapPgError(err)
		if errors.Is(mapped, ErrReferenceNotFound) {
			return ErrBomInUse
		}
		return mapped
	}
	if err := recalcPricesTx(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) ReplaceBomRows(ctx context.Context, bomID string, rows []resolvedRow) (*Bom, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	apply := func() error {
		if _, err := tx.Exec(ctx, `DELETE FROM bom_rows WHERE bom_id = $1`, bomID); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		columns := []string{"bom_id", "product_id", "quantity", "is_base", "is_excluded", "reference_bom_id", "reference_revision"}
		_, err := tx.CopyFrom(ctx, pgx.Identifier{"bom_rows"}, columns, pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			row := rows[i]
			var quantity pgtype.Numeric
			if err := quantity.Scan(row.Quantity); err != nil {
				return nil, err
			}
			return []any{bomID, row.ProductID, quantity, row.IsBase, row.IsExcluded, row.ReferenceBomID, row.ReferenceRevision}, nil
		}))
		return err
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetBom(ctx, bomID)
}

func (r *Repo) AddRow(ctx context.Context, bomID, productID, quantity string, isBase, isExcluded bool, referenceBomID *string, referenceRevision *int) (*BomRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var row *BomRow
	apply := func() error {
		const q = `
			INSERT INTO bom_rows (bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision
		`
		var err error
		row, err = scanBomRow(tx.QueryRow(ctx, q, bomID, productID, quantity, isBase, isExcluded, referenceBomID, referenceRevision))
		return err
	}
	if err := r.withRevision(ctx, tx, bomID, apply); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}

func (r *Repo) UpdateRow(ctx context.Context, id, productID, quantity string, isBase, isExcluded bool, referenceBomID *string, referenceRevision *int) (*BomRow, error) {
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

	var row *BomRow
	apply := func() error {
		const q = `
			UPDATE bom_rows
			SET product_id = $2, quantity = $3, is_base = $4, is_excluded = $5, reference_bom_id = $6, reference_revision = $7
			WHERE id = $1
			RETURNING id, bom_id, product_id, quantity, is_base, is_excluded, reference_bom_id, reference_revision
		`
		var err error
		row, err = scanBomRow(tx.QueryRow(ctx, q, id, productID, quantity, isBase, isExcluded, referenceBomID, referenceRevision))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
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

	out := []*BomRevision{}
	for rows.Next() {
		rv := &BomRevision{}
		if err := rows.Scan(&rv.BomID, &rv.Revision, &rv.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func (r *Repo) GetRevision(ctx context.Context, bomID string, revision int) ([]*BomRow, []*RouteRow, time.Time, error) {
	const q = `SELECT rows, routes, changed_at FROM bom_revisions WHERE bom_id = $1 AND revision = $2`
	var raw, rawRoutes []byte
	var changedAt time.Time
	err := r.pool.QueryRow(ctx, q, bomID, revision).Scan(&raw, &rawRoutes, &changedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, time.Time{}, ErrNotFound
	}
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	var snaps []bomRowSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return nil, nil, time.Time{}, err
	}
	out := make([]*BomRow, 0, len(snaps))
	for _, s := range snaps {
		if s.IsExcluded {
			continue
		}
		var refBomID *string
		if s.ReferenceBomID != "" {
			value := s.ReferenceBomID
			refBomID = &value
		}
		var refRev *int
		if s.ReferenceRevision != 0 {
			value := s.ReferenceRevision
			refRev = &value
		}
		out = append(out, &BomRow{
			ProductID:         s.ProductID,
			Quantity:          s.Quantity,
			IsBase:            s.IsBase,
			IsExcluded:        s.IsExcluded,
			ReferenceBomID:    refBomID,
			ReferenceRevision: refRev,
		})
	}
	routes, err := decodeRouteRowSnapshots(bomID, rawRoutes)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	return out, routes, changedAt, nil
}

type pathNode struct {
	key  string
	prev *pathNode
}

func (p *pathNode) has(key string) bool {
	for n := p; n != nil; n = n.prev {
		if n.key == key {
			return true
		}
	}
	return false
}

type expandItem struct {
	bomID      string
	revision   int
	productID  string
	multiplier float64
	depth      int
	path       *pathNode
}

type bomExpander struct {
	ctx       context.Context
	q         querier
	live      map[string][]*BomRow
	snapshots map[string][]*BomRow
	defaults  map[string]string
	known     map[string]bool
}

func newBomExpander(ctx context.Context, q querier) *bomExpander {
	return &bomExpander{
		ctx:       ctx,
		q:         q,
		live:      map[string][]*BomRow{},
		snapshots: map[string][]*BomRow{},
		defaults:  map[string]string{},
		known:     map[string]bool{},
	}
}

func (e *bomExpander) liveRows(bomID string) ([]*BomRow, error) {
	if rows, ok := e.live[bomID]; ok {
		return rows, nil
	}
	rows, err := effectiveRowsQ(e.ctx, e.q, bomID)
	if err != nil {
		return nil, err
	}
	e.live[bomID] = rows
	return rows, nil
}

func (e *bomExpander) snapshotRows(bomID string, revision int) ([]*BomRow, error) {
	key := bomID + ":" + strconv.Itoa(revision)
	if rows, ok := e.snapshots[key]; ok {
		return rows, nil
	}
	const q = `SELECT rows FROM bom_revisions WHERE bom_id = $1 AND revision = $2`
	var raw []byte
	err := e.q.QueryRow(e.ctx, q, bomID, revision).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var snaps []bomRowSnapshot
	if err := json.Unmarshal(raw, &snaps); err != nil {
		return nil, err
	}
	out := make([]*BomRow, 0, len(snaps))
	for _, s := range snaps {
		if s.IsExcluded {
			continue
		}
		var refBomID *string
		if s.ReferenceBomID != "" {
			value := s.ReferenceBomID
			refBomID = &value
		}
		var refRev *int
		if s.ReferenceRevision != 0 {
			value := s.ReferenceRevision
			refRev = &value
		}
		out = append(out, &BomRow{
			BomID:             bomID,
			ProductID:         s.ProductID,
			Quantity:          s.Quantity,
			IsBase:            s.IsBase,
			IsExcluded:        s.IsExcluded,
			ReferenceBomID:    refBomID,
			ReferenceRevision: refRev,
		})
	}
	e.snapshots[key] = out
	return out, nil
}

func (e *bomExpander) resolveDefaults(productIDs []string) error {
	missing := []string{}
	seen := map[string]bool{}
	for _, id := range productIDs {
		if id == "" || e.known[id] || seen[id] {
			continue
		}
		seen[id] = true
		missing = append(missing, id)
	}
	if len(missing) == 0 {
		return nil
	}
	placeholders := make([]string, len(missing))
	args := make([]any, len(missing))
	for i, id := range missing {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	query := `SELECT product_id, id FROM boms WHERE is_default = true AND product_id IN (` + strings.Join(placeholders, ", ") + `)`
	rows, err := e.q.Query(e.ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var productID, bomID string
		if err := rows.Scan(&productID, &bomID); err != nil {
			return err
		}
		e.defaults[productID] = bomID
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range missing {
		e.known[id] = true
	}
	return nil
}

func (r *Repo) ExpandBom(ctx context.Context, bomID string) ([]*BomRow, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	root, err := findBomQ(ctx, tx, bomID)
	if err != nil {
		return nil, err
	}
	exp := newBomExpander(ctx, tx)
	rootRows, err := exp.liveRows(bomID)
	if err != nil {
		return nil, err
	}
	if len(rootRows) == 0 {
		return []*BomRow{}, nil
	}

	stack := []expandItem{{
		bomID:      bomID,
		productID:  root.ProductID,
		multiplier: 1,
		path:       &pathNode{key: "live:" + bomID},
	}}
	results := map[string]float64{}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if item.depth > 100 {
			continue
		}

		var rows []*BomRow
		if item.revision > 0 {
			rows, err = exp.snapshotRows(item.bomID, item.revision)
		} else {
			rows, err = exp.liveRows(item.bomID)
		}
		if err != nil {
			return nil, err
		}

		if len(rows) == 0 {
			if item.path.prev != nil {
				results[item.productID] += item.multiplier
			}
			continue
		}

		productIDs := make([]string, 0, len(rows))
		for _, row := range rows {
			if row.ReferenceBomID == nil {
				productIDs = append(productIDs, row.ProductID)
			}
		}
		if err := exp.resolveDefaults(productIDs); err != nil {
			return nil, err
		}

		for _, row := range rows {
			qty := item.multiplier * parseQty(row.Quantity)
			if row.ReferenceBomID != nil {
				revision := 0
				if row.ReferenceRevision != nil {
					revision = *row.ReferenceRevision
				}
				if revision <= 0 {
					return nil, ErrInvalidRow
				}
				key := "rev:" + *row.ReferenceBomID + ":" + strconv.Itoa(revision)
				if item.path.has(key) {
					continue
				}
				stack = append(stack, expandItem{
					bomID:      *row.ReferenceBomID,
					revision:   revision,
					productID:  row.ProductID,
					multiplier: qty,
					depth:      item.depth + 1,
					path:       &pathNode{key: key, prev: item.path},
				})
				continue
			}
			defBomID, ok := exp.defaults[row.ProductID]
			if !ok {
				results[row.ProductID] += qty
				continue
			}
			key := "live:" + defBomID
			if item.path.has(key) {
				continue
			}
			stack = append(stack, expandItem{
				bomID:      defBomID,
				productID:  row.ProductID,
				multiplier: qty,
				depth:      item.depth + 1,
				path:       &pathNode{key: key, prev: item.path},
			})
		}
	}

	out := make([]*BomRow, 0, len(results))
	for productID, qty := range results {
		out = append(out, &BomRow{ProductID: productID, Quantity: formatQty(qty)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out, nil
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		switch pgErr.ConstraintName {
		case "uniq_bom_name_per_product":
			return ErrBomNameExists
		case "bom_route_rows_seq_unique":
			return ErrRouteSeqExists
		}
		return ErrRowExists
	case "23503":
		switch pgErr.ConstraintName {
		case "bom_rows_reference_revision_fk", "bom_route_rows_reference_fk":
			return ErrReferenceNotFound
		case "bom_rows_bom_id_fkey", "bom_revisions_bom_product_fk":
			return ErrNotFound
		case "bom_route_rows_operation_class_id_fkey", "bom_route_rows_equipment_id_fkey", "bom_route_rows_tool_id_fkey":
			return ErrNotFound
		}
		return ErrInvalidRow
	case "23514":
		if pgErr.ConstraintName == "boms_price_check" {
			return ErrInvalidPrice
		}
		if strings.HasPrefix(pgErr.ConstraintName, "bom_route_rows_") {
			return ErrInvalidRouteRow
		}
		return ErrInvalidRow
	case "23001":
		if pgErr.ConstraintName == "bom_rows_reference_revision_fk" || pgErr.ConstraintName == "bom_route_rows_reference_fk" {
			return ErrReferenceNotFound
		}
		return ErrInvalidRow
	}
	return err
}

func parseQty(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func formatQty(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
