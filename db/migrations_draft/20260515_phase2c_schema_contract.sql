-- Phase 2C-3-pre draft only.
-- Do not execute in this round.
-- Scope is intentionally limited to additive schema alignment for Phase 2C-1 contract.

CREATE TABLE IF NOT EXISTS pos_order_status_dim (
    status_code SMALLINT PRIMARY KEY,
    status_name TEXT NOT NULL,
    status_bucket TEXT NOT NULL,
    is_sales BOOLEAN NOT NULL DEFAULT false,
    is_void BOOLEAN NOT NULL DEFAULT false,
    is_cancelled_like BOOLEAN NOT NULL DEFAULT false,
    is_excluded BOOLEAN NOT NULL DEFAULT false,
    description TEXT NOT NULL DEFAULT '',
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

COMMENT ON TABLE pos_order_status_dim IS 'Phase 2C schema contract status dimension. Fixes raw order status semantics for validation, documentation, and future order fact use.';
COMMENT ON COLUMN pos_order_status_dim.status_code IS 'Raw order status code from source system. Primary key is intentionally the raw code.';
COMMENT ON COLUMN pos_order_status_dim.status_name IS 'Canonical label for the raw status code.';
COMMENT ON COLUMN pos_order_status_dim.status_bucket IS 'High-level bucket: sales, void, or excluded.';
COMMENT ON COLUMN pos_order_status_dim.is_sales IS 'True only when the raw status belongs to the sales bucket.';
COMMENT ON COLUMN pos_order_status_dim.is_void IS 'True only when the raw status belongs to the void bucket.';
COMMENT ON COLUMN pos_order_status_dim.is_cancelled_like IS 'True for excluded statuses that behave like cancelled records in reporting semantics.';
COMMENT ON COLUMN pos_order_status_dim.is_excluded IS 'True when the raw status must be excluded from the primary sales fact path.';
COMMENT ON COLUMN pos_order_status_dim.description IS 'Human-readable explanation of the reporting semantics for this status.';
COMMENT ON COLUMN pos_order_status_dim.sort_order IS 'Stable display and review ordering for status rows.';
COMMENT ON COLUMN pos_order_status_dim.is_active IS 'Soft active flag for future status code lifecycle management.';
COMMENT ON COLUMN pos_order_status_dim.updated_at IS 'Draft audit timestamp for status dimension maintenance.';

INSERT INTO pos_order_status_dim (
    status_code,
    status_name,
    status_bucket,
    is_sales,
    is_void,
    is_cancelled_like,
    is_excluded,
    description,
    sort_order,
    is_active,
    updated_at
)
VALUES
    (1, 'normal_sales', 'sales', true, false, false, false, '正常銷售主口徑', 10, true, now()),
    (-2, 'void', 'void', false, true, false, false, '作廢主口徑', 20, true, now()),
    (-1, 'cancelled_like', 'excluded', false, false, true, true, '排除於 sales / void 主口徑', 30, true, now()),
    (2, 'other_excluded', 'excluded', false, false, false, true, '其他排除狀態', 40, true, now())
ON CONFLICT (status_code) DO UPDATE
SET
    status_name = EXCLUDED.status_name,
    status_bucket = EXCLUDED.status_bucket,
    is_sales = EXCLUDED.is_sales,
    is_void = EXCLUDED.is_void,
    is_cancelled_like = EXCLUDED.is_cancelled_like,
    is_excluded = EXCLUDED.is_excluded,
    description = EXCLUDED.description,
    sort_order = EXCLUDED.sort_order,
    is_active = EXCLUDED.is_active,
    updated_at = EXCLUDED.updated_at;

ALTER TABLE pos_product_dim
    ADD COLUMN IF NOT EXISTS cate_no TEXT NULL;

ALTER TABLE pos_product_dim
    ADD COLUMN IF NOT EXISTS cate_name TEXT NULL;

COMMENT ON COLUMN pos_product_dim.cate_no IS 'Phase 2C schema contract category code. Product attribute only; do not duplicate into pos_sales_hourly_fact.';
COMMENT ON COLUMN pos_product_dim.cate_name IS 'Phase 2C schema contract category name. Product attribute only; do not duplicate into pos_sales_hourly_fact.';

ALTER TABLE pos_branch_dim
    ADD COLUMN IF NOT EXISTS group_code TEXT NULL;

COMMENT ON COLUMN pos_branch_dim.group_code IS 'Phase 2C schema contract branch group code. Branch attribute only; branch options remain view or presentation derived.';

COMMENT ON COLUMN pos_sales_hourly_fact.business_date IS 'Phase 2C schema contract: business_date is fixed to sale_period semantics. Do not use this column as tr_date or t_open_date; those dates belong to future order or payment facts.';
