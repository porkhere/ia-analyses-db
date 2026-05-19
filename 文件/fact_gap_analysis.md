# Fact Gap Analysis

## 總結判定

- 目前的 pos_sales_hourly_fact 適合作為「商品層級銷售彙總 fact」，不適合作為所有 QuickSight 報表的唯一來源。
- 如果目標只是支援基本銷售、商品銷售、稅額與折扣分析，它已經有不錯的骨架。
- 如果目標是完整覆蓋客戶現有 10 份 QuickSight 分析，現況明顯不足，且不足點是結構性問題，不是多補幾個 measure 就能解決。

## 目前 fact 的強項

- 已有 branch_id、product_no、order_type_id、payment_type_id，可承接銷售主軸的維度切片。
- 已有 qty_milli、gross_sales_milli、discount_milli、surcharge_milli、net_sales_milli、sales_ex_tax_milli、tax_milli，可承接主銷售 measure。
- 小時粒度對日報、時段分析、後續日/月匯總都算合理。

## 主要結構性缺口

### 1. 只有單一 business_date，無法同時支援 tr_date 與 sale_period

- 目前 schema 只有 business_date。
- 但現行 QuickSight 同時存在結單日報表與營業日報表。
- 產品銷售、付款別、門店對帳單還會混用 t_open_date、tr_date、sale_period。
- 結論：若不增加日期語意，Priority A 的每日營業額總計表就已經無法完整一比一重建。

### 2. 缺少安全可聚合的 order_num

- 訂單類型統計、杯數/杯單價、作廢統計都需要 order count。
- 現有 fact grain 是 owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id。
- 在這個 grain 下直接存 order_num 幾乎必然重複計數，尤其遇到多品項或混合付款時。
- 結論：訂單筆數應屬於獨立的 order-level fact，不應硬塞進目前 sales fact。

### 3. 缺少 raw payment 維度與 raw payment amount

- 付款別報表依賴 PAYMENT.name、PAYMENT.memo1，且 amount 是 SUM(amount - change)。
- 現有 fact 只有 canonical payment_type_id，沒有 raw payment name、memo1，也沒有 payment amount measure。
- 即使未來把銷售額按付款方式分攤，也不等於 QuickSight 目前使用的收款額口徑。
- 結論：付款別報表不能以目前 pos_sales_hourly_fact 精確取代。

### 4. 缺少產品部門資訊

- 產品銷售報表、杯數/杯單價、營業額杯數比較都依賴 cate_name，部分 SQL 還會依賴 cate_no = '1000' 來排除 other 類別。
- 現行 pos_product_dim 只有 product_no、product_name，沒有 cate_no、cate_name。
- 結論：若不補產品部門維度，產品類報表只能做到部分相容。

### 5. 缺少 void lifecycle fact

- 銷售作廢統計報表不是單純看 status = -2 總額，而是要拆 current_void 與 diff_void。
- 這個邏輯依賴 void_sale_period 與 sale_period 的對照。
- 現有 sales fact 沒有 status 欄位，也沒有 current/diff void measures。
- 結論：void 報表需要獨立的 order status / void fact。

### 6. 缺少 condiment / modifier 資料域

- 調味報表完全依賴 order_item_condiments_parquet。
- 這類欄位與商品銷售 fact 的 grain 不同，硬塞進 pos_sales_hourly_fact 只會讓 grain 更混亂。
- 結論：調味資料應獨立 fact 化，不應強行併入主 sales fact。

### 7. 缺少營業時間資料域

- 各店門市營業時間報表使用 branch_opening_status_daily，包含 open_time、close_time、terminal_no、first/last 交易資訊。
- 這不是銷售彙總問題，而是門店營運狀態問題。
- 結論：營業時間報表應獨立於 sales fact。

### 8. 門店對帳單所需欄位不完整

- 現有 fact 已有 discount、surcharge、tax、net sales 等 measure。
- 但門店對帳單還需要 customer_subtotal、service_charge_subtotal、promotion_subtotal、revalue_subtotal 等欄位。
- 這些欄位目前不存在，而且多數偏向訂單層拆項，不宜用產品小時 fact 勉強承載。

## 各報表支援度判定

