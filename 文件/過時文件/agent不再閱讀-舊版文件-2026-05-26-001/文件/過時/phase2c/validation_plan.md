# Validation Plan

## 目的

- 本文件定義未來 fact 設計確認後的驗證順序。
- 本輪只定義比對方法與抽樣範圍，不做 PostgreSQL schema 實作，也不做 sync 寫入實作。

## 驗證前必須先凍結的前提

- 先定義 business_date 到底代表 tr_date 還是 sale_period；若要同時支援兩者，必須在 fact 或 view 層明確暴露雙日期欄位。
- 先定義 ORD.total 對應到哪個 fact measure，預設應以 net_sales_milli 為最近似值，但必須經抽樣驗證。
- 先定義 ITEM.current_subtotal 是否對應 gross_sales_milli，避免產品銷售報表語意漂移。
- 先決定付款別報表是要保留 raw payment name / memo1，還是接受 canonical payment type 重新定義。
- 若要驗證產品類與杯數類報表，需先決定 product dim 是否補 cate_no、cate_name。

## 驗證原則

- 先驗證 Priority A，再擴到 Priority B，最後才碰 Priority C。
- 先用單店小窗驗證口徑，再做全店長窗驗證。
- 每一階段都要同時比對 QuickSight dataset SQL、Athena 手工查核結果、ia-analyses-db 輸出，以及未來 PostgreSQL fact 結果。
- 只有在前一階段通過後，才放大日期範圍或門店範圍。

## 分階段驗證

| Stage | 範圍 | 目標報表 | 主要檢核項目 | 通過條件 |
| --- | --- | --- | --- | --- |
| Stage 1 | branch_id = TWA2001，2025-01-01 ~ 2025-01-03 | 每日營業額總計表、產品銷售報表、付款別報表 | 日期口徑、branch filter、total / qty / product amount / payment amount 對齊 | Priority A 核心欄位可逐列對上，差異可被明確解釋 |
| Stage 2 | branch_id = TWA2001，2025-01-01 ~ 2025-01-07 | Priority A 全部 + 銷售作廢統計、訂單類型統計、杯數/杯單價 | 小窗延伸到 7 天，驗證 void、order_num、destination、other 類別排除 | 所有已宣告支援的欄位穩定對齊，沒有日期漂移 |
| Stage 3 | 全分店，2025-01-01 ~ 2025-01-07 | Priority A + Priority B | 驗證全店 filter、group_code、branch join、跨店聚合 | 總額與分店彙總都對齊，抽樣店家無異常外點 |
| Stage 4 | 全分店，2025-01-01 ~ 2025-01-31 | 全部已宣告支援的報表 | 長窗穩定性、月度聚合、period-over-period 指標 | 31 天內不出現系統性偏差，再考慮進入正式寫入 |

## 各報表建議比對欄位

### 50lan_每日營業額總計表

- 維度：日期、branch_id、branch_name。
- 指標：total、qty。
- 要分開驗證兩種日期語意：tr_date 版本與 sale_period 版本。

### 50lan_產品銷售報表

- 維度：日期、branch_id、cate_name、product_name。
- 指標：qty_subtotal、current_subtotal。
- 若 cate_name 尚未補齊，這張報表只能做部分驗證，不應宣告 fully supported。

### 50lan_付款別報表

- 維度：tr_date、branch_id、payment name、memo1。
- 指標：amount。
- 若未來只保留 canonical payment type，必須另外做「新口徑 vs 舊 QuickSight 口徑」差異說明，不能直接當成同一報表驗過。

### 50lan_銷售作廢統計報表

- 維度：sale_period、branch_id。
- 指標：sale_num、sale_total、current_void_num、current_void_total、diff_void_num、diff_void_total。
- 這張報表一定要做 current vs diff void 拆分驗證，不能只驗總 void 金額。

### 50lan_訂單類型統計報表

- 維度：tr_date、branch_id、destination。
- 指標：total、qty、order_num。
- order_num 若沒有 order-level fact，不應進入正式驗證階段。

