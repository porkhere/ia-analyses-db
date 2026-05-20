# QuickSight Analysis Inventory

## 範圍與方法

- 來源僅使用 QuickSight metadata：describe-analysis-definition、describe-data-set，以及已整理出的 .quicksight_analysis_metadata.json。
- 本文件不包含 PostgreSQL schema 調整、不包含 sync 實作，也不依賴大型 Athena 掃描。
- 本輪共盤點 10 份現行分析，依需求分成 Priority A、B、C。

## 全體共通觀察

- 10 份分析皆以 SPICE dataset + Custom SQL 為主。
- 銷售主線報表預設都以 orders_parquet 的 status = 1 作為有效銷售資料。
- 只有 50lan_銷售作廢統計報表 明確讀取 status = -2，並搭配 void_sale_period 區分同日作廢與跨日作廢。
- 本次盤點中沒有任何報表使用 status = -1，也沒有任何報表使用 status = 2。
- 本次盤點中沒有任何報表使用 order_additions_parquet。
- 本次盤點中沒有任何報表使用 order_item_taxes。
- 只有 50lan_付款別報表 使用 order_payments_parquet。
- 只有 50lan_調味報表 使用 order_item_condiments_parquet。
- 只有 50lan_各店門市營業時間報表 使用 branch_opening_status_daily。

## Priority A 詳細盤點

| Analysis | 主要口徑 | 日期欄位 | 門店欄位 | 商品欄位 | 數量欄位 | 金額欄位 | 來源表 | status 規則 | 補充 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 50lan_每日營業額總計表 | 每日門店營業額總表 | tr_date、sale_period | branch_id、branch_name、options | 無 | SUM(qty_subtotal) AS qty | SUM(total) AS total | orders_parquet、branches | status = 1 | 分成結單日版與營業日版，無計算欄位 |
| 50lan_產品銷售報表 | 商品/部門銷售與佔比 | t_open_date、tr_date | branch_id、branch_name、options、group_code | cate_name、product_name | SUM(ITEM.current_qty) AS qty_subtotal | SUM(ITEM.current_subtotal) AS current_subtotal | orders_parquet、order_items_parquet、branches | status = 1 | 佔比以 QuickSight calculated fields 計算 |
| 50lan_付款別報表 | 付款別金額與佔比 | t_open_date、tr_date | branch_id、branch_name、options、group_code | 無 | 無 | SUM(PAYMENT.amount - PAYMENT.change) AS amount | orders_parquet、order_payments_parquet、branches | status = 1 | 付款維度使用原始 name、memo1，不是 canonical payment type |

### 50lan_每日營業額總計表

- Dataset 分成結單日與營業日兩套新版，另保留舊版 dataset。
- 結單日版用 tr_date，營業日版用 sale_period，兩者都從 orders_parquet 聚合 total 與 qty_subtotal。
- 門店過濾與顯示使用 branch_id、branch_name 與 CONCAT(branch_id, ' ', branch_name) 組出的 options。
- 沒有 product、payment、void、additions、condiments、tax 明細層欄位。
- 視覺化主體是「日期 x 門店」的 pivot table，因此核心需求就是日期口徑與分店總額一致。

### 50lan_產品銷售報表

- 基礎 SQL 是 orders_parquet 與 order_items_parquet 的 join，並以 status = 1 篩選有效銷售。
- 商品維度依賴 cate_name、product_name；門店維度依賴 branch_id、branch_name、group_code、options。
- 金額欄位不是 orders.total，而是 item 層級的 current_subtotal；數量欄位是 item.current_qty。
- QuickSight 內另有 qty_percentage、qty_percentage_by_date_id、revenue_percentage 等佔比計算欄位。
- 多個 visual 會同時用到日期、分店、商品部門、商品名稱，因此實際上需要完整的商品維度，不只是 product_no。

### 50lan_付款別報表

- 基礎 SQL 是 orders_parquet 與 order_payments_parquet 的 join，並以 status = 1 篩選有效銷售。
- 付款維度不是 canonical 類別，而是 PAYMENT.name 與 PAYMENT.memo1 原始欄位。
- 金額欄位是 SUM(PAYMENT.amount - PAYMENT.change)，語意上是實際收款額，而不是商品銷售額分攤。
- amount_percentage 在 QuickSight 端以 percentOfTotal(sum(amount), [tr_date, branch_id]) 計算。
- 視覺化包括 pie chart 與 pivot table，並且對 name 有排除式 filter，因此 raw payment name 口徑很重要。

## Priority B / C Inventory

| Analysis | Priority | 日期口徑 | 主要維度 | 主要指標 | 來源表 | status / 特殊邏輯 |
| --- | --- | --- | --- | --- | --- | --- |
| 50lan_銷售作廢統計報表 | B | sale_period | branch_id、branch_name、options | total、num、current_num、current_total、current_void_num、current_void_total、diff_void_num、diff_void_total | orders_parquet、branches | 同時使用 status = 1 與 status = -2；以 void_sale_period 區分同日作廢與跨日作廢 |
| 50lan_訂單類型統計報表 | B | tr_date | branch_id、branch_name、destination | total、qty、order_num、total_percentage、qty_percentage、order_num_percentage | orders_parquet、branches | status = 1；destination 是報表主維度 |
| 50lan_杯數/杯單價報表 | B | tr_date | branch_id、branch_name、destination | total、qty、order_num、total_for_other、qty_for_other、total_deduct、qty_deduct、杯單價 | orders_parquet、order_items_parquet、branches | status = 1；以 cate_no = '1000' 作為 other 類別扣除 |
| 50lan_調味報表 | C | tr_date | branch_id、branch_name、condiment_group_name、name | qty_subtotal、subtotal | orders_parquet、order_item_condiments_parquet、branches | status = 1；直接依賴 condiment 明細 |
| 50lan_門店對帳單 | C | t_open_date、tr_date、sale_period | branch_id、branch_name、destination | customer_subtotal、qty_subtotal、item_subtotal、service_charge_subtotal、tax_subtotal、surcharge_subtotal、discount_subtotal、promotion_subtotal、revalue_subtotal、sale | orders_parquet、branches | status = 1；偏向訂單層對帳與拆項 |
| 50lan_營業額杯數比較報表 | C | DT | branch_id、branch_name | total、qty、total_for_other、qty_for_other、total_deduct、qty_deduct、月比/年比欄位 | orders_parquet、order_items_parquet、branches | status = 1；依賴 other 類別扣除與 period-over-period calculated fields |
| 50lan_各店門市營業時間報表 | C | sale_period | branch_id、branch_name、terminal_no | open_time、first_time、first_qty_all、first_other_qty、first_total、last_time、last_qty_all、last_other_qty、last_total、close_time | branch_opening_status_daily、branches | 不依賴 order status；branches dataset 會過濾 active branch |

## 目前最重要的 inventory 結論

- Priority A 只有每日營業額總計表最接近單純銷售彙總；產品銷售報表與付款別報表都已經超出單一總額 fact 的範圍。
- Priority B 中，銷售作廢統計報表 與 訂單類型統計報表 都強烈依賴訂單層語意，尤其是 void 狀態與 order_num。
- Priority C 中，調味報表 與 各店門市營業時間報表 本質上就是不同資料域，不適合硬塞進 pos_sales_hourly_fact。
- 目前 QuickSight inventory 對 order_additions_parquet 與 order_item_taxes 沒有直接需求，因此它們不是這一輪 fact 設計的首要壓力點。