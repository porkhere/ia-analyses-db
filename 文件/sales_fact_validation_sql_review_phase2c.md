# Phase 2C Sales Fact Validation SQL Review

## 目的

本文件審查 Phase 2C-4.5 新增的 validation SQL draft，確認它們對齊 [sales_fact_validation_contract_phase2c.md](sales_fact_validation_contract_phase2c.md) 的要求，但本輪不執行 SQL、不連 Athena、不連 PostgreSQL。

對應檔案：

- [../db/validation_draft/sales_fact_source_metrics.sql](../db/validation_draft/sales_fact_source_metrics.sql)
- [../db/validation_draft/sales_fact_target_metrics.sql](../db/validation_draft/sales_fact_target_metrics.sql)
- [../db/validation_draft/sales_fact_compare_metrics.sql](../db/validation_draft/sales_fact_compare_metrics.sql)

補充：

- Phase 2C-4.5 只覆蓋 metrics reconciliation draft。
- Phase 2C-4.6 另以 [sales_fact_validation_gate_review_phase2c.md](sales_fact_validation_gate_review_phase2c.md) 補齊 dimension gate 與 negative schema gate review。

## 範圍確認

本輪只做 SQL 草案，不做以下事情：

- 不執行 Athena
- 不執行 PostgreSQL
- 不修改 [../db/init/001_schema.sql](../db/init/001_schema.sql)
- 不新增 migration
- 不修改 [../db/patches/003_phase2c_schema_contract.sql](../db/patches/003_phase2c_schema_contract.sql)
- 不修改 Go
- 不改 `sync-athena`
- 不做資料回填
- 不進 PG write path

## Source Metrics Draft 審查

[sales_fact_source_metrics.sql](../db/validation_draft/sales_fact_source_metrics.sql) 的定位正確：

- 明確標示它是 Athena / source candidate 草案，不保證可直接在 PostgreSQL 執行。
- 使用 placeholder：`:owner_user_id`、`:start_date`、`:end_date`。
- 固定輸出 source hard gate metrics：
  - `row_count`
  - `gross_sales_milli`
  - `discount_milli`
  - `surcharge_milli`
  - `net_sales_milli`
  - `sales_ex_tax_milli`
  - `tax_milli`
  - `included_tax_milli`
  - `excluded_tax_milli`
  - `qty_milli`
  - `item_count`
- 補了 source-path hard gates：
  - `status_1_rows`
  - `non_status_1_rows`
  - `latest_status_rows`
- warning gate 只保留 placeholder，不把 exploratory warning 變成正式 hard gate。

判定：

- 符合 contract 對 `status = 1` latest-row source path 的要求。
- 也符合 `item_count` 僅作 validation-only control metric 的要求。

## Target Metrics Draft 審查

[sales_fact_target_metrics.sql](../db/validation_draft/sales_fact_target_metrics.sql) 的定位正確：

- 明確以 PostgreSQL `public.pos_sales_hourly_fact` 為查詢對象。
- 使用 placeholder：`:owner_user_id`、`:start_date`、`:end_date`。
- 固定輸出 persisted target metrics：
  - `row_count`
  - `gross_sales_milli`
  - `discount_milli`
  - `surcharge_milli`
  - `net_sales_milli`
  - `sales_ex_tax_milli`
  - `tax_milli`
  - `included_tax_milli`
  - `excluded_tax_milli`
  - `qty_milli`
- 明確排除 `item_count`，因為它不是 persisted fact column。

判定：

- 符合 contract 對 target metrics 的定義。
- 沒有把 validation-only metric 錯塞回 schema。

## Compare Metrics Draft 審查

[sales_fact_compare_metrics.sql](../db/validation_draft/sales_fact_compare_metrics.sql) 的定位正確：

- 不直接跨 Athena / PostgreSQL 執行。
- 以 `source_metrics_input` 與 `target_metrics_input` 兩個 CTE 模擬輸入。
- 使用 placeholder：`:owner_user_id`、`:start_date`、`:end_date`。
- 固定輸出 compare deltas：
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
- 區分 `target_scope = persisted_fact` 與 `target_scope = pre_insert_candidate`。
- 明確把 `item_count` hard gate 限定在 `pre_insert_candidate` compare。
- 明確把 warning gate 限定為 exploratory rounding placeholder，不覆蓋正式 exact-match contract。

判定：

- 符合 contract 對 compare shape、delta 欄位與 gate 定義的要求。
- 也保留了未來實作時把 persisted compare 與 pre-insert compare 分開落地的空間。

## README Markdown 修正

本輪另外檢查了 [README.md](../README.md) 的 markdown code fence。

結果：

- `常用指令` 區塊原本只有 opening fence，沒有 closing fence。
- 本輪會補上 closing fence，但只修 markdown fence，不改動其他 README 內容。

## 風險與待確認

- source draft 仍使用 placeholder relation `athena_sales_fact_source_candidate_draft`；正式落地前，必須替換成 Phase 2B status-aware sales candidate 的真實 SQL。
- compare draft 目前是 shape-first 設計，不包含跨系統資料搬運方法；正式執行前還需要決定 source metrics 如何餵進 compare SQL。
- warning gate 在目前 contract 中預設為未啟用；若未來要正式使用 rounding warning，必須先更新 validation contract，而不是直接改 compare SQL 行為。
- `item_count` 目前仍是 validation-only control metric；Phase 2C-5 實作時，必須在 pre-insert candidate 階段保留這個 control total。

## 本輪結論

Phase 2C-4.5 已完成 sales fact validation SQL draft，但仍維持 no-op 狀態：

- 只建立 SQL 草案檔
- 沒有執行 Athena
- 沒有執行 PostgreSQL
- 沒有修改 schema / patch / Go / sync-athena
- 沒有進入 PG write path

但 Phase 2C-5 仍然不能開始，因為 metrics reconciliation draft 之外，還需要 Phase 2C-4.6 的 dimension gate 與 negative schema gate draft 也完成 review。
