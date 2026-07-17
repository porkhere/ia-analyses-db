BEGIN;

-- IA Signals：獨立於 sales fact 的訊號持久層（Phase 2 Task 2.1）。
-- 這三張表只放外部/營運訊號本身，不會、也不應該加欄位到 pos_sales_hourly_fact 或其他 sales fact 表。

CREATE TABLE IF NOT EXISTS public.ia_signal_weather (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES public.ia_users(id),
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

CREATE TABLE IF NOT EXISTS public.ia_signal_promotion (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES public.ia_users(id),
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

CREATE TABLE IF NOT EXISTS public.ia_signal_availability (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES public.ia_users(id),
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

CREATE INDEX IF NOT EXISTS idx_ia_signal_weather_owner_date
    ON public.ia_signal_weather (owner_user_id, signal_date, observation_kind);

CREATE INDEX IF NOT EXISTS idx_ia_signal_promotion_owner_date
    ON public.ia_signal_promotion (owner_user_id, signal_date);

CREATE INDEX IF NOT EXISTS idx_ia_signal_availability_owner_date
    ON public.ia_signal_availability (owner_user_id, signal_date);

CREATE INDEX IF NOT EXISTS idx_ia_signal_availability_item
    ON public.ia_signal_availability (owner_user_id, item, signal_date);

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

COMMIT;
