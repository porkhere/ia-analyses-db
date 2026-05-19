-- Phase 2C-4.5 draft only.
-- PostgreSQL target metrics validation SQL for public.pos_sales_hourly_fact.
-- Required placeholders:
--   :owner_user_id
--   :start_date
--   :end_date
--
-- Hard gate metrics:
--   row_count
--   gross_sales_milli
--   discount_milli
--   surcharge_milli
--   net_sales_milli
--   sales_ex_tax_milli
--   tax_milli
--   included_tax_milli
--   excluded_tax_milli
--   qty_milli
--
-- Warning gate metrics:
--   warning_rounding_delta_milli
--   warning_gate_note
-- Current Phase 2C-4 contract does not define any required warning gate metric by default.
-- item_count is intentionally excluded here because it is not a persisted pos_sales_hourly_fact column.

WITH params AS (
    SELECT
        CAST(:owner_user_id AS BIGINT) AS owner_user_id,
        CAST(:start_date AS DATE) AS start_date,
        CAST(:end_date AS DATE) AS end_date
),
target_metrics AS (
    SELECT
        f.owner_user_id,
        f.business_date AS sale_period,
        COUNT(*) AS row_count, -- hard gate
        SUM(f.gross_sales_milli) AS gross_sales_milli, -- hard gate
        SUM(f.discount_milli) AS discount_milli, -- hard gate
        SUM(f.surcharge_milli) AS surcharge_milli, -- hard gate
        SUM(f.net_sales_milli) AS net_sales_milli, -- hard gate
        SUM(f.sales_ex_tax_milli) AS sales_ex_tax_milli, -- hard gate
        SUM(f.tax_milli) AS tax_milli, -- hard gate
        SUM(f.included_tax_milli) AS included_tax_milli, -- hard gate
        SUM(f.excluded_tax_milli) AS excluded_tax_milli, -- hard gate
        SUM(f.qty_milli) AS qty_milli, -- hard gate
        CAST(NULL AS BIGINT) AS warning_rounding_delta_milli, -- warning gate placeholder
        CAST(NULL AS TEXT) AS warning_gate_note -- warning gate placeholder
    FROM public.pos_sales_hourly_fact f
    JOIN params p
      ON p.owner_user_id = f.owner_user_id
    WHERE f.business_date BETWEEN p.start_date AND p.end_date
    GROUP BY f.owner_user_id, f.business_date
)
SELECT *
FROM target_metrics
ORDER BY owner_user_id, sale_period;
