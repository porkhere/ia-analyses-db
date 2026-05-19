# 多 Fact 資料模型設計

## 設計依據

本文件依據以下現有內容整理：

- `文件/quicksight_analysis_inventory.md`
- `文件/quicksight_metric_mapping.md`
- `文件/fact_gap_analysis.md`
- `文件/validation_plan.md`
- `db/init/001_schema.sql`
- `internal/athena/sql.go`
- `README.md`
- `文件/架構指南.md`
- `文件/更新紀錄.md`

其中 `internal/athena/sql.go` 已確認目前正式 preview sales path 來自 status-aware source：`orders_sales_candidate = status = 1 latest row`，`orders_void_candidate = status = -2 latest row`，`orders_excluded_candidate = status IN (-1, 2)`。

## 設計原則

1. 一張 fact 只承接一種穩定 grain，不為了少建表而混合多種資料域。
2. QuickSight 現行報表若依賴 raw 維度，就不能只用 canonical dim 取代。
3. 日期語意必須顯式凍結，不允許同一欄位同時代表 `sale_period` 與 `tr_date`。
4. 商品部門、raw payment、void lifecycle、condiment、branch opening 都是獨立問題，不能靠擴充 sales fact 解決。
5. PostgreSQL 寫入只能發生在 fact 邊界定稿後，不能反過來用寫入骨架推動模型。

## Phase 2C-1 Schema Contract 定稿

### 1. business_date

- Phase 2C-1 不改現有實體欄位名稱，仍使用 `business_date`。
- 從這一版開始，`business_date` 的正式商業語意凍結為 `sale_period`。
- `business_date` 不再允許兼任 `tr_date` 或 `t_open_date`。
- `tr_date` 與 `t_open_date` 之後只會進 order fact / payment fact，不會回寫到 sales fact。
- 若未來要把欄位名稱從 `business_date` 改成 `sale_period`，那會是後續 migration 議題，不屬於 Phase 2C-1 contract。

### 2. pos_sales_hourly_fact

- 保持商品層級銷售彙總 fact 定位，不擴張成萬用 fact。
- 保留現有 grain：`owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`。
- 其中 `business_date = sale_period`，`order_type_id` / `payment_type_id` 都只代表 canonical 維度。
- 正式 source rule：只吃 status-aware `status = 1` latest row。
- 明確排除欄位：`tr_date`、`t_open_date`、raw payment `name` / `memo1`、`order_count`、`void_*`、`refund_*`、`cancelled_order_count`、`cate_no`、`cate_name`、`group_code`、`options`。

### 3. pos_product_dim

- Phase 2C-1 contract 定稿為：`product_no` 繼續保留 text 自然鍵，不改 surrogate-only lookup。
- `pos_product_dim` 後續必補 `cate_no`、`cate_name`。
- `product_name`、`product_name_normalized` 保留。
- 商品部門資訊只進 product dim，不進 sales fact。

### 4. pos_branch_dim

- Phase 2C-1 contract 定稿為：`branch_id` 繼續保留 text 自然鍵。
- `pos_branch_dim` 後續必補 `group_code`。
- `branch_name`、`branch_name_normalized` 保留。
- `options` 不列為正式 schema contract 欄位；它定義為展示層或 view 層以 `branch_id + ' ' + branch_name` 組字串產生。

### 5. pos_order_status_dim

- Phase 2C-1 定稿新增 `pos_order_status_dim`，作為 raw order status 的正式語意表。
- 主鍵應直接使用 raw status code，避免再做第二層轉碼。
- 最小欄位 contract：`id`、`code`、`name`、`bucket`、`description`、`included_in_sales_fact`、`included_in_order_fact`、`included_in_void_metrics`、`sort_order`。
- v1 bucket 定稿如下：

| raw status | code | bucket | included_in_sales_fact | included_in_order_fact | included_in_void_metrics | 說明 |
| --- | --- | --- | --- | --- | --- | --- |
| `1` | `sales_completed` | `sales` | `true` | `true` | `false` | 正常銷售主口徑 |
| `-2` | `voided` | `void` | `false` | `true` | `true` | 作廢主口徑，後續搭配 `void_sale_period` |
| `-1` | `excluded_minus_1` | `excluded` | `false` | `true` | `false` | 目前只確定排除於 sales / void 主口徑，不過度宣稱業務語意 |
| `2` | `excluded_2` | `excluded` | `false` | `true` | `false` | 目前只確定排除於 sales / void 主口徑，不過度宣稱業務語意 |

### 6. payment raw 保留策略

