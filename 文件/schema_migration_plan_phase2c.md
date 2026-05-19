# Phase 2C-2 Schema Migration Plan

## 0. 依據

本文件將 Phase 2C-1 schema contract 轉成可執行的 migration plan 草案，但本輪不實作 migration、不修改 PostgreSQL schema。

本文依據以下既有文件與 schema：

- [README.md](../README.md)
- [架構指南.md](./架構指南.md)
- [更新紀錄.md](./更新紀錄.md)
- [data_model_design.md](./data_model_design.md)
- [fact_implementation_sequence.md](./fact_implementation_sequence.md)
- [fact_gap_analysis.md](./fact_gap_analysis.md)
- [validation_plan.md](./validation_plan.md)
- [001_schema.sql](../db/init/001_schema.sql)

## 1. 目標

本次 Phase 2C-2 migration plan 的目的如下：

- 讓現有 schema 對齊 Phase 2C-1 schema contract。
- 讓 `pos_sales_hourly_fact` 專注於商品層級銷售彙總。
- 補齊 product / branch / order status dimension。
- 把 order-level / payment-level / condiment / branch opening 的需求排除出 sales fact。
- 為後續 PG day-level replace + validation 做準備。

## 2. 不做事項

Phase 2C-2 本輪明確不做以下事情：

- 不新增 PG 寫入。
- 不實作 day-level replace。
- 不跑 Athena。
- 不改 `sync-athena`。
- 不建立 order / payment / condiment / branch opening fact。
- 不 drop 現有欄位，除非明確列入未來版本。
- 不把 raw payment `name` / `memo1` 塞進 sales fact。
- 不把 `cate_name`、branch `options` 塞進 sales fact。
- 不修改 [001_schema.sql](../db/init/001_schema.sql)。
- 不建立 migration 檔。

## 3. 現有 schema 審查

### 3.1 目前 baseline schema 實際有哪些表

依 [001_schema.sql](../db/init/001_schema.sql)，目前 baseline schema 共有 6 張表：

| 表名 | 用途 | Phase 2C-2 判定 |
| --- | --- | --- |
| `ia_users` | owner key 正規化與內部整數鍵 | 保留 |
| `pos_order_type_dim` | canonical order type seed 維度 | 保留 |
| `pos_payment_type_dim` | canonical payment type seed 維度 | 保留 |
| `pos_product_dim` | 商品維度 | 需補 `cate_no`、`cate_name` |
| `pos_branch_dim` | 門市維度 | 需補 `group_code` |
| `pos_sales_hourly_fact` | 商品層級銷售彙總 fact | 保留主體，補 contract 註記，不擴張 grain |

目前 baseline schema 中不存在 `pos_order_status_dim`，因此它是 Phase 2C-2 的新增表候選。

### 3.2 ia_users 現況

目前欄位如下：

- `id`
- `owner_user_key`
- `display_name`
- `source_system`
- `is_active`
- `created_at`
- `updated_at`

判定：

- `ia_users` 不需 Phase 2C-2 schema 變更。
- 後續 migration 只會繼續引用它的 `id` 作為 fact / dim 的 `owner_user_id`。

### 3.3 pos_order_type_dim 現況

目前欄位如下：

- `id`
- `code`
- `name`
- `description`
- `sort_order`
- `is_active`
- `created_at`

判定：

- 這張表是 canonical order type 維度，Phase 2C-2 不需變更。
- `order_type_id` 仍只代表 canonical order type，不延伸成 raw `destination` mirror。

### 3.4 pos_payment_type_dim 現況

目前欄位如下：

- `id`
- `code`
- `name`
- `description`
- `sort_order`
- `is_active`
- `created_at`

判定：

- 這張表是 canonical payment type 維度，Phase 2C-2 不需變更。
- `payment_type_id` 仍只代表 canonical payment type，不承接 raw payment `name` / `memo1`。

### 3.5 pos_sales_hourly_fact 現況

