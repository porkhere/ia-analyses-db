# Fact Implementation Sequence

最後檢視：2026-05-19

註：2026-05-19 起，正式 Makefile 入口改為 `dev-*` / `prod-*`。文內若出現 `db-*`，均屬歷史命名。

## 設計原則

- 先凍結語意，再做 schema 與寫入。
- 先完成最接近可落地的 fact，再擴到其他資料域。
- 每一階段都要有明確禁止事項，避免把未定義問題偷塞進實作。

## Phase 2C-1：凍結日期語意與 product dim

### 目標

- 凍結 `pos_sales_hourly_fact.business_date` 的正式語意
- 凍結 `tr_date` / `sale_period` / `t_open_date` 的 fact 分工
- 凍結 `pos_product_dim` 的 category contract
- 凍結 `pos_branch_dim` 的 `group_code` / `options` contract
- 凍結 `pos_order_status_dim` 與 payment raw 保留策略

### 交付物

- 已定稿的 schema contract，至少包含：
- `business_date = sale_period`，但 Phase 2C-1 不改欄位名
- `pos_product_dim` 補 `cate_no` / `cate_name`
- `pos_branch_dim` 補 `group_code`，`options` 不進 schema contract
- 新增 `pos_order_status_dim`
- raw payment `name` / `memo1` 由 payment fact 保留，不進 sales fact / payment dim
- `pos_sales_hourly_fact` 僅保留商品層級銷售欄位

### 禁止事項

- 不做 PostgreSQL schema 變更
- 不做 migration
- 不做 PG 寫入
- 不做 day-level replace
- 不改 Athena sync 查詢邏輯

### 驗收條件

- `business_date` 已明確凍結為 `sale_period`
- `cate_no` / `cate_name` 已明確凍結進 `pos_product_dim`
- `group_code` 已明確凍結進 `pos_branch_dim`
- `options` 已明確凍結為 view / 展示層組字串，不進 schema contract
- `pos_order_status_dim` 的最小欄位與四個 raw status bucket 已明確定義
- raw payment `name` / `memo1` 的保留策略已明確定義
- sales fact contract 的 included / excluded columns 已明確定義

## Phase 2C-2：完成 schema migration plan 草案

### 目標

- 先完成 [schema_migration_plan_phase2c.md](schema_migration_plan_phase2c.md)
- 將 Phase 2C-1 contract 轉成 additive schema migration plan
- 先定義新增欄位 / 新增表、deprecated 策略、回填策略與 validation 影響

### 交付物

- [schema_migration_plan_phase2c.md](schema_migration_plan_phase2c.md)
- `pos_product_dim.cate_no` / `cate_name` 的 migration 草案
- `pos_branch_dim.group_code` 的 migration 草案
- `pos_order_status_dim` 的建表與 seed 草案
- `pos_sales_hourly_fact` 的 deprecated / not-add 欄位策略

### 禁止事項

- 不建立 migration
- 不修改 `db/init/001_schema.sql`
- 不增加 raw payment `name` / `memo1` 到 sales fact
- 不增加 `order_count` / `order_num` 到 sales fact
- 不增加 `current_void_*` / `diff_void_*` / `refund_*` 到 sales fact 作為完整 void 方案
- 不加入 condiment 或 branch opening 欄位
- 不 drop deprecated 欄位
- plan 審查通過前，不進 PG write path

### 驗收條件

- [schema_migration_plan_phase2c.md](schema_migration_plan_phase2c.md) 已完成
- README / 架構指南 / 更新紀錄 已引用這份 plan
- `business_date = sale_period` 的 schema contract 與不新增欄位清單已明確寫出
- `pos_product_dim`、`pos_branch_dim`、`pos_order_status_dim` 的 migration 草案已明確寫出
- plan 審查通過後，下一輪才建立 migration

## Phase 2C-3：本機開發 DB schema patch 驗證

### 目標

- 套用既有 schema patch 到本機開發 DB
- 驗證 Phase 2C schema contract 已真正落在 DB schema
- 確認 schema patch 可重播、可查詢、且不破壞現有開發 DB

### 交付物

- `make db-apply-patches` 執行成功
- `pos_order_status_dim` 存在且 seed 正確
- `pos_order_status_dim.updated_at = TIMESTAMPTZ`
- `pos_product_dim.cate_no/cate_name` 存在
- `pos_branch_dim.group_code` 存在
- `pos_sales_hourly_fact.business_date` comment 存在
- `make db-size` 可正常查詢

### 禁止事項

- 不做資料回填
- 不做 PG sync
- 不改 Go
- 不改 `sync-athena`
- 不跑 Athena
- 不 drop 欄位

### 驗收條件

- 既有 patch 可在本機開發 DB 套用成功
- schema contract 對應欄位、table、comment 與 seed 已全部通過驗證
- 開發 DB 仍可正常查詢