- raw payment `name` / `memo1` 不會放進 `pos_sales_hourly_fact`。
- raw payment `name` / `memo1` 也不會被 canonical `payment_type_id` 吃掉或覆蓋。
- Phase 2C-1 定稿策略是：raw payment 維度必須在後續 payment fact 中原樣保留，canonical `payment_type_id` 只作輔助映射。
- `pos_payment_type_dim` 仍只承接 canonical payment 類別，不混入 raw payment 欄位。
- 若未來需要 mapping 治理，可另補 raw payment mapping table；但 raw payment 欄位本身仍必須保留可查。

## 不做單一萬用 fact 的理由

- QuickSight 同時需要 `sale_period`、`tr_date`、`t_open_date`，單一日期欄位無法正確覆蓋。
- `order_num` 在商品粒度與付款粒度中很容易重複計數。
- raw payment `name` / `memo1` 與 canonical `payment_type_id` 屬於不同語意層。
- `status = -2` 與 `void_sale_period` 是訂單生命週期問題，不是商品銷售 measure 問題。
- condiment 與 branch opening 本質上就是不同 grain 的資料域。
- 門店對帳單需要訂單層拆項，不適合從商品小時 fact 反推。

## pos_sales_hourly_fact 現況審查

### 目前 grain

- `owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`

### 目前欄位與用途

| 欄位 | 目前用途 | 是否保留在 sales fact | 設計判定 |
| --- | --- | --- | --- |
| `owner_user_id` | 租戶 / owner 隔離鍵 | 保留 | 必要 |
| `business_date` | 目前唯一日期欄位 | 保留，但必須凍結語意 | 最終應代表 `sale_period`；不應同時兼任 `tr_date` / `t_open_date` |
| `hour_of_day` | 小時級切片 | 保留 | 符合小時級 sales fact 責任 |
| `branch_id` | 門店自然鍵 | 保留 | 先維持 text，避免過早 surrogate key 化 |
| `product_no` | 商品自然鍵 | 保留 | 先維持 text，透過 product dim 取名稱與部門 |
| `order_type_id` | canonical 訂單型態 | 保留 | 支援 canonical order type 分析；不能取代 raw `destination` |
| `payment_type_id` | canonical 付款型態 | 保留 | 支援 canonical payment 分析；不能取代 raw payment `name` / `memo1` |
| `qty_milli` | 商品數量 | 保留 | 支援杯數 / 銷售數量分析 |
| `gross_sales_milli` | 商品層級原始售額近似值 | 保留 | 可支援產品銷售與部分對帳欄位 |
| `discount_milli` | 折扣 | 保留 | 支援折扣分析 |
| `surcharge_milli` | 附加費 | 保留 | 支援加價分析 |
| `net_sales_milli` | 含稅折後實認列銷售額 | 保留 | 可支援營業額主 measure |
| `sales_ex_tax_milli` | 未稅銷售額 | 保留 | 支援稅前分析 |
| `included_tax_milli` | 內含稅 | 保留 | 稅額細分 |
| `excluded_tax_milli` | 外加稅 | 保留 | 稅額細分 |
| `tax_milli` | 稅額總計 | 保留 | 可支援 tax report / 對帳欄位的一部分 |
| `created_at` / `updated_at` | 寫入審計欄位 | 保留 | 一般審計欄位 |

### 目前可支援的 QuickSight metrics

- `qty`
- `total` / `sale` 類營業額 measure 的 sales-side 近似值
- `current_subtotal` 類商品銷售金額的近似值
- `discount_subtotal`
- `surcharge_subtotal`
- `tax_subtotal`
- `sales_ex_tax`
- canonical `order_type_id` / `payment_type_id` 切片下的銷售分析

### 目前不能支援的 QuickSight metrics

- `tr_date` / `sale_period` / `t_open_date` 同時存在時的正確日期切換
- 安全可聚合 `order_num`
- raw payment `name` / `memo1` / `amount - change`
- `current_void_num` / `current_void_total`
- `diff_void_num` / `diff_void_total`
- condiment `qty_subtotal` / `subtotal`
- branch opening `open_time` / `close_time` / `first_time` / `last_time`
- 門店對帳單的 `customer_subtotal` / `service_charge_subtotal` / `promotion_subtotal` / `revalue_subtotal`

### 是否需要新增欄位

- 不建議在本輪直接對 sales fact 新增大量欄位。
- 優先處理的是語意凍結，而不是欄位堆疊。
- 若未來要新增欄位，應只限於 sales fact 真正負責的商品層級 measure，不應新增 raw payment、void lifecycle、condiment、branch opening 欄位。