目前 baseline schema 的欄位如下：

- `id`
- `owner_user_id`
- `business_date`
- `hour_of_day`
- `branch_id`
- `product_no`
- `order_type_id`
- `payment_type_id`
- `qty_milli`
- `gross_sales_milli`
- `discount_milli`
- `surcharge_milli`
- `net_sales_milli`
- `sales_ex_tax_milli`
- `included_tax_milli`
- `excluded_tax_milli`
- `tax_milli`
- `created_at`
- `updated_at`

#### 保留

以下欄位應保留在 `pos_sales_hourly_fact`：

- `owner_user_id`
- `business_date`
- `hour_of_day`
- `branch_id`
- `product_no`
- `order_type_id`
- `payment_type_id`
- `qty_milli`
- `gross_sales_milli`
- `discount_milli`
- `surcharge_milli`
- `net_sales_milli`
- `sales_ex_tax_milli`
- `included_tax_milli`
- `excluded_tax_milli`
- `tax_milli`
- `created_at`
- `updated_at`

#### 改語意 / 補註解

- `business_date`
  - Phase 2C-2 只補 contract，不改實體欄位名。
  - 正式語意固定為 `sale_period`。
  - 不允許再把它解讀成 `tr_date` 或 `t_open_date`。
- `updated_at`
  - 保留為審計欄位。
  - 不承接任何 business event date 語意。

#### deprecated / phase-out

以 [001_schema.sql](../db/init/001_schema.sql) 為準，baseline schema 目前沒有下列欄位：

- `order_count`
- `void_milli`
- `refund_milli`
- `completed_order_count`
- `void_order_count`
- `refund_order_count`
- `cancelled_order_count`

Phase 2C-2 的判定是：

- baseline schema 不新增這些欄位。
- 如果現場開發 DB 曾因臨時 patch 出現這些欄位，應先標記為 deprecated / unused，不在 Phase 2C-2 直接 drop。
- 真正的 drop 時機應放到後續 migration 已完成替代 fact、且下游查詢與 validation 已完成切換之後。

#### 不應新增

以下欄位明確不應新增到 `pos_sales_hourly_fact`：

- `raw_payment_name`
- `raw_payment_memo1`
- `cate_no`
- `cate_name`
- `group_code`
- `branch_options`
- `tr_date`
- `t_open_date`
- `void_sale_period`
- `order_num`
- `item_count`

### 3.6 pos_product_dim 現況

目前欄位如下：

- `id`
- `owner_user_id`
- `product_no`
- `product_name`
- `product_name_normalized`
- `is_active`
- `last_seen_at`
- `created_at`
- `updated_at`

現況判定：

- 已有商品自然鍵與名稱欄位。
- 目前缺少 `cate_no`。
- 目前缺少 `cate_name`。
- 目前唯一鍵仍是 `UNIQUE (owner_user_id, product_no)`，這個設計應保留。

### 3.7 pos_branch_dim 現況

目前欄位如下：

- `id`
- `owner_user_id`
- `branch_id`
- `branch_name`
- `branch_name_normalized`
- `is_active`
- `last_seen_at`
- `created_at`
- `updated_at`

現況判定：

- 已有門市自然鍵與名稱欄位。
- 目前缺少 `group_code`。
- `options` 目前不在 schema 中，且 Phase 2C-2 維持不進正式 schema。
- `branch_id` 應繼續保留為 text natural key。

### 3.8 status dimension 現況

目前 baseline schema 沒有 `pos_order_status_dim`。

Phase 2C-2 判定：

- 這張表應列為新增表。
- 目的不是立即讓 `pos_sales_hourly_fact` 建 FK，而是先固定 raw status 語意，供未來 order fact、validation、文件與報表邊界使用。

## 4. Phase 2C-2 建議 migration 草案

### A. pos_product_dim

建議新增欄位：

```sql
cate_no text NULL,
cate_name text NULL
```

設計說明：

