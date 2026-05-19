# ia-analyses-db minimal analytics baseline plan

最後檢視：2026-05-20

## 目標

建立一套可重現的 dev 最小 analytics baseline，讓新電腦或全新 workspace 在最少步驟下得到「非空、可查、可做商品排行榜 smoke test」的 PostgreSQL 狀態，而不是只有空 schema。

目標流程：

1. `make dev-env`
2. `make dev-up`
3. `make dev-restore-baseline` 或等價流程
4. `make dev-smoke-analytics`

完成後至少保證：

- `pos_product_dim` 非空
- `pos_branch_dim` 非空
- `pos_sales_hourly_fact` 非空
- 可直接執行商品排行榜 smoke test

## 現況盤點

- 目前已有通用 `dev-backup` / `dev-restore` / `dev-sync-seeds` / `dev-migrate` 機制
- `backup/dev/` 與 `backup/prod/` 目前只有 `.gitkeep`，沒有已提交的 baseline dump
- `db/init/001_schema.sql` 只負責 schema 與 canonical seed（order type / payment type / order status），不會帶入 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact`
- 現有 `reports/phase2c_sales_fact_pipe_summary_*.md` 與 `state/sales_fact_pipe_state.json` 只保存摘要，不保存可 restore 的資料快照
- 現場也沒有可用 AWS shared config / credential，因此不能把「clone 後自動 replay Athena」視為可靠 baseline 方案

結論：目前 repo 尚不存在「可重建非空 analytics baseline」的正式載體。

## 方案比較

| 方案 | POC 適配度 | 跨電腦重建 | drift 風險 | 與現有治理一致性 | 評語 |
| --- | --- | --- | --- | --- | --- |
| backup 入 git（POC 小資料） | 高 | 高 | 中 | 高 | 最符合現在的 restore / backup / Makefile 路徑 |
| seed SQL | 中 | 高 | 中高 | 中 | 可 diff，但維護 fact/dim 小樣本會很快變成人工資料腳本 |
| parquet sample | 中低 | 中 | 中 | 低 | 還要補 importer / replay 流程，無法走現有 restore 主路徑 |
| local snapshot restore | 中 | 低 | 中 | 中 | 本機可用，但不適合 clone 後跨電腦重建 |
| pipeline 自動 replay | 低 | 低 | 低 | 低 | 最接近 source truth，但依賴 Athena / AWS 權限，不符合目前環境條件 |
| hybrid：tracked backup + manifest + smoke validation | 高 | 高 | 低中 | 高 | 最適合目前 POC 的實際推薦方案 |

## 明確結論

### 哪個方案最符合目前 POC 階段

- 若只選單一載體，`backup 入 git（POC 小資料）` 最符合目前 POC 階段。
- 原因是 repo 已有正式 `dev-restore` / `prod-restore` 路徑，agent-rule 也明確允許且要求 POC backup 可入 git。

### 哪個方案最容易跨電腦重建

- `backup 入 git` 與 `hybrid` 最容易跨電腦重建。
- clone repo 後不依賴 Athena、不依賴 AWS credential、不依賴長時間 replay，只要 restore 即可。

### 哪個方案最不容易 drift

- 若只談理論上的 source 對齊，`pipeline 自動 replay` 最不容易 drift。
- 但在目前 POC 環境，最實際且最不容易 drift 的方案是 `hybrid`：用已提交 baseline dump 當 restore 載體，再配 manifest、row count 與 smoke validation 固定基準。

### 哪個方案最符合目前 agent-rule 與 Makefile 治理方向

- `backup 入 git` 與 `hybrid` 最符合目前治理方向。
- 現有 Makefile、backup/restore script、agent-rule、README 都已把 backup/restore 定義成正式入口，直接延伸這條路徑最一致。

## 建議方案

建議採用 `hybrid：tracked minimal backup + manifest + smoke validation`。

其中真正承載資料的是「已提交到 git 的 dev baseline dump」，其餘配套用來降低 drift 與維護成本。

### 核心設計

1. 以一個很小的已驗證資料窗建立 dev baseline dump。
2. baseline dump 放在 `backup/dev/baseline/` 子目錄，不和一般輪替備份混在一起。
3. 一般 `dev-backup` 仍只寫入 `backup/dev/*.dump`，維持 5 份輪替；tracked baseline 不參與自動 prune。
4. baseline 另附 manifest，記錄資料窗、owner、row count、預期 smoke query 結果與 dump checksum。
5. restore 完成後執行固定 smoke validation，確認非空 fact/dim 與排行榜查詢可跑。

### 為什麼 baseline dump 要放子目錄

目前 `db_backup.sh`、`db_restore.sh`、`db_del_backup.sh` 都只掃 `backup/$APP_ENV` 的第一層 `*.dump`。

若把 tracked baseline 直接放在 `backup/dev/*.dump`：

- 可能被 `dev-backup` 的自動 prune 誤刪
- 可能被 `dev-del-backup ALL=1` 一起刪除
- 也會混入一般 restore 選單，降低可預測性

因此建議改放：

- `backup/dev/baseline/<timestamp>.dump`

例如：

- `backup/dev/baseline/2026-05-20-12-00.dump`

這樣仍符合 backup 路徑治理，但不會被現有通用 backup rotation 腳本誤處理。

## 建議流程草圖

### 維護者建立 baseline

1. 在本機建立一份最小但非空的已驗證資料庫
2. 資料範圍先鎖定在單一 owner、單一小日期窗
3. 匯出 baseline dump 到 `backup/dev/baseline/<timestamp>.dump`
4. 產生 manifest：owner、日期窗、核心表 row count、預期 smoke 結果、checksum
5. 在乾淨 dev 容器上重新 restore 驗證
6. 更新 README / 文件 / 更新紀錄
7. commit / push

### 新電腦使用者重建 baseline

1. `make dev-env`
2. `make dev-up`
3. `make dev-restore-baseline`
4. `make dev-smoke-analytics`

## Makefile 建議新增或調整入口

### 建議新增

- `dev-restore-baseline`
  - restore repo 內已提交的 tracked minimal baseline dump
  - 內部走既有 `db_restore.sh`，但傳入明確 baseline 路徑

- `dev-smoke-analytics`
  - 驗證 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact` 皆非空
  - 跑一個固定商品排行榜 smoke query
  - 後續若商品語意分類完成，再補「特殊交易項不應進排行榜」檢查

- `dev-baseline-info`
  - 顯示目前 baseline dump 路徑、manifest 版本、日期窗、row count 摘要

### 可選但不必在下一輪一次做完

- `dev-refresh-baseline`
  - 維護者專用，從目前本機 PG 重新產 baseline dump 與 manifest
  - 不建議一開始就自動化太多，先手動流程落地即可

### 現有入口建議調整

- `help` 補一組「最小 analytics baseline」說明
- README 的 backup / restore 章節補上「一般輪替備份」與「tracked baseline backup」的差異

## backup / seed / restore 治理方式

### backup 治理

- 一般輪替備份：`backup/dev/*.dump`、`backup/prod/*.dump`
- tracked baseline：`backup/dev/baseline/*.dump`
- 一般輪替備份仍維持最多 5 份
- tracked baseline 不參與自動 prune，也不應被 `ALL=1` 清除

### seed 治理

- `db/init/001_schema.sql` 繼續只管 schema 與 canonical seed
- 不把 analytics smoke dataset 直接塞進 `db/init/001_schema.sql`
- 原因是 analytics baseline 與 schema seed 的生命週期不同，混在同一份 SQL 容易造成 drift 與維護成本失控

### restore 治理

- 一般 restore 維持現行互動式 restore
- tracked baseline restore 用專屬入口，避免使用者從一串臨時 dump 中手動選檔
- baseline restore 完成後仍需補跑 migrate 與 smoke validation
- restore 屬高風險操作，繼續沿用現有 pre-restore backup 規則

## 下一輪真正要實作的最小範圍

1. 選定一個小日期窗，建立單一 owner 的 tracked dev baseline dump
2. 把 baseline dump 放到 `backup/dev/baseline/`
3. 新增 baseline manifest
4. 新增 `make dev-restore-baseline`
5. 新增 `make dev-smoke-analytics`
6. 驗證 restore 後三張核心 analytics 表非空，且商品排行榜 smoke query 可跑

## 建議的 baseline 內容下限

- `ia_users` 至少 1 筆
- `pos_product_dim` 至少 10 筆
- `pos_branch_dim` 至少 2 筆
- `pos_sales_hourly_fact` 至少 1 個日期窗、可做 Top-N 商品排行

若下一輪有餘裕，可再額外納入一筆疑似非正常商品項，作為商品語意分類 smoke case；但這不是建立第一版可重現 baseline 的前置條件。