### 是否有欄位應改名

- `business_date` 不應再維持模糊語意。
- Phase 2C-1 定稿：欄位名稱先保留 `business_date`，但正式語意固定為 `sale_period`。
- 本階段不做 rename migration。

### 是否有欄位應移出

- 現有欄位不需要移出。
- 真正的問題不是現有欄位太多，而是未來不應把不屬於 sales grain 的欄位再塞進來。

### 特別審查結論

- `business_date`：定稿為 `sale_period` 語意，但本階段先不改欄位名。
- `branch_id`：暫時保留 text 合理，因為它本來就是跨資料源共享的自然鍵。
- `product_no`：暫時保留 text 合理，因為產品分析本來就依賴自然鍵與商品維度映射。
- `cate_no` / `cate_name`：定稿補到 `pos_product_dim`，不進 sales fact。
- `order_count`：不應放進商品粒度 sales fact，因為會在多品項或多付款情境下重複計數。
- `payment_type_id`：只能代表 canonical payment 類別，不能取代 QuickSight 的 raw payment `name` / `memo1`。
- `void_milli` / `refund_milli` / `cancelled_order_count`：Phase 2C-1 定稿不進 sales fact。即使未來保留 convenience 欄位，也不能宣稱等同完整 void / refund 支援。

## A. pos_sales_hourly_fact

### 定位

- 商品層級銷售彙總
- 支援產品銷售、營業額、杯數、稅、折扣、附加費、canonical order type / payment type 分析
- 不支援 raw payment 對帳
- 不支援 condiment 明細
- 不支援營業時間
- 不支援完整 order_num / void lifecycle

### Schema contract grain

- `owner_user_id + sale_period + hour_of_day + branch_id + product_no + canonical_order_type_id + canonical_payment_type_id`

備註：實體 schema 仍沿用 `business_date`，但 contract 上等價於 `sale_period`。

### Schema contract dimensions

- `owner_user_id`
- `sale_period`
- `hour_of_day`
- `branch_id`
- `product_no`
- `order_type_id`
- `payment_type_id`

### Schema contract measures

- `qty_milli`
- `gross_sales_milli`
- `discount_milli`
- `surcharge_milli`
- `net_sales_milli`
- `sales_ex_tax_milli`
- `included_tax_milli`
- `excluded_tax_milli`
- `tax_milli`

### Schema contract exclusions

- `tr_date`
- `t_open_date`
- `raw_payment_name`
- `raw_payment_memo1`
- `order_count`
- `current_void_*`
- `diff_void_*`
- `refund_*`
- `cancelled_order_count`
- `cate_no`
- `cate_name`
- `group_code`
- `options`

### status 規則

- source path 只取 status-aware `status = 1` latest row
- `status = -2` 不進 sales fact
- `status = -1 / 2` 不進 sales fact

### date semantics

- 主日期：`sale_period`
- 不承接：`tr_date`、`t_open_date`

### 支援報表

- 50lan_產品銷售報表
- 50lan_營業額杯數比較報表
- 50lan_杯數/杯單價報表的商品銷售 / 杯數 / 金額側 measure
- 50lan_每日營業額總計表的 sale-side reconciliation 或 sale_period 版本輔助驗證

### 不支援報表

- 50lan_付款別報表
- 50lan_銷售作廢統計報表
- 50lan_調味報表
- 50lan_各店門市營業時間報表
- 50lan_門店對帳單完整版本
- 任何依賴 raw payment、完整 order count、void lifecycle 的報表

## B. pos_order_daily_fact

### 判定：daily 比 hourly 合理

理由：

- 現行需要這張 fact 的 QuickSight 報表全部以日為主，不以小時為主。
- `sale_period`、`tr_date`、`t_open_date`、`void_sale_period` 都是日級語意；做 hourly 只會放大複雜度。
- `order_num` 與 void lifecycle 是訂單日級統計問題，先做 daily 最穩定。

### 定位

- 訂單層日級統計 fact
- 承接安全可聚合 `order_num`
- 承接訂單狀態統計與 void lifecycle
- 承接門店對帳單的訂單側拆項

### 建議 grain

- `owner_user_id + sale_period + tr_date + open_date + branch_id + raw_destination + canonical_order_type_id`

### 建議 dimensions

- `owner_user_id`
- `sale_period`
- `tr_date`
- `open_date`
- `branch_id`
- `raw_destination`
- `order_type_id`

### 建議 measures

