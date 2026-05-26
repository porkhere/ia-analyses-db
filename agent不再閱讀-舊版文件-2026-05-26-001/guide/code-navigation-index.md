# ia-analyses-db code navigation index

更新日期：2026-05-24-02:36
校準日期：2026-05-24-02:36

| 功能線 | 功能入口 | 核心 flow | 資料存取 / 外部依賴 | 回傳 / 輸出契約 | 最小驗證 | 相關文件 |
|---|---|---|---|---|---|---|
| Sales pipe controller | `cmd/sales-pipe/main.go` | `main.go -> internal/salespipe/controller.go -> internal/sales/write_skeleton.go -> internal/athena/sales_candidate_provider.go` | Athena、PostgreSQL、`state/sales_fact_pipe_state.json`、`reports/phase2c_sales_fact_pipe_summary_<run_id>.md` | `salespipe.Result`、controller state summary、daily summary | path-check + `make help` | `文件/架構指南.md` |
| Sales dimension sync | `cmd/sync-sales-dims/main.go` | `main.go -> internal/athena/sales_dim_sync.go -> internal/postgres/sales_dim_sync_writer.go` | Athena、PostgreSQL | `salesdims.PlanResult`、`salesdims.ApplyResult` | path-check + `make help` | `文件/架構指南.md` |
| Candidate / validation gate | `internal/athena/sales_candidate_provider.go` | `sales_candidate_provider.go -> validation reader -> internal/validation/gates.go` | Athena source metrics、persisted target metrics、dimension / negative schema gate | `sales.CandidateBuildResult`、`validation.PreInsertReport`、`validation.PostInsertReport` | path-check | `guide/task-routing.md` |