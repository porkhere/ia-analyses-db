# Phase 2C Restart Baseline Report

最後更新：2026-05-20

## 本輪範圍

- 只做 dev 環境、容器、migration、核心表與商品語意調查
- 不做一個月長時間 pipeline、不做大量重跑、不做 restore 或其他破壞性 DB 操作

## Dev 基準結果

- `git status --short` 無輸出，`git rev-list --left-right --count HEAD...@{upstream}` = `0 0`
- `make dev-env`、`make dev-up`、`docker compose ps`、`make dev-migrate`、`make dev-size` 均成功
- PostgreSQL 容器健康，Adminer 正常啟動
- `make dev-migrate` 可重播既有 patch，顯示已套用 patch 002 與 003
- 當前 dev database size 約 `7.84 MB`

## 核心表與資料可用性

以下核心表皆存在：`ia_users`、`pos_order_type_dim`、`pos_payment_type_dim`、`pos_order_status_dim`、`pos_product_dim`、`pos_branch_dim`、`pos_sales_hourly_fact`

目前 row count：

| table | rows |
| --- | ---: |
| ia_users | 0 |
| pos_order_type_dim | 10 |
| pos_payment_type_dim | 8 |
| pos_order_status_dim | 4 |
| pos_product_dim | 0 |
| pos_branch_dim | 0 |
| pos_sales_hourly_fact | 0 |

結論：目前 dev PostgreSQL 是「schema 與 seed 已就緒，但 sales fact / product dim / branch dim 尚未載入」的重新開始基準，尚不適合直接做商品排行分析。

## `sales-pipe-status` 的正確解讀

- `make sales-pipe-status` 本輪可正常執行
- 但它讀的是 [state/sales_fact_pipe_state.json](../state/sales_fact_pipe_state.json) 的 controller state，而不是用它來證明目前 dev PostgreSQL 已有資料
- 現有 state 仍記錄 2026-05-18 的 31 天成功執行與 `3698110` 筆歷史寫入摘要；這和當前 dev PG row count = `0` 並不衝突，代表目前本機資料庫不是當時那份已載入快照

## 商品語意調查

- `pos_product_dim` 現有欄位只有 `product_no`、`product_name`、`product_name_normalized`、`cate_no`、`cate_name`、`is_active`、`last_seen_at` 與 audit timestamps
- schema 目前沒有 `normal_sales_item`、`product_semantic_type`、`item_kind`、`is_merchandise` 之類可直接區分「正常販售商品」與「支付/折抵/促銷/特殊交易項」的欄位
- 直接查本機 PG 的「雲林幣」與名稱關鍵字疑似非商品項，結果都是 `0 rows`，原因是目前 `pos_product_dim` 與 `pos_sales_hourly_fact` 都是空表，不是因為已證明來源不存在
- 嘗試用 Athena 補來源證據時，現場缺少可用 AWS shared config / credential；`make sync-sales-dims-plan ...` 失敗於 `failed to get shared config profile, default`，因此本輪無法補做來源實證

## 為什麼「雲林幣」不應直接納入商品排行榜

- 產品排行榜應衡量正常販售商品的銷售表現，而不是支付、折抵、贈與、服務費或其他特殊交易項
- 目前產品銷售相關查詢與 QuickSight 盤點證據，主要依賴 `product_name` + `cate_name`；既有排除規則只看到 `cate_name = 其它` 類型的排除，還不足以排掉「幣、券、折抵、點數、贈」這類名稱項目
- 一旦來源 `order_items_parquet` 把此類特殊交易項當成 item row 帶進來，而維度沒有額外語意旗標，排行查詢就會把它們誤視為正常商品
- 因此「雲林幣」這類項目即使真的存在於 item source，也不應直接進入正常商品排行；它更像支付/折抵/促銷/特殊交易語意，而不是一般飲品 SKU

## 最小修正方案

1. 最小查詢層修正：先在商品排行查詢層加 exclusion rule，至少支援 `product_no` 白黑名單與 `product_name` 關鍵字排除，並把命中的排除結果輸出成 audit 清單。
2. 較穩定的維度修正：為商品建立獨立 semantic mapping，例如 `owner_user_id + product_no -> semantic_type / normal_sales_item`；排行只吃 `normal_sales_item = true`。
3. 若一定要放在既有 dim，最小 schema 變更可考慮在 `pos_product_dim` 新增 `normal_sales_item BOOLEAN`，必要時再加 `product_semantic_type TEXT`；`cate_no/cate_name` 只保留類別資訊，不直接兼任交易語意分類。
4. `product_name` / `cate_name` 關鍵字規則只適合 bootstrap，不應當作最終長期真相來源。

## 下一輪建議

- 下一輪應先補「商品語意分類 / 排除規則」，再做一個月資料分析
- 但在開始前，至少要先滿足二選一：
- 讓 dev PG 載入可分析的 product dim / sales fact 資料
- 或恢復 Athena 讀取權限，先做小範圍來源審計，確認「雲林幣」與其他疑似非商品項的實際清單
- 在上述條件未滿足前，不建議直接進行一個月商品排行分析