# ia-analyses-db

更新日期：2026-05-24-16:27
校準日期：2026-05-24-16:27

註：2026-05-19 起，正式操作入口改為 `make dev-*` / `make prod-*`。本文若出現 `db-*`，除非明確標示為歷史紀錄，否則一律以新命名為準。

註：2026-05-20-22:54 起已啟用 `agent-rule/rule-add.md` 附則第 1 條。現行落地只有受保護 `.gitignore` 規則與本機 dump 管理；repo 不再建立或維護 backup manifest。本文歷史段落若出現 `backup/manifest`、`committed manifest`、`dev-mark-runtime-aligned` 等字樣，均視為 2026-05-20 過度擴張 round 的歷史紀錄，不是現行制度。

這個 repo 是 IA 分析資料庫的第一階段骨架，目標是先把 PostgreSQL、資料表結構、備份還原流程，以及 `sync-athena` 的 CLI 入口建立起來，讓後續 Athena 同步邏輯可以在同一個倉庫內逐步落地。

## 2026-05-24 16:27 附則第 1 條校準結果

- `scripts/db_write_backup_manifest.sh`、`scripts/db_mark_runtime_aligned.sh` 與對應 Makefile target 已移除，DB backup / restore 回到 local dump only
- 現行正式落地是：`.gitignore` 保護 `backup/**/*.dump`、備份檔名帶日期時間、一般 backup 維持每環境最多 5 份
- `backup/manifest/` 若仍留在 repo，只視為 2026-05-20 歷史 round 的 legacy archive，不再是 active truth source，也不得再擴張或回填
- baseline 流程收斂為：本機 `backup/dev/baseline/*.dump` + `make dev-restore-baseline` + `make dev-smoke-analytics`

## 文件入口

- 主要文件索引見 [文件/README.md](文件/README.md)。
- 長期主線架構與文件分層規則見 [文件/架構指南.md](文件/架構指南.md)。
- repo 導航入口見 [guide/index.md](guide/index.md)。
- workspace 跨 repo 對照見 [../ia-analyses-ws-map/ws-map.md](../ia-analyses-ws-map/ws-map.md)。
- 新工作預設先從這兩份文件進入；`文件/過時/` 內的歷史 review / validation / migration / QuickSight 盤點文件不應直接當成最新指令來源。

## 為什麼不是 1:1 複製 Athena 原始表

目前已確認 `50lan_new` 的原始表量大、欄位髒值多，而且付款與稅額邏輯需要先聚合、清洗、再映射。直接把八張 Athena 原表完整複製到 PostgreSQL，會同時帶來以下問題：

- 儲存成本偏高，但分析價值沒有等比例增加。
- `order_payments_parquet` 直接 join 會放大列數，容易把銷售額乘爆。
- 稅額實際上以 `orders.included_tax_subtotal` 為主，不適合照原表照搬。
- AI / 報表要的是穩定、可聚合、可解釋的事實表，不是完整 OLTP-ish 明細鏡像。

因此第一階段先採用 slim schema：一張小時級 sales fact 表，搭配使用者、商品、門市、訂單型態、付款型態等維度與映射表。

但這個 repo 的下一階段方向已正式收斂成多 fact 模型，而不是把所有 QuickSight 報表硬塞進單一萬用 fact。現有 `pos_sales_hourly_fact` 只保留「商品層級銷售彙總 fact」的責任；訂單狀態、raw payment、condiment、branch opening 等資料域，將由獨立 fact 承接。

## 目前資料表

- `ia_users`: 外部 `owner_user_key` 與內部 `owner_user_id` 的對照。
- `pos_sales_hourly_fact`: 小時級銷售事實表，grain 為 `owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`。
- `pos_product_dim`: 商品維度，保留 `product_no`、名稱，以及 `cate_no` / `cate_name`。
- `pos_branch_dim`: 門市維度，保留 `branch_id`、名稱，以及 `group_code`。
- `pos_order_type_dim`: 訂單型態映射表。
- `pos_payment_type_dim`: 付款型態映射表。
- `pos_order_status_dim`: 訂單狀態語意表，固定 raw status code 與 sales / void / excluded bucket。

## 2026-05-20 23:26 全 repo 更新模式 runtime 對齊結果

- 依 `agent-rule` 第 2.6 條、本輪再次完成最低啟動檢查：`ia-analyses-db`、`ia-analyses-go`、`ia-analyses-guide` 在 `git fetch --all --prune` 後皆維持 clean，`HEAD...@{upstream}` 皆為 `0 0`
- 本輪再次確認只有 `ia-analyses-db` 屬於 runtime repo；`ia-analyses-go`、`ia-analyses-guide` 的 README / Makefile 仍明確標示為文件骨架，免重啟
- DB dev runtime 對齊已通過：`make dev-env`、`make dev-restart RESTORE=1 BACKUP_FILE=2026-05-20-22-06.dump`、schema drift 檢查、`make dev-smoke-analytics`、`make dev-size`、`make dev-backup` 均通過；restore validation = `ia_analyses|7`
- schema drift 檢查結果為 `public tables = 7`、`missing_required_tables = 0`、`missing_required_columns = 0`；核心表 row count 為 `ia_users = 1`、`pos_order_type_dim = 10`、`pos_payment_type_dim = 8`、`pos_order_status_dim = 4`、`pos_product_dim = 581`、`pos_branch_dim = 278`、`pos_sales_hourly_fact = 3698110`
- `make dev-smoke-analytics` 通過後，關鍵字排除後 leaderboard row count = `580`；`make dev-size` 顯示目前 dev database size 約 `962.80 MB`
- restore 流程新增一般 dev pre-restore backup `2026-05-20-23-24.dump`，對齊完成後新增一般 dev backup `2026-05-20-23-26.dump`；依輪替策略，最舊的 `2026-05-20-17-37.dump` 與 `2026-05-20-17-51.dump` 對應 manifest 已移除，當前保留的 5 份 manifest 為 [backup/manifest/dev/2026-05-20-18-28.md](backup/manifest/dev/2026-05-20-18-28.md)、[backup/manifest/dev/2026-05-20-22-03.md](backup/manifest/dev/2026-05-20-22-03.md)、[backup/manifest/dev/2026-05-20-22-06.md](backup/manifest/dev/2026-05-20-22-06.md)、[backup/manifest/dev/2026-05-20-23-24.md](backup/manifest/dev/2026-05-20-23-24.md)、[backup/manifest/dev/2026-05-20-23-26.md](backup/manifest/dev/2026-05-20-23-26.md)
- 為了讓附則第 1 條的 manifest 狀態能完整記錄 restore 後的 migrate / schema drift / restart / validation 結果，本輪新增 `make dev-mark-runtime-aligned BACKUP_FILE=...`；`2026-05-20-22-06.dump` 的 manifest 已用這個入口補齊為全 true

