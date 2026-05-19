# Phase 2C-4 Sales Fact Validation Contract

## 0. 範圍

本文件定義 `pos_sales_hourly_fact` 在正式實作 PostgreSQL write path 前，必須先凍結的 validation contract。

本輪只定義 contract 與 SQL 草案，不做以下事情：

- 不實作 `sync-athena` 寫入
- 不跑 Athena
- 不寫 PostgreSQL
- 不修改 schema
- 不新增 migration
- 不做資料回填

本文件建立在以下已完成前提上：

- Phase 2C-1 schema contract 已定稿
- Phase 2C-2 migration plan 已定稿
- Phase 2C-3 本機開發 DB schema patch 驗證已通過
- `pos_order_status_dim`、`pos_product_dim.cate_no/cate_name`、`pos_branch_dim.group_code` 已在本機開發 DB 驗證存在

## 1. 寫入前必須驗證的 source metrics

`sales fact write candidate` 在真正進入 PostgreSQL 前，必須先產出下列 source metrics，作為 expected totals：

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

這一層的 source metrics 定義如下：

- 必須來自與未來 write path 相同的 sales aggregation candidate query。
- 必須只使用 status-aware `status = 1` latest-row source path。
- 必須以 `owner_user_id + sale_period` 作為最小 validation slice。
- 若要做更細粒度抽樣，應再下鑽到 `branch_id` 或 `hour_of_day`，但 owner/date totals 仍是最小必驗層。

補充：

- `item_count` 是 validation-only control metric。
- `item_count` 代表進入 sales aggregation 的 item line 數量或等價控制總數，不代表最終 `pos_sales_hourly_fact` persisted 欄位。
- `item_count` 必須在 source candidate 與 pre-insert target candidate 層完成對帳，但不要求寫入 `pos_sales_hourly_fact` schema。

## 2. 寫入後必須驗證的 target metrics

真正完成 day-level replace 後，PG target 端至少必須驗證：

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

target metrics 定義如下：

- 來源是 `pos_sales_hourly_fact` 在指定 `owner_user_id + business_date` 的實際 persisted rows。
- `business_date` 在 target 端固定等同 `sale_period`。
- target metrics 必須能和 source candidate metrics 逐日對齊。

關於 `item_count`：

- 因為目前 `pos_sales_hourly_fact` schema 故意不持久化 `item_count`，所以 `item_count` 不是 persisted target metric。
- `item_count` 必須改在 pre-insert target candidate query 層驗證，也就是「即將被 insert 的 aggregation result」與 source candidate 的控制總數是否一致。
- Phase 2C-5 write path 若沒有額外 staging table，則 `item_count` 應在 transaction 內的 insert 前 compare 完成。

## 3. Athena source 與 PG target 的 reconciliation 欄位

Phase 2C-4 固定以下 reconciliation 欄位：

| 欄位 | source candidate | PG target | 規則 |
| --- | --- | --- | --- |
| `row_count` | 必須有 | 必須有 | 完全相等 |
| `gross_sales_milli` | 必須有 | 必須有 | 完全相等 |
| `discount_milli` | 必須有 | 必須有 | 完全相等 |
| `surcharge_milli` | 必須有 | 必須有 | 完全相等 |
| `net_sales_milli` | 必須有 | 必須有 | 完全相等 |
| `sales_ex_tax_milli` | 必須有 | 必須有 | 完全相等 |
| `tax_milli` | 必須有 | 必須有 | 完全相等 |
| `included_tax_milli` | 必須有 | 必須有 | 完全相等 |
| `excluded_tax_milli` | 必須有 | 必須有 | 完全相等 |
| `qty_milli` | 必須有 | 必須有 | 完全相等 |
| `item_count` | 必須有 | 僅 pre-insert candidate 有 | 完全相等 |

這裡的 contract 是：

- persisted fact metrics 原則上全部做 exact match。
- `item_count` 也做 exact match，但比較位置在 pre-insert candidate，不在 persisted fact schema。
- 若任一欄位 mismatch，該日 write 必須視為 failed validation，不得繼續視為成功。

## 4. business_date = sale_period 的驗證方式

必須同時做兩層檢查：

### A. schema contract 檢查

- `pos_sales_hourly_fact.business_date` 的 column comment 必須包含 `business_date is fixed to sale_period semantics`。
- write path 不得再引入 `tr_date` 或 `t_open_date` 到 sales fact insert list。

### B. data contract 檢查

