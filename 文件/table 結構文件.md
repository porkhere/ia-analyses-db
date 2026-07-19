# ia-analyses-db table 結構文件

## 核心結構總覽

目前初始化 schema 由 `db/init/001_schema.sql` 定義，主體是 7 張核心表與對應 index / seed。資料流方向是：`ia_users` 提供 owner 對照，商品與門市由 dimension sync 寫入維度表，銷售資料由 `sales-pipe` 寫入 `pos_sales_hourly_fact`。

另外還有 3 張 IA Signals 表（`ia_signal_weather`、`ia_signal_promotion`、`ia_signal_availability`，由 `db/patches/004_ia_signals.sql` 建立），以及 1 張分店地理與 CWA 對照表（`ia_branch_location_mapping`，由 `db/patches/005_branch_location_mapping.sql` 建立）。這些表都獨立於 sales fact 存在，細節見下方各節。

## 關聯圖概念

- `ia_users.id` 被 `pos_product_dim.owner_user_id`、`pos_branch_dim.owner_user_id`、`pos_sales_hourly_fact.owner_user_id` 參照
- `pos_order_type_dim.id` 被 `pos_sales_hourly_fact.order_type_id` 參照
- `pos_payment_type_dim.id` 被 `pos_sales_hourly_fact.payment_type_id` 參照
- `pos_order_status_dim` 目前主要提供狀態語意對照與驗證，不直接成為現行 sales fact 的外鍵
- `ia_users.id` 也被 `ia_signal_weather.owner_user_id`、`ia_signal_promotion.owner_user_id`、`ia_signal_availability.owner_user_id` 參照（tenant scoping）；這三張表跟 `pos_sales_hourly_fact` 之間**沒有** FK 關聯，刻意保持獨立
- `ia_users.id` 也被 `ia_branch_location_mapping.owner_user_id` 參照（tenant scoping）；`ia_branch_location_mapping.branch_id` 刻意**沒有**參照 `pos_branch_dim.branch_id` 的 FK，讓分店維度改名或變動時仍能保留歷史對照

## 資料表說明

### ia_users

用途：管理外部 `owner_user_key` 與內部整數 `id` 的對照。

主要欄位：

- `id`：主鍵
- `owner_user_key`：外部 owner key，唯一
- `display_name`：顯示名稱
- `source_system`：來源系統，預設 `athena`
- `is_active`：是否啟用
- `created_at`、`updated_at`：稽核時間

### pos_order_type_dim

用途：統一訂單型態的 canonical mapping。

主要欄位：

- `id`：主鍵
- `code`：穩定代碼
- `name`：顯示名稱
- `description`：說明
- `sort_order`：排序
- `is_active`、`created_at`

目前 seed 固定 10 筆，包含 `unknown`、`in_store`、`foodpanda`、`delivery`、`pickup`、`ubereats`、`quick_pickup`、`quick_delivery`、`qr_order`、`other`。

### pos_payment_type_dim

用途：統一付款型態的 canonical mapping。

主要欄位：

- `id`：主鍵
- `code`：穩定代碼
- `name`：顯示名稱
- `description`：說明
- `sort_order`：排序
- `is_active`、`created_at`

目前 seed 固定 8 筆，包含 `unknown_payment`、`cash`、`card`、`e_wallet`、`platform_payment`、`coupon`、`mixed`、`other`。

### pos_order_status_dim

用途：定義 raw order status 的報表語意。

主要欄位：

- `status_code`：主鍵，直接使用 raw status code
- `status_name`：canonical 名稱
- `status_bucket`：高階分桶，現有 `sales`、`void`、`excluded`
- `is_sales`、`is_void`、`is_cancelled_like`、`is_excluded`
- `description`：語意說明
- `sort_order`、`is_active`、`updated_at`

目前 seed 固定 4 筆，對應 `1`、`-2`、`-1`、`2` 等主要狀態。

### pos_product_dim

用途：紀錄 owner 名下的商品維度。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `product_no`：來源商品編號
- `product_name`：商品名稱
- `product_name_normalized`：正規化名稱
- `cate_no`、`cate_name`：商品分類欄位
- `is_active`、`last_seen_at`、`created_at`、`updated_at`

唯一鍵：`(owner_user_id, product_no)`

來源註記：
- introduced by init: `db/init/001_schema.sql`（初始定義）
- 若欄位由 patch 新增，請標註為 `introduced by patch <patch-filename>` 並註明 patch 名稱。

### pos_branch_dim