## 2026-05-20 22:54 附則第 1 條落地結果

- 已啟用「DB backup 實體檔不入 git」：一般 backup 與未來 baseline dump 都留在本機 `backup/`，不再提交到 git
- `.gitignore` 已改為忽略 `backup/**/*.dump`；backup 追蹤證據改由 `backup/manifest/` 提交
- 目前現存 5 份 dev dump manifest 為 [backup/manifest/dev/2026-05-20-18-28.md](backup/manifest/dev/2026-05-20-18-28.md)、[backup/manifest/dev/2026-05-20-22-03.md](backup/manifest/dev/2026-05-20-22-03.md)、[backup/manifest/dev/2026-05-20-22-06.md](backup/manifest/dev/2026-05-20-22-06.md)、[backup/manifest/dev/2026-05-20-23-24.md](backup/manifest/dev/2026-05-20-23-24.md)、[backup/manifest/dev/2026-05-20-23-26.md](backup/manifest/dev/2026-05-20-23-26.md)
- `make dev-backup` 會在建立 local dump 後同步寫入 manifest；`make dev-restore` 會更新對應 manifest 的 restore 狀態；manifest 現固定包含 `storage_type`、`local_path` 與 `availability_scope`；若整輪 DB runtime 對齊完成，則用 `make dev-mark-runtime-aligned BACKUP_FILE=...` 補齊 manifest 的 migrate / schema drift / restart / validation 狀態

## 2026-05-20 22:09 全 repo 更新模式 runtime 對齊結果

- 依 `agent-rule` 第 2.6 條與第 5.6 條，在 DB repo pull 最新 6 commits 後完成 dev runtime 對齊：`make dev-env`、`make dev-up`、`make dev-restore BACKUP_FILE=2026-05-20-18-28.dump`、`make dev-migrate`、schema drift 檢查、`make dev-restart`、`make dev-smoke-analytics`、`make dev-size`、`make dev-backup` 均通過
- restore validation = `ia_analyses|7`；schema drift 檢查確認 `public tables = 7`、`missing_required_tables = 0`、`missing_required_columns = 0`，且 `ia_users = 1`
- 當前核心表 row count 為 `pos_product_dim = 581`、`pos_branch_dim = 278`、`pos_sales_hourly_fact = 3698110`；`make dev-smoke-analytics` 通過後，關鍵字排除後 leaderboard row count = `580`
- `make dev-size` 顯示目前 dev database size 約 `962.73 MB`
- 本輪新增一般 dev pre-restore backup `2026-05-20-22-03.dump` 與對齊後 backup `2026-05-20-22-06.dump`；依附則第 1 條，實體 dump 留在本機 `backup/dev/`，對應追蹤證據改由 [backup/manifest/dev/2026-05-20-22-03.md](backup/manifest/dev/2026-05-20-22-03.md) 與 [backup/manifest/dev/2026-05-20-22-06.md](backup/manifest/dev/2026-05-20-22-06.md) 提交

## 2026-05-20 18:29 全 repo 更新模式 runtime 對齊結果

- 依 `agent-rule` 第 2.6 條與第 5.6 條再次完成 dev runtime 對齊：`make dev-env`、`make dev-restart RESTORE=1 BACKUP_FILE=2026-05-20-17-51.dump`、`make dev-smoke-analytics` 均通過；restore validation = `ia_analyses|7`
- restore 前自動建立一般 dev pre-restore backup [backup/dev/2026-05-20-18-28.dump](backup/dev/2026-05-20-18-28.dump)；目前最新一般 dev restore backup inventory 為 [backup/dev/2026-05-20-18-28.dump](backup/dev/2026-05-20-18-28.dump)、[backup/dev/2026-05-20-17-51.dump](backup/dev/2026-05-20-17-51.dump)、[backup/dev/2026-05-20-17-37.dump](backup/dev/2026-05-20-17-37.dump)
- schema drift 檢查通過：`public tables = 7`、`missing_required_tables = 0`、`missing_required_columns = 0`；核心表 row count 為 `ia_users = 1`、`pos_order_type_dim = 10`、`pos_payment_type_dim = 8`、`pos_order_status_dim = 4`、`pos_product_dim = 581`、`pos_branch_dim = 278`、`pos_sales_hourly_fact = 3698110`
- patch effect / schema contract 驗證通過：`pos_order_status_dim.updated_at = TIMESTAMPTZ`、`pos_branch_dim.group_code = text`，且 `pos_sales_hourly_fact.business_date` comment 仍固定為 `sale_period` 語意
- `make dev-smoke-analytics` 已通過；關鍵字排除後 leaderboard row count = `580`，top 5 商品查詢可正常回傳

## 2026-05-20 17:53 全 repo 更新模式 runtime 對齊結果

- 依 `agent-rule` 第 2.6 條與第 5.6 條再次完成 dev runtime 對齊：`make dev-env`、`make dev-restart RESTORE=1 BACKUP_FILE=2026-05-20-17-37.dump`、`make dev-smoke-analytics` 均通過；restore validation = `ia_analyses|7`
- 當前 dev PostgreSQL 核心表 row count 為：`pos_product_dim = 581`、`pos_branch_dim = 278`、`pos_sales_hourly_fact = 3698110`
- 目前最新一般 dev restore backup 為 [backup/dev/2026-05-20-17-51.dump](backup/dev/2026-05-20-17-51.dump)；上一份 [backup/dev/2026-05-20-17-37.dump](backup/dev/2026-05-20-17-37.dump) 仍保留，可供 `make dev-restore` 使用，但兩者都不取代 `backup/dev/baseline/` 的 baseline 專用途徑
- `make dev-smoke-analytics` 已通過；關鍵字排除後 leaderboard row count = `580`，top 5 商品查詢可正常回傳

