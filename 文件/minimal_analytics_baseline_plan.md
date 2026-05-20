# ia-analyses-db minimal analytics baseline plan

更新日期：2026-05-20-22:54
校準日期：2026-05-20-22:54

註：2026-05-20-22:54 起已啟用附則第 1 條。DB backup 實體檔不入 git；本文件中的現行方案一律改以「local dump + committed manifest」理解，舊的 `backup 入 git`、`tracked baseline dump` 字樣若未另外標示，均視為已被新政策覆蓋。

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
| local backup + committed manifest（POC 小資料） | 中高 | 中 | 中 | 高 | 符合附則第 1 條，但重建前提改成同機器或可取得同一份 local dump |
| seed SQL | 中 | 高 | 中高 | 中 | 可 diff，但維護 fact/dim 小樣本會很快變成人工資料腳本 |
| parquet sample | 中低 | 中 | 中 | 低 | 還要補 importer / replay 流程，無法走現有 restore 主路徑 |
| local snapshot restore | 中 | 低 | 中 | 中 | 本機可用，但不適合 clone 後跨電腦重建 |
| pipeline 自動 replay | 低 | 低 | 低 | 低 | 最接近 source truth，但依賴 Athena / AWS 權限，不符合目前環境條件 |
| hybrid：local baseline dump + committed manifest + smoke validation | 高 | 中 | 低中 | 高 | 最符合附則第 1 條的實際推薦方案 |

## 明確結論

### 哪個方案最符合目前 POC 階段

- 附則第 1 條生效後，若只選單一載體，`local backup + committed manifest（POC 小資料）` 最符合目前 POC 階段。
- 原因是 repo 既有 `dev-restore` / `prod-restore` 路徑仍可直接使用，只需把追蹤責任從 dump 本體轉成 manifest。

### 哪個方案最容易跨電腦重建

- 在附則第 1 條下，任何不把 dump 入 git 的方案都不再保證「clone 後立即跨電腦重建」。
- 現行可行前提改成：同一台機器保留 local dump，或能從 manifest 指到的儲存位置取回同一份 dump。

### 哪個方案最不容易 drift

- 若只談理論上的 source 對齊，`pipeline 自動 replay` 最不容易 drift。
- 但在目前 POC 環境，最實際且最不容易 drift 的方案仍是 `hybrid`：用 local baseline dump 當 restore 載體，再配 committed manifest、row count 與 smoke validation 固定基準。

### 哪個方案最符合目前 agent-rule 與 Makefile 治理方向

- 在附則第 1 條生效後，`local backup + committed manifest` 與 `hybrid` 最符合目前治理方向。
- 現有 Makefile、backup/restore script、README 仍可保留正式入口，只需把 git 追蹤對象收斂到 manifest、metadata 與文件。

## 建議方案

建議採用 `hybrid：local minimal backup + committed manifest + smoke validation`。

其中真正承載資料的是「本機可還原的 dev baseline dump」，repo 內提交的是 manifest、restore 指令、checksum 與文件，用來降低 drift 與維護成本。

### 核心設計

1. 以一個很小的已驗證資料窗建立 dev baseline dump。
2. baseline dump 放在本機 `backup/dev/baseline/` 子目錄，不和一般輪替備份混在一起，也不入 git。
3. 一般 `dev-backup` 仍只寫入本機 `backup/dev/*.dump`，維持 5 份輪替；baseline 不參與自動 prune。
4. baseline 另附 committed manifest，記錄資料窗、owner、row count、預期 smoke query 結果與 dump checksum；manifest 預設提交到 `backup/manifest/dev/baseline/`。
5. restore 完成後執行固定 smoke validation，確認非空 fact/dim 與排行榜查詢可跑。

## 2026-05-20 本輪實作狀態

