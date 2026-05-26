# Sales Fact Regression Windows Phase 2C

## 目的

這份文件定義 Phase 2C-5.R 使用的小窗口 regression plan。目的不是擴張功能，而是在不改 semantic contract、不改 schema、不改 validation gate 的前提下，確認 Athena -> candidate -> validation -> local PG sales fact pipe 對不同小窗口都能穩定產生正確結果。

## Expected Checks

每個 window 都要先跑以下 read-only 檢查：

- validate-only
- write-plan

read-only 的 expected result 固定為：

- validate-only 不寫 PG
- write-plan 不寫 PG
- actual_write_enabled = false
- source / candidate metrics delta = 0
- product_dim_miss_count = 0
- branch_dim_miss_count = 0
- forbidden_column_count = 0

若該 window 允許 actual write，還要再檢查：

- actual write 只允許既有開放範圍
- post-insert target metrics delta = 0
- PG row count 與 candidate row count 相等
- row count 不累加
- updated_at 有刷新

## Fail Classification

若任何 window fail，先分類，不直接修程式：

- data_semantics_issue
- dimension_bootstrap_issue
- source_candidate_mismatch
- post_insert_mismatch
- validation_gate_issue
- expected_guard_rejection
- actual_code_bug

只有分類為 actual_code_bug 時，後續才考慮修程式。

## Windows

### 2025-01-01

- role: 已知通過日
- validate-only expected checks: read-only；all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；all gates pass；source / candidate delta = 0
- actual write allowed: yes
- PG row count check: business_date = 2025-01-01 的 row count 必須等於 candidate row count 103545
- metrics delta expected result: pre-insert = 0；post-insert = 0
- fail classification hints:
  - source / candidate compare fail -> source_candidate_mismatch
  - post-insert compare fail -> post_insert_mismatch
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue

### 2025-01-02

- role: 相鄰日
- validate-only expected checks: read-only；all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；all gates pass；source / candidate delta = 0
- actual write allowed: yes
- PG row count check: business_date = 2025-01-02 的 row count 必須等於該日 candidate row count
- metrics delta expected result: pre-insert = 0；post-insert = 0
- fail classification hints:
  - source / candidate compare fail -> source_candidate_mismatch
  - post-insert compare fail -> post_insert_mismatch
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue

### 2025-01-01 ~ 2025-01-02

- role: 已知 2-day write 通過窗口
- validate-only expected checks: read-only；兩天 all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；兩天 all gates pass；source / candidate delta = 0
- actual write allowed: yes
- PG row count check: 2025-01-01 與 2025-01-02 的 row count 都必須與各日 candidate row count 相等
- metrics delta expected result: 兩天 pre-insert = 0；兩天 post-insert = 0
- fail classification hints:
  - 第一天 fail 且第二天未處理 -> validation_gate_issue / source_candidate_mismatch / post_insert_mismatch
  - 第二天 fail -> post_insert_mismatch / validation_gate_issue / actual_code_bug

### 2025-01-07

- role: 非相鄰單日
- validate-only expected checks: read-only；all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；all gates pass；source / candidate delta = 0
- actual write allowed: no
- PG row count check: not required in 5.R
- metrics delta expected result: pre-insert = 0
- fail classification hints:
  - compare fail -> source_candidate_mismatch
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue

### 2025-01-15

- role: 月中單日
- validate-only expected checks: read-only；all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；all gates pass；source / candidate delta = 0
- actual write allowed: no
- PG row count check: not required in 5.R
- metrics delta expected result: pre-insert = 0
- fail classification hints:
  - compare fail -> source_candidate_mismatch
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue

### 2025-01-31

- role: 月底單日
- validate-only expected checks: read-only；all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；all gates pass；source / candidate delta = 0
- actual write allowed: no
- PG row count check: not required in 5.R
- metrics delta expected result: pre-insert = 0
- fail classification hints:
  - compare fail -> source_candidate_mismatch
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue

### 2025-01-31 ~ 2025-02-01

- role: 跨月兩日，只 validate / write-plan，不 actual write
- validate-only expected checks: read-only；兩天 all gates pass；source / candidate delta = 0
- write-plan expected checks: read-only；兩天 all gates pass；source / candidate delta = 0
- actual write allowed: no
- PG row count check: not required in 5.R
- metrics delta expected result: 兩天 pre-insert = 0
- fail classification hints:
  - compare fail -> source_candidate_mismatch 或 data_semantics_issue
  - gate fail -> validation_gate_issue 或 dimension_bootstrap_issue
  - actual write attempt blocked -> expected_guard_rejection

## Scope Boundary

Phase 2C-5.R 只做 small-window regression validation：

- 不做 7-day actual write
- 不做 31-day actual write
- 不改 schema
- 不改 semantic contract
- 不改 validation gate
- 不新增其他 fact

5.R 通過後，下一步才回到 Phase 2C-5.6 controlled 7-day local write validation。

## Observed Results

### Read-only Pass Windows

- 2025-01-01: validate-only pass；write-plan pass；`actual_write_enabled = false`
- 2025-01-02: validate-only pass；write-plan pass；`actual_write_enabled = false`
- 2025-01-01 ~ 2025-01-02: validate-only pass；write-plan pass；兩天 source / candidate delta = `0`

### Read-only Failed Windows

- 2025-01-07: pre-insert hard gate failed；`product_dim_miss_count = 24`、`branch_dim_miss_count = 162`；分類 `dimension_bootstrap_issue`
- 2025-01-15: pre-insert hard gate failed；`product_dim_miss_count = 19`、`branch_dim_miss_count = 138`；分類 `dimension_bootstrap_issue`
- 2025-01-31: pre-insert hard gate failed；`product_dim_miss_count = 31`、`branch_dim_miss_count = 989`；分類 `dimension_bootstrap_issue`
- 2025-01-31 ~ 2025-02-01: 在 2025-01-31 即被同一組 dimension gate 擋下，未繼續到 2025-02-01；分類 `dimension_bootstrap_issue`

### Actual Write Regression Results

- 2025-01-01: actual write pass；PG row count = `103545`；post-insert delta = `0`；`updated_at` 刷新
- 2025-01-02: actual write pass；PG row count = `98167`；post-insert delta = `0`；`updated_at` 刷新
- 2025-01-01 ~ 2025-01-02: actual write pass；兩天 row count 維持 `103545` / `98167`；沒有累加；兩天 `updated_at` 都刷新

### Current Disposition

- 5.R 尚未全窗口通過
- 目前沒有 evidence 指向 `source_candidate_mismatch`、`post_insert_mismatch` 或 `actual_code_bug`
- 5.6 controlled 7-day local write validation 維持暫停，待 dimension bootstrap 問題完成 disposition 後再恢復