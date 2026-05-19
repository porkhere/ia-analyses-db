-- Phase 2C-4.5 draft only.
-- This file is an Athena / source candidate validation SQL draft.
-- It is not expected to run directly in PostgreSQL without replacing the placeholder source CTE.
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
--   item_count
--   status_1_rows
--   non_status_1_rows
--   latest_status_rows
--
-- Warning gate metrics:
--   warning_rounding_delta_milli
--   warning_gate_note
-- Current Phase 2C-4 contract does not define any required warning gate metric by default.

WITH params AS (
    SELECT
        CAST(:owner_user_id AS BIGINT) AS owner_user_id,
        CAST(:start_date AS DATE) AS start_date,
        CAST(:end_date AS DATE) AS end_date
),
sales_fact_source_candidate AS (
    /*
      Replace this SELECT with the real Phase 2B status-aware sales candidate query.
      Required output columns:
        owner_user_id
        sale_period
        status
        is_latest_status_row
        gross_sales_milli
        discount_milli
        surcharge_milli
        net_sales_milli
        sales_ex_tax_milli
        tax_milli
        included_tax_milli
        excluded_tax_milli
        qty_milli
        item_count

      The placeholder relation name below is intentional and marks this file as
      Athena/source-candidate oriented draft SQL.
    */
    SELECT
        c.owner_user_id,
        c.sale_period,
        c.status,
        c.is_latest_status_row,
        c.gross_sales_milli,
        c.discount_milli,
        c.surcharge_milli,
        c.net_sales_milli,
        c.sales_ex_tax_milli,
        c.tax_milli,
        c.included_tax_milli,
        c.excluded_tax_milli,
        c.qty_milli,
        c.item_count
    FROM athena_sales_fact_source_candidate_draft c
    JOIN params p
      ON p.owner_user_id = c.owner_user_id
    WHERE c.sale_period BETWEEN p.start_date AND p.end_date
),
source_metrics AS (
    SELECT
        owner_user_id,
        sale_period,
        COUNT(*) AS row_count, -- hard gate
        SUM(gross_sales_milli) AS gross_sales_milli, -- hard gate
        SUM(discount_milli) AS discount_milli, -- hard gate
        SUM(surcharge_milli) AS surcharge_milli, -- hard gate
        SUM(net_sales_milli) AS net_sales_milli, -- hard gate
        SUM(sales_ex_tax_milli) AS sales_ex_tax_milli, -- hard gate
        SUM(tax_milli) AS tax_milli, -- hard gate
        SUM(included_tax_milli) AS included_tax_milli, -- hard gate
        SUM(excluded_tax_milli) AS excluded_tax_milli, -- hard gate
        SUM(qty_milli) AS qty_milli, -- hard gate
        SUM(item_count) AS item_count, -- hard gate; validation-only control metric
        SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS status_1_rows, -- hard gate
        SUM(CASE WHEN status <> 1 THEN 1 ELSE 0 END) AS non_status_1_rows, -- hard gate; must be 0
        SUM(CASE WHEN is_latest_status_row THEN 1 ELSE 0 END) AS latest_status_rows, -- hard gate; must equal status_1_rows
        CAST(NULL AS BIGINT) AS warning_rounding_delta_milli, -- warning gate placeholder
        CAST(NULL AS VARCHAR) AS warning_gate_note -- warning gate placeholder
    FROM sales_fact_source_candidate
    GROUP BY owner_user_id, sale_period
)
SELECT *
FROM source_metrics
ORDER BY owner_user_id, sale_period;
