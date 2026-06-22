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
 - [x] 收斂 bridge copy 的生命週期（已完成）
   - 決議：
     1. `ia-analyses-db` 可保留現有 `cmd/` 與 `internal/` bridge copy，直到前端 MVP 與 `ia-analyses-go` 的 Go pipeline 穩定為止。
     2. 不允許在 `ia-analyses-db` 新增任何 Go pipeline 功能；所有新 Go pipeline 工作必須落在 `ia-analyses-go`。
     3. 未來步驟：在 `ia-analyses-go` 穩定後，開一個獨立的 cleanup checkpoint（archive / remove bridge copy），屆時再決定封存或刪除策略。
     4. 本次變更不會刪除或移動任何檔案；僅記錄政策與未來清理計畫。
   - 關聯檔案：`Makefile`、`cmd/`、`internal/`、`文件/架構指南.md`

 - [x] 確認 `db/init` 與 `db/patches` 的漂移管理方式（已完成）
  - 現況：`001_schema.sql` 與 `db/patches/` 都存在 Phase 2C 相關內容，過去曾出現 init 與 patch 在不同檔案描述同一變更的情況，需制定明確流程以避免 drift。
  - 關聯檔案：`db/init/001_schema.sql`、`db/patches/002_adjust_qty_and_sales_ex_tax.sql`、`db/patches/003_phase2c_schema_contract.sql`
  - 決議（2026/06/22）：建立以下 drift 管理規範（policy）：
    1. 新資料庫（New DB）建立路徑：`db/init/*` 為 authoritative，任何在 `db/init/` 的 schema 定義為新庫初始化時唯一信源。
    2. 現有資料庫升級路徑（Existing DB upgrade）：`db/patches/*` 為 authoritative，所有要在既有安裝上套用的變更必須以新 patch 檔實作。
    3. 每次 schema 變更流程（mandatory steps）：在提出任何 schema 變更前後，請務必同時完成：
       - 在 `db/init/001_schema.sql` 中更新/加入最終的初始化定義（代表新庫狀態）。
       - 新增一個 `db/patches/` patch 檔以支援既有庫的升級路徑（patch 檔名需以遞增編號開頭並包含簡短說明）。
       - 更新 `文件/table 結構文件.md`，在對應 table 條目中標註該變更是：`introduced by init`、或 `introduced by patch <patch-filename>`，若為修改則註記 `modified by patch <patch-filename>`。
    4. 文件與驗證：新增變更時，請在 PR 描述包含：受影響檔案清單（init / patch / docs）、執行順序（init-only vs patch-on-existing）、以及在本地執行 `make dev-smoke-analytics` / `bash -n scripts/db_smoke_analytics.sh` 的驗證步驟摘要。
    5. 回溯一致性檢查（periodic check）：每個主要 release 前應執行一次自動或人工檢查，確認 `db/init/001_schema.sql` 與歷史 `db/patches` 的最終狀態在語義上相容（例如檢查欄位存在性與型別兼容），將檢查結果寫入 release note。
  - 建議作法：每次 schema 變更都明確記錄「新庫 init 直接包含」與「舊庫 patch 演進」兩條路徑；新增 table 文件時標明來源是 init 還是 patch。

 - [x] 補 smoke analytics 的前端分析口徑檢查（已完成）
 - 證明（2026/06/22）：
   - 已新增 product-summary grain minimal aggregation query（count + top5 preview）到 `scripts/db_smoke_analytics.sh`。
   - `bash -n scripts/db_smoke_analytics.sh` → PASS。
   - `make dev-smoke-analytics` → executed; preview and other checks returned expected non-zero results and top5 preview after fixing an ambiguous `owner_user_id` reference。
 - 關聯檔案：`scripts/db_smoke_analytics.sh`、`ia-analyses-go/internal/postgres/stat_feed_reader.go`

 - [x] 決定 `pos_branch_dim.group_code` 的授權來源（已決定）
  - 現況：`pos_branch_dim.group_code` 欄位存在於 schema（`db/init/001_schema.sql`），但目前同步流程不會寫入該欄位（現有 sync 未提供 group_code 值，因此多數紀錄為 NULL）。
  - 決議（2026/06/22）：
    - `group_code` 為非 authoritative 欄位目前（read-only metadata placeholder）。
    - 前端/POC 不應承諾或依賴 branch-group filtering，`group_code` 目前不可作為 POC filter 條件。
    - 若未來要支援 branch-group filter，必須先：
      1. 定義 authoritative source（例如新增專用 table 或由外部 config 提供），
      2. 更新 `ia-analyses-go` 的 sync pipeline 使其寫入 `pos_branch_dim.group_code`，
      3. 同時在 `db/init/001_schema.sql` 與新增 patch（`db/patches/`）中記錄變更，並在 `文件/table 結構文件.md` 註記 `modified by patch <patch-filename>` 或 `introduced by patch <patch-filename>`。
  - 建議作法：目前不變更 schema / patch / sync；在需要支援前端 filter 時，按以上步驟執行並在 PR 中標明驗證計畫。

## 已完成的文件整理

- [x] 目前基本文件足夠：`已知的問題`、`當前工作摘要`、`架構指南`、`table 結構文件`、本檔
- [x] 舊階段文件已在 `文件/過時文件/agent不再閱讀-舊版文件-2026-05-26-001/`
