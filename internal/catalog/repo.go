package catalog

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

func (r *Repo) Create(ctx context.Context, parentID *string, name string) (*Node, error) {
	const q = `
		INSERT INTO nodes (parent_id, name)
		VALUES ($1, $2)
		RETURNING id, parent_id, name
	`
	n := &Node{}
	err := r.pool.QueryRow(ctx, q, parentID, name).Scan(&n.ID, &n.ParentID, &n.Name)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *Repo) Get(ctx context.Context, id string) (*Node, error) {
	const q = `SELECT id, parent_id, name FROM nodes WHERE id = $1`
	n := &Node{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&n.ID, &n.ParentID, &n.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *Repo) ListChildren(ctx context.Context, parentID string) ([]*Node, error) {
	const q = `SELECT id, parent_id, name FROM nodes WHERE parent_id = $1 ORDER BY name`
	rows, err := r.pool.Query(ctx, q, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		n := &Node{}
		if err := rows.Scan(&n.ID, &n.ParentID, &n.Name); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (r *Repo) Update(ctx context.Context, id, name string) (*Node, error) {
	const q = `
		UPDATE nodes SET name = $2 WHERE id = $1
		RETURNING id, parent_id, name
	`
	n := &Node{}
	err := r.pool.QueryRow(ctx, q, id, name).Scan(&n.ID, &n.ParentID, &n.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM nodes WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