| Analysis | 現況判定 | 主要原因 |
| --- | --- | --- |
| 50lan_每日營業額總計表 | Partial | 核心 total、qty 可對上，但 tr_date / sale_period 雙日期語意尚未解決 |
| 50lan_產品銷售報表 | Partial | 核心 qty、gross 類 measure 可對上，但缺 cate_name / cate_no 與多日期語意 |
| 50lan_付款別報表 | Unsupported | 缺 raw payment name、memo1、payment amount 口徑 |
| 50lan_銷售作廢統計報表 | Unsupported | 缺 status = -2 與 void_sale_period 拆分欄位 |
| 50lan_訂單類型統計報表 | Partial | order_type 可映射，但缺安全可聚合的 order_num |
| 50lan_杯數/杯單價報表 | Partial | 基礎 qty / amount 存在，但缺 cate_no = '1000' 邏輯與 order_num |
| 50lan_調味報表 | Unsupported | 缺 condiment fact |
| 50lan_門店對帳單 | Unsupported | 缺多個對帳拆項欄位，且報表偏訂單層語意 |
| 50lan_營業額杯數比較報表 | Partial | 基礎 total / qty 可重建，但缺產品部門與 other 類別排除邏輯 |
| 50lan_各店門市營業時間報表 | Unsupported | 屬於營運時段資料，不是 sales fact 可替代 |

## 是否需要第二張 fact table

- 需要，而且只靠「再加一張萬用 fact」其實仍不夠。
- 如果只看最小可行方向，第一張新增 fact 應該是 order-level fact，而不是再把主 sales fact 擴胖。
- 原因很直接：目前最大的缺口集中在 order_num、void lifecycle、對帳拆項、raw payment，這些都比較接近訂單層，不接近商品小時彙總層。

## 建議的未來分工方向

### 保留 pos_sales_hourly_fact 的責任

- 專注在商品層級銷售、數量、折扣、附加費、稅額、order type / payment type 的 canonical 聚合。
- 補齊 Priority A 所需的最小缺口時，優先考慮日期語意與產品部門維度，不要先塞入 void、condiment、營業時間。

### 優先新增的 fact 類型

- order-level fact：承接 order_num、void 狀態、void_sale_period 拆分、門店對帳單所需訂單層拆項。
- payment fact：承接 raw payment name、memo1、amount - change 等真實付款口徑。
- condiment fact：承接 condiment_group_name、condiment name、qty、subtotal。
- branch opening fact：承接營業時間與 terminal 粒度資訊。

## 這一輪不建議做的事

- 不建議把 condiment 報表強塞進 pos_sales_hourly_fact。
- 不建議把營業時間報表視為 sales fact 的延伸欄位。
- 不建議為了補 order_num 而在現行 grain 直接新增 count 欄位，因為極容易重複計數。
- 不建議先做 PostgreSQL 寫入 skeleton，再回頭定義 date / payment / void 語意。

## 這一輪 gap analysis 的結論

- 若目標是「Priority A 先上線」，可以在不動大量資料域的前提下先處理日期語意與產品部門維度。
- 若目標是「完整替換既有 QuickSight 10 份分析」，單靠目前 pos_sales_hourly_fact 不足，必須接受多 fact 設計。
- 因此下一步應先決定 fact 邊界，而不是直接開始做 PG 寫入。

## 下一步不應直接寫 PG

- 不應先建立新的 PostgreSQL sync skeleton，再回頭補資料模型定義。
- 不應先做 day-level replace，然後才決定 `business_date`、`tr_date`、`sale_period` 的分工。
- 不應把 raw payment、void lifecycle、condiment、branch opening 先塞進 `pos_sales_hourly_fact` 來換取短期進度。
- 正確順序應該是：先定 data model boundary，再進入 fact schema contract 與寫入實作。

## Priority A 未來支援方式

| Analysis | 建議主 fact | 補充 |
| --- | --- | --- |
| 50lan_每日營業額總計表 | `pos_order_daily_fact` | 這張報表的核心是訂單日級總額與雙日期語意；`pos_sales_hourly_fact` 可做 sale_period 口徑的交叉驗證，但不應作為唯一來源 |
| 50lan_產品銷售報表 | `pos_sales_hourly_fact` + `pos_product_dim` | 前提是 `pos_product_dim` 補 `cate_no` / `cate_name`，且 `business_date` 語意凍結 |
| 50lan_付款別報表 | `pos_payment_daily_fact` | 必須保留 raw payment `name` / `memo1` 與 `amount - change`，不能用 canonical `payment_type_id` 取代 |

因此，Priority A 可以由 sales fact + order fact + payment fact 的組合支援，但不應強行要求單一 sales fact 一次包完三張報表。