## 2026-05-20 16:11 重新開始基準（歷史）

- `make dev-env`、`make dev-up`、`docker compose ps`、`make dev-migrate`、`make dev-size` 本輪均通過；目前 dev database size 約 `7.84 MB`
- 當時 dev PostgreSQL 的核心表都存在，但 row count 顯示它是空資料基準：`ia_users = 0`、`pos_product_dim = 0`、`pos_branch_dim = 0`、`pos_sales_hourly_fact = 0`；只有 `pos_order_type_dim = 10`、`pos_payment_type_dim = 8`、`pos_order_status_dim = 4` 這些 seed 維度已存在
- `make sales-pipe-status` 本輪雖可正常執行，但它讀的是 [state/sales_fact_pipe_state.json](state/sales_fact_pipe_state.json) 的歷史 controller state，不代表目前 dev PostgreSQL 已載入 31 天資料
- `pos_product_dim` 目前沒有 `normal_sales_item` / `product_semantic_type` 一類的商品語意旗標；既有 QuickSight 盤點只看到 `cate_name = 其它` 類型排除，還不足以排除「雲林幣」這類支付 / 折抵 / 特殊交易項
- 本輪短報告見 [reports/phase2c_restart_baseline_20260520.md](reports/phase2c_restart_baseline_20260520.md)

## 2026-05-20 16:11 最小 analytics baseline 設計結論（歷史）

