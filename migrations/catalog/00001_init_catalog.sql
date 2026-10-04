-- +goose Up
CREATE TABLE nodes (
  id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id uuid REFERENCES nodes(id) ON DELETE RESTRICT,
  name      text NOT NULL
);

CREATE INDEX idx_nodes_parent ON nodes(parent_id);

CREATE TABLE products (
  id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id uuid NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
  name    text NOT NULL
);

CREATE INDEX idx_products_node ON products(node_id);

CREATE TABLE components (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id     uuid NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
  name        text NOT NULL,
  archived_at timestamptz
);

CREATE INDEX idx_components_node ON components(node_id);

-- +goose Down
DROP TABLE components;
DROP TABLE products;
DROP TABLE nodes;