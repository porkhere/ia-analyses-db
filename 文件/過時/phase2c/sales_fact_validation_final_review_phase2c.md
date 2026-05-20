# Phase 2C Sales Fact Validation Final Review

## 目的

本文件是 Phase 2C-4.7 的最終整併審查，目的是把 Phase 2C-4.5 metrics reconciliation SQL draft、Phase 2C-4.6 validation gate SQL draft，以及既有 validation contract 收斂成單一結論，判定是否可以進入 Phase 2C-5。

本輪只做文件整併審查，不做以下事情：

- 不執行 SQL
- 不執行 Athena
- 不執行 PostgreSQL
- 不修改 schema
- 不新增 migration
- 不修改 Go
- 不改 `sync-athena`
- 不做資料回填
- 不進 PG write path

## 1. 目前已存在的 validation SQL draft 清單

目前已存在的 validation SQL draft 如下：

- [../../../db/validation_draft/sales_fact_source_metrics.sql](../../../db/validation_draft/sales_fact_source_metrics.sql)
- [../../../db/validation_draft/sales_fact_target_metrics.sql](../../../db/validation_draft/sales_fact_target_metrics.sql)
- [../../../db/validation_draft/sales_fact_compare_metrics.sql](../../../db/validation_draft/sales_fact_compare_metrics.sql)
- [../../../db/validation_draft/sales_fact_dimension_gate_checks.sql](../../../db/validation_draft/sales_fact_dimension_gate_checks.sql)
- [../../../db/validation_draft/sales_fact_negative_schema_checks.sql](../../../db/validation_draft/sales_fact_negative_schema_checks.sql)

## 2. 每一份 SQL draft 的責任邊界

### sales_fact_source_metrics.sql

責任邊界：

- 定義 Athena / source candidate 應產生的 expected metrics。
- 定義 `status = 1` latest-row sales source path 的 source-side hard gates。
- 提供 `row_count`、金額欄位、`qty_milli` 與 `item_count` 的 source totals。
- 不負責 persisted fact query。

### sales_fact_target_metrics.sql

責任邊界：

- 定義 PostgreSQL `pos_sales_hourly_fact` 的 actual persisted metrics。
- 查詢對象固定為 `public.pos_sales_hourly_fact`。
- 提供 persisted fact 的 row count 與 sum totals。
- 不負責 `item_count`，因為 `item_count` 不得進 persisted fact schema。

### sales_fact_compare_metrics.sql

責任邊界：

- 定義 source vs target / pre-insert candidate 的 compare shape、delta 欄位與 hard gate pass/fail 邏輯。
- 允許 `target_scope = persisted_fact` 與 `target_scope = pre_insert_candidate` 兩種比較場景。
- 將 `item_count_delta` 明確限定在 `pre_insert_candidate` compare。
- 不直接跨 Athena / PostgreSQL 執行，只定義 compare contract。

### sales_fact_dimension_gate_checks.sql

責任邊界：

- 定義 write 前必須通過的 dimension gate 與 source-path gate。
- 檢查 product / branch / order_type / payment_type dim 是否 miss。
- 檢查 `business_date = sale_period` 是否成立。
- 檢查是否混入 `status != 1` 或非 latest-row source。
- 不負責 sum metrics compare。

### sales_fact_negative_schema_checks.sql

責任邊界：

- 防止 `pos_sales_hourly_fact` 被污染成萬用表。
- 透過 PostgreSQL `information_schema` 檢查 forbidden columns 是否存在。
- 把 raw payment、void/refund、order-level metrics，以及 `item_count` 從 persisted fact schema 層直接擋掉。
- 不負責 query output negative check；那一層仍由 validation contract 定義。

## 3. hard gate 清單

目前已固定的 hard gate 如下：

### reconciliation hard gates

- `row_count_delta`
- `gross_sales_milli_delta`
- `discount_milli_delta`
- `surcharge_milli_delta`
- `net_sales_milli_delta`
- `sales_ex_tax_milli_delta`
- `tax_milli_delta`
- `included_tax_milli_delta`
- `excluded_tax_milli_delta`
- `qty_milli_delta`
- `item_count_delta`

補充：

- `item_count_delta` 只適用於 `pre_insert_candidate` compare。
- `item_count` 不得進 persisted fact schema。

### dimension / source-path hard gates

