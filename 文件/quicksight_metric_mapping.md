# QuickSight Metric Mapping

## 判定規則

- Direct：目前 fact 或現有 dimension 已可直接支援，只需正常聚合或 join。
- Partial：有接近欄位，但仍缺少日期語意、明細維度、或需要額外衍生規則。
- Missing：目前 fact/dimension 沒有對應欄位，不能直接重建 QuickSight 口徑。

## 共用維度對照

| QuickSight 欄位 | 使用報表 | 目前對應 | 判定 | 說明 |
| --- | --- | --- | --- | --- |
| tr_date | 每日營業額、產品銷售、付款別、訂單類型、杯數/杯單價、門店對帳單 | pos_sales_hourly_fact.business_date | Partial | 現況只有單一 business_date，無法同時保留 tr_date 與 sale_period 語意 |
| sale_period | 每日營業額、銷售作廢、門店對帳單、營業時間 | pos_sales_hourly_fact.business_date | Partial | 同上；若 business_date 定義為結單日，營業日報表會失真 |
| t_open_date | 產品銷售、付款別、門店對帳單 | 無 | Missing | 目前 fact 沒有原始開單日期欄位 |
| branch_id | 全部銷售類報表 | pos_sales_hourly_fact.branch_id | Direct | 可直接聚合 |
| branch_name / options | 多數報表 | pos_branch_dim.branch_name + 組字串 | Direct | 需 join branch dim 或 view 組合欄位 |
| product_no | 現有 fact 主鍵 | pos_sales_hourly_fact.product_no | Direct | QuickSight 多數直接顯示 product_name 而不是 product_no |
| product_name | 產品銷售相關 | pos_product_dim.product_name | Direct | 需要可靠的 product dim 映射 |
| cate_name | 產品銷售、杯數/杯單價、營業額杯數比較 | 無 | Missing | 現行 product dim 沒有產品部門資訊 |
| destination | 訂單類型、杯數/杯單價、門店對帳單 | pos_order_type_dim | Direct | 目前只有 canonical order_type，若需保留 raw destination 仍需額外欄位或 view |
| payment name / memo1 | 付款別報表 | pos_payment_type_dim.payment type only | Missing | 現有只有 canonical payment_type_id，沒有 raw name、memo1 |
| condiment_group_name / condiment name | 調味報表 | 無 | Missing | 完全缺少 condiment 維度 |
| terminal_no | 營業時間報表 | 無 | Missing | 目前 fact 沒有 terminal 維度 |
| open_time / close_time / first_time / last_time | 營業時間報表 | 無 | Missing | 這是營運時段資料，不是銷售聚合欄位 |

## Priority A 指標對照

| QuickSight 指標 | QuickSight 來源公式 | 目前對應 | 判定 | 說明 |
| --- | --- | --- | --- | --- |
| 每日營業額總計表.total | SUM(orders.total) | pos_sales_hourly_fact.net_sales_milli | Partial | 金額語意最接近，但是否等於現有 ORD.total 仍須以 validation 確認；另受日期語意限制 |
| 每日營業額總計表.qty | SUM(orders.qty_subtotal) | pos_sales_hourly_fact.qty_milli | Partial | 基礎數量可對上，但同樣受 tr_date / sale_period 二選一限制 |
| 產品銷售.qty_subtotal | SUM(order_items.current_qty) | pos_sales_hourly_fact.qty_milli | Partial | 可從商品粒度聚合，但缺 cate_name 與日期雙語意 |
| 產品銷售.current_subtotal | SUM(order_items.current_subtotal) | pos_sales_hourly_fact.gross_sales_milli | Partial | 最接近的是 gross_sales_milli，但要先確認 gross 的定義與 current_subtotal 是否一致 |
| qty_percentage | percentOfTotal(sum(qty_subtotal)) | 由 BI 層重算 | Partial | 前提是 qty_subtotal 基礎欄位先成立 |
| qty_percentage_by_date_id | percentOfTotal(sum(qty_subtotal), [tr_date, branch_id]) | 由 BI 層重算 | Partial | 需要 tr_date 可用 |
| revenue_percentage | percentOfTotal(sum(current_subtotal)) | 由 BI 層重算 | Partial | 需要 current_subtotal 基礎欄位可還原 |
| 付款別.amount | SUM(order_payments.amount - order_payments.change) | 無精確對應 | Missing | 目前 fact 沒有 raw payment amount measure |
| 付款別.amount_percentage | percentOfTotal(sum(amount), [tr_date, branch_id]) | 由 BI 層重算 | Missing | 基礎 amount 缺失，因此比例也無法精確重建 |

