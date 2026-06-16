# Sales Fact Correctness Basis Phase 2C

## 目的

這份文件定義 Phase 2C sales fact correctness 的判準來源，避免把「Athena 有資料」誤解成「任何 raw row 都可直接視為正確 persisted fact」。

目前 correctness 的基礎不是單一 raw row mirror，而是：

- Athena raw POS tables 是 source of record
- sales fact 的正確性要經過已定稿的 semantic contract
- semantic contract 的結果必須再通過 validation gates 與 post-insert compare

## Source Of Record

- Athena raw POS tables 仍是 source of record
- 目前主要來源表包含：
  - orders_parquet
  - order_items_parquet
  - order_additions_parquet
  - order_payments_parquet
- local PostgreSQL 的 public.pos_sales_hourly_fact 是受控 materialization，不是 raw mirror

## Semantic Contract

目前 sales fact correctness 依據以下 semantic contract：

- status = 1 的 latest row 才能進 sales fact
- status = -2 保留給後續 void fact，不進 sales fact
- status = -1 與 status = 2 屬於 excluded path，不進 sales fact
- business_date 的正式語意固定為 sale_period
- 金額與數量欄位使用 milli bigint
- item_count 只允許 validation-only，不進 persisted sales fact
- raw payment name / memo1 不進 sales fact
- void / refund / order-level metrics 不進 sales fact

## Correctness Checks

Phase 2C 目前把 correctness 落成以下可執行檢查：

1. source metrics 必須等於 candidate metrics
2. candidate metrics 必須等於 post-insert target metrics
3. dimension gate 必須通過
4. negative schema gate 必須通過

### Source To Candidate

source metrics 與 candidate metrics 必須對齊下列欄位：

- row_count
- gross_sales_milli
- discount_milli
- surcharge_milli
- net_sales_milli
- sales_ex_tax_milli
- tax_milli
- included_tax_milli
- excluded_tax_milli
- qty_milli
- item_count

其中 item_count 只作 validation-only control metric；它可以參與 source / candidate compare，但不得進 persisted fact。

### Candidate To Persisted Target

candidate metrics 與 post-insert target metrics 必須對齊下列 persisted 指標：

- row_count
- gross_sales_milli
- discount_milli
- surcharge_milli
- net_sales_milli
- sales_ex_tax_milli
- tax_milli
- included_tax_milli
- excluded_tax_milli
- qty_milli

這一層不再比較 item_count，因為 item_count 不屬於 persisted schema。

### Dimension Gate

下列 gate 必須為 0：

- product_dim_miss_count
- branch_dim_miss_count
- order_type_dim_miss_count
- payment_type_dim_miss_count
- business_date_not_equal_sale_period_count
- non_status_1_count
- not_latest_status_count

### Negative Schema Gate

forbidden_column_count 必須為 0。禁止欄位包含：

- item_count
- raw_payment_name
- raw_payment_memo1
- void_milli
- refund_milli
- order_count
- completed_order_count
- void_order_count
- refund_order_count
- cancelled_order_count
- tr_date
- t_open_date
- void_sale_period
- order_num

## Benchmark Positioning

- Athena source + semantic contract + validation gates + post-insert compare，才是目前 row-level correctness 的直接依據
- QuickSight 是 business benchmark，可用來驗證報表與商業口徑方向
- QuickSight 不是所有 row-level fact 的唯一 truth，也不是 persisted sales fact schema 的唯一設計來源

## Current Practical Interpretation

目前 Phase 2C 的 correctness 可以簡化成以下判斷：

- source of record 來自 Athena raw POS tables
- business semantics 由 status-aware contract 凍結
- row-level materialization 正確性由 source -> candidate -> persisted target 的 compare 鏈保證
- schema 邊界由 negative schema gate 保證
- dimension completeness 由 dimension gate 保證

任何 regression 驗證或小窗口 actual write，只要違反以上任何一點，就不能視為 correctness pass。