用途：紀錄 owner 名下的門市維度。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `branch_id`：來源門市代碼
- `branch_name`：門市名稱
- `branch_name_normalized`：正規化名稱
- `group_code`：門市群組代碼
- `is_active`、`last_seen_at`、`created_at`、`updated_at`

唯一鍵：`(owner_user_id, branch_id)`

`group_code` 欄位授權來源說明：
- 目前狀況：`group_code` 已在 `db/init/001_schema.sql` 定義，但現有同步流程（`ia-analyses-go` 的 sync）並不提供 authoritative `group_code` 值，因而多數紀錄為 NULL。
- Policy：在 POC 階段 `group_code` 為非 authoritative placeholder；前端不應使用 `group_code` 作為 filter 條件。在未定義 authoritative source 之前，不要承諾 branch-group filter。
- 若未來要啟用：請先定義 authoritative source、更新 sync pipeline、在 `db/patches/` 提交 patch 並更新 `文件/table 結構文件.md` 中該欄位註記為 `introduced/modified by patch <patch-filename>`。

### pos_sales_hourly_fact

用途：承接商品層級的小時銷售彙總。

grain：`owner_user_id + business_date + hour_of_day + branch_id + product_no + order_type_id + payment_type_id`

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`
- `business_date`：正式語意固定為 `sale_period`
- `hour_of_day`：0 到 23 的小時值
- `branch_id`：來源門市代碼
- `product_no`：來源商品編號
- `order_type_id`：參照 `pos_order_type_dim.id`
- `payment_type_id`：參照 `pos_payment_type_dim.id`
- `qty_milli`：數量，千分位整數
- `gross_sales_milli`、`discount_milli`、`surcharge_milli`
- `net_sales_milli`、`sales_ex_tax_milli`
- `included_tax_milli`、`excluded_tax_milli`、`tax_milli`
- `created_at`、`updated_at`

註記：`pos_sales_hourly_fact` 與 `pos_product_dim` 已在 smoke analytics 的 product-summary 檢查中被 JOIN 用於產生 product-summary grain（aggregation by `owner_user_id` + `product_no` + `product_name`），以驗證前端所需口徑能否由現有 fact/dim 計算得出。

來源註記：
- introduced by init: `db/init/001_schema.sql`
- 如果有 patch 修改（例如 rename/新增欄位），請在該欄位下註記 `modified by patch <patch-filename>`。

唯一鍵就是這張 fact 的正式 grain。

### ia_branch_location_mapping

用途：保存分店代碼與地址、行政區、郵遞區號、鄉鎮市區代碼、CWA 測站及經緯度的獨立對照。這張表是地理參考資料，不是 `pos_branch_dim` 的欄位延伸，也不會把地理資料塞進 `pos_sales_hourly_fact` 或 IA Signal 表。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`，用來隔離租戶資料
- `branch_id`：來源分店代碼；不建立到 `pos_branch_dim.branch_id` 的外鍵
- `address`、`city_county`、`township_district`、`postal_code`、`township_code`：地址與行政區對照欄位
- `station_id`：外部氣象測站代碼，只有在有來源依據時才可填寫
- `latitude`、`longitude`：經緯度；必須同時為 NULL 或同時有值，且分別限制在 -90 到 90、-180 到 180
- `distance_meters`：對照距離，允許 NULL，但有值時不可為負數
- `source_type`、`source_reference`：來源類型與可追溯的來源識別，不能是空白字串
- `source_metadata`：JSONB 結構化來源補充資料，NOT NULL，沒有補充資料時使用 `{}`；不得用來藏未驗證的猜測值
- `verification_status`：只能是 `unverified`、`verified`、`needs_review` 或 `rejected`
- `verified_at`、`verified_by`：驗證時間與驗證者；狀態為 `verified` 時兩者都必須有值，且 `verified_by` 不能是空白
- `valid_from`、`valid_to`：對照版本的生效區間，`valid_to` 可為 NULL，但有值時不得早於 `valid_from`
- `created_at`、`updated_at`：稽核時間

版本粒度與限制：唯一版本粒度是 `(owner_user_id, branch_id, valid_from)`。`owner_user_id` 是唯一的租戶範圍控制；`branch_id` 只作歷史對照識別，不對目前的 `pos_branch_dim` 建 FK，避免目前分店維度變動時破壞歷史資料。經緯度必須成對出現，座標與距離都由資料庫 CHECK constraint 做基本合法性限制。

