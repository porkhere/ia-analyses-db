# Backup Manifest

- backup_file_name: missing
- backup_created_at: missing
- file_size_bytes: missing
- sha256: missing
- storage_type: local
- local_path: missing
- availability_scope: unknown
- restore_command: unavailable
- not_in_git_reason: 依憲法附則第 1 條：DB backup 實體檔不入 git
- rule_addendum: 附則第 1 條：DB backup 實體檔不入 git
- restore_completed: false
- migrate_completed: false
- schema_drift_checked: false
- restart_completed: false
- validation_completed: false
- manifest_written_at: 2026-05-24-03:15

## baseline 基本資訊

- baseline_name: dev-minimal-analytics-baseline-v1-placeholder
- baseline_dump_status: missing
- expected_local_dump_glob: backup/dev/baseline/*.dump

## 資料範圍

- owner_user_id: 未定；目前 repo 內沒有已驗證 baseline dump
- date_window: 未定；目前 repo 內沒有已驗證 baseline dump

## 主要表 row count

- ia_users: 0
- pos_product_dim: 0
- pos_branch_dim: 0
- pos_sales_hourly_fact: 0
- pos_order_type_dim: 10
- pos_payment_type_dim: 8
- pos_order_status_dim: 4

## smoke query expectation

- make dev-restore-baseline：在沒有本機 `backup/dev/baseline/*.dump` 的情況下，必須明確失敗並提示缺少 local baseline dump
- make dev-smoke-analytics：在目前空資料 baseline 上，必須明確輸出 `smoke failed` 並以 non-zero 結束
- 商品排行榜 smoke query 必須使用名稱排除關鍵字：`幣`、`券`、`折抵`、`折扣`、`點數`、`贈`、`服務費`、`運費`、`調整`、`測試`、`test`

## 注意事項

- 目前本機沒有任何可直接 restore 的 local baseline dump；不得假造看似真實的 50 嵐銷售資料補位
- future baseline dump 只允許放在本機 `backup/dev/baseline/`，不得混入一般 `backup/dev/*.dump` 的 5 份輪替機制
- future baseline dump 的對應 backup manifest 應提交到 `backup/manifest/dev/baseline/*.md`
- baseline restore 仍沿用高風險 restore 規則：執行前先做 pre-restore backup
- 後續若要新增正式 baseline dump，必須先確認資料來源、owner_user_id、日期窗與 row count，並回填本 manifest