- `order_count`
- `status_1_order_count`
- `status_minus_2_order_count`
- `status_minus_1_order_count`
- `status_2_order_count`
- `sale_total_milli`
- `current_void_order_count`
- `current_void_total_milli`
- `diff_void_order_count`
- `diff_void_total_milli`
- `customer_subtotal_milli`
- `item_subtotal_milli`
- `service_charge_subtotal_milli`
- `tax_subtotal_milli`
- `surcharge_subtotal_milli`
- `discount_subtotal_milli`
- `promotion_subtotal_milli`
- `revalue_subtotal_milli`

### status 規則

- 保留 `status = 1 / -2 / -1 / 2` 的統計
- `current_void`：`status = -2` 且 `void_sale_period = sale_period`
- `diff_void`：`status = -2` 且 `void_sale_period != sale_period`

### date semantics

- `sale_period`：營業日
- `tr_date`：結單日 / transaction date
- `open_date`：開單日，對應 `t_open_date`
- `void_sale_period`：只在 void bucket / measure 計算中使用

### 可支援報表

- 50lan_每日營業額總計表
- 50lan_銷售作廢統計報表
- 50lan_訂單類型統計報表
- 50lan_杯數/杯單價報表的一部分
- 50lan_門店對帳單的一部分

### 不支援內容

- raw payment 維度與實收金額
- condiment 明細
- branch opening
- 商品層銷售比例與品項排行

## C. pos_payment_daily_fact

### 判定：daily 比 hourly 合理

理由：

- 50lan_付款別報表以日為主要分析粒度。
- 目前 QuickSight 沒有 payment-hour 報表需求。
- 保留日級即可覆蓋付款別佔比與門店付款彙總，且資料量較穩定。

### 定位

- 付款側日級統計 fact
- 保留 raw payment 維度與實收金額口徑
- 作為付款別報表與門店對帳付款側的正式來源

### 建議 grain

- `owner_user_id + sale_period + tr_date + open_date + branch_id + raw_payment_name + raw_payment_memo1 + canonical_payment_type_id`

### 建議 dimensions

- `owner_user_id`
- `sale_period`
- `tr_date`
- `open_date`
- `branch_id`
- `raw_payment_name`
- `raw_payment_memo1`
- `payment_type_id`

### 建議 measures

- `payment_amount_milli`
- `payment_change_milli`
- `payment_net_milli`
- `payment_order_count`

### status 規則

- v1 建議只納入 status-aware `status = 1` latest row
- 不把 `status = -2 / -1 / 2` 混入主付款報表口徑
- 若未來需要退款 / chargeback / 反向付款，應另定 payment adjustment 口徑，而不是直接混入 v1 payment fact

### 可支援報表

- 50lan_付款別報表
- 50lan_門店對帳單的付款側需求

### 關鍵原則

- `pos_sales_hourly_fact.payment_type_id` 不等於 QuickSight payment `name` / `memo1`
- 若要相容既有報表，raw payment 維度必須保留在 payment fact

## D. pos_condiment_hourly_fact

### 定位

- condiment / modifier 明細 fact
- 只做邊界設計，不在本輪實作

### 建議 grain

- `owner_user_id + sale_period + tr_date + hour_of_day + branch_id + product_no + condiment_group_name + condiment_name`

### 建議 dimensions

- `owner_user_id`
- `sale_period`
- `tr_date`
- `hour_of_day`
- `branch_id`
- `product_no`
- `condiment_group_name`
- `condiment_name`

### 建議 measures

- `qty_milli`
- `subtotal_milli`

### 支援報表

- 50lan_調味報表

### 邊界說明

- 不應塞進 `pos_sales_hourly_fact`
- 原因是 condiment grain 與商品銷售 grain 不同，若混在 sales fact 內，會破壞商品銷售聚合的穩定性

## E. pos_branch_opening_daily_fact

### 定位

- 門店營運狀態 / 營業時間 fact
- 只做邊界設計，不在本輪實作

### 建議 grain

- `owner_user_id + sale_period + branch_id + terminal_no`

### 建議 dimensions

- `owner_user_id`
- `sale_period`
- `branch_id`
- `terminal_no`

### 建議 measures / 屬性

- `open_time`
- `close_time`
- `first_time`
- `last_time`
- `first_total_milli`
- `last_total_milli`
- `first_qty_all_milli`
- `last_qty_all_milli`

### 支援報表

- 50lan_各店門市營業時間報表

### 邊界說明

- 這是營運狀態資料，不是 sales fact
- 不應為了方便報表而塞進 `pos_sales_hourly_fact`

