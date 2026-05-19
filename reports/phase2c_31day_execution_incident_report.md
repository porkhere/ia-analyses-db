# Phase 2C 31-Day Execution Incident Report

## Incident Summary

- 31-day long shell orchestration 未完成。
- 使用者查到 PostgreSQL 至少曾寫入 2025-01-01 ~ 2025-01-10。
- incident snapshot 的 total_rows = 1086960。
- incident snapshot 的 database_size = 357 MB。
- 排除目前查詢自身 backend 後，`pg_stat_activity` 沒有 active pipeline query。

## What The Incident Does Not Prove

- 沒有證據證明 Athena 壞掉。
- 沒有證據證明 PostgreSQL 跑不動。
- 沒有證據證明 sales fact 資料模型錯誤。
- 沒有證據證明 31 天資料科學上不可處理。

## Most Likely Failure Class

- 最可能問題是 long command orchestration / observability / interrupt handling。
- 問題重點在於 shell 長命令不可觀測、中途 interrupt 後沒有可靠 state、沒有 failure summary report、也沒有 resume 控制器。

## Immediate Follow-Up

- 下一步不是要求使用者手動縮小日期範圍。
- 下一步是建立 Go controller 內部分段、day-level transaction、progress state、resume、status 與 summary report。
- controller 完成後，長日期範圍由 controller 內部 chunk / day jobs 管理，而不是把切段責任丟回 user interface。

## Current Controller Direction

- Makefile 只保留 thin entrypoint。
- 複雜任務編排交給 Go execution controller。
- state 檔固定落在 `state/sales_fact_pipe_state.json`。
- summary report 固定落在 `reports/phase2c_sales_fact_pipe_summary_<run_id>.md`。
- summary report 只保留摘要，不保存逐筆 candidate row 或逐筆 insert row log。