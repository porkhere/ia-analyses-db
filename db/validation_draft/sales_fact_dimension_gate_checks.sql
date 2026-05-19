-- Phase 2C-4.6 draft only.
-- Dimension and source-path validation gates for sales fact write candidate.
-- This is a shape-first draft and is not executed in this phase.
-- Required placeholders:
--   :owner_user_id
--   :start_date
--   :end_date
--
-- Hard gate outputs:
--   product_dim_miss_count
--   branch_dim_miss_count
--   order_type_dim_miss_count
--   payment_type_dim_miss_count
--   business_date_not_equal_sale_period_count
--   non_status_1_count
--   not_latest_status_count
--
-- Warning gate outputs:
--   warning_gate_note
-- Current Phase 2C-4 contract does not define any mandatory warning gate here.

WITH params AS (
    SELECT
        CAST(:owner_user_id AS BIGINT) AS owner_user_id,
        CAST(:start_date AS DATE) AS start_date,
        CAST(:end_date AS DATE) AS end_date
),
sales_fact_target_candidate AS (
    /*
      Replace this CTE with the real pre-insert sales fact candidate query.
      Required columns:
        owner_user_id
        sale_period
        business_date
        product_no
        branch_id
        order_type_id
        payment_type_id
        status
        is_latest_status_row
    */
    SELECT
        c.owner_user_id,
        c.sale_period,
        c.business_date,
        c.product_no,
        c.branch_id,
        c.order_type_id,
        c.payment_type_id,
        c.status,
        c.is_latest_status_row
    FROM sales_fact_target_candidate_draft c
    JOIN params p
      ON p.owner_user_id = c.owner_user_id
    WHERE c.sale_period BETWEEN p.start_date AND p.end_date
),
dimension_gate_checks AS (
    SELECT
        c.owner_user_id,
        c.sale_period,
        SUM(CASE WHEN pd.id IS NULL THEN 1 ELSE 0 END) AS product_dim_miss_count, -- hard gate
        SUM(CASE WHEN bd.id IS NULL THEN 1 ELSE 0 END) AS branch_dim_miss_count, -- hard gate
        SUM(CASE WHEN otd.id IS NULL THEN 1 ELSE 0 END) AS order_type_dim_miss_count, -- hard gate
        SUM(CASE WHEN ptd.id IS NULL THEN 1 ELSE 0 END) AS payment_type_dim_miss_count, -- hard gate
        SUM(CASE WHEN c.business_date IS DISTINCT FROM c.sale_period THEN 1 ELSE 0 END) AS business_date_not_equal_sale_period_count, -- hard gate
        SUM(CASE WHEN c.status <> 1 THEN 1 ELSE 0 END) AS non_status_1_count, -- hard gate
        SUM(CASE WHEN COALESCE(c.is_latest_status_row, false) = false THEN 1 ELSE 0 END) AS not_latest_status_count, -- hard gate
        CAST(NULL AS TEXT) AS warning_gate_note -- warning gate placeholder
    FROM sales_fact_target_candidate c
    LEFT JOIN public.pos_product_dim pd
      ON pd.owner_user_id = c.owner_user_id
     AND pd.product_no = c.product_no
    LEFT JOIN public.pos_branch_dim bd
      ON bd.owner_user_id = c.owner_user_id
     AND bd.branch_id = c.branch_id
    LEFT JOIN public.pos_order_type_dim otd
      ON otd.id = c.order_type_id
    LEFT JOIN public.pos_payment_type_dim ptd
      ON ptd.id = c.payment_type_id
    GROUP BY c.owner_user_id, c.sale_period
)
SELECT *
FROM dimension_gate_checks
ORDER BY owner_user_id, sale_period;
