# ia-analyses-db POC 目標

建立時間：2026/06/16

## POC 定位

`ia-analyses-db` 是 POC 的分析資料庫與 schema 主權 repo。它永遠只被 `ia-analyses-go` 接觸；前端、Groq、MCP、`ia-analyses-stat` 都不能直接連 DB。

## POC 需要支援的資料能力

- 保存 Athena pipeline 同步來的 POS 銷售資料。
- 支援 product-summary、top/bottom products、period comparison、forecast history window、branch/product grain 查詢。
- 保留多租戶演進空間：目前以 50 嵐作為開發樣本，但 schema / seed / 文件不能把單一客戶寫死成永久規則。

## Tenant 目標

- `ia_users` 是 POC 的 tenant / owner 對照核心。
- POC 可以先有單一 dev tenant，但必須明確命名，例如 `50lan-dev` 或後續決定的 tenant key。
- `demo-owner` 只能是暫時 seed，不應成為前端或 Groq tool 的語意來源。
- 所有 fact / dim query 都要保留 `owner_user_id` 條件。

## Schema 目標

POC 第一版不需要為 Groq / MCP 新增 schema。MCP 是工具協定，不是資料模型。

優先保持現有 7 張核心表：

- `ia_users`
- `pos_order_type_dim`
- `pos_payment_type_dim`
- `pos_order_status_dim`
- `pos_product_dim`
- `pos_branch_dim`
- `pos_sales_hourly_fact`

若要回答「材料準備」：

- POC 第一版不新增材料 BOM 表。
- 先由 Go 以商品銷售 forecast 回答「備料 proxy」。
- 真正材料表應是 POC 後續階段，屆時才新增 ingredient / recipe / product-material mapping schema。

## DB 邊界

- 不直接提供 HTTP API。
- 不提供 MCP server。
- 不把 Groq credentials、前端 session 或 prompt log 放進 DB，除非後續明確設計 audit/log schema。
- 不在 DB repo 新增 Go pipeline 功能；Go pipeline 主責在 `ia-analyses-go`。

## POC 驗收

- Go 能用 tenant context 查到 product summary / period comparison / forecast history 所需資料。
- DB 文件清楚說明 tenant key 仍是 POC decision point。
- `db_smoke_analytics` 或後續 smoke 能覆蓋 product-summary grain 的基本可用性。
