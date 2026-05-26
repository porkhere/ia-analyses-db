# ia-analyses-db table 結構文件

## 核心結構總覽

目前初始化 schema 由 `db/init/001_schema.sql` 定義，主體是 7 張核心表與對應 index / seed。資料流方向是：`ia_users` 提供 owner 對照，商品與門市由 dimension sync 寫入維度表，銷售資料由 `sales-pipe` 寫入 `pos_sales_hourly_fact`。

## 關聯圖概念

- `ia_users.id` 被 `pos_product_dim.owner_user_id`、`pos_branch_dim.owner_user_id`、`pos_sales_hourly_fact.owner_user_id` 參照
- `pos_order_type_dim.id` 被 `pos_sales_hourly_fact.order_type_id` 參照
- `pos_payment_type_dim.id` 被 `pos_sales_hourly_fact.payment_type_id` 參照
- `pos_order_status_dim` 目前主要提供狀態語意對照與驗證，不直接成為現行 sales fact 的外鍵

## 資料表說明

### ia_users

用途：管理外部 `owner_user_key` 與內部整數 `id` 的對照。

主要欄位：

- `id`：主鍵
- `owner_user_key`：外部 owner key，唯一
- `display_name`：顯示名稱
- `source_system`：來源系統，預設 `athena`
- `is_active`：是否啟用
- `created_at`、`updated_at`：稽核時間

### pos_order_type_dim

用途：統一訂單型態的 canonical mapping。

主要欄位：

- `id`：主鍵
- `code`：穩定代碼
- `name`：顯示名稱
- `description`：說明
- `sort_order`：排序
- `is_active`、`created_at`

目前 seed 固定 10 筆，包含 `unknown`、`in_store`、`foodpanda`、`delivery`、`pickup`、`ubereats`、`quick_pickup`、`quick_delivery`、`qr_order`、`other`。

### pos_payment_type_dim

用途：統一付款型態的 canonical mapping。

主要欄位：

- `id`：主鍵
- `code`：穩定代碼
- `name`：顯示名稱
- `description`：說明
- `sort_order`：排序
- `is_active`、`created_at`

目前 seed 固定 8 筆，包含 `unknown_payment`、`cash`、`card`、`e_wallet`、`platform_payment`、`coupon`、`mixed`、`other`。

### pos_order_status_dim

用途：定義 raw order status 的報表語意。

主要欄位：

- `status_code`：主鍵，直接使用 raw status code
- `status_name`：canonical 名稱
- `status_bucket`：高階分桶，現有 `sales`、`void`、`excluded`
- `is_sales`、`is_void`、`is_cancelled_like`、`is_excluded`
- `description`：語意說明
- `sort_order`、`is_active`、`updated_at`

目前 seed 固定 4 筆，對應 `1`、`-2`、`-1`、`2` 等主要狀態。

### pos_product_dim

用途：紀錄 owner 名下的商品維度。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `product_no`：來源商品編號
- `product_name`：商品名稱
- `product_name_normalized`：正規化名稱
- `cate_no`、`cate_name`：商品分類欄位
- `is_active`、`last_seen_at`、`created_at`、`updated_at`

唯一鍵：`(owner_user_id, product_no)`

### pos_branch_dim

用途：紀錄 owner 名下的門市維度。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `branch_id`：來源門市代碼
- `branch_name`：門市名稱
- `branch_name_normalized`：正規化名稱
- `group_code`：門市群組代碼
- `is_active`、`last_seen_at`、`created_at`、`updated_at`

唯一鍵：`(owner_user_id, branch_id)`

### pos_sales_hourly_fact

用途：承接商品層級的小時銷售彙總。

grain：`owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `business_date`：正式語意固定為 `sale_period`
- `hour_of_day`：0 到 23 的小時值
- `branch_id`：來源門市代碼
- `product_no`：來源商品編號
- `order_type_id`：參照 `pos_order_type_dim.id`
- `payment_type_id`：參照 `pos_payment_type_dim.id`
- `qty_milli`：數量，千分位整數
- `gross_sales_milli`、`discount_milli`、`surcharge_milli`
- `net_sales_milli`、`sales_ex_tax_milli`
- `included_tax_milli`、`excluded_tax_milli`、`tax_milli`
- `created_at`、`updated_at`

唯一鍵就是這張 fact 的正式 grain。

## Index 重點

- `idx_ia_users_active`：owner lookup
- `idx_pos_product_dim_lookup`：商品 lookup
- `idx_pos_branch_dim_lookup`：門市 lookup
- `idx_pos_sales_hourly_fact_date`：日期與小時查詢
- `idx_pos_sales_hourly_fact_branch`：門市分析
- `idx_pos_sales_hourly_fact_product`：商品分析

## 寫入來源

- `ia-analyses-go` 的 `sync-sales-dims` 寫入 `pos_product_dim` 與 `pos_branch_dim`
- `ia-analyses-go` 的 `sales-pipe` 寫入 `pos_sales_hourly_fact`
- 本 repo 的 seed 與 patch 保證 canonical 維度與 schema contract 維持一致