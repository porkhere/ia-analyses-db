# Phase 2C Schema Migration Review

## 目的

本文件審查 [schema_migration_plan_phase2c.md](schema_migration_plan_phase2c.md) 對應的 Phase 2C-3-pre migration SQL draft，確認 forward draft 與 rollback draft 的邊界符合本輪限制。

對應草案檔：

- [20260515_phase2c_schema_contract.sql](../db/migrations_draft/20260515_phase2c_schema_contract.sql)
- [20260515_phase2c_schema_contract_rollback.sql](../db/migrations_draft/20260515_phase2c_schema_contract_rollback.sql)

## 本輪範圍確認

Phase 2C-3-pre 只建立 migration 草案，不執行 migration，不修改現有 DB。

本輪 forward draft 僅包含以下操作：

- `CREATE TABLE pos_order_status_dim`
- seed `pos_order_status_dim`
- `ALTER TABLE pos_product_dim ADD COLUMN cate_no text NULL`
- `ALTER TABLE pos_product_dim ADD COLUMN cate_name text NULL`
- `ALTER TABLE pos_branch_dim ADD COLUMN group_code text NULL`
- `COMMENT ON TABLE / COMMENT ON COLUMN`

本輪明確沒有做以下事情：

- 不執行 migration
- 不修改 [001_schema.sql](../db/init/001_schema.sql)
- 不修改 Go
- 不改 `sync-athena`
- 不跑 Athena
- 不寫 PostgreSQL
- 不 drop 欄位
- 不新增 order / payment / condiment / branch opening fact

## Forward Draft 審查

### 1. pos_order_status_dim

forward draft 建立 `pos_order_status_dim`，欄位與 [schema_migration_plan_phase2c.md](schema_migration_plan_phase2c.md) 一致：

- `status_code`
- `status_name`
- `status_bucket`
- `is_sales`
- `is_void`
- `is_cancelled_like`
- `is_excluded`
- `description`
- `sort_order`
- `is_active`
- `updated_at`

seed 也與 plan 一致：

- `1 -> normal_sales / sales`
- `-2 -> void / void`
- `-1 -> cancelled_like / excluded`
- `2 -> other_excluded / excluded`

判定：

- 符合「先固定 raw status 語意，不急著建立 FK」的 Phase 2C-2 原則。
- 沒有擴張到 order fact 或 status-aware metrics 寫入。

### 2. pos_product_dim

forward draft 只新增：

- `cate_no text NULL`
- `cate_name text NULL`

判定：

- 符合 product dim 補 category contract 的設計。
- 沒有把 `cate_no` / `cate_name` 塞進 `pos_sales_hourly_fact`。
- 沒有附帶 index、backfill、或資料修正邏輯。

### 3. pos_branch_dim

forward draft 只新增：

- `group_code text NULL`

判定：

- 符合 branch dim 補 group code contract 的設計。
- 沒有把 `options` 寫進 schema。
- 沒有附帶來源猜測或回填邏輯。

### 4. pos_sales_hourly_fact.business_date 註解

forward draft 對 `pos_sales_hourly_fact.business_date` 加上 comment，固定：

- `business_date = sale_period`
- 不可當 `tr_date`
- 不可當 `t_open_date`

判定：

- 符合 Phase 2C-1 contract。
- 這是低風險、可回顧的 schema 註記，不涉及欄位 rename。

## Rollback Draft 審查

rollback draft 採保守策略，只包含：

- 清掉 `pos_sales_hourly_fact.business_date` comment
- 清掉 `pos_product_dim.cate_no` comment
- 清掉 `pos_product_dim.cate_name` comment
- 清掉 `pos_branch_dim.group_code` comment
- `DROP TABLE IF EXISTS pos_order_status_dim`

判定：

- 這份 rollback draft 是刻意設計的部分回退，而不是完整 schema 還原。
- 原因是本輪明確禁止 drop 欄位，因此不會在 rollback draft 中加入：
  - `ALTER TABLE pos_product_dim DROP COLUMN cate_no`
  - `ALTER TABLE pos_product_dim DROP COLUMN cate_name`
  - `ALTER TABLE pos_branch_dim DROP COLUMN group_code`
- 這代表 rollback 後，新增的 nullable columns 仍會保留，但 status table 與 comment 可被回退。

## 風險與待確認

- `pos_order_status_dim.updated_at` 目前沿用 plan 指定的 `timestamp without time zone`，和現有 `TIMESTAMPTZ` audit 慣例不同；正式實作前要再確認是否統一。
- rollback draft 不是完整 reversal；如果未來要完整回退新增欄位，必須另外審查 drop-column migration。
- 這份 draft 沒有包含 backfill SQL，符合本輪限制，但也表示 `cate_no`、`cate_name`、`group_code` 新增後仍會是空欄位，直到後續另行實作回填。
- 這份 draft 沒有加 index，符合 migration plan；若未來查詢證明 `cate_no` 或 `group_code` 成為高頻 filter，再另補 index migration。

## 建議審查清單

在進入正式 migration 前，至少再確認以下事項：

1. 是否接受 `pos_order_status_dim` 先不建立 FK，只作語意表。
2. 是否接受 rollback draft 為部分回退，新增欄位不在本輪移除。
3. 是否接受 `business_date` 先用 comment 固定語意，而不是 rename。
4. 是否接受 category 與 group code 先只加 nullable column，不在本輪做 backfill。
5. 是否接受正式執行 migration 前，還要再補 execution checklist 與 validation checklist。

## 本輪結論

Phase 2C-3-pre 已把 Phase 2C-2 migration plan 轉成 SQL draft，但仍維持 strict no-op 狀態：

- 只建立草案檔
- 沒有執行 migration
- 沒有修改現有 DB
- 沒有修改 baseline schema
- 沒有修改 Go 或 `sync-athena`
