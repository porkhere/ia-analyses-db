BEGIN;

CREATE TABLE IF NOT EXISTS ia_users (
    id BIGSERIAL PRIMARY KEY,
    owner_user_key TEXT NOT NULL UNIQUE,
    display_name TEXT,
    source_system TEXT NOT NULL DEFAULT 'athena',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_order_type_dim (
    id SMALLINT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_payment_type_dim (
    id SMALLINT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_order_status_dim (
    status_code SMALLINT PRIMARY KEY,
    status_name TEXT NOT NULL,
    status_bucket TEXT NOT NULL,
    is_sales BOOLEAN NOT NULL DEFAULT FALSE,
    is_void BOOLEAN NOT NULL DEFAULT FALSE,
    is_cancelled_like BOOLEAN NOT NULL DEFAULT FALSE,
    is_excluded BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_product_dim (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    product_no TEXT NOT NULL,
    product_name TEXT NOT NULL,
    product_name_normalized TEXT,
    cate_no TEXT,
    cate_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_user_id, product_no)
);

CREATE TABLE IF NOT EXISTS pos_branch_dim (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    branch_id TEXT NOT NULL,
    branch_name TEXT NOT NULL,
    branch_name_normalized TEXT,
    group_code TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_user_id, branch_id)
);

CREATE TABLE IF NOT EXISTS pos_sales_hourly_fact (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    business_date DATE NOT NULL,
    hour_of_day SMALLINT NOT NULL CHECK (hour_of_day BETWEEN 0 AND 23),
    branch_id TEXT NOT NULL,
    product_no TEXT NOT NULL,
    order_type_id SMALLINT NOT NULL REFERENCES pos_order_type_dim(id),
    payment_type_id SMALLINT NOT NULL REFERENCES pos_payment_type_dim(id),
    qty_milli BIGINT NOT NULL DEFAULT 0,
    gross_sales_milli BIGINT NOT NULL DEFAULT 0,
    discount_milli BIGINT NOT NULL DEFAULT 0,
    surcharge_milli BIGINT NOT NULL DEFAULT 0,
    net_sales_milli BIGINT NOT NULL DEFAULT 0,
    sales_ex_tax_milli BIGINT NOT NULL DEFAULT 0,
    included_tax_milli BIGINT NOT NULL DEFAULT 0,
    excluded_tax_milli BIGINT NOT NULL DEFAULT 0,
    tax_milli BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (
        owner_user_id,
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id
    )
);

CREATE INDEX IF NOT EXISTS idx_ia_users_active
    ON ia_users (is_active, owner_user_key);

CREATE INDEX IF NOT EXISTS idx_pos_product_dim_lookup
    ON pos_product_dim (owner_user_id, product_no, is_active);

CREATE INDEX IF NOT EXISTS idx_pos_branch_dim_lookup
    ON pos_branch_dim (owner_user_id, branch_id, is_active);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_date
    ON pos_sales_hourly_fact (owner_user_id, business_date, hour_of_day);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_branch
    ON pos_sales_hourly_fact (owner_user_id, business_date, branch_id);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_product
    ON pos_sales_hourly_fact (owner_user_id, business_date, product_no);

COMMENT ON TABLE public.pos_order_status_dim IS 'Phase 2C schema contract status dimension. Fixes raw order status semantics for validation, documentation, and future order fact use.';
COMMENT ON COLUMN public.pos_order_status_dim.status_code IS 'Raw order status code from source system. Primary key is intentionally the raw code.';
COMMENT ON COLUMN public.pos_order_status_dim.status_name IS 'Canonical label for the raw status code.';
COMMENT ON COLUMN public.pos_order_status_dim.status_bucket IS 'High-level bucket: sales, void, or excluded.';
COMMENT ON COLUMN public.pos_order_status_dim.is_sales IS 'True only when the raw status belongs to the sales bucket.';
COMMENT ON COLUMN public.pos_order_status_dim.is_void IS 'True only when the raw status belongs to the void bucket.';
COMMENT ON COLUMN public.pos_order_status_dim.is_cancelled_like IS 'True for excluded statuses that behave like cancelled records in reporting semantics.';
COMMENT ON COLUMN public.pos_order_status_dim.is_excluded IS 'True when the raw status must be excluded from the primary sales fact path.';
COMMENT ON COLUMN public.pos_order_status_dim.description IS 'Human-readable explanation of the reporting semantics for this status.';
COMMENT ON COLUMN public.pos_order_status_dim.sort_order IS 'Stable display and review ordering for status rows.';
COMMENT ON COLUMN public.pos_order_status_dim.is_active IS 'Soft active flag for future status code lifecycle management.';
COMMENT ON COLUMN public.pos_order_status_dim.updated_at IS 'Audit timestamp for status dimension maintenance.';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pos_product_dim'
          AND column_name = 'cate_no'
    ) THEN
        COMMENT ON COLUMN public.pos_product_dim.cate_no IS 'Phase 2C schema contract category code. Product attribute only; do not duplicate into pos_sales_hourly_fact.';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pos_product_dim'
          AND column_name = 'cate_name'
    ) THEN
        COMMENT ON COLUMN public.pos_product_dim.cate_name IS 'Phase 2C schema contract category name. Product attribute only; do not duplicate into pos_sales_hourly_fact.';
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pos_branch_dim'
          AND column_name = 'group_code'
    ) THEN
        COMMENT ON COLUMN public.pos_branch_dim.group_code IS 'Phase 2C schema contract branch group code. Branch attribute only; branch options remain view or presentation derived.';
    END IF;
