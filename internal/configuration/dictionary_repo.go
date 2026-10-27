package configuration

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapDictionaryError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return ErrDictionaryCodeExists
	case "23503", "23001":
		return ErrDictionaryInUse
	}
	return err
}

func (r *Repo) ListOperationClasses(ctx context.Context) ([]*OperationClass, error) {
	const q = `SELECT id, code, name, hourly_rate FROM operation_classes ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*OperationClass{}
	for rows.Next() {
		item := &OperationClass{}
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.HourlyRate); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repo) GetOperationClass(ctx context.Context, id string) (*OperationClass, error) {
	const q = `SELECT id, code, name, hourly_rate FROM operation_classes WHERE id = $1`
	item := &OperationClass{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Code, &item.Name, &item.HourlyRate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *Repo) CreateOperationClass(ctx context.Context, code, name, hourlyRate string) (*OperationClass, error) {
	const q = `
		INSERT INTO operation_classes (code, name, hourly_rate)
		VALUES ($1, $2, $3)
		RETURNING id, code, name, hourly_rate
	`
	item := &OperationClass{}
	err := r.pool.QueryRow(ctx, q, code, name, hourlyRate).Scan(&item.ID, &item.Code, &item.Name, &item.HourlyRate)
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	return item, nil
}

func (r *Repo) UpdateOperationClass(ctx context.Context, id, code, name, hourlyRate string) (*OperationClass, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return nil, err
	}
	const q = `
		UPDATE operation_classes
		SET code = $2, name = $3, hourly_rate = $4
		WHERE id = $1
		RETURNING id, code, name, hourly_rate
	`
	item := &OperationClass{}
	err = tx.QueryRow(ctx, q, id, code, name, hourlyRate).Scan(&item.ID, &item.Code, &item.Name, &item.HourlyRate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	if err := recalcPricesTx(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repo) DeleteOperationClass(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM operation_classes WHERE id = $1`, id)
	if err != nil {
		return mapDictionaryError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ListEquipment(ctx context.Context) ([]*Equipment, error) {
	const q = `SELECT id, code, name FROM equipment ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Equipment{}
	for rows.Next() {
		item := &Equipment{}
		if err := rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repo) GetEquipment(ctx context.Context, id string) (*Equipment, error) {
	const q = `SELECT id, code, name FROM equipment WHERE id = $1`
	item := &Equipment{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Code, &item.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *Repo) CreateEquipment(ctx context.Context, code, name string) (*Equipment, error) {
	const q = `INSERT INTO equipment (code, name) VALUES ($1, $2) RETURNING id, code, name`
	item := &Equipment{}
	err := r.pool.QueryRow(ctx, q, code, name).Scan(&item.ID, &item.Code, &item.Name)
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	return item, nil
}

func (r *Repo) UpdateEquipment(ctx context.Context, id, code, name string) (*Equipment, error) {
	const q = `UPDATE equipment SET code = $2, name = $3 WHERE id = $1 RETURNING id, code, name`
	item := &Equipment{}
	err := r.pool.QueryRow(ctx, q, id, code, name).Scan(&item.ID, &item.Code, &item.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	return item, nil
}

func (r *Repo) DeleteEquipment(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM equipment WHERE id = $1`, id)
	if err != nil {
		return mapDictionaryError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ListTools(ctx context.Context) ([]*Tool, error) {
	const q = `SELECT id, code, name FROM tools ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Tool{}
	for rows.Next() {
		item := &Tool{}
		if err := rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repo) GetTool(ctx context.Context, id string) (*Tool, error) {
	const q = `SELECT id, code, name FROM tools WHERE id = $1`
	item := &Tool{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Code, &item.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *Repo) CreateTool(ctx context.Context, code, name string) (*Tool, error) {
	const q = `INSERT INTO tools (code, name) VALUES ($1, $2) RETURNING id, code, name`
	item := &Tool{}
	err := r.pool.QueryRow(ctx, q, code, name).Scan(&item.ID, &item.Code, &item.Name)
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	return item, nil
}

func (r *Repo) UpdateTool(ctx context.Context, id, code, name string) (*Tool, error) {
	const q = `UPDATE tools SET code = $2, name = $3 WHERE id = $1 RETURNING id, code, name`
	item := &Tool{}
	err := r.pool.QueryRow(ctx, q, id, code, name).Scan(&item.ID, &item.Code, &item.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapDictionaryError(err)
	}
	return item, nil
}

func (r *Repo) DeleteTool(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tools WHERE id = $1`, id)
	if err != nil {
		return mapDictionaryError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