- 在 2026-05-20 16:11 的盤點時，repo 沒有已提交的非空 analytics baseline 載體；當時 `backup/dev/`、`backup/prod/` 只有 `.gitkeep`
- `db/init/001_schema.sql` 與 `make dev-sync-seeds` 只會建立 schema 與 canonical seed，不會產生非空 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact`
- 現有 summary report 與 state 只保存摘要，不是可 restore 的資料快照
- 建議方案採 `hybrid`：以「tracked minimal backup dump」作為 restore 載體，再配 baseline manifest 與 smoke validation 降低 drift
- baseline dump 應放在本機 `backup/dev/baseline/`，避免被一般 `dev-backup` 輪替或 `dev-del-backup ALL=1` 誤刪
- 本輪已先建立 baseline 規劃 manifest（現位於 `backup/manifest/dev/baseline/manifest.md`）、`make dev-restore-baseline` 與 `make dev-smoke-analytics`
- 在該次盤點時，由於 repo 內沒有任何可用 dump，且本機 dev DB 查核仍為空資料 baseline，因此沒有提交 baseline dump；`make dev-restore-baseline` 會明確提示缺少 local baseline dump，`make dev-smoke-analytics` 會明確回報 `smoke failed`
- 詳細設計見 [文件/minimal_analytics_baseline_plan.md](文件/minimal_analytics_baseline_plan.md)

## 2026-05-20 analytics productization gap 結論

- 目前最接近產品化的基礎是：`pos_sales_hourly_fact`、六張核心維度 / 映射表、`sales-pipe-*` controller、`sync-sales-dims*` 與 `sync-athena-*` pipeline，以及 `文件/過時/quicksight/quicksight_metric_mapping.md`
- 目前最可直接產品化的統計是商品排行、門店排行、時段銷售、canonical payment mix、order type mix、稅/折扣/附加費摘要
- 目前最大的 productization gap 不是再擴 schema，而是：`可重現 baseline data`、`商品語意分類`、`API layer`、`aggregation / materialized view layer`
- 產品 roadmap 建議先從 `baseline restore + semantics + read-only analytics API` 開始，不直接跳大型 forecast 或 ML framework
- 詳細盤點見 [reports/analytics_productization_gap_20260520.md](reports/analytics_productization_gap_20260520.md)

## 設計決策

### 為什麼現在不是單一 fact 模型

根據 QuickSight 10 份 analysis 盤點、metric mapping 與 fact gap analysis，已確認以下缺口是結構性問題，不是多補幾個欄位就能解決：

- `tr_date` / `sale_period` / `t_open_date` 三種日期語意
- 安全可聚合的 `order_num`
- raw payment `name` / `memo1` / `amount - change`
- 商品部門 `cate_no` / `cate_name`
- `status = -2` 與 `void_sale_period` 的 void lifecycle
- condiment / modifier 明細
- branch opening / 營業時間
- 門店對帳單需要的訂單層拆項

因此後續資料模型會拆成：

- `pos_sales_hourly_fact`：商品層級銷售彙總
- `pos_order_daily_fact`：訂單層日級統計與 void / 對帳拆項
- `pos_payment_daily_fact`：付款側 raw 維度與實收金額
- `pos_condiment_hourly_fact`：調味 / modifier 明細
- `pos_branch_opening_daily_fact`：營業時間與 terminal 狀態

### 為什麼 fact 表用 `owner_user_id`

外部 `owner_user_key` 可能較長，也可能在不同資料流中重複出現。fact 表改存內部整數鍵有兩個好處：

- 減少重複字串佔用。
- 後續如果同一個 external key 需要掛更多屬性，可以集中在 `ia_users` 管理。

### 為什麼要有 `order_type_id` / `payment_type_id`

來源資料的命名不會永遠穩定。先收斂到一組可控的 canonical mapping，可以把 Athena 原始欄位命名變動，隔離在同步邏輯內，而不是擴散到所有查詢與 AI prompt。

### 為什麼金額都用 milli-TWD

這批資料已經觀察到浮點值、科學記號、極端異常值。先在同步階段把有效金額轉成整數 milli-TWD，可以避免 PostgreSQL 與下游分析在四捨五入上互相漂移。

### 為什麼數量也改成 `qty_milli`

這一版把數量欄位從 `qty_total` 改成 `qty_milli`，1 單位 = 1000。目的是讓 fact table 完全避免 numeric，並維持欄位型別的一致性。

### `pos_sales_hourly_fact` 的正式定位

- grain：`owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`
- 實際責任：商品層級銷售彙總，支援產品銷售、杯數、稅額、折扣、附加費與 canonical order/payment 分析
- 正式 source path：只吃 status-aware `status = 1` latest row
- 不承接：raw payment 對帳、完整 order count、void lifecycle、condiment 明細、branch opening
- `payment_type_id` 只代表 canonical payment 類別，不能取代 QuickSight 的 raw payment `name` / `memo1`

## Multi-Fact 方向

### 目前不是單一萬用 fact

- `pos_sales_hourly_fact` 不再被定義成全部 QuickSight 報表的唯一來源。
- 每一張 fact 必須只承接一種穩定 grain，避免因為多日期語意或 raw 維度而讓 aggregation 失真。
- Priority A / B / C 報表會由不同 fact 組合支援，而不是強迫共用同一張事實表。

### 已收斂的多 fact 分工

- `pos_sales_hourly_fact`：產品銷售、杯數、稅、折扣、附加費、canonical order/payment 分析
- `pos_order_daily_fact`：每日營業額、void 統計、訂單類型、門店對帳單的訂單側欄位
- `pos_payment_daily_fact`：付款別報表與門店對帳單付款側需求
- `pos_condiment_hourly_fact`：調味報表
- `pos_branch_opening_daily_fact`：各店門市營業時間報表

### Phase 2C-1 schema contract

在開始 PostgreSQL 寫入前，Phase 2C-1 已先定稿以下 contract：

- `business_date` 暫不改名，但正式語意固定為 `sale_period`
- `tr_date` / `t_open_date` 不進 sales fact，後續由 order/payment fact 承接
- `pos_product_dim` 必補 `cate_no` / `cate_name`
- `pos_branch_dim` 必補 `group_code`，`options` 由 view / 展示層生成，不列入 schema contract
- raw payment `name` / `memo1` 必須保留，但保留位置是 payment fact，不是 sales fact 或 canonical payment dim
- void lifecycle 由 order fact 承接，不在 sales fact 先補 convenience 欄位

### Phase 2C-2 schema migration plan

- Phase 2C-2 的輸出是 [文件/過時/phase2c/schema_migration_plan_phase2c.md](文件/過時/phase2c/schema_migration_plan_phase2c.md)，不是 migration 實作。
- 這一輪只整理 schema 變更計畫與 migration 順序，不修改 `db/init/001_schema.sql`、不建立 migration、也不寫 PostgreSQL。
- 目前下一步是審查 [文件/過時/phase2c/schema_migration_plan_phase2c.md](文件/過時/phase2c/schema_migration_plan_phase2c.md)，確認 additive schema、回填策略、deprecated 欄位策略與 validation 影響。

### Phase 2C-3-pre formal migration

- 已建立正式 patch：[db/patches/003_phase2c_schema_contract.sql](db/patches/003_phase2c_schema_contract.sql)。
- 已同步更新 [db/init/001_schema.sql](db/init/001_schema.sql)，讓新建 DB baseline 與 Phase 2C schema contract 對齊。
- 本輪沒有執行 migration，也沒有寫 PostgreSQL；patch 只建立檔案，供後續審查與手動套用。
- 這個正式 patch 只包含 `pos_order_status_dim`、`pos_product_dim.cate_no/cate_name`、`pos_branch_dim.group_code`，以及 `pos_sales_hourly_fact.business_date` 的 schema comment。

### Phase 2C-3 schema patch validation

- 本機開發 DB 已成功套用 [db/patches/003_phase2c_schema_contract.sql](db/patches/003_phase2c_schema_contract.sql)。
- `make db-apply-patches` 已驗證 patch runner 可安全重播既有 patch，003 patch 本身可正常套用。
- `pos_order_status_dim` 已存在，4 筆 seed 正確，`updated_at` 型別已確認為 `TIMESTAMPTZ`。
- `pos_product_dim.cate_no/cate_name`、`pos_branch_dim.group_code` 與 `pos_sales_hourly_fact.business_date` comment 也都已在本機開發 DB 驗證通過。

### Phase 2C-4 sales fact validation contract

- Phase 2C-4 的輸出是 [文件/過時/phase2c/sales_fact_validation_contract_phase2c.md](文件/過時/phase2c/sales_fact_validation_contract_phase2c.md)。
- 這一輪只定義 sales fact 寫入前的 validation contract 與 SQL 草案，不實作 `sync-athena` 寫入，不跑 Athena，不寫 PostgreSQL。
- contract 內容固定了 source metrics、target metrics、reconciliation 欄位、`business_date = sale_period` 驗證、`status = 1` latest-row source path 驗證、dim miss 檢查、negative checks、day-level replace 流程與 tolerance 原則。
- `item_count` 在這份 contract 中被定義為 validation-only control metric，不新增進 `pos_sales_hourly_fact` schema。
- Phase 2C-4.5 已新增 metrics reconciliation SQL 草案：`sales_fact_source_metrics.sql`、`sales_fact_target_metrics.sql`、`sales_fact_compare_metrics.sql`。
- Phase 2C-4.6 已補齊 validation gate SQL 草案：`sales_fact_dimension_gate_checks.sql`、`sales_fact_negative_schema_checks.sql`。
- Phase 2C-4.7 final review 已完成，文件為 [文件/過時/phase2c/sales_fact_validation_final_review_phase2c.md](文件/過時/phase2c/sales_fact_validation_final_review_phase2c.md)；它把 contract、metrics review 與 gate review 收斂成單一最終結論。
- Phase 2C-5 的前置條件已固定為：contract 定稿、[文件/過時/phase2c/sales_fact_validation_sql_review_phase2c.md](文件/過時/phase2c/sales_fact_validation_sql_review_phase2c.md) 通過、[文件/過時/phase2c/sales_fact_validation_gate_review_phase2c.md](文件/過時/phase2c/sales_fact_validation_gate_review_phase2c.md) 通過，以及 [文件/過時/phase2c/sales_fact_validation_final_review_phase2c.md](文件/過時/phase2c/sales_fact_validation_final_review_phase2c.md) 完成。
- final review 的結論已固定為：Phase 2C-5 可以開始，但僅限 sales fact PG write path skeleton + validation gate 整合。
- Phase 2C-5 仍然禁止：order fact、payment fact、condiment fact、branch opening fact，以及 raw payment、void / refund / order-level metrics、`item_count` 寫進 persisted sales fact；也不得繞過 validation gate。
- Phase 2C-5 必須遵守：day-level replace、transaction boundary、validation first、pre-insert compare、dimension gate、negative schema gate、post-insert compare，且 hard gate fail 必須 stop / rollback。
- 本輪已進入 Phase 2C-5.1：status-aware sales source candidate provider 已接上 Athena；validate-only / write-plan 會真的產生 candidate rows、source metrics、pre-insert metrics 與 validation gate 輸出。
- `sync-athena` 已新增 `--write-pg`、`--validate-only` 與 `--owner-user-id`；`--write-pg` 目前仍維持 disabled writer，不會真正寫 PostgreSQL，但會執行 Athena candidate build 與 pre-insert gate。
- `sync-athena-write-plan` 與 `sync-athena-validate` 不再停在 placeholder；兩者都會走 status-aware source candidate provider，validation gate 仍不可繞過。
- 本輪已完成 Phase 2C-5.2：新增 `sync-sales-dims-plan` 與 `sync-sales-dims`，先做 sales fact dimension bootstrap / sync，讓 local PostgreSQL 的 `pos_product_dim` / `pos_branch_dim` 能接住 sales candidate。
- `sync-sales-dims-plan` 只查 Athena、顯示 planned upsert count 與 conflict summary，不寫 PostgreSQL；`sync-sales-dims` 才允許 upsert `pos_product_dim` / `pos_branch_dim`。
- 2C-5.2 仍明確禁止寫入任何 fact table；`sales_fact_written` 必須維持 `false`。`pos_branch_dim.group_code` 因來源未定稿，本輪固定寫 `NULL`，不猜值。
- 2025-01-01 ~ 2025-01-02 驗證結果：planned upsert `pos_product_dim = 487`、`pos_branch_dim = 277`，product conflict = `0`、branch conflict = `6`；dim sync 後重新執行 validate-only，`product_dim_miss_count = 0`、`branch_dim_miss_count = 0`，source / candidate metrics delta 仍為 `0`，`actual_write_enabled = false`。
- 本輪已進入 Phase 2C-5.3：已開啟 local-only sales fact actual write，但只允許 `public.pos_sales_hourly_fact` 的單日 day-level replace。
- local-only actual write 必須同時滿足：`--write-pg` + explicit `--local-only-actual-write`、target 必須是 local PostgreSQL、`START_DATE = END_DATE`、先通過 pre-insert validation / dimension gate / negative schema gate、transaction 內 delete + insert、post-insert compare 歸零後才 commit。
- `make sync-athena-validate` 與 `make sync-athena-write-plan` 在 2C-5.3 仍保持 read-only，不寫 PostgreSQL；`make db-sync-athena` 仍是安全停止入口。
- 2025-01-01 單日驗證結果：`sync-athena-write-local` 實際寫入 `pos_sales_hourly_fact = 103545` 筆，post-insert target metrics 與 candidate metrics delta = `0`，`actual_write_enabled = true`，transaction result = `commit`。
- `item_count` 仍只允許 validation-only，不進 persisted fact；raw payment、void / refund / order-level metrics 也都不進 `pos_sales_hourly_fact`。validate-only 重新驗證後，`forbidden_column_count = 0`、`product_dim_miss_count = 0`、`branch_dim_miss_count = 0`。
- `branch conflict count = 6` 目前仍視為 dimension warning，不阻擋 sales fact write，但後續必須追蹤清理。
- 本輪已進入 Phase 2C-5.4：sales fact local write idempotency validation，確認 2025-01-01 的單日 day-level replace 可以安全重跑。
- 2025-01-01 的 rerun 驗證結果：重跑前 PG row count = `103545`、重跑後 PG row count = `103545`，沒有累加成 `207090`；post-insert target metrics delta 仍為 `0`。
- 直接查詢 PostgreSQL 可見 `updated_at` 在 rerun 後有刷新，代表這一輪確實重新執行 day-level replace，而不是跳過既有資料。
- `make sync-athena-validate` 與 `make sync-athena-write-plan` 在 2C-5.4 重新驗證後仍保持 read-only，`actual_write_enabled = false`，不會 delete / insert / commit。
- 多日 `make sync-athena-write-local START_DATE=2025-01-01 END_DATE=2025-01-02` 在 2C-5.4 仍會直接拒絕，不開放多日 actual write。
- 本輪已進入 Phase 2C-5.5：controlled 2-day local actual write，將 local-only actual write 從單日擴成最多 2 天的小範圍驗證。
- `make sync-athena-write-local` 現在最多只允許 2 天；超過 2 天會直接拒絕，`make db-sync-athena` 仍是安全停止入口。
- 2C-5.5 的 2025-01-01 ~ 2025-01-02 驗證結果：2025-01-01 寫入 `103545` 筆、2025-01-02 寫入 `98167` 筆；兩天 post-insert target metrics delta 都為 `0`，兩天 `product_dim_miss_count = 0`、`branch_dim_miss_count = 0`、`forbidden_column_count = 0`。
- 2C-5.5 仍採逐日 day-level transaction，不使用跨兩天的大 transaction；每一天都固定走 pre-insert validation、delete、insert、post-insert compare、commit/rollback。若前一天失敗，不得繼續後一天。
- `make sync-athena-validate` 與 `make sync-athena-write-plan` 在 2C-5.5 重新驗證後仍保持 read-only，`actual_write_enabled = false`，不會 delete / insert / commit。
- 本輪已進入 Phase 2C-5.R：small-window regression validation，先整理 [文件/過時/phase2c/sales_fact_correctness_basis_phase2c.md](文件/過時/phase2c/sales_fact_correctness_basis_phase2c.md) 與 [文件/過時/phase2c/sales_fact_regression_windows_phase2c.md](文件/過時/phase2c/sales_fact_regression_windows_phase2c.md)，再用多個小窗口重跑 validate-only / write-plan，並只在既有允許範圍內重跑 actual write。
- Phase 2C-5.R 的 correctness basis 已固定為：Athena raw POS tables 是 source of record，但正確性不是直接相信 raw rows，而是依據 status-aware semantic contract、source -> candidate compare、dimension gate、negative schema gate 與 candidate -> post-insert target compare；QuickSight 是 business benchmark，不是所有 row-level fact 的唯一 truth。
- 5.R read-only regression 結果：2025-01-01、2025-01-02、2025-01-01 ~ 2025-01-02 的 validate-only / write-plan 都通過，`actual_write_enabled = false`，source / candidate metrics delta = `0`。
- 5.R read-only regression 也確認：2025-01-07、2025-01-15、2025-01-31、2025-01-31 ~ 2025-02-01 都不是 source / candidate mismatch，也不是 forbidden schema 問題；它們是在 pre-insert hard gate 被 `product_dim_miss_count` / `branch_dim_miss_count` 擋下，先分類為 `dimension_bootstrap_issue`，不視為 `actual_code_bug`。
- 5.R actual write regression 只重跑既有允許範圍：2025-01-01、2025-01-02、2025-01-01 ~ 2025-01-02；PG row count 仍維持 `103545` / `98167`，沒有累加，post-insert target metrics delta = `0`，`updated_at` 也都有刷新。
- 本輪暫停 7-day write 推進；只有在 5.R 全窗口通過並完成 failure disposition 後，才回到 5.6 controlled 7-day local write validation。
- 本輪已進入 Phase 2C-5.X-control：建立真正的 sales fact pipe execution controller，修正「31-day shell 長命令黑箱、不可觀測、無 state、無 resume」的 orchestration 問題。
- Makefile 現在只應作 thin entrypoint；複雜任務編排改由 Go controller 處理，入口為 `make sales-pipe-status`、`make sales-pipe-plan`、`make sales-pipe-validate`、`make sales-pipe-write-local`、`make sales-pipe-resume`、`make sales-pipe-report`。
- user-facing interface 不再要求 user 手動縮小日期範圍；controller 會接收完整日期區間，並在內部自動做 day-level execution、chunking、progress state、resume 與 summary report。
- 31-day long shell command 已被判定為不合格 orchestration；目前沒有證據證明 Athena 壞、PostgreSQL 跑不動、資料模型錯，真正缺的是 controller-based observability / interrupt handling / resume。
- controller state 固定落在 [state/sales_fact_pipe_state.json](state/sales_fact_pipe_state.json)，summary report 固定落在 `reports/phase2c_sales_fact_pipe_summary_<run_id>.md`；summary report 只保留耗時、每日 row count、validation summary、table size / DB size 與 report size，不保存逐筆資料 log。
- Phase 2C-5.X-control 的 smoke 結果：`make sales-pipe-plan` 已能對 2025-01-01 ~ 2025-01-31 展開 day jobs / chunks；`make sales-pipe-validate` 與 `make sales-pipe-write-local` 已對 2025-01-11 單日通過，state 與 summary report 皆有更新，post-insert delta = `0`，`product_dim_miss_count = 0`，`branch_dim_miss_count = 0`，`forbidden_column_count = 0`。
- 已完成 Phase 2C-5.X-resume：controller 的 `write-plan` / `resume` 語義已修正，`write-plan` 不再把 `planned` 汙染主 state，`resume` 會合併顯式 owner/date range，並以 state + PostgreSQL persisted dates 判定真正已完成日期。
- 31-day resume 前確認：PostgreSQL 已有 2025-01-01 ~ 2025-01-11，共 `1217900` 筆；`make sales-pipe-plan OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31` 已正確顯示 11 天 `completed`、20 天 `pending`。
- Phase 2C-5.X-resume 的實際結果：`make sales-pipe-resume OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31 CONFIRM_LONG_RUN=1` 已在 `32m51s` 內完成；controller 自動略過 2025-01-01 ~ 2025-01-11，成功寫入 2025-01-12 ~ 2025-01-31 共 20 天，`hard_gate_failed_count = 0`，`post_insert_delta_all_zero = true`，`product_dim_miss_total = 0`，`branch_dim_miss_total = 0`，`forbidden_column_count = 0`。
- 31 天 local PG 最終結果：`public.pos_sales_hourly_fact` 在 2025-01-01 ~ 2025-01-31 共 `3698110` 筆；`active_pipeline_process = false`，active `pg_stat_activity` query count = `0`；table total size = `1029 MB`、table size = `590 MB`、indexes size = `439 MB`、database size = `1037 MB`。
- 本輪 31-day summary report 為 [reports/phase2c_sales_fact_pipe_summary_20260518T073547Z.md](reports/phase2c_sales_fact_pipe_summary_20260518T073547Z.md)，檔案大小 `2.5 KiB`；仍不保存逐筆 candidate / insert row log。

## 快速開始

1. 選擇環境並產生目前工作 `.env`：`make dev-env`
2. 依需要調整 `.env.dev` 或 `.env.prod`，再重新執行對應的 `make dev-env` / `make prod-env`
3. 啟動 PostgreSQL：`make dev-up`
4. 套用 schema / patch：`make dev-migrate`
5. 若本機已有 local baseline dump，再執行：`make dev-restore-baseline`
6. 執行 analytics smoke：`make dev-smoke-analytics`
7. 檢查資料庫體積：`make dev-size`
8. 檢查或建立一般備份：`make dev-backup`

## 常用指令

```bash
make dev-env
make dev-up
make dev-up RESTORE=1
make dev-size
make dev-backup
make dev-backup-list
make dev-restore BACKUP_FILE=2026-05-19-09-30.dump
make dev-restore-baseline
make dev-smoke-analytics
make dev-migrate
make prod-env
make prod-up
make prod-backup
make sync-athena-dry-fast OWNER_USER_KEY=demo-owner START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-athena-dry-full OWNER_USER_KEY=demo-owner START_DATE=2025-01-01 END_DATE=2025-01-02 PREVIEW_LIMIT=5
make dev-sync-seeds
make dev-del-backup BACKUP_FILE=2026-05-19-09-30.dump
make sales-pipe-status
make sales-pipe-plan OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31
make sales-pipe-validate OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-11 END_DATE=2025-01-11
make sales-pipe-write-local OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-11 END_DATE=2025-01-11
make sales-pipe-resume OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31 CONFIRM_LONG_RUN=1
make sales-pipe-report
make sync-athena-dry OWNER_USER_KEY=demo-owner START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-athena-dry OWNER_USER_KEY=demo-owner START_DATE=2025-01-01 END_DATE=2025-01-02 PREVIEW_LIMIT=5
make sync-athena-validate OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-athena-write-plan OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-athena-write-local OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-sales-dims-plan OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-02
make sync-sales-dims OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-02
```

補充：所有 `sales-pipe-*`、`sync-athena-*` 與 `sync-sales-dims*` 都會使用目前 `.env`。切換環境前請先執行 `make dev-env` 或 `make prod-env`。

補充：`make db-sync-athena` 已停用，避免在 Phase 2C-5 skeleton 期間誤觸非 dry-run 入口。從 5.X-control 開始，優先使用 `sales-pipe-*` controller entrypoints；`sync-athena-*` 保留作低階工具，不再負責複雜 orchestration。

目前已進入 Phase 2A：`sync-athena --dry-run` 會真的查 Athena，但仍然不會寫 PostgreSQL。Phase 2B 收斂後，dry-run 分成兩種模式：

- `make sync-athena-dry` 與 `make sync-athena-dry-fast`：fast mode，預設用來跑 7 天、31 天這種長日期窗驗證，只保留必要查詢。
- `make sync-athena-dry-full`：full mode，保留短日期窗 root cause debug 用的完整輸出，不建議直接拿去跑長日期窗。
- `make sync-athena-validate`：跑 Phase 2C-5.5 status-aware Athena candidate build、source / pre-insert metrics 與 validation gate，不做 write。
- `make sync-athena-write-plan`：跑 Phase 2C-5.5 Athena candidate build、輸出 candidate / metrics / gate 與 day-level replace plan，但仍不做實際 insert。
- `make sync-athena-write-local`：跑 Phase 2C-5.5 controlled 2-day local actual write；最多允許 2 天，並且每一天都在獨立 transaction 內完成 delete + insert + post-insert compare + commit/rollback。
- `make sales-pipe-status`：controller status mode；不跑 Athena、不寫 PG，只回報 state、DB size、已落地日期摘要與是否有 active pipeline process。
- `make sales-pipe-plan`：controller plan mode；不寫 PG，由 controller 自動展開 day jobs / chunks 並產生 plan summary。
- `make sales-pipe-validate`：controller validate-only mode；不要求 user 手動縮小日期範圍，由 controller 內部自動切日執行。
- `make sales-pipe-write-local`：controller write-local mode；由 controller 內部自動 day-level transaction、progress state、resume 與 summary report。
- `make sales-pipe-resume`：根據 state 與 PG 已存在資料自動略過已成功日期，繼續未完成日期；若 `FORCE=1` 才允許重跑已成功日期。
- `make sales-pipe-report`：根據 state 重建或更新 summary report，不重新跑 Athena，也不輸出逐筆資料。
- `make sync-sales-dims-plan`：跑 Phase 2C-5.2 sales fact dimension bootstrap plan，只查 Athena 並輸出 `pos_product_dim` / `pos_branch_dim` 預計 upsert 數量與 conflict summary。
- `make sync-sales-dims`：跑 Phase 2C-5.2 sales fact dimension bootstrap apply，只 upsert `pos_product_dim` / `pos_branch_dim`，不寫任何 fact table。
- 預計使用的 Athena database / workgroup / output
- 預計讀取的來源表

### Fast mode 會輸出

- 目標 owner user
- 日期區間
- dry run mode
- 各來源表的實際 row count
- 轉換後 aggregation 的實際 row count
- source vs preview reconciliation summary
- top tax delta sample
- status excluded summary

### Full mode 另外會輸出

- order destination mapping summary
- payment name mapping summary
- tax reconciliation breakdown
- additions tax debug
- rounding debug
- top tax delta order trace
- duplicate order summary / trace
- status dedup candidate summary
- status dedup before/after comparison
- preview sample table

### Phase 2B status-aware source 口徑

- sales preview / future sales fact：只吃 `status = 1` 的 latest row。
- void candidate / future void fact：只保留 `status = -2` 的 latest row。
- `status = -1` 與 `status = 2`：只進 excluded/debug，不進正常銷售，也不進 void 主口徑。
- 不再使用「全 status 混在一起取 latest row」的做法。

這個切分的原因不是要改 tax formula，而是為了避免 `orders_parquet` 同一個 `order_id + t_open_date` 的多生命週期版本同時混入 sales path，進而造成 item / tax / net 的 join amplification。

### Phase 2B 長日期窗 fast 驗證

- 7 天，`2025-01-01 ~ 2025-01-07`
	- fast mode 總耗時：64.17 秒
	- result_aggregation row_count：775,736
	- source_order_count：538,543
	- tax delta：-9,297 milli
	- net delta：365 milli
	- sales_ex_tax delta：9,662 milli
	- top tax delta 最大絕對值：6 milli
	- 結論：仍維持 rounding 級別，適合當長窗穩定性驗證
- 31 天，`2025-01-01 ~ 2025-01-31`
	- fast mode 總耗時：87.27 秒
	- result_aggregation row_count：3,698,110
	- source_order_count：2,410,592
	- tax delta：-45,109 milli
	- net delta：1,511 milli
	- sales_ex_tax delta：46,620 milli
	- top tax delta 最大絕對值：7 milli
	- 結論：差異仍集中在 order-level rounding 級別，可進下一階段的多 fact 邊界定稿與日期語意凍結
- 目標 grain
- 一張終端機欄位預覽表

目前 dry-run 會真的送出 Athena 查詢，query metric 來自 Athena `GetQueryExecution` 的實際統計，不再是 Phase 1.5 的靜態 heuristics。這一版仍然只做 read-only dry-run，不會寫入 PostgreSQL。

目前 preview aggregation 有兩個明確邊界：

- `--preview-limit` 只在 full mode 的 preview sample table 有作用；fast mode 會接受參數，但不會額外查 preview sample。
- `qty_milli` 目前以 `try_cast(current_qty AS decimal(20,3)) * 1000` 轉成 bigint，不再在 preview 中顯示 `qty_total`。
- `order_additions_parquet` 目前會把 `current_discount` / `current_surcharge` 以 order-level 比例分攤進 preview discount/surcharge；`include_tax` 目前只保留在 reconciliation summary，不直接併入 preview tax，避免與 `orders.included_tax_subtotal` 重複計算。
- `net_sales_milli` 現在代表含稅、折扣與加價後的實際認列銷售額；`sales_ex_tax_milli` 才代表未稅銷售額。
目前 `make sync-athena-dry-full` 除了既有的 preview table、mapping summary、reconciliation summary 之外，還會額外輸出：

## Phase 2B reconciliation debug

目前 `make sync-athena-dry` 除了既有的 preview table、mapping summary、reconciliation summary 之外，還會額外輸出：

- `tax_reconciliation_breakdown`
- `additions_tax_debug`
- `rounding_debug`
- `top_tax_delta_sample`

這一輪的目的不是直接改正式 tax 公式，而是先把 source 與 preview 的差異拆清楚。debug 區塊目前採用以下邊界：

- `source_orders_all`：日期範圍內 `orders_parquet` 原始訂單彙總。
- `source_orders_valid`：排除非有限值與金額 outlier 後的訂單彙總；目前 outlier 門檻明確定義為 `abs(order_total) <= 100000 TWD`。
- `source_completed_orders`：目前 completed proxy 是 `status = 1 AND transaction_voided IS NULL`。`orders_parquet` 沒有明確的 cancelled / unresolved 欄位，因此這一版只能先用這個 proxy 做 debug。
- `preview_allocatable_orders`：completed + valid + 有可用 item + tax allocation denominator > 0 的訂單。
- `preview_excluded_zero_denominator`：有 item，但 `order_net_sales` denominator 近似 0，導致 preview tax allocation 目前會落成 0 的訂單。
- `preview_excluded_no_items`：completed + valid，但沒有可進入 `item_lines` 的 item，因此不會出現在 preview allocation。
- `preview_excluded_status`：valid 但不符合 completed proxy 的訂單，用來觀察 source 與 preview 是否處理同一批 completed orders。

因此，source 與 preview 可能有差異，不一定代表 rounding 問題，還可能來自：

- source / preview 使用的訂單集合不同
- completed proxy 與目前 preview 的實際 scope 不一致
- no-items 或 zero-denominator 訂單被排除在 allocation 之外
- `orders.included_tax_subtotal` 與 `order_additions.include_tax` 的口徑是否重疊
- line-level allocation 後再轉成 milli bigint 所帶來的 order-level delta

這一版仍然只做 read-only dry-run debug，不寫 PostgreSQL，也不會啟用正式 `db-sync-athena`。

目前 preview table 欄位順序為：

- `business_date`
- `hour_of_day`
- `branch_id`
- `product_no`
- `order_type_id`
- `payment_type_id`
- `qty_milli`
- `gross_sales_milli`
- `discount_milli`
- `surcharge_milli`
- `net_sales_milli`
- `sales_ex_tax_milli`
- `tax_milli`

非 dry-run 模式目前會直接回傳「尚未實作 PostgreSQL 寫入同步」，這是刻意保留的 Phase 2A 邊界。

## Athena 環境需求

- `ATHENA_OUTPUT_LOCATION` 必須是可寫入的 S3 路徑。
- `AWS_PROFILE` 與 `AWS_REGION` 必須能查詢對應 Athena workgroup。
- 目前這個環境下，SQL 需要搭配 `QueryExecutionContext.Database=50lan_new`，並在查詢中使用未限定 database 的 table 名稱。

## 備份與還原

- 一般 backup 與 baseline dump 實體檔固定留在本機 `backup/dev/` 或 `backup/prod/`，依附則第 1 條不入 git
- repo 不再建立或維護 backup manifest；若需要交接 baseline 資訊，應寫在 active 文件或交接材料，而不是回填 `backup/manifest/`
- 備份檔名格式固定為 `YYYY-MM-DD-HH-MM.dump`，每個環境最多保留 5 份，超過時會自動刪除最舊檔
- 若要建立可重現的 dev analytics baseline，實體 dump 應放在本機 `backup/dev/baseline/` 子目錄，不和一般輪替備份混放
- `make dev-size` / `make prod-size` 會用 PostgreSQL 內建函式計算當前資料庫大小，單位 MB
- `make dev-backup` / `make prod-backup` 會建立目前環境的 local `.dump` 備份
- `make dev-backup-list` / `make prod-backup-list` 會依新到舊列出目前環境可還原備份與大小
- `make dev-sync-seeds` / `make prod-sync-seeds` 會安全地重跑 `db/init/001_schema.sql`，把最新 seed upsert 到現有 DB，不需要刪 volume 重建
- `make dev-apply-patches` / `make prod-apply-patches` 會套用 `db/patches/*.sql`，用安全方式更新現有 DB schema
- `make dev-restore` / `make prod-restore` 會先列出本機備份、接受數字選擇，輸入 `n` 可退出；restore 完成後只做基本 PostgreSQL 驗證，不再回寫 manifest
- `make dev-restore-baseline` 只會讀本機 `backup/dev/baseline/*.dump`；若目前尚無 local baseline dump，會明確失敗，不會假裝 restore 成功
- `make dev-smoke-analytics` 會檢查 `pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact` 是否非空，並執行帶有排除關鍵字的商品排行榜 smoke query；資料不足時必須 `smoke failed`
- restore 屬高風險操作，執行前會先自動做一份當前環境 backup，完成後再執行基本 PostgreSQL 驗證
- `make dev-up RESTORE=1` / `make prod-up RESTORE=1` 會在容器啟動後進入 restore 流程，然後自動補跑 migrate
- baseline 的日期窗、row count、checksum 與交接資訊若需保留，應寫入 [文件/minimal_analytics_baseline_plan.md](文件/minimal_analytics_baseline_plan.md) 或本輪交接材料

## 下一階段預計補上

- Phase 2C-1：凍結日期語意、產品部門維度與 raw/canonical 邊界
- Phase 2C-2：審查 [文件/過時/phase2c/schema_migration_plan_phase2c.md](文件/過時/phase2c/schema_migration_plan_phase2c.md)，確認 additive schema migration 順序
- Phase 2C-3：本機開發 DB schema patch 驗證通過
- Phase 2C-4：審查 [文件/過時/phase2c/sales_fact_validation_contract_phase2c.md](文件/過時/phase2c/sales_fact_validation_contract_phase2c.md)
- Phase 2C-4.5：metrics reconciliation SQL drafts review 已完成
- Phase 2C-4.6：dimension / negative schema gate SQL drafts review 已完成
- Phase 2C-4.7：final review 已完成，確認可進入受限的 Phase 2C-5
- Phase 2C-5：已完成 Phase 2C-5.1 Athena-backed sales candidate provider 與 pre-insert gate 路徑；actual production write 仍未完成
- Phase 2D：設計並驗證 `pos_order_daily_fact`
- Phase 2E：設計並驗證 `pos_payment_daily_fact`
- Phase 2F：設計 `pos_condiment_hourly_fact`
- Phase 2G：設計 `pos_branch_opening_daily_fact`