END $$;

COMMENT ON COLUMN public.pos_sales_hourly_fact.business_date IS 'Phase 2C schema contract: business_date is fixed to sale_period semantics. Do not use this column as tr_date or t_open_date; those dates belong to future order or payment facts.';

INSERT INTO ia_users (id, owner_user_key, display_name, source_system)
VALUES (1, 'demo-owner', 'Demo Owner', 'athena')
ON CONFLICT DO NOTHING;

INSERT INTO pos_order_type_dim (id, code, name, description, sort_order)
VALUES
    (0, 'unknown', '未知', '尚未辨識或無法映射的訂單型態', 0),
    (1, 'in_store', '來店', '對應 raw value 如來店、內用、店內點餐', 10),
    (2, 'foodpanda', 'Foodpanda', '對應 raw value 如熊貓、FOODPANDA、Foodpanda', 20),
    (3, 'delivery', '外送', '一般外送或未指名平台的配送訂單', 30),
    (4, 'pickup', '自取', '對應 raw value 如自取、到店自取', 40),
    (5, 'ubereats', 'Uber Eats', '對應 raw value 如UberEats、UBER EATS', 50),
    (6, 'quick_pickup', '快一點自取', '對應 raw value 如快一點自取', 60),
    (7, 'quick_delivery', '快一點外送', '對應 raw value 如快一點外送', 70),
    (8, 'qr_order', '掃碼點單', '對應 raw value 如掃碼點單、QR order', 80),
    (9, 'other', '其他', '保留給後續擴充的訂單型態', 90)
ON CONFLICT (id) DO UPDATE
SET
    code = EXCLUDED.code,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    sort_order = EXCLUDED.sort_order,
    is_active = TRUE;

INSERT INTO pos_payment_type_dim (id, code, name, description, sort_order)
VALUES
    (0, 'unknown_payment', '未知付款', '尚未辨識、空值或無法映射的付款型態', 0),
    (1, 'cash', '現金', '現金支付', 10),
    (2, 'card', '卡片', '信用卡、金融卡或卡片支付', 20),
    (3, 'e_wallet', '電子支付', '對應 raw value 如linepay、easycard、uupay 等', 30),
    (4, 'platform_payment', '平台付款', '對應 raw value 如FOODPANDA、UBER EATS 等平台代收', 40),
    (5, 'coupon', '票券折抵', '票券、折抵、禮券或優惠券類付款', 50),
    (8, 'mixed', '混合付款', '同一筆訂單經聚合後含多種付款型態', 80),
    (9, 'other', '其他', '保留給後續擴充的付款型態', 90)
ON CONFLICT (id) DO UPDATE
SET
    code = EXCLUDED.code,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    sort_order = EXCLUDED.sort_order,
    is_active = TRUE;

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
    (1, 'normal_sales', 'sales', TRUE, FALSE, FALSE, FALSE, '正常銷售主口徑', 10, TRUE, NOW()),
    (-2, 'void', 'void', FALSE, TRUE, FALSE, FALSE, '作廢主口徑', 20, TRUE, NOW()),
    (-1, 'cancelled_like', 'excluded', FALSE, FALSE, TRUE, TRUE, '排除於 sales / void 主口徑', 30, TRUE, NOW()),
    (2, 'other_excluded', 'excluded', FALSE, FALSE, FALSE, TRUE, '其他排除狀態', 40, TRUE, NOW())
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

COMMIT;