## F. Dimension tables schema contract

### `pos_product_dim`

- 定稿新增：`cate_no`, `cate_name`
- `product_no` 維持 text 自然鍵
- 理由：產品銷售、杯數/杯單價、營業額杯數比較都直接依賴產品部門
- `cate_no` / `cate_name` 不進 sales fact

### `pos_branch_dim`

- 定稿新增：`group_code`
- `branch_id`、`branch_name` 維持現有自然鍵 + 顯示名稱策略
- `options` 不列入 schema contract，統一定義為 view / 展示層組字串

### `pos_order_type_dim`

- 保留 canonical order type
- raw `destination` 不應消失，應由 order fact 保留原值
- 若後續 mapping 複雜度升高，可補 raw-to-canonical mapping table

### `pos_payment_type_dim`

- 保留 canonical payment type
- raw payment `name` / `memo1` 不進 payment dim，應由 payment fact 保留原值
- 若後續 mapping 需要治理，可補 raw payment mapping table，但不能取代 raw 欄位保存

### `pos_order_status_dim`

- 定稿新增
- 最小欄位 contract：`id`, `code`, `name`, `bucket`, `description`, `included_in_sales_fact`, `included_in_order_fact`, `included_in_void_metrics`, `sort_order`
- 主要用途是凍結 raw order status 的正式語意，而不是讓每張 fact 自己重複解讀 status

### `pos_date_dim`

- 非立即阻塞，但建議列為後續維度
- 原因：目前多張報表同時需要 `sale_period`、`tr_date` 與 period-over-period 分析
- 在 facts 已先凍結日期語意後，再導入 date dim 會比先建 dim 再倒推 facts 更安全

## 各 QuickSight analysis 對應 fact

| Analysis | 建議主 fact | 需要的 dimension / 補充 |
| --- | --- | --- |
| 50lan_每日營業額總計表 | `pos_order_daily_fact` | `pos_branch_dim`；同時承接 `sale_period` 與 `tr_date` |
| 50lan_產品銷售報表 | `pos_sales_hourly_fact` | `pos_product_dim` 必須補 `cate_no` / `cate_name` |
| 50lan_付款別報表 | `pos_payment_daily_fact` | raw payment `name` / `memo1` 必須保留 |
| 50lan_銷售作廢統計報表 | `pos_order_daily_fact` | 需要 void lifecycle 與 `void_sale_period` |
| 50lan_訂單類型統計報表 | `pos_order_daily_fact` | raw `destination` + canonical `order_type_id` |
| 50lan_杯數/杯單價報表 | `pos_sales_hourly_fact` + `pos_order_daily_fact` | 商品 qty / 金額來自 sales fact；`order_num` 來自 order fact；產品部門來自 product dim |
| 50lan_調味報表 | `pos_condiment_hourly_fact` | 需要 condiment 維度 |
| 50lan_門店對帳單 | `pos_order_daily_fact` + `pos_payment_daily_fact` | 訂單側拆項來自 order fact；付款側來自 payment fact |
| 50lan_營業額杯數比較報表 | `pos_sales_hourly_fact` | `pos_product_dim` 需補 category 以處理 other 類別扣除 |
| 50lan_各店門市營業時間報表 | `pos_branch_opening_daily_fact` | 營運狀態資料域，不與 sales fact 混用 |

## Phase 2C 建議先做哪一張 fact

### 結論

- Phase 2C 不應直接等同 PostgreSQL 寫入。
- 新的 Phase 2C 應先完成多 fact 邊界凍結與 sales fact schema contract 確認。
- 第一張實際落地的 fact 仍建議是 `pos_sales_hourly_fact`，但它只能在邊界明確後落地。

### 建議順序

1. 先凍結 `pos_sales_hourly_fact` 的日期語意與 product dim 邊界
2. 再做 sales fact 的 PostgreSQL day-level replace 與 validation
3. 再往 `pos_order_daily_fact`
4. 再往 `pos_payment_daily_fact`
5. 最後處理 condiment 與 branch opening

### 原因

- `pos_sales_hourly_fact` 目前已最接近可實作狀態，而且 status-aware sales path 已在 Athena 端收斂。
- 但在 `business_date`、`cate_no` / `cate_name`、raw/canonical payment 邊界未凍結前，不應直接寫 PG。
- `pos_order_daily_fact` 與 `pos_payment_daily_fact` 會是 Priority A / B 報表相容性的第二與第三支柱，但不應與 sales fact 同時亂開工。