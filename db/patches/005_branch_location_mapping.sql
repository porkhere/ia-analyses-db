BEGIN;

-- 分店地理與 CWA 對照獨立存放，保留歷史版本，不碰既有分店維度。
CREATE TABLE IF NOT EXISTS public.ia_branch_location_mapping (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES public.ia_users(id),
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

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_owner_branch
    ON public.ia_branch_location_mapping (owner_user_id, branch_id, valid_from DESC);

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_verified_current
    ON public.ia_branch_location_mapping (owner_user_id, branch_id, valid_from DESC)
    WHERE verification_status = 'verified' AND valid_to IS NULL;

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

COMMIT;
