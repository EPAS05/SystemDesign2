-- +goose Up
CREATE TABLE boms (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id uuid NOT NULL,
  is_default boolean NOT NULL DEFAULT false,
  name       text
);

CREATE UNIQUE INDEX uniq_default_bom_per_product
  ON boms(product_id) WHERE is_default = true;

CREATE TABLE bom_rows (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  bom_id       uuid NOT NULL REFERENCES boms(id) ON DELETE CASCADE,
  component_id uuid NOT NULL,
  quantity     numeric(12,3) NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_bom_rows_bom       ON bom_rows(bom_id);
CREATE INDEX idx_bom_rows_component ON bom_rows(component_id);

-- +goose Down
DROP TABLE bom_rows;
DROP TABLE boms;