驗證規則：新進或尚未人工確認的資料使用 `unverified` 或 `needs_review`；只有留下 `verified_at` 與非空 `verified_by` 才能使用 `verified`。`rejected` 保留被判定不可用的來源紀錄，不能當成已驗證對照使用。

來源中繼資料政策：`source_type` 與 `source_reference` 是必填且不得空白，`source_metadata` 用 JSONB 保存可追溯的來源補充資訊。這張表不接受猜測的地址、分店名稱、CWA 測站或座標；後續若要寫入，必須由明確來源或人工查核流程提供依據。

CSV 匯入契約（Task 2.5.3.2）：正式輸入必須是 UTF-8 CSV，且 header 必須精確為 `customer,branch_id,location`。每個檔案只能包含一個 customer；欄位不可空白、CSV 格式錯誤、重複 `branch_id`、不存在 customer 或不屬於該 customer 的 branch 都會拒絕。`location` 只寫入 `address`，`source_type` 為 `csv`，`verification_status` 固定為 `unverified`；不從分店名稱或地址推導 CWA 測站、座標或 verified 狀態。

操作入口在 `ia-analyses-go`：`make branch-location-import FILE=<path> DRY_RUN=1` 只做完整檔案與 tenant/branch 驗證並回報 counts/errors，不寫 DB；`make branch-location-import FILE=<path> MODE=replace` 會先完成整檔驗證，再以單一 atomic transaction 替換指定 customer 的 current mappings。header-only 或零資料列檔案會拒絕，避免意外清空 mapping。

`ia_branch_location_mapping_import_audit` 會保存來源檔案 SHA-256、匯入前 mapping snapshot、匯入 row snapshot 與 replaced counts；既有 mapping rows 會先關閉有效期間，同日重跑也不會失去 audit。匯入器不修改 sales facts、`pos_branch_dim` 或 IA Signals。

來源註記：
- fresh schema: `db/init/001_schema.sql`
- introduced by patch: `db/patches/005_branch_location_mapping.sql`

### ia_branch_location_mapping_import_audit

用途：保存 Task 2.5.3.2 CSV replace 匯入的批次稽核資訊與 mapping snapshot。這張表不參與 Weather evidence，也不取代 mapping table 的版本資料。

主要欄位：`owner_user_id`、`customer`、`source_reference`、`source_sha256`、`row_count`、`previous_current_count`、`closed_count`、`inserted_count`、`previous_mappings`、`imported_mappings`、`created_at`。

來源註記：`db/init/001_schema.sql`；incremental patch：`db/patches/006_branch_location_import_audit.sql`。

## IA Signals 訊號表

**設計前提（硬規則）**：這三張表是 IA Signals 的持久層，**獨立於 sales fact 存在**，絕不加欄位到 `pos_sales_hourly_fact` 或任何其他 sales fact 表。三張表都用 generic commerce naming（`location`、`item`、`tenant` 概念的 DB 內部對應是 `owner_user_id`），不引入新的 `branch_id`、`product_no` 或其他 50 嵐專屬命名。

**Grain 設計說明**：`location`（對應 `pos_branch_dim.branch_id` 概念）與部分表的 `item`（對應 `pos_product_dim.product_no` 概念）可為 `NULL`，代表「租戶層級／不分特定門市或品項」的訊號。唯一鍵用 PostgreSQL 16 的 `UNIQUE NULLS NOT DISTINCT` 實作（而非把這些欄位直接放進 `PRIMARY KEY`，因為 `PRIMARY KEY` 欄位在 PostgreSQL 中隱含 `NOT NULL`，無法承載「NULL＝租戶層級」的語意）。技術主鍵維持 `id BIGSERIAL`，延續 `pos_product_dim` / `pos_branch_dim` / `pos_sales_hourly_fact` 既有的「surrogate id + UNIQUE 自然鍵」慣例。

### ia_signal_weather