- source candidate 的日期欄位必須明確命名或等價定義為 `sale_period`。
- target query 必須以 `business_date` 做 group，並與 source candidate 的 `sale_period` totals 對比。
- 任一 validation SQL 中，若出現把 sales fact 的 `business_date` 解讀成 `tr_date` 的邏輯，視為 contract 違反。

SQL 草案：

```sql
SELECT col_description('public.pos_sales_hourly_fact'::regclass, ordinal_position)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'pos_sales_hourly_fact'
  AND column_name = 'business_date';
```

## 5. status = 1 latest-row source path 的驗證方式

sales fact 的 source path 必須固定為：

- 只取 `status = 1`
- 只取該 status bucket 的 latest row
- 不得混入 `status = -2 / -1 / 2`

Phase 2C-4 必須驗證兩件事：

### A. candidate 組成檢查

- source candidate query 的輸入訂單集合，只允許來自 `status = 1` latest-row source。
- `status = -2 / -1 / 2` 只能存在於 debug、order fact 或未來 void flow，不得出現在 sales fact write candidate。

### B. aggregate 結果檢查

- write candidate 的聚合總額，必須能回溯到 `status = 1` latest-row candidate。
- 若 validation 抽樣發現任何 `status != 1` 的來源混入 sales fact write candidate，該日 write 必須中止。

SQL 草案：

```sql
WITH sales_source AS (
    -- 以實際 Phase 2B status-aware sales candidate query 為準
    SELECT status, is_latest_status_row
    FROM sales_fact_source_candidate
)
SELECT
    SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS status_1_rows,
    SUM(CASE WHEN status <> 1 THEN 1 ELSE 0 END) AS non_status_1_rows,
    SUM(CASE WHEN is_latest_status_row THEN 1 ELSE 0 END) AS latest_rows
FROM sales_source;
```

通過條件：

- `non_status_1_rows = 0`
- `latest_rows = status_1_rows`

## 6. product_no 是否能對到 pos_product_dim

FK / dim completeness 必須在 write 前先檢查。

規則：

- 每一筆 write candidate 的 `owner_user_id + product_no`，都必須能對到 `pos_product_dim`。
- 若 `product_name`、`cate_no`、`cate_name` 尚未回填完整，不阻止 dim row 存在；但 `product_no` 本身不能 miss。
- miss rate 原則上必須為 0，否則該日 write 失敗。

SQL 草案：

```sql
SELECT COUNT(*) AS missing_product_dim_rows
FROM sales_fact_target_candidate c
LEFT JOIN pos_product_dim d
  ON d.owner_user_id = c.owner_user_id
 AND d.product_no = c.product_no
WHERE d.id IS NULL;
```

## 7. branch_id 是否能對到 pos_branch_dim

規則：

- 每一筆 write candidate 的 `owner_user_id + branch_id`，都必須能對到 `pos_branch_dim`。
- `group_code` 可以是 null，但 dim row 不可缺失。
- miss rate 原則上必須為 0。

SQL 草案：

```sql
SELECT COUNT(*) AS missing_branch_dim_rows
FROM sales_fact_target_candidate c
LEFT JOIN pos_branch_dim d
  ON d.owner_user_id = c.owner_user_id
 AND d.branch_id = c.branch_id
WHERE d.id IS NULL;
```

## 8. order_type_id / payment_type_id 是否能對到 dim

規則：

- `order_type_id` 必須能對到 `pos_order_type_dim`。
- `payment_type_id` 必須能對到 `pos_payment_type_dim`。
- 這兩個欄位允許映射到 canonical `unknown` 類型，但不允許 dimension miss。

SQL 草案：

```sql
SELECT COUNT(*) AS missing_order_type_rows
FROM sales_fact_target_candidate c
LEFT JOIN pos_order_type_dim d
  ON d.id = c.order_type_id
WHERE d.id IS NULL;

SELECT COUNT(*) AS missing_payment_type_rows
FROM sales_fact_target_candidate c
LEFT JOIN pos_payment_type_dim d
  ON d.id = c.payment_type_id
WHERE d.id IS NULL;
```

Phase 2C-4.6 對應 SQL 草案：

- [../db/validation_draft/sales_fact_dimension_gate_checks.sql](../db/validation_draft/sales_fact_dimension_gate_checks.sql)
- hard gate 欄位固定為：
- `product_dim_miss_count`
- `branch_dim_miss_count`
- `order_type_dim_miss_count`
- `payment_type_dim_miss_count`
- `business_date_not_equal_sale_period_count`
- `non_status_1_count`
- `not_latest_status_count`

