# Phase 2C Sales Fact Pipe Summary

## Basic Information

- report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T073538Z.md
- run_id: 20260518T073538Z
- owner_user_key: demo-owner
- owner_user_id: 1
- source_schema: 50lan_new
- start_date: 2025-01-01
- end_date: 2025-01-31
- requested_days: 31
- internal_chunk_size: 7
- execution_mode: write-plan
- actual_write_enabled: false
- status: success

## Timing

- started_at: 2026-05-18T15:35:38+08:00
- finished_at: 2026-05-18T15:35:38+08:00
- elapsed_seconds: 0.000
- elapsed_human_readable: 0s

## Write Summary

- total_rows_written: 1217900
- processed_days: 11
- succeeded_days: 11
- failed_days: 0
- rollback_days: 0
- skipped_days: 0

| business_date | row_count | status |
|---|---:|---|
| 2025-01-01 | 103545 | completed |
| 2025-01-02 | 98167 | completed |
| 2025-01-03 | 112298 | completed |
| 2025-01-04 | 137469 | completed |
| 2025-01-05 | 121797 | completed |
| 2025-01-06 | 98455 | completed |
| 2025-01-07 | 104005 | completed |
| 2025-01-08 | 107173 | completed |
| 2025-01-09 | 92923 | completed |
| 2025-01-10 | 111128 | completed |
| 2025-01-11 | 130940 | completed |
| 2025-01-12 | 0 | pending |
| 2025-01-13 | 0 | pending |
| 2025-01-14 | 0 | pending |
| 2025-01-15 | 0 | pending |
| 2025-01-16 | 0 | pending |
| 2025-01-17 | 0 | pending |
| 2025-01-18 | 0 | pending |
| 2025-01-19 | 0 | pending |
| 2025-01-20 | 0 | pending |
| 2025-01-21 | 0 | pending |
| 2025-01-22 | 0 | pending |
| 2025-01-23 | 0 | pending |
| 2025-01-24 | 0 | pending |
| 2025-01-25 | 0 | pending |
| 2025-01-26 | 0 | pending |
| 2025-01-27 | 0 | pending |
| 2025-01-28 | 0 | pending |
| 2025-01-29 | 0 | pending |
| 2025-01-30 | 0 | pending |
| 2025-01-31 | 0 | pending |

## Validation Summary

- source_candidate_delta_all_zero: true
- post_insert_delta_all_zero: false
- product_dim_miss_total: 0
- branch_dim_miss_total: 0
- order_type_dim_miss_total: 0
- payment_type_dim_miss_total: 0
- forbidden_column_count: 0
- hard_gate_failed_count: 0

## PostgreSQL Size Summary

- pos_sales_hourly_fact_total_size: 380 MB
- pos_sales_hourly_fact_table_size: 194 MB
- pos_sales_hourly_fact_indexes_size: 185 MB
- database_size: 388 MB

## Report Summary

- summary_report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T073538Z.md
- summary_report_file_size: 2.4 KiB
- whether_raw_row_log_saved: no
