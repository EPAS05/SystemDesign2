-- +goose Up
ALTER TABLE boms
    ADD COLUMN price numeric(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN price_manual boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT boms_price_check CHECK (price >= 0);

ALTER TABLE bom_revisions
    ADD COLUMN routes jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT bom_revisions_routes_array_check CHECK (jsonb_typeof(routes) = 'array');

CREATE TABLE operation_classes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    hourly_rate numeric(12, 2) NOT NULL DEFAULT 0,
    CONSTRAINT operation_classes_hourly_rate_check CHECK (hourly_rate >= 0)
);

CREATE TABLE equipment (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE TABLE tools (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE TABLE bom_route_rows (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    bom_id uuid NOT NULL REFERENCES boms (id) ON DELETE CASCADE,
    seq integer NOT NULL,
    operation_class_id uuid NOT NULL REFERENCES operation_classes (id) ON DELETE RESTRICT,
    equipment_id uuid REFERENCES equipment (id) ON DELETE RESTRICT,
    tool_id uuid REFERENCES tools (id) ON DELETE RESTRICT,
    product_id uuid NOT NULL,
    quantity numeric(12, 3) NOT NULL DEFAULT 0,
    labor_norm numeric(12, 3) NOT NULL DEFAULT 0,
    is_base boolean NOT NULL DEFAULT false,
    is_excluded boolean NOT NULL DEFAULT false,
    reference_bom_id uuid,
    reference_revision integer,
    CONSTRAINT bom_route_rows_seq_unique UNIQUE (bom_id, seq),
    CONSTRAINT bom_route_rows_quantity_check CHECK (quantity >= 0),
    CONSTRAINT bom_route_rows_labor_norm_check CHECK (labor_norm >= 0),
    CONSTRAINT bom_route_rows_base_excluded_check CHECK (NOT (is_base AND is_excluded)),
    CONSTRAINT bom_route_rows_reference_pair_check CHECK (
        (reference_bom_id IS NULL) = (reference_revision IS NULL)
    ),
    CONSTRAINT bom_route_rows_reference_revision_check CHECK (
        reference_revision IS NULL OR reference_revision > 0
    ),
    CONSTRAINT bom_route_rows_self_reference_check CHECK (
        reference_bom_id IS NULL OR reference_bom_id <> bom_id
    ),
    CONSTRAINT bom_route_rows_reference_fk FOREIGN KEY (
        reference_bom_id,
        reference_revision,
        product_id
    ) REFERENCES bom_revisions (bom_id, revision, product_id) ON DELETE RESTRICT
);

CREATE INDEX bom_route_rows_product_idx ON bom_route_rows (product_id);
CREATE INDEX bom_route_rows_reference_idx ON bom_route_rows (reference_bom_id, reference_revision);
CREATE INDEX bom_route_rows_operation_class_idx ON bom_route_rows (operation_class_id);
CREATE INDEX bom_route_rows_equipment_idx ON bom_route_rows (equipment_id);
CREATE INDEX bom_route_rows_tool_idx ON bom_route_rows (tool_id);

-- +goose Down
DROP TABLE IF EXISTS bom_route_rows;
DROP TABLE IF EXISTS tools;
DROP TABLE IF EXISTS equipment;
DROP TABLE IF EXISTS operation_classes;

ALTER TABLE bom_revisions DROP CONSTRAINT IF EXISTS bom_revisions_routes_array_check;
ALTER TABLE bom_revisions DROP COLUMN IF EXISTS routes;

ALTER TABLE boms DROP CONSTRAINT IF EXISTS boms_price_check;
ALTER TABLE boms DROP COLUMN IF EXISTS price_manual;
ALTER TABLE boms DROP COLUMN IF EXISTS price;
