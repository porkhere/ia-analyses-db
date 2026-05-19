# Phase 2C Sales Fact Pipe Summary

## Basic Information

- report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T071726Z.md
- run_id: 20260518T071726Z
- owner_user_key: demo-owner
- owner_user_id: 1
- source_schema: 50lan_new
- start_date: 2025-01-11
- end_date: 2025-01-11
- requested_days: 1
- internal_chunk_size: 7
- execution_mode: write-local
- actual_write_enabled: true
- status: success
- current_business_date: 2025-01-11

## Timing

- started_at: 2026-05-18T15:17:26+08:00
- finished_at: 2026-05-18T15:19:11+08:00
- elapsed_seconds: 105.000
- elapsed_human_readable: 1m45s

## Write Summary

- total_rows_written: 130940
- processed_days: 1
- succeeded_days: 1
- failed_days: 0
- rollback_days: 0
- skipped_days: 0

| business_date | row_count | status |
|---|---:|---|
| 2025-01-11 | 130940 | success |

## Validation Summary

- source_candidate_delta_all_zero: true
- post_insert_delta_all_zero: true
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

- summary_report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T071726Z.md
- summary_report_file_size: 1.5 KiB
- whether_raw_row_log_saved: no