### 50lan_杯數/杯單價報表 與 50lan_營業額杯數比較報表

- 維度：日期、branch_id、destination 或 branch_id。
- 指標：total、qty、total_for_other、qty_for_other、total_deduct、qty_deduct、杯單價。
- 先確認 cate_no = '1000' 的 other 類別能在新模型中被正確標記，再做驗證。

### 50lan_調味報表

- 維度：tr_date、branch_id、condiment_group_name、name。
- 指標：qty_subtotal、subtotal。
- 只有在 condiment fact 成立後才建議進驗證。

### 50lan_門店對帳單

- 維度：t_open_date、tr_date、sale_period、branch_id、destination。
- 指標：customer_subtotal、item_subtotal、service_charge_subtotal、tax_subtotal、surcharge_subtotal、discount_subtotal、promotion_subtotal、revalue_subtotal、sale。
- 若缺任一訂單層拆項欄位，就只能做 partial validation，不可宣告完成替代。

### 50lan_各店門市營業時間報表

- 維度：sale_period、branch_id、terminal_no。
- 指標：open_time、first_time、first_total、last_time、last_total、close_time。
- 這張報表應獨立於 sales fact 驗證。

## 建議的資料來源優先順序

1. QuickSight dataset Custom SQL：確認現行定義沒有理解偏差。
2. Athena 手工查核 SQL：用最小窗確認 source result。
3. ia-analyses-db 預覽或 dry-run 結果：確認新轉換規則沒有偏移。
4. PostgreSQL fact query：確認寫入後聚合結果一致。

## 執行上的 guardrails

- Stage 1 與 Stage 2 優先只做單店，避免一開始就走大掃描。
- Priority C 報表不要在銷售主 fact 尚未定義清楚前提前驗證。
- 付款別與作廢統計如果口徑仍未定案，不應被放進「已可上線」清單。

## 後續需依 fact 分開驗證

### Sales fact validation

- 驗證 `pos_sales_hourly_fact` 是否正確支援產品銷售、杯數、稅、折扣、附加費。
- 驗證 `business_date` 是否已明確收斂為 `sale_period`。
- 驗證 `pos_product_dim` 是否已補 `cate_no` / `cate_name`，足以支援商品部門分析。

### Order fact validation

- 驗證 `pos_order_daily_fact` 的 `order_count` 是否安全可聚合。
- 驗證 `status = 1 / -2 / -1 / 2` 的統計與 `void_sale_period` 拆分。
- 驗證每日營業額總計表、銷售作廢統計報表、訂單類型統計報表與門店對帳單訂單側欄位。

### Payment fact validation

- 驗證 `pos_payment_daily_fact` 是否保留 raw payment `name` / `memo1`。
- 驗證 `amount`、`change`、`amount - change` 是否與 QuickSight 付款別報表一致。
- 驗證 canonical `payment_type_id` 與 raw payment 維度的 mapping 是否只作輔助，不覆蓋原始報表口徑。

### Condiment fact validation

- 驗證 `pos_condiment_hourly_fact` 的 condiment group / name、qty、subtotal。
- 驗證這張 fact 與商品銷售 fact 的 grain 分離是否正確，避免重複計數。

### Branch opening fact validation

- 驗證 `pos_branch_opening_daily_fact` 的 `sale_period`、`terminal_no`、`open_time`、`close_time`、first/last 指標。
- 驗證這張 fact 作為營運狀態資料，而非 sales fact 延伸欄位。

## 本輪結論

- 驗證順序應先解決語意，再做數值比對。
- 最先需要被凍結的是日期語意、產品部門維度、付款口徑與 void 拆分口徑。
- validation 不應只驗 `pos_sales_hourly_fact`；必須依 sales / order / payment / condiment / branch opening 各 fact 分開驗證。
- 只有這些前提清楚後，後續 PostgreSQL fact 驗證才有意義。