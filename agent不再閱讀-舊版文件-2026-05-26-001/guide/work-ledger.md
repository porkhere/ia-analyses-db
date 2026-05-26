- date: 2026-05-24-16:27
  type: docs
  summary: 移除 backup manifest 制度並把附則第 1 條收斂回 dump ignore 與 local baseline
  touched_impl: Makefile; scripts/db_backup.sh; scripts/db_restore.sh; scripts/db_del_backup.sh; scripts/db_restore_baseline.sh
  touched_docs: README.md; 文件/架構指南.md; 文件/更新紀錄.md; 文件/minimal_analytics_baseline_plan.md; guide/task-routing.md
  feature_line: db-backup-addendum-1
  verify: bash -n scripts/db_backup.sh scripts/db_restore.sh scripts/db_del_backup.sh scripts/db_restore_baseline.sh + make help
  next_hint: 下次碰 backup 先看 .gitignore、Makefile 與 scripts/db_restore_baseline.sh，別再引回 manifest

- date: 2026-05-24-02:36
  type: guide
  summary: 建立 sales-pipe、sales-dims 與 validation gate 的 source-level code nav 基線
  touched_impl: cmd/sales-pipe/main.go; internal/salespipe/controller.go; internal/sales/write_skeleton.go; cmd/sync-sales-dims/main.go; internal/athena/sales_candidate_provider.go
  touched_docs: README.md; 文件/README.md; 文件/架構指南.md; guide/code-navigation-index.md; guide/change-impact-map.md
  feature_line: sales-pipe / sales-dims / validation-gate
  verify: path-check + make help
  next_hint: 下次碰 pipeline 先看 cmd/sales-pipe/main.go，再沿 controller.go 進 write_skeleton.go 與 sales_candidate_provider.go