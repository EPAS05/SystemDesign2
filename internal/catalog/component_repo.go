package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) CreateComponent(ctx context.Context, nodeID, name string) (*Component, error) {
	const q = `
		INSERT INTO components (node_id, name)
		VALUES ($1, $2)
		RETURNING id, node_id, name
	`
	c := &Component{}
	err := r.pool.QueryRow(ctx, q, nodeID, name).Scan(&c.ID, &c.NodeID, &c.Name)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repo) GetComponent(ctx context.Context, id string) (*Component, error) {
	const q = `SELECT id, node_id, name FROM components WHERE id = $1 AND archived_at IS NULL`
	c := &Component{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.NodeID, &c.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repo) ListComponentsByNode(ctx context.Context, nodeID string) ([]*Component, error) {
	const q = `
		SELECT id, node_id, name
		FROM components
		WHERE node_id = $1 AND archived_at IS NULL
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []*Component
	for rows.Next() {
		c := &Component{}
		if err := rows.Scan(&c.ID, &c.NodeID, &c.Name); err != nil {
			return nil, err
		}
		components = append(components, c)
	}
	return components, rows.Err()
}

func (r *Repo) UpdateComponent(ctx context.Context, id, name string) (*Component, error) {
	const q = `
		UPDATE components SET name = $2
		WHERE id = $1 AND archived_at IS NULL
		RETURNING id, node_id, name
	`
	c := &Component{}
	err := r.pool.QueryRow(ctx, q, id, name).Scan(&c.ID, &c.NodeID, &c.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repo) DeleteComponent(ctx context.Context, id string) error {
	const q = `UPDATE components SET archived_at = now() WHERE id = $1 AND archived_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