## Phase 2C-4：sales fact 寫入前 validation contract

### 目標

- 完成 [sales_fact_validation_contract_phase2c.md](sales_fact_validation_contract_phase2c.md)
- 在真正實作 PG write path 前，先固定 validation gate 與 compare 規則
- 將 source metrics、target metrics、negative checks 與 day-level replace 流程寫成可執行規格

### 交付物

- [sales_fact_validation_contract_phase2c.md](sales_fact_validation_contract_phase2c.md)
- source metrics / target metrics / reconciliation 欄位清單
- `business_date = sale_period` 驗證方式
- `status = 1` latest-row source path 驗證方式
- product / branch / order_type / payment_type dim miss 檢查
- raw payment / void / refund / order-level metrics 的 negative checks
- day-level replace 驗證流程與 tolerance 原則
- validation SQL 草案

### 禁止事項

- 不改 `db/init/001_schema.sql`
- 不新增 migration
- 不改 `db/patches`
- 不改 Go
- 不改 `sync-athena`
- 不跑 Athena
- 不寫 PostgreSQL
- 不做資料回填

### 驗收條件

- `pos_sales_hourly_fact` 寫入前必驗欄位與 compare 規則已固定
- `item_count` 已明確定義為 validation-only control metric，不進 sales fact schema
- day-level replace 的失敗停止條件已固定
- Phase 2C-5 前，不再開放直接開始 PG write path 實作

## Phase 2C-4.7：sales fact validation draft final review

### 目標

- 完成 [sales_fact_validation_final_review_phase2c.md](sales_fact_validation_final_review_phase2c.md)
- 將 Phase 2C-4.5 metrics reconciliation draft 與 Phase 2C-4.6 gate draft 收斂成單一最終結論
- 固定 Phase 2C-5 的開始條件與實作限制

### 交付物

- [sales_fact_validation_final_review_phase2c.md](sales_fact_validation_final_review_phase2c.md)
- 五份 validation SQL draft 的責任邊界整理
- hard gate / warning gate 清單
- insert 前檢查、insert 後檢查與 rollback / stop 條件
- 是否可進入 Phase 2C-5 的明確結論

### 禁止事項

- 不新增 validation SQL
- 不改 `db/init/001_schema.sql`
- 不新增 migration
- 不改 `db/patches`
- 不改 Go
- 不改 `sync-athena`
- 不跑 Athena
- 不寫 PostgreSQL
- 不做資料回填

### 驗收條件

- 五份 validation SQL draft 的責任邊界已固定
- `item_count` 已維持為 validation-only control metric，且不得進 sales fact schema
- Phase 2C-5 的實作限制已固定為 sales fact PG write path skeleton + validation gate 整合
- final review 已明確給出 go / no-go 結論

## Phase 2C-5：實作 sales fact PG day-level replace + validation

### 目標

- 只落地 `pos_sales_hourly_fact`
- 完成 PostgreSQL day-level replace
- 依據 Phase 2C-4 validation contract 與 Phase 2C-4.7 final review 實作 write gate 與 post-write compare
- 僅限 sales fact PG write path skeleton + validation gate 整合

### 範圍

- `ia_users` / branch / product dim upsert
- sales fact 寫入與 replace
- sales fact validation

### 本輪現況

