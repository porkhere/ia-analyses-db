# ia-analyses-db task routing

更新日期：2026-05-24-18:05
校準日期：2026-05-24-18:05

| 任務 | 先讀 | 再進入 | 不要直接做 |
|---|---|---|---|
| DB 文件 / guide 校準 | `README.md`、`文件/README.md`、`文件/架構指南.md` | `guide/index.md` | 不要先進 `文件/過時/` |
| DB runtime / backup 問題 | `README.md`、`文件/架構指南.md`、`.gitignore` | `Makefile`、`scripts/`、`backup/dev/` | 未授權前不要做 restore / 高風險 compose |
| sales-pipe 導航或排查 | `../ia-analyses-go/guide/code-navigation-index.md` | 必要時再對照 `cmd/sales-pipe/main.go`、`internal/salespipe/controller.go`、`internal/sales/write_skeleton.go` | 不要再把本 repo 當主要操作入口 |
| sync-sales-dims 導航或排查 | `../ia-analyses-go/guide/code-navigation-index.md` | 必要時再對照 `cmd/sync-sales-dims/main.go`、`internal/athena/sales_dim_sync.go`、`internal/postgres/sales_dim_sync_writer.go` | 不要把 dimension sync 誤當 sales fact write |
| validation gate 理解 | `guide/code-navigation-index.md` | `internal/athena/sales_candidate_provider.go`、`internal/validation/gates.go` | 不要只看 README 就假設 gate 細節 |