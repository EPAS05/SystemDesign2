package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) CreateProduct(ctx context.Context, nodeID, name string) (*Product, error) {
	const q = `
		INSERT INTO products (node_id, name)
		VALUES ($1, $2)
		RETURNING id, node_id, name
	`
	p := &Product{}
	err := r.pool.QueryRow(ctx, q, nodeID, name).Scan(&p.ID, &p.NodeID, &p.Name)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repo) GetProduct(ctx context.Context, id string) (*Product, error) {
	const q = `SELECT id, node_id, name FROM products WHERE id = $1`
	p := &Product{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&p.ID, &p.NodeID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repo) ListProductsByNode(ctx context.Context, nodeID string) ([]*Product, error) {
	const q = `
		SELECT id, node_id, name
		FROM products
		WHERE node_id = $1
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		p := &Product{}
		if err := rows.Scan(&p.ID, &p.NodeID, &p.Name); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *Repo) UpdateProduct(ctx context.Context, id, name string) (*Product, error) {
	const q = `
		UPDATE products SET name = $2 WHERE id = $1
		RETURNING id, node_id, name
	`
	p := &Product{}
	err := r.pool.QueryRow(ctx, q, id, name).Scan(&p.ID, &p.NodeID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repo) DeleteProduct(ctx context.Context, id string) error {
	const q = `DELETE FROM products WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