- `product_dim_miss_count`
- `branch_dim_miss_count`
- `order_type_dim_miss_count`
- `payment_type_dim_miss_count`
- `business_date_not_equal_sale_period_count`
- `non_status_1_count`
- `not_latest_status_count`

### negative schema hard gates

- `forbidden_column_count`

## 4. warning gate 清單

目前可以作為 warning gate 或後續建議的項目如下：

- `warning_rounding_delta_milli`
- `warning_gate_note`
- `group_code unknown rate`
- `cate_no / cate_name unknown rate`
- `product_no` 多名稱或多分類衝突
- `branch_id` 多名稱衝突

判定：

- 只有 `warning_rounding_delta_milli` 與 `warning_gate_note` 已在 SQL draft 內保留 placeholder。
- `group_code unknown rate`、`cate_no / cate_name unknown rate`、`product_no` 多名稱或多分類衝突、`branch_id` 多名稱衝突，目前尚未定義成正式 warning gate SQL。
- 這些項目目前應明確標記為「後續建議，不是 Phase 2C-5 blocker」。

## 5. insert 前必須執行的檢查

Phase 2C-5 在真正 delete / insert 之前，至少必須執行：

1. source metrics
2. pre-insert candidate metrics
3. compare metrics 中 `pre_insert_candidate` scope 的比較
4. dimension gate checks
5. negative schema checks

補充原則：

- insert 前 hard gate 只要有任一失敗，就不得開始 target date delete。
- `item_count` 的 compare 必須在這個階段完成，不能等到 persisted fact 才補驗。

## 6. insert 後必須執行的檢查

Phase 2C-5 完成 day-level replace 後，至少必須執行：

1. target metrics
2. source vs persisted target compare
3. day-level replace 後的 row count / sum compare
4. 若寫入失敗或 compare 失敗，必須 rollback 或停止

補充原則：

- persisted compare 的 hard gate 仍然是 exact match。
- insert 後不得因為「資料已經寫進去」就跳過 compare。

## 7. rollback / stop 條件

Phase 2C-5 必須遵守以下 stop 規則：

- hard gate 失敗必須停止。
- insert 前 hard gate 失敗不得 delete target date。
- insert 後 hard gate 失敗必須 rollback transaction。
- 若無法 rollback，必須停止後續日期處理並回報錯誤。

這代表 Phase 2C-5 不允許存在 validation bypass，也不允許在 compare fail 時繼續處理下一個日期。

## 8. Phase 2C-5 實作限制

Phase 2C-5 可以開始實作 `sync-athena` PG write path，但必須遵守以下限制：

- `day-level replace`
- `transaction boundary`
- `validation first`
- `no validation bypass`
- 不得把 raw payment、void、refund、order-level metrics 寫進 `pos_sales_hourly_fact`
- `item_count` 只能存在於 source candidate / pre-insert candidate，不得進 persisted fact
- `business_date` 必須維持 `sale_period` 語意
- source 只吃 `status = 1` latest-row sales candidate
- `status = -2 / -1 / 2` 不進 sales fact

補充說明：

- 這裡允許開始的範圍，只是 sales fact PG write path skeleton + validation gate 整合。
- 不包含 order fact、payment fact、condiment fact、branch opening fact。
- 不包含 raw payment、void lifecycle 或 order-level metrics 的擴張。

## 9. 是否可以進 Phase 2C-5 的結論

交叉檢查結果如下：

- validation contract 已固定 source metrics、target metrics、negative checks、day-level replace 與 tolerance 原則。
- metrics reconciliation SQL draft 已齊備。
- dimension gate 與 negative schema gate SQL draft 已齊備。
- `item_count` 的角色已一致：它是 validation-only control metric，但禁止進 persisted fact schema。
- README、架構指南、更新紀錄與實作順序目前只需要同步標示 final review 已完成與 2C-5 的前置條件。
- warning gate 類項目仍有後續建議，但目前不構成 blocker。

最終結論：

**Phase 2C-5 可以開始，但僅限 sales fact PG write path skeleton + validation gate 整合。**

這個結論的前提是：

- 仍然不得跳過 validation gate
- 仍然不得把 `item_count` 寫進 persisted fact
- 仍然不得把 raw payment、void、refund、order-level metrics 寫進 `pos_sales_hourly_fact`
- 仍然必須以 `status = 1` latest-row source path 作為唯一 sales source