- `cate_no` / `cate_name` 應放在 product dim，而不是 sales fact。
- 原因是 category 是商品屬性，不是商品小時銷售 grain 的核心維度值；若直接塞進 fact，只會讓同一商品屬性重複存放並增加 drift 風險。
- `pos_sales_hourly_fact` 只保留 `product_no`，需要 category 時再透過 dim join。

index 建議：

- Phase 2C-2 不建議為 `cate_no` / `cate_name` 額外先加 index。
- 現有 `idx_pos_product_dim_lookup (owner_user_id, product_no, is_active)` 已滿足主要 lookup 路徑。
- 若後續驗證證明 `cate_no` 成為高頻 filter，再考慮新增 `(owner_user_id, cate_no)` 類型 index。

回填來源預期：

- `order_items_parquet.product_no`
- `order_items_parquet.product_name`
- `order_items_parquet.cate_no`
- `order_items_parquet.cate_name`

同一 `product_no` 出現多個 `cate_no` / `cate_name` 的處理：

- Phase 2C-2 不直接 silent overwrite。
- 本輪建議先做 dry-run profiling 與 conflict profiling。
- 建議產出 `product_dim_conflict_report` 作為 validation artifact，而不是直接落成正式表。
- 真正實作回填時，預設候選策略可採「most frequent `cate_no + cate_name` 組合，若頻率相同再用 latest seen tie-break」，但只有在 conflict rate 可接受時才執行。
- 若 conflict rate 偏高，應先人工確認或補 mapping 規則，再進 Phase 2C-3 之後的實作。

### B. pos_branch_dim

建議新增欄位：

```sql
group_code text NULL
```

設計說明：

- `group_code` 應放在 branch dim，因為它是門市屬性，不是 sales fact measure。
- `branch_id` 仍保留為 text natural key，不做 surrogate-only 查找。
- `options` 不進正式 schema，因為它本質上是展示字串，可由 view 或展示層用 `branch_id + ' ' + branch_name` 組出。
- 若把 `options` 寫進 schema，會把展示責任誤植到資料模型，且增加同步時字串一致性問題。

index 建議：

- Phase 2C-2 不建議先為 `group_code` 額外加 index。
- 現有 `idx_pos_branch_dim_lookup (owner_user_id, branch_id, is_active)` 仍是主要 lookup 路徑。
- 若後續出現固定以 `group_code` 做大範圍篩選的查詢，再考慮補 `(owner_user_id, group_code)` 類型 index。

### C. pos_order_status_dim

建議新增表：

```sql
CREATE TABLE pos_order_status_dim (
    status_code smallint PRIMARY KEY,
    status_name text NOT NULL,
    status_bucket text NOT NULL,
    is_sales boolean NOT NULL DEFAULT false,
    is_void boolean NOT NULL DEFAULT false,
    is_cancelled_like boolean NOT NULL DEFAULT false,
    is_excluded boolean NOT NULL DEFAULT false,
    description text NOT NULL DEFAULT '',
    sort_order smallint NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    updated_at timestamp without time zone NOT NULL DEFAULT now()
);
```

seed 草案：

| status_code | status_name | status_bucket | is_sales | is_void | is_cancelled_like | is_excluded | description | sort_order |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `1` | `normal_sales` | `sales` | `true` | `false` | `false` | `false` | 正常銷售主口徑 | `10` |
| `-2` | `void` | `void` | `false` | `true` | `false` | `false` | 作廢主口徑 | `20` |
| `-1` | `cancelled_like` | `excluded` | `false` | `false` | `true` | `true` | 排除於 sales / void 主口徑 | `30` |
| `2` | `other_excluded` | `excluded` | `false` | `false` | `false` | `true` | 其他排除狀態 | `40` |

設計說明：

- 這張表先不一定被 `pos_sales_hourly_fact` FK 使用。
- 它的第一目的，是固定 raw status 語意，供 order fact、validation、文件與未來報表使用。
- 這張表也能避免不同 SQL 或文件各自解讀 `status = -1 / 2`。
- current Phase 2C-2 是 schema alignment，不是 status-aware metrics 落地。