- 已建立 `backup/dev/baseline/manifest.md`，把 baseline 名稱、建立時間、owner/date window 狀態、主要表 row count、smoke expectation 與注意事項固定下來
- 已新增 `make dev-restore-baseline`，只接受本機 `backup/dev/baseline/*.dump`；若目前沒有 local baseline dump，必須明確失敗
- 已新增 `make dev-smoke-analytics`，會檢查 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact`，並執行帶排除關鍵字的商品排行榜 smoke query；資料不足時必須 non-zero 結束
- 已驗證目前 repo 內沒有任何 `.dump` / snapshot 可直接作為 baseline，且本機 dev PostgreSQL 目前仍是空資料 baseline，因此本輪沒有提交正式 baseline dump
- 本輪沒有假造 50 嵐資料、沒有修改 schema、沒有跑完整一個月 pipeline，也沒有把 local baseline 混入一般 backup rotation

### 為什麼 baseline dump 要放子目錄

目前 `db_backup.sh`、`db_restore.sh`、`db_del_backup.sh` 都只掃 `backup/$APP_ENV` 的第一層 `*.dump`。

若把 local baseline 直接放在 `backup/dev/*.dump`：

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
4. 產生 backup manifest：owner、日期窗、核心表 row count、預期 smoke 結果、checksum
5. 在乾淨 dev 容器上重新 restore 驗證
6. 更新 README / 文件 / 更新紀錄，並提交 manifest / metadata / 文件
7. commit / push

### 新電腦使用者重建 baseline

1. `make dev-env`
2. `make dev-up`
3. `make dev-restore-baseline`
4. `make dev-smoke-analytics`

## Makefile 建議新增或調整入口

### 已實作

- `dev-restore-baseline`
  - restore 本機既有的 minimal baseline dump
  - 內部走既有 `db_restore.sh`，但傳入明確 baseline 路徑
  - 若沒有 local baseline dump，必須明確失敗

- `dev-smoke-analytics`
  - 驗證 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact` 皆非空
  - 跑一個固定商品排行榜 smoke query
  - 目前已先用名稱關鍵字排除 `幣`、`券`、`折抵`、`折扣`、`點數`、`贈`、`服務費`、`運費`、`調整`、`測試`、`test`

### 可選但不必在下一輪一次做完

- `dev-refresh-baseline`
  - 維護者專用，從目前本機 PG 重新產 baseline dump 與 manifest
  - 不建議一開始就自動化太多，先手動流程落地即可

### 現有入口建議調整

- `help` 補一組「最小 analytics baseline」說明
- README 的 backup / restore 章節補上「一般輪替備份」與「local baseline dump / committed manifest」的差異

## backup / seed / restore 治理方式

### backup 治理

- 一般輪替備份：本機 `backup/dev/*.dump`、`backup/prod/*.dump`
- baseline dump：本機 `backup/dev/baseline/*.dump`
- 提交到 git 的追蹤證據：`backup/manifest/**/*.md` 與 `backup/dev/baseline/manifest.md`
- 一般輪替備份仍維持最多 5 份
- baseline dump 不參與自動 prune，也不應被 `ALL=1` 清除

### seed 治理

- `db/init/001_schema.sql` 繼續只管 schema 與 canonical seed
- 不把 analytics smoke dataset 直接塞進 `db/init/001_schema.sql`
- 原因是 analytics baseline 與 schema seed 的生命週期不同，混在同一份 SQL 容易造成 drift 與維護成本失控

### restore 治理

- 一般 restore 維持現行互動式 restore
- local baseline restore 用專屬入口，避免使用者從一串臨時 dump 中手動選檔
- baseline restore 完成後仍需補跑 migrate 與 smoke validation
- restore 屬高風險操作，繼續沿用現有 pre-restore backup 規則

## 下一輪真正要完成的剩餘最小範圍

1. 從已驗證的本機 PostgreSQL 或其他已確認來源，建立單一 owner、小日期窗的 local dev baseline dump
2. 把實際 dump 放到本機 `backup/dev/baseline/`
3. 依實際 dump 回填 `backup/dev/baseline/manifest.md` 與對應 `backup/manifest/dev/baseline/*.md` 的 owner、日期窗、row count、checksum 與 smoke expectation
4. 驗證 restore 後三張核心 analytics 表非空，且商品排行榜 smoke query 可跑

## 建議的 baseline 內容下限

- `ia_users` 至少 1 筆
- `pos_product_dim` 至少 10 筆
- `pos_branch_dim` 至少 2 筆
- `pos_sales_hourly_fact` 至少 1 個日期窗、可做 Top-N 商品排行

若下一輪有餘裕，可再額外納入一筆疑似非正常商品項，作為商品語意分類 smoke case；但這不是建立第一版可重現 baseline 的前置條件。