## Priority B / C 指標對照

| QuickSight 指標 | 使用報表 | 目前對應 | 判定 | 說明 |
| --- | --- | --- | --- | --- |
| current_void_num / diff_void_num | 銷售作廢統計報表 | 無 | Missing | 需要 status = -2 與 void_sale_period 邏輯 |
| current_void_total / diff_void_total | 銷售作廢統計報表 | 無 | Missing | 目前 sales fact 沒有 void 拆分欄位 |
| order_num | 訂單類型統計、杯數/杯單價 | 無 | Missing | 現行 grain 是 product x payment x hour，不能安全直接放訂單筆數 |
| total | 訂單類型統計 | pos_sales_hourly_fact.net_sales_milli | Partial | 金額可聚合，但仍缺 order_num 與原始日期語意 |
| qty | 訂單類型統計 | pos_sales_hourly_fact.qty_milli | Partial | 分類可依 order_type_id，但 order_num 仍缺 |
| total_for_other / qty_for_other | 杯數/杯單價、營業額杯數比較 | 無直接對應 | Missing | 需要產品類別 cate_no = '1000' 的標記 |
| total_deduct / qty_deduct | 杯數/杯單價、營業額杯數比較 | 需以 base sales 排除 other 類別後衍生 | Partial | 前提是 product dim 先補 cate_no / cate_name |
| 杯單價 | 杯數/杯單價 | 由 total_deduct / qty_deduct 重算 | Partial | 依賴上列衍生欄位 |
| subtotal | 調味報表 | 無 | Missing | 需要 condiment fact |
| qty_subtotal（調味） | 調味報表 | 無 | Missing | 需要 condiment fact |
| customer_subtotal | 門店對帳單 | 無 | Missing | 現行 fact 沒有 customer subtotal |
| item_subtotal | 門店對帳單 | pos_sales_hourly_fact.gross_sales_milli | Partial | 接近，但需先確認 gross 定義與 QuickSight item_subtotal 一致 |
| service_charge_subtotal | 門店對帳單 | 無 | Missing | 目前沒有 service charge measure |
| tax_subtotal | 門店對帳單 | pos_sales_hourly_fact.tax_milli | Direct | 稅額欄位存在 |
| surcharge_subtotal | 門店對帳單 | pos_sales_hourly_fact.surcharge_milli | Direct | 附加費欄位存在 |
| discount_subtotal | 門店對帳單 | pos_sales_hourly_fact.discount_milli | Direct | 折扣欄位存在 |
| promotion_subtotal | 門店對帳單 | 無 | Missing | 目前沒有 promotion 拆項 |
| revalue_subtotal | 門店對帳單 | 無 | Missing | 目前沒有 revalue 拆項 |
| sale | 門店對帳單 | pos_sales_hourly_fact.net_sales_milli | Partial | 金額接近，但整張報表仍缺多個對帳欄位 |
| open_time / close_time / first_time / last_time | 各店門市營業時間報表 | 無 | Missing | 需來自 branch_opening_status_daily 類型的營運資料 |

## 結論

- 目前 pos_sales_hourly_fact 能直接承接的，主要是 branch/product/order_type/payment_type 粒度下的銷售金額、折扣、附加費、稅額、數量聚合。
- 真正卡住 QuickSight 相容性的，不是單純 measure 不夠，而是日期語意、原始付款維度、產品部門、訂單筆數、void 邏輯、condiment 明細與營業時間資料域不同。
- 付款別報表與作廢統計報表是目前最不適合硬映射到單一 sales fact 的兩個案例。