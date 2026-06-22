# code-review 改善事項

建立時間：2026/06/16

## Review 範圍

- 程式/SQL：`db/init/001_schema.sql`、`db/patches/`、`scripts/`、`Makefile`
- 文件：`文件/*`、`README.md`
- 關聯 repo：`ia-analyses-go` 會寫入與讀取本 repo 的 PostgreSQL schema

## Checkpoint

- [ ] 定義正式 owner / tenant key 規則
  - 現況：`ia_users` seed 仍是 `demo-owner` / id `1`，文件已標為已知問題。
  - 關聯檔案：`db/init/001_schema.sql`、`文件/已知的問題.md`、`ia-analyses-go/internal/frontendapi/service.go`
  - 建議作法：前端 MVP 可先固定單 tenant，但要決定 `owner_user_key` 是否對應 Athena database、客戶代碼或登入租戶；定案後同步 Go repo 的預設值與既有落地資料 migration。

- [ ] 收斂 bridge copy 的生命週期
  - 現況：本 repo 仍保留 `cmd/`、`internal/` 作為 Go bridge copy；正式 Go pipeline 已在 `ia-analyses-go`。
  - 關聯檔案：`Makefile`、`cmd/`、`internal/`、`文件/架構指南.md`
  - 建議作法：保留到前端 MVP 前可以接受，但新增 Go pipeline 功能不要再落在 DB repo。等 `ia-analyses-go` 穩定後，建立一次清除 checkpoint，把 bridge copy 改成封存或移除。

- [ ] 確認 `db/init` 與 `db/patches` 的漂移管理方式
  - 現況：`001_schema.sql` 已包含 phase2c 結構，`003_phase2c_schema_contract.sql` 也存在正式 patch；這對新 DB 與舊 DB 都合理，但後續容易出現 init/patch 雙處修改不一致。
  - 關聯檔案：`db/init/001_schema.sql`、`db/patches/002_adjust_qty_and_sales_ex_tax.sql`、`db/patches/003_phase2c_schema_contract.sql`
  - 建議作法：每次 schema 變更都明確記錄「新庫 init 直接包含」與「舊庫 patch 演進」兩條路徑；新增 table 文件時標明來源是 init 還是 patch。

- [ ] 補 smoke analytics 的前端分析口徑檢查
  - 現況：`scripts/db_smoke_analytics.sh` 已檢查基本表數與 join；前端要看的 product summary / top-bottom / period comparison 口徑主要由 Go SQL 承接。
  - 關聯檔案：`scripts/db_smoke_analytics.sh`、`ia-analyses-go/internal/postgres/stat_feed_reader.go`
  - 建議作法：補一個小型 smoke 查詢，驗證 `pos_sales_hourly_fact` join `pos_product_dim` 後能產出 product-summary grain，避免前端展示時才發現資料口徑缺口。

- [x] 補 smoke analytics 的前端分析口徑檢查（部分完成）
  - 現況更新（2026/06/22）：已在 `scripts/db_smoke_analytics.sh` 中新增一個 minimal product-summary grain aggregation query（count 與 top5 preview），該腳本經 `bash -n` 語法檢查通過。
  - 執行狀態：已新增 query 並提交，但 **尚未在本環境執行實際 runtime smoke**（若需要我可以嘗試執行 `make dev-smoke-analytics`，但可能因 Docker/容器未啟動或環境限制而無法連線）。
  - 關聯檔案：`scripts/db_smoke_analytics.sh`、`ia-analyses-go/internal/postgres/stat_feed_reader.go`
  - 建議作法：若要標記為完全完成，需在可連到容器的環境執行 `make dev-smoke-analytics` 並確認有非零的 preview 結果。

- [ ] 決定 `pos_branch_dim.group_code` 的授權來源
  - 現況：schema 有欄位，但目前同步刻意寫 NULL；這對 MVP 不阻塞，但若前端要 branch group filter 會變成需求缺口。
  - 關聯檔案：`db/init/001_schema.sql`、`文件/table 結構文件.md`
  - 建議作法：前端 MVP 先不要承諾 branch group filter；若需要，先決定來源表或設定檔，再補 sync 與文件。

## 已完成的文件整理

- [x] 目前基本文件足夠：`已知的問題`、`當前工作摘要`、`架構指南`、`table 結構文件`、本檔
- [x] 舊階段文件已在 `文件/過時文件/agent不再閱讀-舊版文件-2026-05-26-001/`
