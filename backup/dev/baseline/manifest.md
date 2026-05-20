# Dev Minimal Analytics Baseline Manifest

最後更新：2026-05-20-22:54

註：2026-05-20-22:54 起已啟用附則第 1 條。若未來建立 baseline dump，實體檔只保留在本機 `backup/dev/baseline/`，不入 git；repo 內提交的追蹤證據改為本檔與 `backup/manifest/dev/baseline/*.md`。

## baseline 基本資訊

- baseline 名稱：`dev-minimal-analytics-baseline-v1-placeholder`
- 建立時間：`2026-05-20`
- baseline dump 狀態：`missing`
- baseline dump 路徑：`backup/dev/baseline/*.dump`

## 資料範圍

- owner_user_id：`未定；目前 repo 內沒有已驗證 baseline dump`
- 日期窗：`未定；目前 repo 內沒有已驗證 baseline dump`

## 主要表 row count

- `ia_users = 0`
- `pos_product_dim = 0`
- `pos_branch_dim = 0`
- `pos_sales_hourly_fact = 0`
- `pos_order_type_dim = 10`
- `pos_payment_type_dim = 8`
- `pos_order_status_dim = 4`

## smoke query expectation

- `make dev-restore-baseline`：在沒有本機 `backup/dev/baseline/*.dump` 的情況下，必須明確失敗並提示缺少 local baseline dump
- `make dev-smoke-analytics`：在目前空資料 baseline 上，必須明確輸出 `smoke failed` 並以 non-zero 結束
- 商品排行榜 smoke query 必須使用名稱排除關鍵字：`幣`、`券`、`折抵`、`折扣`、`點數`、`贈`、`服務費`、`運費`、`調整`、`測試`、`test`

## 注意事項

- 目前本機沒有任何可直接 restore 的 local baseline dump；不得假造看似真實的 50 嵐銷售資料補位
- future baseline dump 只允許放在本機 `backup/dev/baseline/`，不得混入一般 `backup/dev/*.dump` 的 5 份輪替機制
- future baseline dump 的對應 backup manifest 應提交到 `backup/manifest/dev/baseline/*.md`
- baseline restore 仍沿用高風險 restore 規則：執行前先做 pre-restore backup
- 後續若要新增正式 baseline dump，必須先確認資料來源、owner_user_id、日期窗與 row count，並回填本 manifest