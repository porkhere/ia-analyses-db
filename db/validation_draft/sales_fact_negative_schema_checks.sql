-- Phase 2C-4.6 draft only.
-- Negative schema checks for public.pos_sales_hourly_fact.
-- This file targets PostgreSQL information_schema only and is not executed in this phase.
--
-- Hard gate outputs:
--   forbidden_column_count
--   forbidden_column_names
-- Any non-zero forbidden_column_count means the sales fact schema contract is violated.
--
-- Warning gate outputs:
--   warning_gate_note
-- Current Phase 2C-4 contract does not define any mandatory warning gate here.

WITH forbidden_columns AS (
    SELECT 'raw_payment_name' AS column_name
    UNION ALL SELECT 'raw_payment_memo1'
    UNION ALL SELECT 'item_count'
    UNION ALL SELECT 'void_milli'
    UNION ALL SELECT 'refund_milli'
    UNION ALL SELECT 'order_count'
    UNION ALL SELECT 'completed_order_count'
    UNION ALL SELECT 'void_order_count'
    UNION ALL SELECT 'refund_order_count'
    UNION ALL SELECT 'cancelled_order_count'
    UNION ALL SELECT 'tr_date'
    UNION ALL SELECT 't_open_date'
    UNION ALL SELECT 'void_sale_period'
    UNION ALL SELECT 'order_num'
),
found_forbidden_columns AS (
    SELECT f.column_name
    FROM forbidden_columns f
    JOIN information_schema.columns c
      ON c.table_schema = 'public'
     AND c.table_name = 'pos_sales_hourly_fact'
     AND c.column_name = f.column_name
)
SELECT
    COUNT(*) AS forbidden_column_count, -- hard gate
    COALESCE(string_agg(column_name, ',' ORDER BY column_name), '') AS forbidden_column_names, -- hard gate
    CAST(NULL AS TEXT) AS warning_gate_note -- warning gate placeholder
FROM found_forbidden_columns;
