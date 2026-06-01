# ia-analyses-db

這個 repo 負責 IA Analyses 的 PostgreSQL runtime、schema contract、seed、patch，以及 backup / restore / baseline restore 流程。正式操作入口以 `make dev-*` / `make prod-*` 為準；`sales-pipe-*`、`sync-athena-*`、`sync-sales-dims*` 只保留 bridge copy 對照用途，不再是主要操作面。

## 目前責任

- 啟動與管理 PostgreSQL 與 Adminer 容器
- 維護 `.env.dev` / `.env.prod` 與目前工作用 `.env`
- 管理 `db/init/`、`db/patches/`、seed 與 schema 演進
- 執行 backup、restore、baseline restore、smoke analytics 驗證
- 提供 `ia-analyses-go` 寫入與驗證所依賴的資料庫結構

## 常用操作

1. `make dev-env`
2. `make dev-up`
3. `make dev-migrate`
4. `make dev-smoke-analytics`
5. `make dev-backup`

常見補充情境：

- 還原一般備份：`make dev-restore BACKUP_FILE=YYYY-MM-DD-HH-MM.dump`
- 還原 baseline：`make dev-restore-baseline BASELINE_FILE=YYYY-MM-DD-HH-MM.dump`
- 查看大小：`make dev-size`
- 列出備份：`make dev-backup-list`

正式切到 prod 時，對應入口改用 `make prod-*`。

## 目錄速覽

- `db/init/`：初始 schema 與 seed contract
- `db/patches/`：正式 patch SQL
- `db/migrations_draft/`：草稿中的 DB 結構變更
- `scripts/`：backup、restore、patch、smoke test 等 shell 入口
- `backup/`：本機 dump 放置位置，實體 dump 不入 git
- `schema/`：來源 schema 或對照資料
- `cmd/`、`internal/`：過渡期 bridge copy，供和 `ia-analyses-go` 對照（`sync-athena-*` 等 Makefile target 因 `cmd/sync-athena` 不存在而無法實際執行）

## 資料模型重點

目前核心表共 7 張：

- `ia_users`
- `pos_order_type_dim`
- `pos_payment_type_dim`
- `pos_order_status_dim`
- `pos_product_dim`
- `pos_branch_dim`
- `pos_sales_hourly_fact`

其中 `pos_sales_hourly_fact` 的 grain 固定為 `owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`，`business_date` 的正式語意固定為 `sale_period`。

## 與其他 repo 的關係

- `ia-analyses-db` 擁有 DB runtime、schema 與 restore 邏輯
- `ia-analyses-go` 擁有主要 Go CLI 與 Athena 同步寫入入口
- 進行同步或驗證前，應先確保本 repo 的 DB runtime 已可用
- workspace 關聯總覽見 `../IA-Analyses-ws-map/總關聯指南.md`

## 初始化說明

本輪 `init ws` 前的舊版基線文件已封存到 `agent不再閱讀-舊版文件-2026-05-26-001/`。