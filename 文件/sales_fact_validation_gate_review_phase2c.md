# Phase 2C Sales Fact Validation Gate Review

## 目的

本文件審查 Phase 2C-4.6 新增的 validation gate SQL draft，確認 dimension gate 與 negative schema gate 已補齊到 [sales_fact_validation_contract_phase2c.md](sales_fact_validation_contract_phase2c.md) 所定義的 write gate 要求。

對應檔案：

- [../db/validation_draft/sales_fact_dimension_gate_checks.sql](../db/validation_draft/sales_fact_dimension_gate_checks.sql)
- [../db/validation_draft/sales_fact_negative_schema_checks.sql](../db/validation_draft/sales_fact_negative_schema_checks.sql)
- [sales_fact_validation_sql_review_phase2c.md](sales_fact_validation_sql_review_phase2c.md)

## 範圍確認

本輪只做 SQL 草案與 review，不做以下事情：

- 不執行 Athena
- 不執行 PostgreSQL
- 不修改 [../db/init/001_schema.sql](../db/init/001_schema.sql)
- 不新增 migration
- 不修改 [../db/patches/003_phase2c_schema_contract.sql](../db/patches/003_phase2c_schema_contract.sql)
- 不修改 Go
- 不改 `sync-athena`
- 不做資料回填
- 不進 PG write path

## 與 Phase 2C-4.5 的關係

- Phase 2C-4.5 已完成 metrics reconciliation draft。
- Phase 2C-4.6 補齊 dimension gate 與 negative schema gate。
- 兩者合起來才構成完整的 sales fact validation gate 草案。

## Dimension Gate Draft 審查

[sales_fact_dimension_gate_checks.sql](../db/validation_draft/sales_fact_dimension_gate_checks.sql) 已補齊 contract 要求的 hard gate 欄位：

- `product_dim_miss_count`
- `branch_dim_miss_count`
- `order_type_dim_miss_count`
- `payment_type_dim_miss_count`
- `business_date_not_equal_sale_period_count`
- `non_status_1_count`
- `not_latest_status_count`

判定：

- 使用了要求的 placeholder：`:owner_user_id`、`:start_date`、`:end_date`。
- 同時把 dim completeness、日期語意與 source-path gate 收斂到單一結果集。
- hard gate 都是 count 型態，方便後續直接作為 write stop condition。

## Negative Schema Gate Draft 審查

[sales_fact_negative_schema_checks.sql](../db/validation_draft/sales_fact_negative_schema_checks.sql) 已補齊 PostgreSQL information_schema 檢查：

- `raw_payment_name`
- `raw_payment_memo1`
- `item_count`
- `void_milli`
- `refund_milli`
- `order_count`
- `completed_order_count`
- `void_order_count`
- `refund_order_count`
- `cancelled_order_count`
- `tr_date`
- `t_open_date`
- `void_sale_period`
- `order_num`

判定：

- 查詢對象固定為 `public.pos_sales_hourly_fact`。
- 以 `forbidden_column_count` 與 `forbidden_column_names` 輸出 hard gate 結果。
- 能直接對應 contract 中的 negative schema check。
- `item_count` 雖然是 validation-only control metric，但它不得出現在 `pos_sales_hourly_fact` persisted schema；若需要比對，只能存在於 source candidate 或 pre-insert candidate metrics。

## 目前 Phase 2C-5 的阻擋條件

Phase 2C-5 仍然不能開始，除非以下兩類 review 都通過：

1. Phase 2C-4.5 metrics reconciliation draft review
2. Phase 2C-4.6 dimension / negative schema gate draft review

換句話說，只有當 metrics draft 與 gate draft 都完成 review，才允許開始 sales fact PG write path 的實作。

## 風險與待確認

- `sales_fact_dimension_gate_checks.sql` 仍使用 placeholder relation `sales_fact_target_candidate_draft`；正式落地前必須替換成真正的 pre-insert candidate query。
- `business_date_not_equal_sale_period_count` 目前假設 candidate query 同時輸出 `business_date` 與 `sale_period`；正式實作時必須保留這個 compare 能力。
- negative schema gate 目前只做 schema 層檢查，不會捕捉 query output 中臨時引入 forbidden metrics 的情況；這一層仍要由 validation contract 的 query/output negative check 補足。
- `item_count` 現在被同時定義成 control metric 與 forbidden persisted column；Phase 2C-5 實作時必須確保它只留在 validation 流程，不進最終 sales fact schema。

## 本輪結論

Phase 2C-4.6 已補齊 sales fact validation gate SQL draft，但仍維持 no-op 狀態：

- 只建立 SQL 草案檔
- 沒有執行 Athena
- 沒有執行 PostgreSQL
- 沒有修改 schema / patch / Go / sync-athena
- 沒有進入 PG write path
