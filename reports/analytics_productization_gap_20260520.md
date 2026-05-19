# Analytics Productization Gap Report

最後更新：2026-05-20

## 本輪結論

- `ia-analyses-db` 已具備一條可寫入的核心 analytics fact path，以及對應的 dimension bootstrap、controller state 與 QuickSight 對照文件。
- 目前真正阻礙 productization 的不是 schema 不存在，而是三個缺口：`可重現 baseline data`、`商品語意分類`、`API / serving layer`。
- 因此下一階段不應再陷入大範圍 schema 討論；應改成先把「可重現 smoke baseline + read query contract + 語意排除規則」做出來。

## 目前 analytics 能力盤點

### 1. 已存在的 fact table

- `pos_sales_hourly_fact`
  - grain：`owner_user_id + business_date(sale_period) + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`
  - 指標：`qty_milli`、`gross_sales_milli`、`discount_milli`、`surcharge_milli`、`net_sales_milli`、`sales_ex_tax_milli`、`included_tax_milli`、`excluded_tax_milli`、`tax_milli`

### 2. 已存在的 dimension / mapping

- `ia_users`
- `pos_product_dim`
- `pos_branch_dim`
- `pos_order_type_dim`
- `pos_payment_type_dim`
- `pos_order_status_dim`

### 3. 已存在的 aggregation

- 唯一正式落地的 persisted aggregation 是 `pos_sales_hourly_fact`
- 它本身已是商品 x 門店 x 小時 x canonical order type x canonical payment type 的聚合層
- 目前沒有 materialized views
- 目前沒有第二層 serving aggregation table

### 4. 已存在的 pipeline / execution path

- `sync-athena-dry-fast`
- `sync-athena-dry-full`
- `sync-athena-validate`
- `sync-athena-write-plan`
- `sync-athena-write-local`
- `sync-sales-dims-plan`
- `sync-sales-dims`
- `sales-pipe-status`
- `sales-pipe-plan`
- `sales-pipe-validate`
- `sales-pipe-write-local`
- `sales-pipe-resume`
- `sales-pipe-report`

### 5. 已存在的 state / controller / report

- controller：`internal/salespipe/controller.go`
- CLI：`cmd/sales-pipe/main.go`
- state file：`state/sales_fact_pipe_state.json`
- summary reports：`reports/phase2c_sales_fact_pipe_summary_<run_id>.md`
- 這一層已能提供 plan、status、resume、summary，但不是 API layer

### 6. 已存在的 QuickSight metric mapping

- 已有完整對照文件：`文件/quicksight_metric_mapping.md`
- Direct / Partial / Missing 的分類已經存在，可直接拿來當 product roadmap 的能力盤點基礎
- 目前最可直接承接的是：branch / product / canonical order type / canonical payment type 粒度下的銷售金額、杯數、稅、折扣、附加費
- 目前最明確缺失的是：raw payment、void lifecycle、order_num、condiment、branch opening、對帳拆項

### 7. 已存在、可直接做 API 的統計資料

前提：需要先有可重現 baseline data；目前本機 dev PG 仍是空資料 baseline。

一旦 baseline restore 完成，現有 fact / dim 已可直接支援：

- Top N 商品排行：依 `qty_milli`、`gross_sales_milli`、`net_sales_milli`、`sales_ex_tax_milli`
- Top N 門店排行：依 `net_sales_milli`、`qty_milli`
- 時段銷售：依 `hour_of_day` 聚合
- 訂單型態分析：依 `order_type_id`
- canonical 付款型態分析：依 `payment_type_id`
- 稅 / 折扣 / 附加費摘要
- 依 `sale_period` 的日級營業額趨勢
- 商品 x 門店交叉分析

## A. 目前已可做的 analytics

### 商品排行榜

- 狀態：可做，但需要 baseline data 與商品語意排除規則
- 資料來源：`pos_sales_hourly_fact` + `pos_product_dim`
- 風險：目前沒有 `normal_sales_item` 類欄位，特殊交易項可能混入排行

### 門店排行榜

- 狀態：可做
- 資料來源：`pos_sales_hourly_fact` + `pos_branch_dim`
- 可做指標：營業額、杯數、稅額、折扣、附加費

### 時段銷售

- 狀態：可做
- 資料來源：`pos_sales_hourly_fact.hour_of_day`
- 可做輸出：尖峰時段、門店時段分布、商品時段分布

### payment analysis

- 狀態：部分可做
- 可做範圍：canonical payment type mix
- 缺口：沒有 raw payment `name` / `memo1` / `amount - change`