- Phase 2C-5 已開始，且本輪已完成 Phase 2C-5.1：sales fact Athena status-aware source candidate provider + validation gate integration
- 本輪已完成 Phase 2C-5.2：sales fact dimension bootstrap / sync，新增 `sync-sales-dims-plan` 與 `sync-sales-dims`，先補齊 `pos_product_dim` / `pos_branch_dim`
- 本輪已完成 Phase 2C-5.3：新增 `sync-athena-write-local`，開啟 local-only sales fact actual write，但只允許單日 `public.pos_sales_hourly_fact` day-level replace
- CLI 已新增 / 預留 `--write-pg`、`--validate-only`、`--owner-user-id` 與 `--local-only-actual-write`；只有 `--write-pg` + explicit local-write flag 才會開 actual write
- day-level replace orchestration、pre-insert compare、dimension gate、negative schema gate、post-insert compare 與 hard gate fail stop / rollback 已正式接到 local-only actual write
- `sync-athena-validate` 與 `sync-athena-write-plan` 仍只做 read-only Athena candidate / gate 輸出；`sync-athena-write-local` 才會在 local PostgreSQL 執行 transaction 內 delete + insert + compare
- `sync-sales-dims-plan` 只查 Athena、顯示 planned upsert 與 conflict summary；`sync-sales-dims` 才允許 upsert `pos_product_dim` / `pos_branch_dim`，且仍不得寫 `pos_sales_hourly_fact`
- 2025-01-01 驗證結果：`sync-athena-write-local` 成功寫入 `pos_sales_hourly_fact = 103545` 筆，post-insert target metrics 與 candidate metrics delta = `0`；寫後重新跑 validate-only，`product_dim_miss_count = 0`、`branch_dim_miss_count = 0`
- `branch conflict count = 6` 目前視為 dimension warning，不阻擋 sales fact write，但後續仍需追查 branch 名稱來源一致性
- 本輪已完成 Phase 2C-5.4：sales fact local write idempotency validation；2025-01-01 rerun 前後 PG row count = `103545 -> 103545`，post-insert target metrics delta 仍為 `0`
- 直接查詢 PostgreSQL 已驗證 `updated_at` 在 rerun 後刷新，代表這一輪確實執行 day-level replace，而不是跳過既有資料
- validate-only / write-plan 在 2C-5.4 重新驗證後仍保持 read-only；多日 `sync-athena-write-local` 仍直接拒絕
- 本輪已完成 Phase 2C-5.5：controlled 2-day local actual write；2025-01-01 寫入 `103545` 筆、2025-01-02 寫入 `98167` 筆，兩天 post-insert target metrics delta 都為 `0`
- `sync-athena-write-local` 現在最多允許 2 天，但仍逐日處理；每一天都是獨立 day-level transaction，前一天 fail 時不得繼續後一天
- validate-only / write-plan 在 2C-5.5 重新驗證後仍保持 read-only；超過 2 天的 `sync-athena-write-local` 仍直接拒絕
- 本輪已進入 Phase 2C-5.R：small-window regression validation，先固定 [sales_fact_correctness_basis_phase2c.md](sales_fact_correctness_basis_phase2c.md) 與 [sales_fact_regression_windows_phase2c.md](sales_fact_regression_windows_phase2c.md) 的判準，再做多窗口 regression sweep
- 5.R read-only regression 已確認：2025-01-01、2025-01-02、2025-01-01 ~ 2025-01-02 的 validate-only / write-plan 全部通過，`actual_write_enabled = false`，source / candidate delta = `0`
- 5.R read-only regression 也確認：2025-01-07、2025-01-15、2025-01-31、2025-01-31 ~ 2025-02-01 都在 pre-insert dimension gate 被擋下，先分類為 `dimension_bootstrap_issue`；目前沒有 evidence 指向 `actual_code_bug`
- 5.R actual write regression 只重跑既有允許範圍：2025-01-01、2025-01-02、2025-01-01 ~ 2025-01-02；PG row count 維持 `103545` / `98167`，沒有累加，post-insert target metrics delta = `0`，`updated_at` 有刷新
- source candidate provider、local-only actual writer 與 transaction 內 post-insert target metrics compare 已不再是 placeholder；目前先暫停 5.6 7-day local write validation，需等 5.R 全窗口通過後才恢復
- 本輪已進入 Phase 2C-5.X-control：建立 Go execution controller，修正 31-day shell 長命令不可觀測、interrupt 後無可靠 state、無 failure summary、無 resume 的 orchestration 問題
- Makefile 在 5.X-control 中只保留 thin entrypoint；controller 本體在 [internal/salespipe/controller.go](internal/salespipe/controller.go)，對外提供 status、write-plan、validate-only、write-local、resume、report 六種 mode
- user-facing range 不再要求 user 手動縮小；controller 會接收完整 date range，內部自動做 chunk / day-level execution / resume / summary report
- 5.X-control smoke 已驗證：`make sales-pipe-plan` 可對 2025-01-01 ~ 2025-01-31 展開 day jobs，`make sales-pipe-validate` 與 `make sales-pipe-write-local` 對 2025-01-11 單日通過，post-insert delta = `0`，`product_dim_miss_count = 0`，`branch_dim_miss_count = 0`，`forbidden_column_count = 0`
- 5.X-resume 已完成：`write-plan` 不再把 `planned` 寫成主 state 的 completed signal，`resume` 會合併顯式 owner/date range，並以 state + PostgreSQL persisted dates 判定真正完成日
- `make sales-pipe-plan OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31` 已正確顯示 11 天 `completed`、20 天 `pending`
- `make sales-pipe-resume OWNER_USER_KEY=demo-owner OWNER_USER_ID=1 START_DATE=2025-01-01 END_DATE=2025-01-31 CONFIRM_LONG_RUN=1` 已在 `32m51s` 內完成，跳過 2025-01-01 ~ 2025-01-11，成功寫入 2025-01-12 ~ 2025-01-31 共 20 天；整個 31-day PG row count = `3698110`，`post_insert_delta_all_zero = true`，`hard_gate_failed_count = 0`

### 必守原則

- day-level replace
- transaction boundary
- validation first
- pre-insert compare
- dimension gate
- negative schema gate
- post-insert compare
- hard gate fail 必須 stop / rollback

### 禁止事項

