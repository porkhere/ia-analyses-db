-- Phase 2C-3-pre rollback draft only.
-- Do not execute in this round.
-- This rollback is intentionally conservative: it removes the new status table and clears comments,
-- but does not drop newly added nullable columns because this phase explicitly forbids column drops.

COMMENT ON COLUMN pos_sales_hourly_fact.business_date IS NULL;
COMMENT ON COLUMN pos_product_dim.cate_no IS NULL;
COMMENT ON COLUMN pos_product_dim.cate_name IS NULL;
COMMENT ON COLUMN pos_branch_dim.group_code IS NULL;

DROP TABLE IF EXISTS pos_order_status_dim;
