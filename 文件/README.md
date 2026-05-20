# ia-analyses-db 文件索引

更新日期：2026-05-20-16:11
校準日期：2026-05-20-16:11

本目錄根層只保留文件索引與長期主線文件。新工作預設先讀 active 文件，不應直接把 過時 文件當成最新指令來源。

## Active 文件

- [架構指南.md](架構指南.md)：DB repo 長期架構與職責
- [更新紀錄.md](更新紀錄.md)：變更紀錄
- [table 結構文件.md](table%20結構文件.md)：目前 table/schema 說明
- [data_model_design.md](data_model_design.md)：資料模型設計
- [minimal_analytics_baseline_plan.md](minimal_analytics_baseline_plan.md)：最小 analytics baseline 設計

## 過時文件

- 過時/phase2c/：phase2c sales fact / validation / migration 階段文件
- 過時/quicksight/：QuickSight / BI 盤點與 mapping 歷史文件
- 過時/其他/：暫時歸檔但未分類文件

## 歸檔原則

- 過時文件不是刪除，也不是無效文件。
- 過時文件代表「已完成階段任務」或「歷史 review 紀錄」。
- 新工作預設應先讀 active 文件，不應把過時文件當作最新指令來源。
- 若過時文件內容仍有價值，應透過 active 文件重新整理後再引用。