### category analysis

- 狀態：部分可做
- 可做前提：`pos_product_dim.cate_no/cate_name` 要有完整 baseline
- 缺口：目前 category completeness 與排行排除規則尚未固定

### 其他已具備基礎的統計

- 日級營業額趨勢（以 `sale_period` 語意）
- 訂單型態 mix
- 稅 / 折扣 / 附加費摘要
- 商品 x 門店交叉分析
- 商品 / 門店 / 時段的 share-of-sales 類統計

## B. 目前缺少但應優先補齊的

### 1. 可重現 baseline data

- 這是 productization 的第一阻塞點
- 沒有 baseline，所有 leaderboard / API / smoke test 都無法穩定重建

### 2. 商品語意分類

- 需要能區分正常販售商品與支付 / 折抵 / 特殊交易項
- 否則商品排行榜、低銷量商品、品類分析都會被污染

### 3. API layer

- 目前 repo 沒有 HTTP server、沒有 API route、沒有 query serving contract
- 要產品化，至少要先有 read-only analytics API

### 4. aggregation layer

- 目前只有 `pos_sales_hourly_fact` 這一層
- 還沒有專門面向 API 的 second-layer aggregation

### 5. materialized views

- 目前沒有 materialized view
- 若排行榜 / 趨勢 API 會頻繁查詢，之後應考慮加一層 pre-aggregated serving view

### 6. anomaly detection 基礎

- 目前沒有固定的異常基準、rolling window、baseline compare
- 但現有 fact 已足夠支撐第一版 rule/statistics 型異常檢查

### 7. forecast 基礎

- 目前缺時間序列 feature layer、也沒有穩定 baseline dataset
- 不適合直接跳進大型 forecast 系統

### 8. basket analysis 基礎

- 目前沒有 order-level fact，因此無法穩定做 co-purchase / association
- 這塊依賴後續 order fact

## C. 哪些功能適合用什麼方法

| 類型 | 適合的功能 |
| --- | --- |
| 純 SQL | 商品排行、門店排行、時段銷售、日級趨勢、canonical payment mix、order type mix、稅/折扣/附加費摘要 |
| 統計學 | 銷售異常偵測、移動平均、週期性比較、Top mover 分析、簡易 forecast v1 |
| rule-based | 商品語意分類、特殊交易項排除、baseline smoke validation、資料品質警示、異常門檻告警 |
| ML/AI | 需求預測進階版、商品推薦、自然語言分析助理、複雜異常分類 |
| hybrid | anomaly detection、forecast v1.5、商品分類 bootstrap、basket analysis v1 |

### 具體判斷

- `商品排行榜 / 門店排行榜 / 時段銷售`：純 SQL 即可先上線
- `商品語意分類`：先 rule-based，後續可 hybrid
- `payment analysis`：先純 SQL 做 canonical mix，raw payment 之後再擴充
- `anomaly detection`：先 statistics + rule-based，比直接上 ML 更實際
- `forecast`：先用統計學，後續再考慮 hybrid
- `basket analysis`：等 order fact 後先用 SQL / rule-based，再視價值升級

## D. 飲料 POS analytics system 最有價值的前 10 個功能候選

1. 商品排行榜 API
2. 門店排行榜 API
3. 時段銷售熱圖與尖峰分析
4. 商品部門 / category mix 分析
5. canonical payment / order type mix 分析
6. 商品語意分類與特殊交易項排除治理
7. 異常營收 / 杯數 / 門店波動告警
8. 低銷量商品 / 長尾商品監控
9. 短期需求預測 v1（移動平均 / weekday seasonality）
10. basket analysis v1（共購 / 加購關聯）

## 建議 roadmap 起點

### 第一優先

- tracked minimal baseline restore
- 商品語意排除規則
- Top 商品 / Top 門店 / 時段銷售三個 read-only API query contract

### 第二優先

- category analysis query contract
- canonical payment / order type mix API
- anomaly watchlist v1

### 第三優先

- materialized views / serving aggregation
- forecast v1
- basket analysis v1

## 建議下一輪最小實作範圍

1. 建立可重現 baseline data
2. 固定商品語意排除規則 v1
3. 定義三支 read-only analytics API / SQL contract：商品排行、門店排行、時段銷售
4. 補一個 `dev-smoke-analytics`，驗證 baseline restore 後這三種查詢都可跑

這樣才算真正從 DB 治理階段走到 analytics productization 的第一步。