## 9. 不允許 sales fact 出現 raw payment、void/refund/order-level metrics 的檢查方式

這一層是 negative contract。

### A. schema negative check

`pos_sales_hourly_fact` 不得存在以下欄位：

- `raw_payment_name`
- `raw_payment_memo1`
- `item_count`
- `void_milli`
- `refund_milli`
- `order_count`
- `order_num`
- `completed_order_count`
- `void_order_count`
- `refund_order_count`
- `cancelled_order_count`
- `tr_date`
- `t_open_date`
- `void_sale_period`

SQL 草案：

```sql
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'pos_sales_hourly_fact'
  AND column_name IN (
      'raw_payment_name',
      'raw_payment_memo1',
      'item_count',
      'void_milli',
      'refund_milli',
      'order_count',
      'order_num',
      'completed_order_count',
      'void_order_count',
      'refund_order_count',
      'cancelled_order_count',
      'tr_date',
      't_open_date',
      'void_sale_period'
  );
```

通過條件：

- 回傳 0 rows。
- `item_count` 雖然是 validation-only control metric，但不得出現在 `pos_sales_hourly_fact` persisted schema；它只能存在於 source candidate 或 pre-insert candidate compare。

Phase 2C-4.6 對應 SQL 草案：

- [../db/validation_draft/sales_fact_negative_schema_checks.sql](../db/validation_draft/sales_fact_negative_schema_checks.sql)
- hard gate 結果固定為：
- `forbidden_column_count`
- `forbidden_column_names`

### B. query/output negative check

- write candidate query 的 select list 不得引入 raw payment 欄位。
- write candidate query 的 aggregation output 不得引入 void / refund / order-level metrics。
- 如果 future implementation 需要額外 control totals，必須在 validation / debug query 層存在，而不是進 sales fact schema。

## 10. day-level replace 的驗證流程

Phase 2C-5 write path 只能使用以下流程：

1. 先建立 source candidate totals。
2. 先建立 pre-insert target candidate totals。
3. 先做 source vs candidate reconciliation。
4. 先做 dim / FK miss 檢查。
5. 先做 negative checks。
6. 全部通過後，才開始 target date 的 replace transaction。
7. transaction 內先 delete target date。
8. transaction 內 insert target date。
9. insert 後立刻查 persisted fact totals。
10. persisted fact totals 必須與 source candidate totals compare。
11. 若 compare fail，必須 rollback 或停止，不可視為成功。
12. 只有所有 compare 全部通過，才可 commit。

補充原則：

- 建議把 delete + insert + post-insert compare 放在同一個 transaction。
- 若技術限制導致 compare 不能在 transaction 內完成，則預設行為應是停止並標記 failed，不得自動宣告成功。

## 11. tolerance 原則

Phase 2C-4 的預設 tolerance 如下：

- `row_count`：完全相等
- `qty_milli`：完全相等
- 所有 integer milli 欄位：完全相等
- `item_count`：完全相等
- dim miss count：必須為 0
- negative check：必須為 0

只有在以下情況，才允許例外 tolerance：

- 比較對象不是最終 sales fact candidate，而是更早期的 raw source / exploratory debug query
- 例外 tolerance 已被明確寫進 validation report
- 它的絕對值被限制在極小範圍

預設上限：

- 若真的需要 rounding tolerance，必須明確限定在 `ABS(delta) <= 1 milli` 的極小範圍，且只能用在 exploratory pre-final comparison。
- source candidate vs persisted fact 的正式 compare，預設 tolerance 仍然是 0。

## 12. SQL 草案

### A. source candidate totals

```sql
WITH sales_fact_source_candidate AS (
    -- 以實際 Phase 2B status-aware sales aggregation query 為準
    SELECT
        owner_user_id,
        sale_period,
        gross_sales_milli,
        discount_milli,
        surcharge_milli,
        net_sales_milli,
        sales_ex_tax_milli,
        tax_milli,
        included_tax_milli,
        excluded_tax_milli,
        qty_milli,
        item_count
    FROM ...
)
SELECT
    owner_user_id,
    sale_period,
    COUNT(*) AS row_count,
    SUM(gross_sales_milli) AS gross_sales_milli,
    SUM(discount_milli) AS discount_milli,
    SUM(surcharge_milli) AS surcharge_milli,
    SUM(net_sales_milli) AS net_sales_milli,
    SUM(sales_ex_tax_milli) AS sales_ex_tax_milli,
    SUM(tax_milli) AS tax_milli,
    SUM(included_tax_milli) AS included_tax_milli,
    SUM(excluded_tax_milli) AS excluded_tax_milli,
    SUM(qty_milli) AS qty_milli,
    SUM(item_count) AS item_count
FROM sales_fact_source_candidate
GROUP BY owner_user_id, sale_period;
```

