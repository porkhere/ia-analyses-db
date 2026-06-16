# ia-analyses-db test routing

更新日期：2026-05-24-18:05
校準日期：2026-05-24-18:05

| 任務 | 類型 | 最小驗證 | 說明 |
|---|---|---|---|
| guide / code-nav 文件更新 | path-check | 檢查 `guide/` 與核心 source 入口路徑存在 | 不執行 build |
| Makefile / 操作入口確認 | engineering-validation | `make help` | 只驗證正式入口可見，不是測試 |
| sales-pipe 功能線抽樣 | path-check | 先檢查 `../ia-analyses-go/cmd/sales-pipe/main.go`，必要時再對照本 repo bridge copy | 需要實跑 controller 時另有任務授權 |
| sync-sales-dims 功能線抽樣 | path-check | 先檢查 `../ia-analyses-go/cmd/sync-sales-dims/main.go`，必要時再對照本 repo bridge copy | 不把 path-check 誤報為測試 |
| validation gate 功能線抽樣 | path-check | 檢查 `internal/athena/sales_candidate_provider.go`、`internal/validation/gates.go` | 只確認路由可定位 |