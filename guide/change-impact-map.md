- pattern: cmd/sales-pipe/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - guide/task-routing.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/architecture-map.md
    - 文件/README.md

- pattern: cmd/sync-sales-dims/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - guide/task-routing.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/architecture-map.md

- pattern: internal/athena/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/task-routing.md

- pattern: internal/salespipe/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/architecture-map.md

- pattern: internal/sales/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/task-routing.md

- pattern: internal/validation/**
  category: code
  must_check:
    - guide/code-navigation-index.md
    - guide/code-navigation-backlog.md
    - guide/work-ledger.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
  optional_check:
    - guide/task-routing.md

- pattern: scripts/**/*.sh
  category: runtime
  must_check:
    - README.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
    - guide/task-routing.md
  optional_check:
    - guide/test-routing.md

- pattern: db/**
  category: db
  must_check:
    - 文件/table 結構文件.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
    - guide/doc-registry.md
  optional_check:
    - 文件/data_model_design.md

- pattern: Makefile
  category: makefile
  must_check:
    - README.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
    - guide/task-routing.md
    - guide/test-routing.md
  optional_check:
    - guide/doc-registry.md

- pattern: .env.example
  category: env
  must_check:
    - README.md
    - 文件/架構指南.md
    - 文件/更新紀錄.md
    - guide/doc-registry.md
  optional_check:
    - guide/task-routing.md