用途：天氣訊號（外部 signal，軸一屬 External）。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`（tenant scoping）
- `location`：對應 `branch_id` 概念；`NULL` = 租戶層級
- `signal_date`：訊號描述的業務日期
- `observation_kind`：**只允許 `forecast` 或 `actual`（CHECK constraint 物理強制）**。這是防資料洩漏的關鍵欄位——forecast frame 組裝時只准 JOIN `observation_kind='forecast'` 的資料；`observation_kind='actual'` 是事後回填的實際觀測，只能用於 retrospective（回測）情境，絕不可進 forecast frame。
- `temperature_c`、`rain_mm`、`humidity_pct`：氣象數值（`NUMERIC`）
- `is_typhoon`：是否颱風
- `source`：訊號來源標註
- `captured_at`：訊號被擷取/寫入的時間點（跟 `signal_date` 分開記錄）
- `created_at`、`updated_at`：稽核時間

唯一鍵（grain）：`UNIQUE NULLS NOT DISTINCT (owner_user_id, location, signal_date, observation_kind)`

來源註記：
- introduced by patch: `db/patches/004_ia_signals.sql`

### ia_signal_promotion

用途：促銷排程訊號（營運內部 signal，軸一屬 Operational；但「已排定促銷」在防洩漏軸二屬 `known_ahead`，可用於 forecast frame）。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`（tenant scoping）
- `location`：對應 `branch_id` 概念；`NULL` = 全租戶（跨店）層級的促銷
- `item`：對應 `product_no` 概念；`NULL` = 不分品項、整店或整租戶層級的促銷
- `signal_date`：促銷生效的業務日期
- `is_promotion`：該租戶／門市／品項在該日是否處於促銷排程中
- `promo_type`：促銷型態自由文字標註，目前未定義 canonical enum
- `discount_pct`：折扣百分比，允許 `NULL`
- `created_at`、`updated_at`：稽核時間

唯一鍵（grain）：`UNIQUE NULLS NOT DISTINCT (owner_user_id, location, item, signal_date)`

來源註記：
- introduced by patch: `db/patches/004_ia_signals.sql`

### ia_signal_availability

用途：品項供應狀態訊號（營運內部 signal，軸一屬 Operational；防洩漏軸二屬 `actual_only`，只能用於 retrospective——賣不好不等於需求低，只能做事後解釋，不能拿來做 forecast）。

主要欄位：

- `id`：主鍵
- `owner_user_id`：參照 `ia_users.id`（tenant scoping）
- `location`：對應 `branch_id` 概念；`NULL` = 全租戶（跨店）層級的狀態（例如整體下架）
- `item`：對應 `product_no` 概念；**本欄位為 `NOT NULL`**——availability 一定綁定特定品項，沒有「不分品項」的 availability 語意
- `signal_date`：狀態描述的業務日期
- `is_stockout`：該品項在該日該門市（或全租戶）是否缺貨
- `is_delisted`：該品項在該日是否已下架
- `created_at`、`updated_at`：稽核時間

唯一鍵（grain）：`UNIQUE NULLS NOT DISTINCT (owner_user_id, location, item, signal_date)`

來源註記：
- introduced by patch: `db/patches/004_ia_signals.sql`

## Index 重點

- `idx_ia_users_active`：owner lookup
- `idx_pos_product_dim_lookup`：商品 lookup
- `idx_pos_branch_dim_lookup`：門市 lookup
- `idx_ia_branch_location_mapping_owner_branch`：依租戶、分店與版本日期查詢地理對照
- `idx_ia_branch_location_mapping_verified_current`：只索引已驗證且 `valid_to IS NULL` 的目前開放對照
- `idx_pos_sales_hourly_fact_date`：日期與小時查詢
- `idx_pos_sales_hourly_fact_branch`：門市分析
- `idx_pos_sales_hourly_fact_product`：商品分析
- `idx_ia_signal_weather_owner_date`：weather 依日期＋observation_kind 查詢（forecast/actual 篩選）
- `idx_ia_signal_promotion_owner_date`：promotion 依日期查詢
- `idx_ia_signal_availability_owner_date`：availability 依日期查詢
- `idx_ia_signal_availability_item`：availability 依品項＋日期查詢

## 寫入來源

- `ia-analyses-go` 的 `sync-sales-dims` 寫入 `pos_product_dim` 與 `pos_branch_dim`
- `ia-analyses-go` 的 `sales-pipe` 寫入 `pos_sales_hourly_fact`
- 本 repo 的 seed 與 patch 保證 canonical 維度與 schema contract 維持一致
- `ia_signal_weather` / `ia_signal_promotion` / `ia_signal_availability` 目前**尚無寫入來源**：本輪（Task 2.1）只建表，尚未實作 Signal 載入 CLI（見 `重構執行建議.md` Task 2.2，屬後續 Phase 2.2+ 工作，不在本輪範圍）
- `ia_branch_location_mapping` 的正式寫入來源是 `ia-analyses-go` 的 `branch-location-import`；目前仍沒有匯入任何 business mapping rows。範例檔 `文件/branch-location-import-example.csv` 只含明確標記的 sample customer/rows，不是可直接宣告 verified 的資料。
