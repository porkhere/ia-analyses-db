# ia-analyses-db architecture map

更新日期：2026-05-24-02:36
校準日期：2026-05-24-02:36

| 功能線 | 主入口 | 核心 flow | 資料存取 / 外部依賴 | 驗證入口 |
|---|---|---|---|---|
| DB runtime / backup | `Makefile` | `dev/prod target -> scripts/*.sh -> docker compose / psql` | `.env`、Docker Compose、PostgreSQL、backup/manifest | `make help` |
| Sales fact pipeline controller | `cmd/sales-pipe/main.go` | `main.go -> internal/salespipe/controller.go -> internal/sales/write_skeleton.go` | Athena、PostgreSQL、`state/`、`reports/` | `guide/code-navigation-index.md` |
| Sales dimension sync | `cmd/sync-sales-dims/main.go` | `main.go -> internal/athena/sales_dim_sync.go -> internal/postgres/sales_dim_sync_writer.go` | Athena、PostgreSQL | `guide/code-navigation-index.md` |
| Candidate / validation gate | `internal/athena/sales_candidate_provider.go` | `candidate provider -> validation reader -> pre/post-insert report` | Athena metrics、validation gate、persisted target metrics | `guide/code-navigation-index.md` |

## 補充

- `ia-analyses-db` 是目前 workspace 唯一 coding tier 與唯一 runtime repo
- source-level code nav 已以實際入口檔抽樣建立基線；若要補 code 內導航註記，應在後續碰到該功能線時局部落地