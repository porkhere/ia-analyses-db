- feature_line: sync-athena-cli-entry
  issue: Makefile 已有 `sync-athena-*` 正式入口，但本輪未把對應 source-level entry 收斂到 code-nav 主索引
  suggested_fix: 下輪針對 `sync-athena-*` 補一條從正式入口到 source file 的完整導航鏈
  estimated_scope: medium
  blocking: false

- feature_line: sales-pipe-boundary-comments
  issue: `internal/salespipe/controller.go` 與 `internal/sales/write_skeleton.go` 仍缺少功能邊界短導航註記
  suggested_fix: 下次實際修改該功能線時，在入口檔補最小 `Feature / Flow / Verify` 註記
  estimated_scope: local
  blocking: false

- feature_line: validation-gate-boundary
  issue: `internal/validation/gates.go` 已承接 pre/post-insert gate 規則，但目前仍需直接讀型別才能辨識邊界
  suggested_fix: 下輪若碰 gate 邏輯，補一筆 code-nav 與 task-routing 的交叉索引
  estimated_scope: local
  blocking: false