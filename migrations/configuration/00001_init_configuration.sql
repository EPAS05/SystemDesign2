-- +goose Up
CREATE TABLE boms (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id uuid NOT NULL,
  is_default boolean NOT NULL DEFAULT false,
  name       text,
  CONSTRAINT boms_name_check CHECK (is_default OR name IS NOT NULL)
);

CREATE UNIQUE INDEX uniq_default_bom_per_product
  ON boms(product_id) WHERE is_default = true;

CREATE UNIQUE INDEX uniq_bom_name_per_product
  ON boms(product_id, name) WHERE is_default = false;

CREATE INDEX idx_boms_product_order
  ON boms(product_id, is_default DESC, name);

CREATE UNIQUE INDEX uniq_boms_id_product
  ON boms(id, product_id);

CREATE TABLE bom_revisions (
  bom_id     uuid NOT NULL,
  revision   int  NOT NULL,
  product_id uuid NOT NULL,
  rows       jsonb NOT NULL,
  changed_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT bom_revisions_pkey PRIMARY KEY (bom_id, revision),
  CONSTRAINT bom_revisions_bom_product_fk
    FOREIGN KEY (bom_id, product_id) REFERENCES boms(id, product_id) ON DELETE CASCADE,
  CONSTRAINT bom_revisions_bom_revision_product_key UNIQUE (bom_id, revision, product_id),
  CONSTRAINT bom_revisions_revision_check CHECK (revision > 0),
  CONSTRAINT bom_revisions_rows_check CHECK (jsonb_typeof(rows) = 'array')
);

CREATE TABLE bom_rows (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  bom_id             uuid NOT NULL REFERENCES boms(id) ON DELETE CASCADE,
  product_id         uuid NOT NULL,
  quantity           numeric(12,3) NOT NULL,
  is_base            boolean NOT NULL DEFAULT false,
  is_excluded        boolean NOT NULL DEFAULT false,
  reference_bom_id   uuid,
  reference_revision int,
  CONSTRAINT bom_rows_bom_product_key UNIQUE (bom_id, product_id),
  CONSTRAINT bom_rows_quantity_check CHECK ((is_excluded AND quantity = 0) OR (NOT is_excluded AND quantity > 0)),
  CONSTRAINT bom_rows_flags_check CHECK (NOT (is_base AND is_excluded)),
  CONSTRAINT bom_rows_excluded_reference_check CHECK (NOT is_excluded OR reference_bom_id IS NULL),
  CONSTRAINT bom_rows_reference_pair_check CHECK ((reference_bom_id IS NULL) = (reference_revision IS NULL)),
  CONSTRAINT bom_rows_reference_revision_check CHECK (reference_revision IS NULL OR reference_revision > 0),
  CONSTRAINT bom_rows_self_reference_check CHECK (reference_bom_id IS NULL OR reference_bom_id <> bom_id),
  CONSTRAINT bom_rows_reference_revision_fk
    FOREIGN KEY (reference_bom_id, reference_revision, product_id)
    REFERENCES bom_revisions(bom_id, revision, product_id) ON DELETE RESTRICT
);

CREATE INDEX idx_bom_rows_product ON bom_rows(product_id);
CREATE INDEX idx_bom_rows_reference ON bom_rows(reference_bom_id, reference_revision);
CREATE INDEX idx_bom_rows_base ON bom_rows(bom_id) WHERE is_base AND NOT is_excluded;

-- +goose Down
DROP TABLE bom_rows;
DROP TABLE bom_revisions;
DROP TABLE boms;