### B. persisted target totals

```sql
SELECT
    owner_user_id,
    business_date AS sale_period,
    COUNT(*) AS row_count,
    SUM(gross_sales_milli) AS gross_sales_milli,
    SUM(discount_milli) AS discount_milli,
    SUM(surcharge_milli) AS surcharge_milli,
    SUM(net_sales_milli) AS net_sales_milli,
    SUM(sales_ex_tax_milli) AS sales_ex_tax_milli,
    SUM(tax_milli) AS tax_milli,
    SUM(included_tax_milli) AS included_tax_milli,
    SUM(excluded_tax_milli) AS excluded_tax_milli,
    SUM(qty_milli) AS qty_milli
FROM pos_sales_hourly_fact
WHERE owner_user_id = :owner_user_id
  AND business_date = :sale_period
GROUP BY owner_user_id, business_date;
```

### C. pre-insert target candidate totals

```sql
WITH sales_fact_target_candidate AS (
    -- 寫入前即將 insert 進 pos_sales_hourly_fact 的結果集
    SELECT
        owner_user_id,
        business_date,
        gross_sales_milli,
        discount_milli,
        surcharge_milli,
        net_sales_milli,
        sales_ex_tax_milli,
        tax_milli,
        included_tax_milli,
        excluded_tax_milli,
        qty_milli,
        item_count
    FROM ...
)
SELECT
    owner_user_id,
    business_date,
    COUNT(*) AS row_count,
    SUM(gross_sales_milli) AS gross_sales_milli,
    SUM(discount_milli) AS discount_milli,
    SUM(surcharge_milli) AS surcharge_milli,
    SUM(net_sales_milli) AS net_sales_milli,
    SUM(sales_ex_tax_milli) AS sales_ex_tax_milli,
    SUM(tax_milli) AS tax_milli,
    SUM(included_tax_milli) AS included_tax_milli,
    SUM(excluded_tax_milli) AS excluded_tax_milli,
    SUM(qty_milli) AS qty_milli,
    SUM(item_count) AS item_count
FROM sales_fact_target_candidate
GROUP BY owner_user_id, business_date;
```

### D. compare draft

```sql
SELECT
    s.owner_user_id,
    s.sale_period,
    s.row_count AS source_row_count,
    t.row_count AS target_row_count,
    s.gross_sales_milli - t.gross_sales_milli AS gross_delta,
    s.discount_milli - t.discount_milli AS discount_delta,
    s.surcharge_milli - t.surcharge_milli AS surcharge_delta,
    s.net_sales_milli - t.net_sales_milli AS net_delta,
    s.sales_ex_tax_milli - t.sales_ex_tax_milli AS sales_ex_tax_delta,
    s.tax_milli - t.tax_milli AS tax_delta,
    s.included_tax_milli - t.included_tax_milli AS included_tax_delta,
    s.excluded_tax_milli - t.excluded_tax_milli AS excluded_tax_delta,
    s.qty_milli - t.qty_milli AS qty_delta
FROM source_totals s
JOIN target_totals t
  ON t.owner_user_id = s.owner_user_id
 AND t.sale_period = s.sale_period;
```

## 13. Phase 2C-5 的開始條件

只有在以下條件全部滿足後，才允許開始 Phase 2C-5 sales fact PG write path 實作：

- 本文件已定稿
- Phase 2C-3 schema patch 驗證已通過
- source metrics / target metrics / reconciliation 欄位已固定
- `business_date = sale_period` 驗證方式已固定
- `status = 1` latest-row source path 驗證方式已固定
- dim / FK miss checks 已固定
- negative checks 已固定
- day-level replace validation flow 已固定
- tolerance 已固定
- Phase 2C-4.5 metrics reconciliation draft 已完成 review
- Phase 2C-4.6 dimension / negative schema gate draft 已完成 review

Phase 2C-5 之前，不應直接開始 `sync-athena` 的 PostgreSQL write path 實作。
