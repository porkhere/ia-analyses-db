# ia-analyses-db minimal analytics baseline plan

更新日期：2026-05-24-16:27
校準日期：2026-05-24-16:27

註：附則第 1 條的現行落地只有受保護 `.gitignore` 規則與本機 dump 管理；repo 不再建立或維護 backup manifest。

## 目標

建立一套可重現的 dev 最小 analytics baseline，讓新電腦或全新 workspace 在最少步驟下得到「非空、可查、可做商品排行榜 smoke test」的 PostgreSQL 狀態，而不是只有空 schema。

目標流程：

1. `make dev-env`
2. `make dev-up`
3. `make dev-restore-baseline`
4. `make dev-smoke-analytics`

完成後至少保證：

- `pos_product_dim` 非空
- `pos_branch_dim` 非空
- `pos_sales_hourly_fact` 非空
- 可直接執行商品排行榜 smoke test

## 現況盤點

- 目前已有通用 `dev-backup` / `dev-restore` / `dev-sync-seeds` / `dev-migrate` 機制
- `backup/dev/` 與 `backup/prod/` 內的 `.dump` 一律只保留在本機，不入 git
- `backup/dev/baseline/` 是 local baseline dump 的正式位置；若沒有 local dump，`make dev-restore-baseline` 必須明確失敗
- `db/init/001_schema.sql` 只負責 schema 與 canonical seed，不會帶入 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact`
- `reports/phase2c_sales_fact_pipe_summary_*.md` 與 `state/sales_fact_pipe_state.json` 只保存摘要，不是可 restore 的資料快照
- 目前 repo 不保證 clone 後可直接跨電腦重建 non-empty baseline；若沒有 local dump，就只能得到空 schema + seed

## 明確結論

### 目前最符合治理方向的方案

- 目前正式方案是 `local baseline dump + smoke validation`
- dump 本體只保留在本機 `backup/dev/baseline/`
- repo 內只保留 restore / smoke 入口與文件，不再保留 manifest 制度

### 為什麼不再用 manifest

- 附則第 1 條已收斂為 dump exclusion 的最小落地，不再要求或維護 manifest
- backup / restore 的正式行為已回到 local dump only，避免再次膨脹成 backup 治理子系統
- 若需要 handoff baseline 的日期窗、row count、checksum，應記在 active 文件或交接材料，而不是維護額外 repo 追蹤檔

### 為什麼 baseline dump 要放子目錄

目前 `db_backup.sh`、`db_restore.sh`、`db_del_backup.sh` 都只掃 `backup/$APP_ENV` 第一層 `*.dump`。

把 baseline dump 放在 `backup/dev/baseline/` 的目的，是避免它：

- 被一般 `dev-backup` 的自動 prune 誤刪
- 被 `dev-del-backup ALL=1` 一起刪除
- 混入一般 restore 選單，降低可預測性

## 建議流程

### 維護者建立 baseline

1. 在本機建立一份最小但非空的已驗證資料庫
2. 資料範圍先鎖定在單一 owner、單一小日期窗
3. 匯出 baseline dump 到 `backup/dev/baseline/<timestamp>.dump`
4. 在乾淨 dev 容器上重新執行 `make dev-restore-baseline BASELINE_FILE=<timestamp>.dump`
5. 再執行 `make dev-smoke-analytics`
6. 若需要交接資訊，把日期窗、row count、checksum 記到 README / 架構指南 / 更新紀錄或交接材料

### 新電腦或新 workspace 重建 baseline

1. `make dev-env`
2. `make dev-up`
3. 準備好本機可用的 `backup/dev/baseline/*.dump`
4. `make dev-restore-baseline`
5. `make dev-smoke-analytics`

## 治理邊界

- 一般輪替備份：本機 `backup/dev/*.dump`、`backup/prod/*.dump`
- baseline dump：本機 `backup/dev/baseline/*.dump`
- `.dump` 一律不入 git
- repo 不再建立 `backup/manifest/**/*.md`
- 一般輪替備份仍維持每環境最多 5 份
- baseline dump 不參與自動 prune，也不應被 `ALL=1` 清除
- baseline restore 完成後仍需補跑 smoke validation

## 目前仍存在的限制

- clone repo 本身不足以拿到 non-empty analytics baseline
- baseline handoff 仍需要 out-of-band 傳遞 local dump
- 若未來需要更可追溯的 handoff 機制，應另案設計文件或交接流程，不要重新引回 manifest 制度

## 建議的 baseline 內容下限

- `ia_users` 至少 1 筆
- `pos_product_dim` 至少 10 筆
- `pos_branch_dim` 至少 2 筆
- `pos_sales_hourly_fact` 至少 1 個日期窗、可做 Top-N 商品排行

若下一輪有餘裕，可再額外納入一筆疑似非正常商品項，作為商品語意分類 smoke case；但這不是建立第一版可重現 baseline 的前置條件。
