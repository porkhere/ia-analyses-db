- path: README.md
  type: readme
  truth_source: Makefile, scripts/, cmd/, internal/
  last_synced: 2026-05-24
  verify: make help

- path: 文件/README.md
  type: other
  truth_source: 文件/, guide/
  last_synced: 2026-05-24
  verify: N/A

- path: 文件/架構指南.md
  type: architecture
  truth_source: Makefile, cmd/, internal/, scripts/, db/
  last_synced: 2026-05-24
  verify: make help

- path: 文件/更新紀錄.md
  type: changelog
  truth_source: README.md, guide/work-ledger.md
  last_synced: 2026-05-24
  verify: N/A

- path: 文件/table 結構文件.md
  type: db
  truth_source: db/init/001_schema.sql, db/patches/
  last_synced: 2026-05-24
  verify: N/A

- path: 文件/data_model_design.md
  type: architecture
  truth_source: db/init/001_schema.sql, db/patches/003_phase2c_schema_contract.sql
  last_synced: 2026-05-24
  verify: N/A

- path: 文件/minimal_analytics_baseline_plan.md
  type: db
  truth_source: scripts/db_restore_baseline.sh, scripts/db_smoke_analytics.sh, backup/manifest/
  last_synced: 2026-05-24
  verify: N/A

- path: guide/index.md
  type: guide
  truth_source: guide/, ../ia-analyses-ws-map/ws-map.md
  last_synced: 2026-05-24
  verify: N/A

- path: guide/code-navigation-index.md
  type: code-nav
  truth_source: cmd/, internal/
  last_synced: 2026-05-24
  verify: path-check

- path: guide/change-impact-map.md
  type: guide
  truth_source: repo structure, cmd/, internal/, scripts/, db/
  last_synced: 2026-05-24
  verify: N/A

- path: guide/work-ledger.md
  type: guide
  truth_source: 本輪 source-level 抽樣紀錄
  last_synced: 2026-05-24
  verify: path-check