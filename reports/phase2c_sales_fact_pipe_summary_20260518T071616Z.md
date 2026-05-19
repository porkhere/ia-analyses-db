# Phase 2C Sales Fact Pipe Summary

## Basic Information

- report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T071616Z.md
- run_id: 20260518T071616Z
- owner_user_key: demo-owner
- owner_user_id: 1
- source_schema: 50lan_new
- start_date: 2025-01-11
- end_date: 2025-01-11
- requested_days: 1
- internal_chunk_size: 7
- execution_mode: validate-only
- actual_write_enabled: false
- status: success
- current_business_date: 2025-01-11

## Timing

- started_at: 2026-05-18T15:16:16+08:00
- finished_at: 2026-05-18T15:17:17+08:00
- elapsed_seconds: 61.000
- elapsed_human_readable: 1m1s

## Write Summary

- total_rows_written: 0
- processed_days: 1
- succeeded_days: 1
- failed_days: 0
- rollback_days: 0
- skipped_days: 0

| business_date | row_count | status |
|---|---:|---|
| 2025-01-11 | 130940 | validated |

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

- pos_sales_hourly_fact_total_size: 349 MB
- pos_sales_hourly_fact_table_size: 173 MB
- pos_sales_hourly_fact_indexes_size: 176 MB
- database_size: 357 MB

## Report Summary

- summary_report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T071616Z.md
- summary_report_file_size: 1.5 KiB
- whether_raw_row_log_saved: no
