# Phase 2C Sales Fact Pipe Summary

## Basic Information

- report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T073547Z.md
- run_id: 20260518T073547Z
- owner_user_key: demo-owner
- owner_user_id: 1
- source_schema: 50lan_new
- start_date: 2025-01-01
- end_date: 2025-01-31
- requested_days: 31
- internal_chunk_size: 7
- execution_mode: write-local
- actual_write_enabled: true
- status: success
- current_business_date: 2025-01-31

## Timing

- started_at: 2026-05-18T15:35:47+08:00
- finished_at: 2026-05-18T16:08:38+08:00
- elapsed_seconds: 1971.000
- elapsed_human_readable: 32m51s

## Write Summary

- total_rows_written: 3698110
- processed_days: 20
- succeeded_days: 20
- failed_days: 0
- rollback_days: 0
- skipped_days: 11

| business_date | row_count | status |
|---|---:|---|
| 2025-01-01 | 103545 | skipped |
| 2025-01-02 | 98167 | skipped |
| 2025-01-03 | 112298 | skipped |
| 2025-01-04 | 137469 | skipped |
| 2025-01-05 | 121797 | skipped |
| 2025-01-06 | 98455 | skipped |
| 2025-01-07 | 104005 | skipped |
| 2025-01-08 | 107173 | skipped |
| 2025-01-09 | 92923 | skipped |
| 2025-01-10 | 111128 | skipped |
| 2025-01-11 | 130940 | skipped |
| 2025-01-12 | 129004 | success |
| 2025-01-13 | 88320 | success |
| 2025-01-14 | 100089 | success |
| 2025-01-15 | 101065 | success |
| 2025-01-16 | 104794 | success |
| 2025-01-17 | 120791 | success |
| 2025-01-18 | 136785 | success |
| 2025-01-19 | 128989 | success |
| 2025-01-20 | 115599 | success |
| 2025-01-21 | 119596 | success |
| 2025-01-22 | 116341 | success |
| 2025-01-23 | 122163 | success |
| 2025-01-24 | 141243 | success |
| 2025-01-25 | 143207 | success |
| 2025-01-26 | 134856 | success |
| 2025-01-27 | 142185 | success |
| 2025-01-28 | 113454 | success |
| 2025-01-29 | 141007 | success |
| 2025-01-30 | 150452 | success |
| 2025-01-31 | 130270 | success |

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

- pos_sales_hourly_fact_total_size: 1029 MB
- pos_sales_hourly_fact_table_size: 590 MB
- pos_sales_hourly_fact_indexes_size: 439 MB
- database_size: 1037 MB

## Report Summary

- summary_report_file: /Users/pork/sofone-project/IA-Analyses/ia-analyses-db/reports/phase2c_sales_fact_pipe_summary_20260518T073547Z.md
- summary_report_file_size: 2.5 KiB
- whether_raw_row_log_saved: no