- 不同時實作 order fact
- 不同時實作 payment fact
- 不同時實作 condiment fact
- 不同時實作 branch opening fact
- 不把付款別報表硬接到 sales fact 上
- 不把門店對帳單完整口徑宣稱完成
- 不把 raw payment 寫進 sales fact
- 不把 void / refund / order-level metrics 寫進 sales fact
- 不把 `item_count` 寫進 persisted sales fact
- 本輪 actual write 只允許最多 2 天的 `public.pos_sales_hourly_fact`
- 不做超過 2 天的 actual write
- 不把 controlled 2-day 驗證直接擴張成 7 天、31 天或 production 寫入
- 5.R small-window regression 未全窗口通過前，不回到 5.6 controlled 7-day local write validation
- `pos_branch_dim.group_code` 在來源未定稿前只允許寫 `NULL`，不猜值
- 不允許 validation bypass
- 不把 shell 長命令當成長日期窗 orchestration 的正式方案
- 不保存逐筆 candidate row log
- 不保存逐筆 insert row log

### 驗收條件

- 產品銷售報表的 qty / amount / category 維度可驗證
- sales-side 的營業額 / 稅 / 折扣 / 附加費指標可驗證
- `business_date` 語意無歧義
- write path 具備 Phase 2C-4 定義的 validation gate
- dimension bootstrap 完成後，`product_dim_miss_count` / `branch_dim_miss_count` 必須可驗證歸零
- actual write 目前只允許最多 2 天的 day-level replace；超過 2 天時必須直接拒絕
- post-insert target metrics 與 candidate metrics delta 必須可驗證為 `0`
- rerun 後 PG row count 不得累加，必須維持與 candidate row count 相等
- rerun 後 `updated_at` 必須可驗證刷新
- controlled 2-day actual write 必須逐日 commit；不能把兩天包進同一個 transaction
- 超過 2 天時必須直接拒絕
- validate-only / write-plan 仍不得寫 PostgreSQL
- 5.R small-window regression 必須先完成 failure classification；若有失敗，不得直接跳去 5.6
- 不擴張到 order fact、payment fact、condiment fact、branch opening fact
- raw payment、void / refund / order-level metrics 與 `item_count` 不進 persisted sales fact
- write 流程符合 pre-insert compare、dimension gate、negative schema gate、post-insert compare 與 hard gate fail stop / rollback
- 長日期窗 orchestration 必須由 controller 提供 state、resume、status 與 summary report；不能再要求 user 手動切段
- 每一輪若改動 CLI / validation / phase / schema / write path / source candidate，都必須同步檢查 README、架構指南、fact implementation sequence、更新紀錄

## Phase 2D：order fact 設計與驗證

### 目標

- 建立 `pos_order_daily_fact`
- 補上安全可聚合 `order_num`
- 補上 status / void lifecycle / 對帳拆項

### 禁止事項

- 不把 raw payment 需求塞進 order fact
- 不把商品明細比例問題塞進 order fact

### 驗收條件

- 每日營業額總計表、銷售作廢統計報表、訂單類型統計報表可由 order fact 驗證
- 門店對帳單訂單側欄位可驗證

## Phase 2E：payment fact 設計與驗證

### 目標

- 建立 `pos_payment_daily_fact`
- 保留 raw payment `name` / `memo1`
- 補上 `amount`、`change`、`amount - change`

### 禁止事項

- 不用 canonical `payment_type_id` 取代 raw payment 維度
- 不把付款別報表改寫成 sales allocation 報表

### 驗收條件

- 50lan_付款別報表可驗證
- 門店對帳單付款側需求可驗證

## Phase 2F：condiment fact

### 目標

- 建立 `pos_condiment_hourly_fact` 的 schema 與驗證邊界

### 禁止事項

- 不把 condiment 回寫到 `pos_sales_hourly_fact`

### 驗收條件

- 50lan_調味報表可由獨立 fact 驗證

## Phase 2G：branch opening fact

### 目標

- 建立 `pos_branch_opening_daily_fact` 的 schema 與驗證邊界

### 禁止事項

- 不把營業時間欄位回寫到 sales fact

### 驗收條件

- 50lan_各店門市營業時間報表可由獨立 fact 驗證

## 目前的 Phase 2C 結論

- Phase 2C 不再單指「開始 PG 寫入」。
- 新的 Phase 2C 是：先凍結多 fact 模型、schema contract、schema patch 驗證、sales fact validation contract，以及 validation SQL draft final review，再進入 sales fact 的 PostgreSQL 寫入。
- Phase 2C-4.7 final review 已完成，目前已開始受限版 Phase 2C-5：sales fact PG write path skeleton + validation gate 整合。
- 下一步不是擴張 fact 範圍，而是進入受控 actual write、day-level replace commit 與 post-insert compare / rollback 驗證；在那之前仍須維持 disabled writer。