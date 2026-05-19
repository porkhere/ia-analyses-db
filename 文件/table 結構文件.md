# ia-analyses-db table 結構文件

最後檢視：2026-05-19

## Source of Truth

目前正式 table 結構以 `db/init/001_schema.sql` 與 `db/patches/` 為準；本文件只整理目前已落地的 public schema 核心表與用途。

## 維度與對照表

### ia_users

- 主鍵：`id`
- 唯一鍵：`owner_user_key`
- 用途：把外部 `owner_user_key` 正規化成內部整數鍵，供 fact 與 dim 共用

### pos_order_type_dim

- 主鍵：`id`
- 唯一鍵：`code`
- 用途：canonical 訂單型態對照表

### pos_payment_type_dim

- 主鍵：`id`
- 唯一鍵：`code`
- 用途：canonical 付款型態對照表

### pos_order_status_dim

- 主鍵：`status_code`
- 用途：固定 raw status 的 sales / void / excluded bucket 與 inclusion flags

### pos_product_dim

- 主鍵：`id`
- 唯一鍵：`owner_user_id + product_no`
- 核心欄位：`product_no`、`product_name`、`cate_no`、`cate_name`
- 用途：商品 snapshot 維度

### pos_branch_dim

- 主鍵：`id`
- 唯一鍵：`owner_user_id + branch_id`
- 核心欄位：`branch_id`、`branch_name`、`group_code`
- 用途：門市 snapshot 維度

## 事實表

### pos_sales_hourly_fact

- 主鍵：`id`
- 唯一鍵：`owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`
- grain：商品層級小時銷售彙總
- 核心指標：`qty_milli`、`gross_sales_milli`、`discount_milli`、`surcharge_milli`、`net_sales_milli`、`sales_ex_tax_milli`、`included_tax_milli`、`excluded_tax_milli`、`tax_milli`
- 備註：`business_date` 在 schema contract 中固定代表 `sale_period` 語意

## 目前索引重點

- `idx_ia_users_active`
- `idx_pos_product_dim_lookup`
- `idx_pos_branch_dim_lookup`
- `idx_pos_sales_hourly_fact_date`
- `idx_pos_sales_hourly_fact_branch`
- `idx_pos_sales_hourly_fact_product`

## 文件維護規則

- schema 新增或 patch 變更後，需同步更新本文件
- 若本文件與 `db/init/001_schema.sql` 不一致，以 SQL 為準，並應在同一輪修正文檔
