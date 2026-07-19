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

-- 分店地理與 CWA 對照獨立存放，保留歷史版本，不碰既有分店維度。
CREATE TABLE IF NOT EXISTS ia_branch_location_mapping (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    branch_id TEXT NOT NULL,
    address TEXT,
    city_county TEXT,
    township_district TEXT,
    postal_code TEXT,
    township_code TEXT,
    station_id TEXT,
    latitude NUMERIC,
    longitude NUMERIC,
    distance_meters NUMERIC,
    source_type TEXT NOT NULL,
    source_reference TEXT NOT NULL,
    source_metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    verification_status TEXT NOT NULL,
    verified_at TIMESTAMPTZ,
    verified_by TEXT,
    valid_from DATE NOT NULL,
    valid_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_branch_location_mapping_version_grain
        UNIQUE (owner_user_id, branch_id, valid_from),
    CONSTRAINT ia_branch_location_mapping_valid_period_check
        CHECK (valid_to IS NULL OR valid_to >= valid_from),
    CONSTRAINT ia_branch_location_mapping_coordinates_pair_check
        CHECK ((latitude IS NULL AND longitude IS NULL)
            OR (latitude IS NOT NULL AND longitude IS NOT NULL)),
    CONSTRAINT ia_branch_location_mapping_latitude_bounds_check
        CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    CONSTRAINT ia_branch_location_mapping_longitude_bounds_check
        CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
    CONSTRAINT ia_branch_location_mapping_distance_check
        CHECK (distance_meters IS NULL
            OR (distance_meters <> 'NaN'::NUMERIC AND distance_meters >= 0)),
    CONSTRAINT ia_branch_location_mapping_source_type_check
        CHECK (BTRIM(source_type) <> ''),
    CONSTRAINT ia_branch_location_mapping_source_reference_check
        CHECK (BTRIM(source_reference) <> ''),
    CONSTRAINT ia_branch_location_mapping_verification_status_check
        CHECK (verification_status IN ('unverified', 'verified', 'needs_review', 'rejected')),
    CONSTRAINT ia_branch_location_mapping_verified_fields_check
        CHECK (verification_status <> 'verified'
            OR (verified_at IS NOT NULL AND NULLIF(BTRIM(verified_by), '') IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS ia_branch_location_mapping_import_audit (
    import_id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    customer TEXT NOT NULL,
    mode TEXT NOT NULL,
    source_reference TEXT NOT NULL,
    source_sha256 TEXT NOT NULL,
    row_count INTEGER NOT NULL,
    previous_current_count INTEGER NOT NULL,
    closed_count INTEGER NOT NULL,
    inserted_count INTEGER NOT NULL,
    previous_mappings JSONB NOT NULL DEFAULT '[]'::JSONB,
    imported_mappings JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_branch_location_mapping_import_audit_mode_check
        CHECK (mode = 'replace'),
    CONSTRAINT ia_branch_location_mapping_import_audit_source_check
        CHECK (BTRIM(source_reference) <> '' AND BTRIM(source_sha256) <> ''),
    CONSTRAINT ia_branch_location_mapping_import_audit_counts_check
        CHECK (row_count > 0
            AND previous_current_count >= 0
            AND closed_count >= 0
            AND inserted_count >= 0)
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

-- IA Signals：獨立於 sales fact 的訊號持久層（Phase 2 Task 2.1）。
-- 這三張表只放外部/營運訊號本身，不會、也不應該加欄位到 pos_sales_hourly_fact 或其他 sales fact 表。

CREATE TABLE IF NOT EXISTS ia_signal_weather (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    location TEXT,
    signal_date DATE NOT NULL,
    observation_kind TEXT NOT NULL,
    temperature_c NUMERIC,
    rain_mm NUMERIC,
    humidity_pct NUMERIC,
    is_typhoon BOOLEAN,
    source TEXT,
    captured_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_signal_weather_observation_kind_check
        CHECK (observation_kind IN ('forecast', 'actual')),
    CONSTRAINT ia_signal_weather_grain
        UNIQUE NULLS NOT DISTINCT (owner_user_id, location, signal_date, observation_kind)
);

CREATE TABLE IF NOT EXISTS ia_signal_promotion (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    location TEXT,
    item TEXT,
    signal_date DATE NOT NULL,
    is_promotion BOOLEAN NOT NULL DEFAULT FALSE,
    promo_type TEXT,
    discount_pct NUMERIC,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_signal_promotion_grain
        UNIQUE NULLS NOT DISTINCT (owner_user_id, location, item, signal_date)
);

CREATE TABLE IF NOT EXISTS ia_signal_availability (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES ia_users(id),
    location TEXT,
    item TEXT NOT NULL,
    signal_date DATE NOT NULL,
    is_stockout BOOLEAN NOT NULL DEFAULT FALSE,
    is_delisted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_signal_availability_grain
        UNIQUE NULLS NOT DISTINCT (owner_user_id, location, item, signal_date)
);

CREATE INDEX IF NOT EXISTS idx_ia_users_active
    ON ia_users (is_active, owner_user_key);

CREATE INDEX IF NOT EXISTS idx_pos_product_dim_lookup
    ON pos_product_dim (owner_user_id, product_no, is_active);

CREATE INDEX IF NOT EXISTS idx_pos_branch_dim_lookup
    ON pos_branch_dim (owner_user_id, branch_id, is_active);

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_owner_branch
    ON ia_branch_location_mapping (owner_user_id, branch_id, valid_from DESC);

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_verified_current
    ON ia_branch_location_mapping (owner_user_id, branch_id, valid_from DESC)
    WHERE verification_status = 'verified' AND valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_import_audit_owner_created
    ON ia_branch_location_mapping_import_audit (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_date
    ON pos_sales_hourly_fact (owner_user_id, business_date, hour_of_day);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_branch
    ON pos_sales_hourly_fact (owner_user_id, business_date, branch_id);

CREATE INDEX IF NOT EXISTS idx_pos_sales_hourly_fact_product
    ON pos_sales_hourly_fact (owner_user_id, business_date, product_no);

CREATE INDEX IF NOT EXISTS idx_ia_signal_weather_owner_date
    ON ia_signal_weather (owner_user_id, signal_date, observation_kind);

CREATE INDEX IF NOT EXISTS idx_ia_signal_promotion_owner_date
    ON ia_signal_promotion (owner_user_id, signal_date);

CREATE INDEX IF NOT EXISTS idx_ia_signal_availability_owner_date
    ON ia_signal_availability (owner_user_id, signal_date);

CREATE INDEX IF NOT EXISTS idx_ia_signal_availability_item
    ON ia_signal_availability (owner_user_id, item, signal_date);

COMMENT ON TABLE public.ia_signal_weather IS
    'IA Signals：天氣訊號，獨立表，不併入 sales fact。observation_kind 是防資料洩漏的物理保證。';
COMMENT ON COLUMN public.ia_signal_weather.location IS
    '對應現有 branch_id 概念，這裡改用 generic commerce naming 的 location；NULL 代表租戶層級（不分特定門市）的訊號。';
COMMENT ON COLUMN public.ia_signal_weather.observation_kind IS
    '只允許 forecast 或 actual 兩種值，已用 CHECK constraint physically 限制。forecast frame 組裝時只准 JOIN observation_kind=''forecast'' 的資料；observation_kind=''actual'' 是事後回填的實際觀測，只能用於 retrospective（回測）情境，不可進 forecast frame，避免把未來才會知道的實際天氣資料洩漏進預測路徑。';
COMMENT ON COLUMN public.ia_signal_weather.source IS
    '訊號來源標註（例如資料供應商名稱或人工整理批次），供追溯與品質查核使用。';
COMMENT ON COLUMN public.ia_signal_weather.captured_at IS
    '這筆訊號實際被擷取/寫入的時間點，跟 signal_date（訊號描述的業務日期）分開記錄。';

COMMENT ON TABLE public.ia_signal_promotion IS
    'IA Signals：促銷排程訊號，獨立表，不併入 sales fact。已排定的促銷屬於 known_ahead，可用於 forecast frame。';
COMMENT ON COLUMN public.ia_signal_promotion.location IS
    '對應現有 branch_id 概念；NULL 代表全租戶（跨店）層級的促銷。';
COMMENT ON COLUMN public.ia_signal_promotion.item IS
    '對應現有 product_no 概念；NULL 代表不分品項、整店或整租戶層級的促銷。';
COMMENT ON COLUMN public.ia_signal_promotion.is_promotion IS
    '該租戶／門市／品項在該日是否處於促銷排程中。';
COMMENT ON COLUMN public.ia_signal_promotion.promo_type IS
    '促銷型態的自由文字標註（例如滿額折扣、買一送一），目前尚未定義 canonical enum。';
COMMENT ON COLUMN public.ia_signal_promotion.discount_pct IS
    '折扣百分比，允許 NULL（例如非折扣類促銷）。';

COMMENT ON TABLE public.ia_signal_availability IS
    'IA Signals：品項供應狀態訊號，獨立表，不併入 sales fact。屬 actual_only，只能用於 retrospective；賣不好不等於需求低，只能做事後解釋，不能拿來做 forecast。';
COMMENT ON COLUMN public.ia_signal_availability.location IS
    '對應現有 branch_id 概念；NULL 代表全租戶（跨店）層級的狀態（例如整體下架）。';
COMMENT ON COLUMN public.ia_signal_availability.item IS
    '對應現有 product_no 概念；availability 一定綁定特定品項，因此本欄位為 NOT NULL。';
COMMENT ON COLUMN public.ia_signal_availability.is_stockout IS
    '該品項在該日該門市（或全租戶，視 location 是否為 NULL）是否缺貨。';
COMMENT ON COLUMN public.ia_signal_availability.is_delisted IS
    '該品項在該日是否已下架（可能是單店或跨店層級，視 location 是否為 NULL）。';

COMMENT ON TABLE public.ia_branch_location_mapping IS
    '分店地理與 CWA 測站對照表；這張表自己管版本與驗證，不綁 pos_branch_dim，方便保留歷史對照。';
COMMENT ON COLUMN public.ia_branch_location_mapping.owner_user_id IS
    '租戶範圍，所有對照資料都必須掛在 ia_users.id 底下。';
COMMENT ON COLUMN public.ia_branch_location_mapping.branch_id IS
    '來源分店代碼，只保存對照用文字，不對 pos_branch_dim 建外鍵，避免維度變動吃掉歷史資料。';
COMMENT ON COLUMN public.ia_branch_location_mapping.source_metadata IS
    '來源補充資訊，使用 JSONB 保存原始查詢或人工查核所需的結構化中繼資料；沒有補充資訊時用空物件。';
COMMENT ON COLUMN public.ia_branch_location_mapping.verification_status IS
    '只能是 unverified、verified、needs_review 或 rejected；標成 verified 時必須同時留下 verified_at 與非空 verified_by。';
COMMENT ON COLUMN public.ia_branch_location_mapping.valid_from IS
    '這個對照版本開始生效的日期；與 owner_user_id、branch_id 組成版本唯一粒度。';
COMMENT ON COLUMN public.ia_branch_location_mapping.valid_to IS
    '這個對照版本結束生效的日期；NULL 表示目前仍開放，且不得早於 valid_from。';

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