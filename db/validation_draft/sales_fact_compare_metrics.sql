-- Phase 2C-4.5 draft only.
-- Single-engine compare SQL draft.
-- This file does not execute across Athena and PostgreSQL.
-- Instead, it defines the compare shape by simulating source_metrics and target_metrics inputs via CTEs.
-- Required placeholders:
--   :owner_user_id
--   :start_date
--   :end_date
--
-- Hard gate deltas:
--   row_count_delta
--   gross_sales_milli_delta
--   discount_milli_delta
--   surcharge_milli_delta
--   net_sales_milli_delta
--   sales_ex_tax_milli_delta
--   tax_milli_delta
--   included_tax_milli_delta
--   excluded_tax_milli_delta
--   qty_milli_delta
--   item_count_delta
--
-- Warning gate deltas:
--   warning_rounding_delta_milli
--   warning_gate_note
-- Current Phase 2C-4 contract still treats persisted fact reconciliation as exact match.
-- Warning gates are placeholders for exploratory rounding diagnostics only.

WITH params AS (
    SELECT
        CAST(:owner_user_id AS BIGINT) AS owner_user_id,
        CAST(:start_date AS DATE) AS start_date,
        CAST(:end_date AS DATE) AS end_date
),
source_metrics_input AS (
    /*
      Replace this CTE with the output of sales_fact_source_metrics.sql.
      item_count is a hard gate here because it is a validation-only control metric.
    */
    SELECT
        p.owner_user_id,
        p.start_date AS sale_period,
        CAST(NULL AS BIGINT) AS row_count,
        CAST(NULL AS BIGINT) AS gross_sales_milli,
        CAST(NULL AS BIGINT) AS discount_milli,
        CAST(NULL AS BIGINT) AS surcharge_milli,
        CAST(NULL AS BIGINT) AS net_sales_milli,
        CAST(NULL AS BIGINT) AS sales_ex_tax_milli,
        CAST(NULL AS BIGINT) AS tax_milli,
        CAST(NULL AS BIGINT) AS included_tax_milli,
        CAST(NULL AS BIGINT) AS excluded_tax_milli,
        CAST(NULL AS BIGINT) AS qty_milli,
        CAST(NULL AS BIGINT) AS item_count,
        CAST(NULL AS BIGINT) AS warning_rounding_delta_milli,
        CAST(NULL AS TEXT) AS warning_gate_note
    FROM params p
    WHERE 1 = 0
),
target_metrics_input AS (
    /*
      Replace this CTE with either:
        1) output of sales_fact_target_metrics.sql for persisted fact compare, or
        2) a pre-insert target candidate metric set when item_count must be compared.

      target_scope must be one of:
        - persisted_fact
        - pre_insert_candidate
    */
    SELECT
        p.owner_user_id,
        p.start_date AS sale_period,
        CAST('persisted_fact' AS TEXT) AS target_scope,
        CAST(NULL AS BIGINT) AS row_count,
        CAST(NULL AS BIGINT) AS gross_sales_milli,
        CAST(NULL AS BIGINT) AS discount_milli,
        CAST(NULL AS BIGINT) AS surcharge_milli,
        CAST(NULL AS BIGINT) AS net_sales_milli,
        CAST(NULL AS BIGINT) AS sales_ex_tax_milli,
        CAST(NULL AS BIGINT) AS tax_milli,
        CAST(NULL AS BIGINT) AS included_tax_milli,
        CAST(NULL AS BIGINT) AS excluded_tax_milli,
        CAST(NULL AS BIGINT) AS qty_milli,
        CAST(NULL AS BIGINT) AS item_count,
        CAST(NULL AS BIGINT) AS warning_rounding_delta_milli,
        CAST(NULL AS TEXT) AS warning_gate_note
    FROM params p
    WHERE 1 = 0
),
compare_input AS (
    SELECT
        COALESCE(s.owner_user_id, t.owner_user_id) AS owner_user_id,
        COALESCE(s.sale_period, t.sale_period) AS sale_period,
        t.target_scope,
        s.row_count AS source_row_count,
        t.row_count AS target_row_count,
        s.gross_sales_milli AS source_gross_sales_milli,
        t.gross_sales_milli AS target_gross_sales_milli,
        s.discount_milli AS source_discount_milli,
        t.discount_milli AS target_discount_milli,
        s.surcharge_milli AS source_surcharge_milli,
        t.surcharge_milli AS target_surcharge_milli,
        s.net_sales_milli AS source_net_sales_milli,
        t.net_sales_milli AS target_net_sales_milli,
        s.sales_ex_tax_milli AS source_sales_ex_tax_milli,
        t.sales_ex_tax_milli AS target_sales_ex_tax_milli,
        s.tax_milli AS source_tax_milli,
        t.tax_milli AS target_tax_milli,
        s.included_tax_milli AS source_included_tax_milli,
        t.included_tax_milli AS target_included_tax_milli,
        s.excluded_tax_milli AS source_excluded_tax_milli,
        t.excluded_tax_milli AS target_excluded_tax_milli,
        s.qty_milli AS source_qty_milli,
        t.qty_milli AS target_qty_milli,
        s.item_count AS source_item_count,
        t.item_count AS target_item_count,
        COALESCE(s.warning_rounding_delta_milli, t.warning_rounding_delta_milli) AS warning_rounding_delta_milli,
        COALESCE(s.warning_gate_note, t.warning_gate_note) AS warning_gate_note
    FROM source_metrics_input s
    FULL OUTER JOIN target_metrics_input t
      ON t.owner_user_id = s.owner_user_id
     AND t.sale_period = s.sale_period
),
compare_metrics AS (
    SELECT
        owner_user_id,
        sale_period,
        target_scope,
        source_row_count,
        target_row_count,
        source_gross_sales_milli,
        target_gross_sales_milli,
        source_discount_milli,
        target_discount_milli,
        source_surcharge_milli,
        target_surcharge_milli,
        source_net_sales_milli,
        target_net_sales_milli,
        source_sales_ex_tax_milli,
        target_sales_ex_tax_milli,
        source_tax_milli,
        target_tax_milli,
        source_included_tax_milli,
        target_included_tax_milli,
        source_excluded_tax_milli,
        target_excluded_tax_milli,
        source_qty_milli,
        target_qty_milli,
        source_item_count,
        target_item_count,
        source_row_count - target_row_count AS row_count_delta, -- hard gate
        source_gross_sales_milli - target_gross_sales_milli AS gross_sales_milli_delta, -- hard gate
        source_discount_milli - target_discount_milli AS discount_milli_delta, -- hard gate
        source_surcharge_milli - target_surcharge_milli AS surcharge_milli_delta, -- hard gate
        source_net_sales_milli - target_net_sales_milli AS net_sales_milli_delta, -- hard gate
        source_sales_ex_tax_milli - target_sales_ex_tax_milli AS sales_ex_tax_milli_delta, -- hard gate
        source_tax_milli - target_tax_milli AS tax_milli_delta, -- hard gate
        source_included_tax_milli - target_included_tax_milli AS included_tax_milli_delta, -- hard gate
        source_excluded_tax_milli - target_excluded_tax_milli AS excluded_tax_milli_delta, -- hard gate
        source_qty_milli - target_qty_milli AS qty_milli_delta, -- hard gate
        CASE
            WHEN target_scope = 'pre_insert_candidate' THEN source_item_count - target_item_count
            ELSE NULL
        END AS item_count_delta, -- hard gate only for pre_insert_candidate compare
        CASE
            WHEN source_row_count IS NULL OR target_row_count IS NULL THEN false
            WHEN source_row_count = target_row_count THEN true
            ELSE false
        END AS row_count_hard_gate_pass,
        CASE
            WHEN source_gross_sales_milli IS NULL OR target_gross_sales_milli IS NULL THEN false
            WHEN source_gross_sales_milli = target_gross_sales_milli THEN true
            ELSE false
        END AS gross_sales_hard_gate_pass,
        CASE
            WHEN source_discount_milli IS NULL OR target_discount_milli IS NULL THEN false
            WHEN source_discount_milli = target_discount_milli THEN true
            ELSE false
        END AS discount_hard_gate_pass,
        CASE
            WHEN source_surcharge_milli IS NULL OR target_surcharge_milli IS NULL THEN false
            WHEN source_surcharge_milli = target_surcharge_milli THEN true
            ELSE false
        END AS surcharge_hard_gate_pass,
        CASE
            WHEN source_net_sales_milli IS NULL OR target_net_sales_milli IS NULL THEN false
            WHEN source_net_sales_milli = target_net_sales_milli THEN true
            ELSE false
        END AS net_sales_hard_gate_pass,
        CASE
            WHEN source_sales_ex_tax_milli IS NULL OR target_sales_ex_tax_milli IS NULL THEN false
            WHEN source_sales_ex_tax_milli = target_sales_ex_tax_milli THEN true
            ELSE false
        END AS sales_ex_tax_hard_gate_pass,
        CASE
            WHEN source_tax_milli IS NULL OR target_tax_milli IS NULL THEN false
            WHEN source_tax_milli = target_tax_milli THEN true
            ELSE false
        END AS tax_hard_gate_pass,
        CASE
            WHEN source_included_tax_milli IS NULL OR target_included_tax_milli IS NULL THEN false
            WHEN source_included_tax_milli = target_included_tax_milli THEN true
            ELSE false
        END AS included_tax_hard_gate_pass,
        CASE
            WHEN source_excluded_tax_milli IS NULL OR target_excluded_tax_milli IS NULL THEN false
            WHEN source_excluded_tax_milli = target_excluded_tax_milli THEN true
            ELSE false
        END AS excluded_tax_hard_gate_pass,
        CASE
            WHEN source_qty_milli IS NULL OR target_qty_milli IS NULL THEN false
            WHEN source_qty_milli = target_qty_milli THEN true
            ELSE false
        END AS qty_hard_gate_pass,
        CASE
            WHEN target_scope = 'pre_insert_candidate'
                 AND source_item_count IS NOT NULL
                 AND target_item_count IS NOT NULL
                 AND source_item_count = target_item_count THEN true
            WHEN target_scope = 'pre_insert_candidate' THEN false
            ELSE NULL
        END AS item_count_hard_gate_pass,
        CASE
            WHEN warning_rounding_delta_milli IS NULL THEN false
            WHEN ABS(warning_rounding_delta_milli) <= 1 THEN false
            ELSE true
        END AS warning_gate_failed, -- warning gate only; default should remain false
        COALESCE(
            warning_gate_note,
            CASE
                WHEN target_scope = 'persisted_fact' THEN 'item_count compare is intentionally skipped for persisted_fact; compare it against pre_insert_candidate metrics instead.'
                ELSE NULL
            END
        ) AS resolved_warning_gate_note,
        CASE
            WHEN source_row_count IS NULL OR target_row_count IS NULL THEN true
            WHEN source_row_count <> target_row_count THEN true
            WHEN source_gross_sales_milli <> target_gross_sales_milli THEN true
            WHEN source_discount_milli <> target_discount_milli THEN true
            WHEN source_surcharge_milli <> target_surcharge_milli THEN true
            WHEN source_net_sales_milli <> target_net_sales_milli THEN true
            WHEN source_sales_ex_tax_milli <> target_sales_ex_tax_milli THEN true
            WHEN source_tax_milli <> target_tax_milli THEN true
            WHEN source_included_tax_milli <> target_included_tax_milli THEN true
            WHEN source_excluded_tax_milli <> target_excluded_tax_milli THEN true
            WHEN source_qty_milli <> target_qty_milli THEN true
            WHEN target_scope = 'pre_insert_candidate'
                 AND (source_item_count IS NULL OR target_item_count IS NULL OR source_item_count <> target_item_count) THEN true
            ELSE false
        END AS hard_gate_failed
    FROM compare_input
)
SELECT *
FROM compare_metrics
ORDER BY owner_user_id, sale_period;
