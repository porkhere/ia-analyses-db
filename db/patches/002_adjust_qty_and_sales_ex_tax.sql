BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pos_sales_hourly_fact'
          AND column_name = 'qty_total'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pos_sales_hourly_fact'
          AND column_name = 'qty_milli'
    ) THEN
        ALTER TABLE public.pos_sales_hourly_fact RENAME COLUMN qty_total TO qty_milli;
        UPDATE public.pos_sales_hourly_fact SET qty_milli = qty_milli * 1000;
    END IF;
END $$;

ALTER TABLE public.pos_sales_hourly_fact
    ADD COLUMN IF NOT EXISTS discount_milli BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS surcharge_milli BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sales_ex_tax_milli BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS included_tax_milli BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS excluded_tax_milli BIGINT NOT NULL DEFAULT 0;

COMMIT;