補充待確認：

- 目前 repo 其他表的審計欄位多採 `TIMESTAMPTZ`；上面的 DDL 草案採用較小的 `updated_at timestamp without time zone`，實作輪應再確認是否要統一 audit 欄位型別。

### D. pos_sales_hourly_fact

欄位策略如下。

#### 保留欄位

- `owner_user_id`
- `business_date`
- `hour_of_day`
- `branch_id`
- `product_no`
- `order_type_id`
- `payment_type_id`
- `qty_milli`
- `gross_sales_milli`
- `discount_milli`
- `surcharge_milli`
- `net_sales_milli`
- `sales_ex_tax_milli`
- `included_tax_milli`
- `excluded_tax_milli`
- `tax_milli`
- `updated_at`

補充說明：

- `item_count` 目前不在 baseline schema 中，也不屬於 Phase 2C-2 必要新增欄位。
- `created_at` 可繼續保留作為審計欄位，但 migration plan 的主要 contract 關注點是語意與排除欄位，而不是審計欄位調整。

#### business_date 註解 / contract

必須明確固定以下規則：

- `business_date = sale_period`
- 不新增 `tr_date` / `t_open_date` 到 sales fact
- 若未來要以 `tr_date` 查詢，應使用 order fact / payment fact 或 view
- 若未來要把欄位名從 `business_date` 改成 `sale_period`，那是後續 migration 題目，不屬於 Phase 2C-2

#### deprecated 欄位

針對下列欄位，Phase 2C-2 的策略是「若存在則 deprecated / unused，但不立刻 drop」：

- `order_count`
- `void_milli`
- `refund_milli`
- `completed_order_count`
- `void_order_count`
- `refund_order_count`
- `cancelled_order_count`

說明：

- 這些欄位不適合放在商品粒度 sales fact，因為會引入 order-level 或 void lifecycle 的聚合歧義。
- Phase 2C-2 先不 drop，是為了避免歷史查詢、實驗性 patch、或未切換完成的 downstream reference 立即中斷。
- 合理的 drop 時機，應放在 order fact / payment fact 已建立、validation 已通過、下游查詢已切換之後。
- 以 baseline schema 來看，這些欄位目前不存在，因此本輪重點是明確宣告「不要新增」，不是執行刪除。

#### 不新增欄位

以下欄位明確不應新增到 `pos_sales_hourly_fact`：

- `raw_payment_name`
- `raw_payment_memo1`
- `cate_no`
- `cate_name`
- `group_code`
- `branch_options`
- `tr_date`
- `t_open_date`
- `void_sale_period`
- `order_num`

## 5. migration 順序建議

建議順序如下：

1. 新增 `pos_order_status_dim`。
2. seed `pos_order_status_dim`。
3. 補 `pos_product_dim.cate_no` / `cate_name`。
4. 補 `pos_branch_dim.group_code`。
5. 加上 `business_date = sale_period` 的 schema comment 或文件註記。
6. 將 sales fact 的 order / void / refund / cancel count 類欄位標記為 deprecated，但不 drop。
7. 更新 validation plan 與 validation checklist。
8. plan 審查通過後，下一輪才進 sales fact PG write path。

這樣排序比較安全的原因：

- 先做 additive schema，比先改寫寫入邏輯安全。
- 先固定 status 語意，可以避免後續 order fact / validation 又各自解讀狀態。
- 先補 dim 欄位，再談 sales fact write path，才能避免 sales fact 寫入後又回頭補 category / group_code 對齊問題。
- 對 `business_date` 先加 contract 註記，再做 PG write，能減少 `sale_period` / `tr_date` 混用風險。
- 對歷史欄位先 deprecated、不立即 drop，可以降低未預期相依查詢中斷的風險。

## 6. 回填策略

本節只設計策略，不做實作。

### 6.1 product dim 回填

預期來源：

- `order_items_parquet.product_no`
- `order_items_parquet.product_name`
- `order_items_parquet.cate_no`
- `order_items_parquet.cate_name`

策略建議：

- Phase 2C-2 先做 dry-run profiling，不直接做實際回填。
- profiling 應至少產出：
  - 每個 `owner_user_id + product_no` 的 distinct `product_name` 數
  - 每個 `owner_user_id + product_no` 的 distinct `cate_no + cate_name` 組合數
  - unknown / null `cate_no`、`cate_name` 比率
- 若同一 `product_no` 有多個名稱或多個分類，先記錄到 `product_dim_conflict_report`。
- `product_dim_conflict_report` 建議作為 validation artifact 或 query report，不在 Phase 2C-2 直接新增正式 schema。
- 真正回填可延後到 Phase 2C-3 或後續；若要自動選值，建議以 most frequent tuple 為主、latest seen 為 tie-break。

### 6.2 branch dim 回填

預期來源：

- `orders_parquet.branch_id`
- `orders_parquet.branch`
- QuickSight dataset 或既有 branch source，如可用

策略建議：

- `branch_id` / `branch_name` 仍可沿用現有 branch dim 邏輯。
- `group_code` 來源目前尚未在 baseline schema 與現有文件中被正式固定。
- 因此 `group_code` 應標記為 `needs_source_confirmation`。
- 在 source 尚未確認前，`group_code` 欄位應允許為 null，不應以猜測值回填。

### 6.3 order status seed

策略建議：

- 直接 seed `1 / -2 / -1 / 2` 四個 raw status。
- 這張表不需要 Athena 回填。
- 若未來發現新的 raw status code，再以增量 seed 維護即可。

## 7. validation impact

後續 validation 應新增至少以下檢查：

- `product_no` 是否都能對到 `pos_product_dim`。
- `cate_no` / `cate_name` 的 unknown rate。
- 同一 `product_no` 多名稱 / 多分類的 conflict rate。
- `branch_id` 是否都能對到 `pos_branch_dim`。
- `group_code` unknown rate。
- source 中出現的 `status_code` 是否都能對到 `pos_order_status_dim`。
- `pos_sales_hourly_fact` 不應產生 raw payment 欄位。
- `pos_sales_hourly_fact` 不應產生 void / refund / order-level metrics。
- `business_date` 是否等於 `sale_period` 的 source 邏輯。
- 若實作 comment 或文件 contract，需驗證 downstream 查詢沒有再把 `business_date` 當 `tr_date` 使用。

## 8. 風險與待確認事項

- `product_no` 可能對應多個 `product_name` / `cate_no` / `cate_name`。
- `group_code` 來源尚未確認。
- deprecated 欄位若已存在於某些非 baseline 環境，短期保留可能造成誤用。
- `business_date` 實體欄位名不改，但語意改為 `sale_period`，需要文件與註解嚴格約束。
- payment raw 欄位不進 sales fact，可能讓使用者誤以為付款別報表可由 sales fact 完整支援；文件需避免這個誤會。
- `pos_order_status_dim` 的欄位型別與 audit 欄位慣例，可能需要和既有 `TIMESTAMPTZ` 風格再做一致性確認。

## 9. 建議驗收條件

Phase 2C-2 完成前，建議至少滿足以下條件：

- [schema_migration_plan_phase2c.md](./schema_migration_plan_phase2c.md) 已完成。
- [README.md](../README.md)、[架構指南.md](./架構指南.md)、[更新紀錄.md](./更新紀錄.md) 已引用 Phase 2C-2 migration plan。
- [fact_implementation_sequence.md](./fact_implementation_sequence.md) 已改為「先完成 plan，再建立 migration」。
- 沒有修改 [001_schema.sql](../db/init/001_schema.sql)。
- 沒有新增 migration。
- 沒有改 `sync-athena`。
- 沒有跑 Athena。
- 沒有寫 PG。
