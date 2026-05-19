package athena

import "fmt"

const defaultPreviewLimit = 20

const topTaxDeltaSampleLimit = 10

const topTaxDeltaTraceLimit = 5

const debugOrderTotalOutlierThresholdTWD = 100000.0
const maxMilliBigintInputTWD int64 = 9223372036854775

func BuildSourceCountSQL(tableName string, window QueryWindow) string {
	return fmt.Sprintf(
		"SELECT count(*) AS row_count FROM %s WHERE t_open_date BETWEEN date '%s' AND date '%s'",
		tableName,
		window.StartDate.Format(dateLayout),
		window.EndDate.Format(dateLayout),
	)
}

func BuildPreviewCountSQL(window QueryWindow) string {
	return fmt.Sprintf("%s SELECT count(*) AS row_count FROM status_dedup_final_aggregation", buildPreviewCTE(window))
}

func BuildPreviewSelectSQL(window QueryWindow) string {
	previewLimit := resolvePreviewLimit(window.PreviewLimit)

	return fmt.Sprintf(
		"%s SELECT cast(business_date AS varchar) AS business_date, cast(hour_of_day AS varchar) AS hour_of_day, branch_id, product_no, cast(order_type_id AS varchar) AS order_type_id, cast(payment_type_id AS varchar) AS payment_type_id, cast(qty_milli AS varchar) AS qty_milli, cast(gross_sales_milli AS varchar) AS gross_sales_milli, cast(discount_milli AS varchar) AS discount_milli, cast(surcharge_milli AS varchar) AS surcharge_milli, cast(net_sales_milli AS varchar) AS net_sales_milli, cast(sales_ex_tax_milli AS varchar) AS sales_ex_tax_milli, cast(tax_milli AS varchar) AS tax_milli FROM status_dedup_final_aggregation ORDER BY business_date, hour_of_day, branch_id, product_no, order_type_id, payment_type_id LIMIT %d",
		buildPreviewCTE(window),
		previewLimit,
	)
}

func BuildOrderMappingSummarySQL(window QueryWindow) string {
	return fmt.Sprintf(
		"%s SELECT cast(order_type_id AS varchar) AS canonical_id, %s AS canonical_code, count(*) AS source_rows, count(distinct destination_raw) AS distinct_raw_values FROM status_dedup_order_destinations GROUP BY 1, 2 ORDER BY 1",
		buildPreviewCTE(window),
		orderTypeCodeSQL("order_type_id"),
	)
}

func BuildPaymentMappingSummarySQL(window QueryWindow) string {
	return fmt.Sprintf(
		"%s SELECT cast(canonical_payment_type_id AS varchar) AS canonical_id, %s AS canonical_code, count(*) AS source_rows, count(distinct payment_name_raw) AS distinct_raw_values FROM payment_raw_values GROUP BY 1, 2 ORDER BY 1",
		buildPreviewCTE(window),
		paymentTypeCodeSQL("canonical_payment_type_id"),
	)
}

func BuildReconciliationSummarySQL(window QueryWindow) string {
	return fmt.Sprintf(`%s,
order_financial_summary AS (
    SELECT
        count(*) AS source_order_count,
        cast(round(sum(order_gross_sales) * 1000) AS bigint) AS source_gross_sales_milli,
        cast(round(sum(order_discount_total) * 1000) AS bigint) AS source_discount_milli,
        cast(round(sum(order_surcharge_total) * 1000) AS bigint) AS source_surcharge_milli,
        cast(round(sum(order_net_sales) * 1000) AS bigint) AS source_net_sales_milli,
        cast(round(sum(order_sales_ex_tax) * 1000) AS bigint) AS source_sales_ex_tax_milli,
        cast(round(sum(order_included_tax) * 1000) AS bigint) AS source_included_tax_milli,
        cast(round(sum(order_excluded_tax) * 1000) AS bigint) AS source_excluded_tax_milli,
        cast(round(sum(order_tax_total) * 1000) AS bigint) AS source_tax_milli,
        cast(round(sum(addition_discount_total) * 1000) AS bigint) AS source_addition_discount_milli,
        cast(round(sum(addition_surcharge_total) * 1000) AS bigint) AS source_addition_surcharge_milli,
        cast(round(sum(addition_include_tax_total) * 1000) AS bigint) AS source_addition_include_tax_milli
    FROM status_dedup_order_financials
    WHERE item_count > 0
),
preview_summary AS (
    SELECT
        count(*) AS preview_group_count,
        coalesce(sum(gross_sales_milli), 0) AS preview_gross_sales_milli,
        coalesce(sum(discount_milli), 0) AS preview_discount_milli,
        coalesce(sum(surcharge_milli), 0) AS preview_surcharge_milli,
        coalesce(sum(net_sales_milli), 0) AS preview_net_sales_milli,
        coalesce(sum(sales_ex_tax_milli), 0) AS preview_sales_ex_tax_milli,
        coalesce(sum(included_tax_milli), 0) AS preview_included_tax_milli,
        coalesce(sum(excluded_tax_milli), 0) AS preview_excluded_tax_milli,
        coalesce(sum(tax_milli), 0) AS preview_tax_milli
    FROM status_dedup_final_aggregation
)
SELECT
    cast(ofs.source_order_count AS varchar) AS source_order_count,
    cast(ofs.source_gross_sales_milli AS varchar) AS source_gross_sales_milli,
    cast(ofs.source_discount_milli AS varchar) AS source_discount_milli,
    cast(ofs.source_surcharge_milli AS varchar) AS source_surcharge_milli,
    cast(ofs.source_net_sales_milli AS varchar) AS source_net_sales_milli,
    cast(ofs.source_sales_ex_tax_milli AS varchar) AS source_sales_ex_tax_milli,
    cast(ofs.source_included_tax_milli AS varchar) AS source_included_tax_milli,
    cast(ofs.source_excluded_tax_milli AS varchar) AS source_excluded_tax_milli,
    cast(ofs.source_tax_milli AS varchar) AS source_tax_milli,
    cast(ofs.source_addition_discount_milli AS varchar) AS source_addition_discount_milli,
    cast(ofs.source_addition_surcharge_milli AS varchar) AS source_addition_surcharge_milli,
    cast(ofs.source_addition_include_tax_milli AS varchar) AS source_addition_include_tax_milli,
    cast(ps.preview_group_count AS varchar) AS preview_group_count,
    cast(ps.preview_gross_sales_milli AS varchar) AS preview_gross_sales_milli,
    cast(ps.preview_discount_milli AS varchar) AS preview_discount_milli,
    cast(ps.preview_surcharge_milli AS varchar) AS preview_surcharge_milli,
    cast(ps.preview_net_sales_milli AS varchar) AS preview_net_sales_milli,
    cast(ps.preview_sales_ex_tax_milli AS varchar) AS preview_sales_ex_tax_milli,
    cast(ps.preview_included_tax_milli AS varchar) AS preview_included_tax_milli,
    cast(ps.preview_excluded_tax_milli AS varchar) AS preview_excluded_tax_milli,
    cast(ps.preview_tax_milli AS varchar) AS preview_tax_milli
FROM order_financial_summary ofs
CROSS JOIN preview_summary ps`, buildPreviewCTE(window))
}

func BuildTaxReconciliationBreakdownSQL(window QueryWindow) string {
	return fmt.Sprintf(`%s
SELECT
    metric,
    cast(order_count AS varchar) AS order_count,
    cast(gross_sales_milli AS varchar) AS gross_sales_milli,
    cast(discount_milli AS varchar) AS discount_milli,
    cast(surcharge_milli AS varchar) AS surcharge_milli,
    cast(net_sales_milli AS varchar) AS net_sales_milli,
    cast(included_tax_milli AS varchar) AS included_tax_milli,
    cast(sales_ex_tax_milli AS varchar) AS sales_ex_tax_milli,
    note
FROM status_dedup_tax_reconciliation_breakdown
ORDER BY sort_order`, buildPreviewCTE(window))
}

func BuildAdditionsTaxDebugSQL(window QueryWindow) string {
	return fmt.Sprintf(`%s
SELECT
    metric,
    cast(metric_value AS varchar) AS metric_value,
    note
FROM status_dedup_additions_tax_debug
ORDER BY sort_order`, buildPreviewCTE(window))
}

func BuildRoundingDebugSQL(window QueryWindow) string {
	return fmt.Sprintf(`%s
SELECT
    metric,
    cast(metric_value AS varchar) AS metric_value,
    note
FROM status_dedup_rounding_debug
ORDER BY sort_order`, buildPreviewCTE(window))
}

func BuildTopTaxDeltaSampleSQL(window QueryWindow) string {
	return fmt.Sprintf(`%s
SELECT
    cast(business_date AS varchar) AS business_date,
    order_id,
    cast(source_included_tax_milli AS varchar) AS source_included_tax_milli,
    cast(allocated_included_tax_milli AS varchar) AS allocated_included_tax_milli,
    cast(delta_milli AS varchar) AS delta_milli,
    cast(item_count AS varchar) AS item_count,
    cast(allocation_denominator_milli AS varchar) AS allocation_denominator_milli,
    cast(status AS varchar) AS status,
    destination
FROM status_dedup_top_tax_delta_sample
ORDER BY abs(cast(delta_milli AS bigint)) DESC, business_date, order_id`, buildPreviewCTE(window))
}

func BuildTopTaxDeltaOrderTraceSQL(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`%s,
top_tax_delta_trace_orders AS (
    SELECT
        row_number() OVER (ORDER BY abs(delta_milli) DESC, business_date, order_id) AS trace_rank,
        business_date,
        order_id,
        source_included_tax_milli,
        allocated_included_tax_milli,
        delta_milli,
        item_count,
        allocation_denominator_milli,
        status,
        destination
    FROM status_dedup_top_tax_delta_sample
    ORDER BY abs(delta_milli) DESC, business_date, order_id
    LIMIT %d
),
top_payment_trace_raw AS (
    SELECT
        t.trace_rank,
        p.order_id,
        coalesce(trim(p.id), '') AS payment_row_id,
        coalesce(trim(p.name), '') AS payment_name,
        %s AS normalized_payment_type_id,
        %s AS payment_amount_milli
    FROM top_tax_delta_trace_orders t
    JOIN order_payments_parquet p ON p.order_id = t.order_id
    WHERE p.t_open_date BETWEEN date '%s' AND date '%s'
),
top_payment_trace_grouped AS (
    SELECT
        trace_rank,
        order_id,
        normalized_payment_type_id,
        sum(payment_amount_milli) AS payment_amount_milli
    FROM top_payment_trace_raw
    GROUP BY 1, 2, 3
),
top_payment_trace_summary AS (
    SELECT
        trace_rank,
        order_id,
        CASE
            WHEN count_if(abs(payment_amount_milli) > 0 AND normalized_payment_type_id <> 0) = 0 THEN 0
            WHEN count_if(abs(payment_amount_milli) > 0 AND normalized_payment_type_id <> 0) = 1 THEN max_by(normalized_payment_type_id, abs(payment_amount_milli))
            ELSE 8
        END AS final_payment_type_id,
        CASE WHEN count_if(abs(payment_amount_milli) > 0 AND normalized_payment_type_id <> 0) > 1 THEN 1 ELSE 0 END AS is_mixed,
        CASE WHEN count_if(payment_amount_milli > 0) > 0 AND count_if(payment_amount_milli < 0) > 0 THEN 1 ELSE 0 END AS has_offsetting_payments,
        count(*) AS payment_after_order_aggregation_row_count,
        count_if(abs(payment_amount_milli) > 0 AND normalized_payment_type_id <> 0) AS normalized_payment_type_count
    FROM top_payment_trace_grouped
    GROUP BY 1, 2
),
top_item_raw_trace AS (
    SELECT
        t.trace_rank,
        oi.order_id,
        coalesce(trim(oi.id), '') AS raw_item_id,
        coalesce(nullif(trim(oi.product_no), ''), 'UNKNOWN_PRODUCT') AS product_no,
        coalesce(trim(oi.product_name), '') AS product_name,
        cast(%s AS bigint) AS current_qty_milli,
                %s AS current_subtotal_milli,
                %s AS current_discount_milli,
                %s AS current_surcharge_milli,
                %s AS raw_item_included_tax_milli
    FROM top_tax_delta_trace_orders t
    JOIN order_items_parquet oi ON oi.order_id = t.order_id
    WHERE oi.t_open_date BETWEEN date '%s' AND date '%s'
      AND coalesce(nullif(trim(oi.product_no), ''), '') <> ''
),
top_item_group_meta AS (
    SELECT
        trace_rank,
        order_id,
        product_no,
        CASE
            WHEN count(*) = 1 THEN max(raw_item_id)
            ELSE concat('product_group:', product_no)
        END AS item_id,
        max_by(product_name, length(product_name)) AS product_name,
        count(*) AS raw_item_row_count,
        cast(sum(raw_item_included_tax_milli) AS bigint) AS raw_item_included_tax_milli
    FROM top_item_raw_trace
    GROUP BY 1, 2, 3
),
top_trace_item_counts AS (
    SELECT trace_rank, order_id, count(*) AS raw_item_row_count
    FROM top_item_raw_trace
    GROUP BY 1, 2
),
top_trace_grouped_item_counts AS (
    SELECT trace_rank, order_id, count(*) AS grouped_item_row_count
    FROM top_item_group_meta
    GROUP BY 1, 2
),
top_trace_payment_counts AS (
    SELECT trace_rank, order_id, count(*) AS raw_payment_row_count
    FROM top_payment_trace_raw
    GROUP BY 1, 2
),
top_trace_join_counts AS (
    SELECT t.trace_rank, le.order_id, count(*) AS final_joined_row_count
    FROM top_tax_delta_trace_orders t
    JOIN status_dedup_line_enriched le ON le.order_id = t.order_id
    GROUP BY 1, 2
),
top_trace_addition_counts AS (
    SELECT t.trace_rank, oa.order_id, count(*) AS raw_addition_row_count
    FROM top_tax_delta_trace_orders t
    JOIN order_additions_parquet oa ON oa.order_id = t.order_id
    WHERE oa.t_open_date BETWEEN date '%s' AND date '%s'
    GROUP BY 1, 2
),
order_header_trace AS (
    SELECT
        t.trace_rank,
        1 AS section_order,
        1 AS row_order,
        'order_header' AS section,
        cast(ob.business_date AS varchar) AS business_date,
        t.order_id,
        cast(ob.order_status AS varchar) AS status,
        ob.destination_raw AS destination,
        cast(ob.order_type_id AS varchar) AS normalized_order_type_id,
        ob.branch_id,
        cast(cast(round(ob.order_total * 1000) AS bigint) AS varchar) AS total_milli,
        cast(cast(round(ob.order_item_subtotal * 1000) AS bigint) AS varchar) AS item_subtotal_milli,
        cast(cast(round(ob.order_discount_subtotal * 1000) AS bigint) AS varchar) AS discount_milli,
        cast(cast(round(ob.order_surcharge_subtotal * 1000) AS bigint) AS varchar) AS surcharge_milli,
        cast(cast(round(ob.order_included_tax * 1000) AS bigint) AS varchar) AS included_tax_milli,
        cast(cast(round(ob.order_tax_subtotal * 1000) AS bigint) AS varchar) AS tax_subtotal_milli,
        coalesce(cast(ob.transaction_voided AS varchar), '') AS transaction_voided,
        coalesce(cast(ob.void_sale_period AS varchar), '') AS void_sale_period,
        '' AS payment_row_id,
        '' AS payment_name,
        '' AS payment_amount_milli,
        '' AS normalized_payment_type_id,
        '' AS final_payment_type_id,
        '' AS is_mixed,
        '' AS has_offsetting_payments,
        '' AS item_id,
        '' AS product_no,
        '' AS product_name,
        '' AS current_qty_milli,
        '' AS current_subtotal_milli,
        '' AS current_discount_milli,
        '' AS current_surcharge_milli,
        '' AS raw_item_included_tax_milli,
        '' AS allocation_denominator_milli,
        '' AS allocation_ratio,
        '' AS allocated_included_tax_milli,
        '' AS allocated_discount_milli,
        '' AS allocated_surcharge_milli,
        '' AS source_order_included_tax_milli,
        '' AS sum_allocated_included_tax_milli,
        '' AS delta_milli,
        '' AS source_order_net_milli,
        '' AS sum_allocated_net_milli,
        '' AS net_delta_milli,
        '' AS item_line_count,
        '' AS product_group_count,
        '' AS payment_row_count,
        '' AS normalized_payment_type_count,
        '' AS raw_item_row_count,
        '' AS grouped_item_row_count,
        '' AS raw_payment_row_count,
        '' AS payment_after_order_aggregation_row_count,
        '' AS final_joined_row_count,
        '' AS raw_addition_row_count,
        '' AS additions_after_order_aggregation_row_count,
        'order header 直接讀 status-aware status = 1 latest sales candidate' AS note
    FROM top_tax_delta_trace_orders t
    JOIN orders_sales_candidate ob ON ob.order_id = t.order_id AND ob.business_date = t.business_date
),
payment_trace_rows AS (
    SELECT
        p.trace_rank,
        2 AS section_order,
        row_number() OVER (PARTITION BY p.order_id ORDER BY p.payment_row_id, p.payment_name, p.payment_amount_milli) AS row_order,
        'payment_trace' AS section,
        cast(t.business_date AS varchar) AS business_date,
        p.order_id,
        cast(t.status AS varchar) AS status,
        t.destination,
        '' AS normalized_order_type_id,
        '' AS branch_id,
        '' AS total_milli,
        '' AS item_subtotal_milli,
        '' AS discount_milli,
        '' AS surcharge_milli,
        '' AS included_tax_milli,
        '' AS tax_subtotal_milli,
        '' AS transaction_voided,
        '' AS void_sale_period,
        p.payment_row_id,
        p.payment_name,
        cast(p.payment_amount_milli AS varchar) AS payment_amount_milli,
        cast(p.normalized_payment_type_id AS varchar) AS normalized_payment_type_id,
        cast(coalesce(s.final_payment_type_id, 0) AS varchar) AS final_payment_type_id,
        cast(coalesce(s.is_mixed, 0) AS varchar) AS is_mixed,
        cast(coalesce(s.has_offsetting_payments, 0) AS varchar) AS has_offsetting_payments,
        '' AS item_id,
        '' AS product_no,
        '' AS product_name,
        '' AS current_qty_milli,
        '' AS current_subtotal_milli,
        '' AS current_discount_milli,
        '' AS current_surcharge_milli,
        '' AS raw_item_included_tax_milli,
        '' AS allocation_denominator_milli,
        '' AS allocation_ratio,
        '' AS allocated_included_tax_milli,
        '' AS allocated_discount_milli,
        '' AS allocated_surcharge_milli,
        '' AS source_order_included_tax_milli,
        '' AS sum_allocated_included_tax_milli,
        '' AS delta_milli,
        '' AS source_order_net_milli,
        '' AS sum_allocated_net_milli,
        '' AS net_delta_milli,
        '' AS item_line_count,
        '' AS product_group_count,
        cast(coalesce(pc.raw_payment_row_count, 0) AS varchar) AS payment_row_count,
        cast(coalesce(s.normalized_payment_type_count, 0) AS varchar) AS normalized_payment_type_count,
        '' AS raw_item_row_count,
        '' AS grouped_item_row_count,
        cast(coalesce(pc.raw_payment_row_count, 0) AS varchar) AS raw_payment_row_count,
        cast(coalesce(s.payment_after_order_aggregation_row_count, 0) AS varchar) AS payment_after_order_aggregation_row_count,
        '' AS final_joined_row_count,
        '' AS raw_addition_row_count,
        '' AS additions_after_order_aggregation_row_count,
        CASE
            WHEN coalesce(s.is_mixed, 0) = 1 AND coalesce(s.has_offsetting_payments, 0) = 1 THEN 'mixed payment 且有正負沖銷'
            WHEN coalesce(s.is_mixed, 0) = 1 THEN 'mixed payment'
            WHEN coalesce(s.has_offsetting_payments, 0) = 1 THEN '同單有正負 payment rows'
            ELSE ''
        END AS note
    FROM top_payment_trace_raw p
    JOIN top_tax_delta_trace_orders t ON t.order_id = p.order_id
    LEFT JOIN top_payment_trace_summary s ON s.trace_rank = p.trace_rank AND s.order_id = p.order_id
    LEFT JOIN top_trace_payment_counts pc ON pc.trace_rank = p.trace_rank AND pc.order_id = p.order_id
),
item_allocation_trace_rows AS (
    SELECT
        t.trace_rank,
        3 AS section_order,
        row_number() OVER (PARTITION BY il.order_id ORDER BY il.product_no) AS row_order,
        'item_allocation_trace' AS section,
        cast(il.business_date AS varchar) AS business_date,
        il.order_id,
        cast(os.order_status AS varchar) AS status,
        os.destination_raw AS destination,
        cast(il.order_type_id AS varchar) AS normalized_order_type_id,
        os.branch_id,
        '' AS total_milli,
        '' AS item_subtotal_milli,
        '' AS discount_milli,
        '' AS surcharge_milli,
        '' AS included_tax_milli,
        '' AS tax_subtotal_milli,
        '' AS transaction_voided,
        '' AS void_sale_period,
        '' AS payment_row_id,
        '' AS payment_name,
        '' AS payment_amount_milli,
        '' AS normalized_payment_type_id,
        cast(le.payment_type_id AS varchar) AS final_payment_type_id,
        '' AS is_mixed,
        '' AS has_offsetting_payments,
        gm.item_id,
        il.product_no,
        gm.product_name,
        cast(il.qty_milli AS varchar) AS current_qty_milli,
        cast(cast(round(il.item_net_subtotal * 1000) AS bigint) AS varchar) AS current_subtotal_milli,
        cast(cast(round(il.item_discount_total * 1000) AS bigint) AS varchar) AS current_discount_milli,
        cast(cast(round(il.item_surcharge_total * 1000) AS bigint) AS varchar) AS current_surcharge_milli,
        cast(coalesce(gm.raw_item_included_tax_milli, 0) AS varchar) AS raw_item_included_tax_milli,
        cast(os.allocation_denominator_milli AS varchar) AS allocation_denominator_milli,
        cast(round(CASE
            WHEN abs(of.order_net_sales) <= 0.000001 THEN 0.0
            ELSE (il.item_gross_subtotal - (il.item_discount_total + le.addition_discount_allocated) + (il.item_surcharge_total + le.addition_surcharge_allocated)) / of.order_net_sales
        END, 6) AS varchar) AS allocation_ratio,
        cast(lcm.included_tax_milli AS varchar) AS allocated_included_tax_milli,
        cast(cast(round(le.addition_discount_allocated * 1000) AS bigint) AS varchar) AS allocated_discount_milli,
        cast(cast(round(le.addition_surcharge_allocated * 1000) AS bigint) AS varchar) AS allocated_surcharge_milli,
        '' AS source_order_included_tax_milli,
        '' AS sum_allocated_included_tax_milli,
        '' AS delta_milli,
        '' AS source_order_net_milli,
        '' AS sum_allocated_net_milli,
        '' AS net_delta_milli,
        '' AS item_line_count,
        '' AS product_group_count,
        '' AS payment_row_count,
        '' AS normalized_payment_type_count,
        cast(coalesce(gm.raw_item_row_count, 0) AS varchar) AS raw_item_row_count,
        '' AS grouped_item_row_count,
        '' AS raw_payment_row_count,
        '' AS payment_after_order_aggregation_row_count,
        '' AS final_joined_row_count,
        '' AS raw_addition_row_count,
        '' AS additions_after_order_aggregation_row_count,
        CASE
            WHEN coalesce(gm.raw_item_row_count, 0) > 1 THEN '目前 allocation 以 product_group 粒度進行；此列合併多筆 raw item rows'
            ELSE '單一 raw item row'
        END AS note
    FROM top_tax_delta_trace_orders t
        JOIN status_dedup_item_lines il ON il.order_id = t.order_id
        JOIN status_dedup_line_enriched le
      ON le.order_id = il.order_id
     AND le.business_date = il.business_date
     AND le.hour_of_day = il.hour_of_day
     AND le.branch_id = il.branch_id
     AND le.product_no = il.product_no
     AND le.order_type_id = il.order_type_id
        JOIN status_dedup_line_component_milli lcm
      ON lcm.order_id = il.order_id
     AND lcm.business_date = il.business_date
     AND lcm.hour_of_day = il.hour_of_day
     AND lcm.branch_id = il.branch_id
     AND lcm.product_no = il.product_no
     AND lcm.order_type_id = il.order_type_id
     AND lcm.payment_type_id = le.payment_type_id
        JOIN status_dedup_order_scope os ON os.order_id = il.order_id
        JOIN status_dedup_order_financials of ON of.order_id = il.order_id
    LEFT JOIN top_item_group_meta gm ON gm.trace_rank = t.trace_rank AND gm.order_id = il.order_id AND gm.product_no = il.product_no
),
allocation_summary_rows AS (
    SELECT
        t.trace_rank,
        4 AS section_order,
        1 AS row_order,
        'allocation_summary' AS section,
        cast(t.business_date AS varchar) AS business_date,
        t.order_id,
        cast(t.status AS varchar) AS status,
        t.destination,
        '' AS normalized_order_type_id,
        '' AS branch_id,
        '' AS total_milli,
        '' AS item_subtotal_milli,
        '' AS discount_milli,
        '' AS surcharge_milli,
        '' AS included_tax_milli,
        '' AS tax_subtotal_milli,
        '' AS transaction_voided,
        '' AS void_sale_period,
        '' AS payment_row_id,
        '' AS payment_name,
        '' AS payment_amount_milli,
        '' AS normalized_payment_type_id,
        cast(coalesce(ps.final_payment_type_id, 0) AS varchar) AS final_payment_type_id,
        cast(coalesce(ps.is_mixed, 0) AS varchar) AS is_mixed,
        cast(coalesce(ps.has_offsetting_payments, 0) AS varchar) AS has_offsetting_payments,
        '' AS item_id,
        '' AS product_no,
        '' AS product_name,
        '' AS current_qty_milli,
        '' AS current_subtotal_milli,
        '' AS current_discount_milli,
        '' AS current_surcharge_milli,
        '' AS raw_item_included_tax_milli,
        cast(os.allocation_denominator_milli AS varchar) AS allocation_denominator_milli,
        '' AS allocation_ratio,
        '' AS allocated_included_tax_milli,
        '' AS allocated_discount_milli,
        '' AS allocated_surcharge_milli,
        cast(os.source_included_tax_milli AS varchar) AS source_order_included_tax_milli,
        cast(os.allocated_included_tax_milli AS varchar) AS sum_allocated_included_tax_milli,
        cast(os.allocated_included_tax_milli - os.source_included_tax_milli AS varchar) AS delta_milli,
        cast(os.source_net_sales_milli AS varchar) AS source_order_net_milli,
        cast(os.allocated_net_sales_milli AS varchar) AS sum_allocated_net_milli,
        cast(os.allocated_net_sales_milli - os.source_net_sales_milli AS varchar) AS net_delta_milli,
        cast(coalesce(ic.raw_item_row_count, 0) AS varchar) AS item_line_count,
        cast(coalesce(gic.grouped_item_row_count, 0) AS varchar) AS product_group_count,
        cast(coalesce(pc.raw_payment_row_count, 0) AS varchar) AS payment_row_count,
        cast(coalesce(ps.normalized_payment_type_count, 0) AS varchar) AS normalized_payment_type_count,
        cast(coalesce(ic.raw_item_row_count, 0) AS varchar) AS raw_item_row_count,
        cast(coalesce(gic.grouped_item_row_count, 0) AS varchar) AS grouped_item_row_count,
        cast(coalesce(pc.raw_payment_row_count, 0) AS varchar) AS raw_payment_row_count,
        cast(coalesce(ps.payment_after_order_aggregation_row_count, 0) AS varchar) AS payment_after_order_aggregation_row_count,
        cast(coalesce(jc.final_joined_row_count, 0) AS varchar) AS final_joined_row_count,
        cast(coalesce(ac.raw_addition_row_count, 0) AS varchar) AS raw_addition_row_count,
        cast(CASE WHEN coalesce(ac.raw_addition_row_count, 0) > 0 THEN 1 ELSE 0 END AS varchar) AS additions_after_order_aggregation_row_count,
        'order-level allocation summary；用來直接比對 source vs allocated totals' AS note
    FROM top_tax_delta_trace_orders t
    JOIN status_dedup_order_scope os ON os.order_id = t.order_id
    LEFT JOIN top_payment_trace_summary ps ON ps.trace_rank = t.trace_rank AND ps.order_id = t.order_id
    LEFT JOIN top_trace_item_counts ic ON ic.trace_rank = t.trace_rank AND ic.order_id = t.order_id
    LEFT JOIN top_trace_grouped_item_counts gic ON gic.trace_rank = t.trace_rank AND gic.order_id = t.order_id
    LEFT JOIN top_trace_payment_counts pc ON pc.trace_rank = t.trace_rank AND pc.order_id = t.order_id
    LEFT JOIN top_trace_join_counts jc ON jc.trace_rank = t.trace_rank AND jc.order_id = t.order_id
    LEFT JOIN top_trace_addition_counts ac ON ac.trace_rank = t.trace_rank AND ac.order_id = t.order_id
),
join_duplication_check_rows AS (
    SELECT
        t.trace_rank,
        5 AS section_order,
        1 AS row_order,
        'join_duplication_check' AS section,
        cast(t.business_date AS varchar) AS business_date,
        t.order_id,
        cast(t.status AS varchar) AS status,
        t.destination,
        '' AS normalized_order_type_id,
        '' AS branch_id,
        '' AS total_milli,
        '' AS item_subtotal_milli,
        '' AS discount_milli,
        '' AS surcharge_milli,
        '' AS included_tax_milli,
        '' AS tax_subtotal_milli,
        '' AS transaction_voided,
        '' AS void_sale_period,
        '' AS payment_row_id,
        '' AS payment_name,
        '' AS payment_amount_milli,
        '' AS normalized_payment_type_id,
        '' AS final_payment_type_id,
        '' AS is_mixed,
        '' AS has_offsetting_payments,
        '' AS item_id,
        '' AS product_no,
        '' AS product_name,
        '' AS current_qty_milli,
        '' AS current_subtotal_milli,
        '' AS current_discount_milli,
        '' AS current_surcharge_milli,
        '' AS raw_item_included_tax_milli,
        '' AS allocation_denominator_milli,
        '' AS allocation_ratio,
        '' AS allocated_included_tax_milli,
        '' AS allocated_discount_milli,
        '' AS allocated_surcharge_milli,
        '' AS source_order_included_tax_milli,
        '' AS sum_allocated_included_tax_milli,
        '' AS delta_milli,
        '' AS source_order_net_milli,
        '' AS sum_allocated_net_milli,
        '' AS net_delta_milli,
        '' AS item_line_count,
        '' AS product_group_count,
        '' AS payment_row_count,
        '' AS normalized_payment_type_count,
        cast(coalesce(ic.raw_item_row_count, 0) AS varchar) AS raw_item_row_count,
        cast(coalesce(gic.grouped_item_row_count, 0) AS varchar) AS grouped_item_row_count,
        cast(coalesce(pc.raw_payment_row_count, 0) AS varchar) AS raw_payment_row_count,
        cast(coalesce(ps.payment_after_order_aggregation_row_count, 0) AS varchar) AS payment_after_order_aggregation_row_count,
        cast(coalesce(jc.final_joined_row_count, 0) AS varchar) AS final_joined_row_count,
        cast(coalesce(ac.raw_addition_row_count, 0) AS varchar) AS raw_addition_row_count,
        cast(CASE WHEN coalesce(ac.raw_addition_row_count, 0) > 0 THEN 1 ELSE 0 END AS varchar) AS additions_after_order_aggregation_row_count,
        CASE
            WHEN coalesce(jc.final_joined_row_count, 0) > coalesce(gic.grouped_item_row_count, 0) THEN 'final joined rows > grouped item rows，疑似 join amplification'
            ELSE 'final joined rows = grouped item rows；payment/additions 經 order-level aggregation 後未見 join amplification'
        END AS note
    FROM top_tax_delta_trace_orders t
    LEFT JOIN top_payment_trace_summary ps ON ps.trace_rank = t.trace_rank AND ps.order_id = t.order_id
    LEFT JOIN top_trace_item_counts ic ON ic.trace_rank = t.trace_rank AND ic.order_id = t.order_id
    LEFT JOIN top_trace_grouped_item_counts gic ON gic.trace_rank = t.trace_rank AND gic.order_id = t.order_id
    LEFT JOIN top_trace_payment_counts pc ON pc.trace_rank = t.trace_rank AND pc.order_id = t.order_id
    LEFT JOIN top_trace_join_counts jc ON jc.trace_rank = t.trace_rank AND jc.order_id = t.order_id
    LEFT JOIN top_trace_addition_counts ac ON ac.trace_rank = t.trace_rank AND ac.order_id = t.order_id
)
SELECT
    cast(trace_rank AS varchar) AS trace_rank,
    section,
    cast(row_order AS varchar) AS row_order,
    business_date,
    order_id,
    status,
    destination,
    normalized_order_type_id,
    branch_id,
    cast(total_milli AS varchar) AS total_milli,
    cast(item_subtotal_milli AS varchar) AS item_subtotal_milli,
    cast(discount_milli AS varchar) AS discount_milli,
    cast(surcharge_milli AS varchar) AS surcharge_milli,
    cast(included_tax_milli AS varchar) AS included_tax_milli,
    cast(tax_subtotal_milli AS varchar) AS tax_subtotal_milli,
    transaction_voided,
    void_sale_period,
    payment_row_id,
    payment_name,
    payment_amount_milli,
    normalized_payment_type_id,
    final_payment_type_id,
    is_mixed,
    has_offsetting_payments,
    item_id,
    product_no,
    product_name,
    current_qty_milli,
    current_subtotal_milli,
    current_discount_milli,
    current_surcharge_milli,
    raw_item_included_tax_milli,
    allocation_denominator_milli,
    allocation_ratio,
    allocated_included_tax_milli,
    allocated_discount_milli,
    allocated_surcharge_milli,
    source_order_included_tax_milli,
    sum_allocated_included_tax_milli,
    delta_milli,
    source_order_net_milli,
    sum_allocated_net_milli,
    net_delta_milli,
    item_line_count,
    product_group_count,
    payment_row_count,
    normalized_payment_type_count,
    raw_item_row_count,
    grouped_item_row_count,
    raw_payment_row_count,
    payment_after_order_aggregation_row_count,
    final_joined_row_count,
    raw_addition_row_count,
    additions_after_order_aggregation_row_count,
    note
FROM (
    SELECT * FROM order_header_trace
    UNION ALL
    SELECT * FROM payment_trace_rows
    UNION ALL
    SELECT * FROM item_allocation_trace_rows
    UNION ALL
    SELECT * FROM allocation_summary_rows
    UNION ALL
    SELECT * FROM join_duplication_check_rows
) trace_rows
ORDER BY trace_rank, section_order, row_order`,
		buildPreviewCTE(window),
		topTaxDeltaTraceLimit,
		paymentTypeIDSQL("p.name"),
		safeMilliBigintExpr(safeDoubleExpr("p.amount")),
		startDate,
		endDate,
		qtyMilliExpr("oi.current_qty"),
		safeMilliBigintExpr(safeDoubleExpr("oi.current_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("oi.current_discount")),
		safeMilliBigintExpr(positiveDoubleExpr("oi.current_surcharge")),
		safeMilliBigintExpr(safeDoubleExpr("oi.included_tax")),
		startDate,
		endDate,
		startDate,
		endDate,
	)
}

func BuildDuplicateOrderTraceSQL(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`%s,
duplicate_trace_orders AS (
    SELECT
        row_number() OVER (ORDER BY abs(delta_milli) DESC, business_date, order_id) AS trace_rank,
        business_date AS t_open_date,
        order_id,
        status AS top_sample_status,
        destination AS top_sample_destination
    FROM status_dedup_top_tax_delta_sample
    ORDER BY abs(delta_milli) DESC, business_date, order_id
    LIMIT %d
),
duplicate_order_rows AS (
    SELECT
        dto.trace_rank,
        row_number() OVER (
            PARTITION BY dto.order_id, dto.t_open_date
            ORDER BY o.transaction_submitted DESC NULLS LAST, o.modified DESC NULLS LAST, o.created DESC NULLS LAST, o.transaction_created DESC NULLS LAST, o.sequence DESC NULLS LAST
        ) AS raw_row_rank,
        count(*) OVER (PARTITION BY dto.order_id, dto.t_open_date) AS duplicate_row_count,
        cast(o.t_open_date AS varchar) AS t_open_date,
        o.id AS order_id,
        cast(o.status AS varchar) AS status,
        coalesce(trim(o.destination), '') AS destination,
        coalesce(nullif(trim(o.branch_id), ''), 'UNKNOWN_BRANCH') AS branch_id,
        coalesce(trim(o.branch), '') AS branch,
        %s AS total_milli,
        %s AS item_subtotal_milli,
        %s AS discount_subtotal_milli,
        %s AS payment_subtotal_milli,
        %s AS included_tax_subtotal_milli,
        %s AS tax_subtotal_milli,
        %s AS item_surcharge_subtotal_milli,
        %s AS trans_surcharge_subtotal_milli,
        coalesce(cast(o.transaction_created AS varchar), '') AS transaction_created,
        coalesce(cast(o.transaction_submitted AS varchar), '') AS transaction_submitted,
        coalesce(cast(o.transaction_voided AS varchar), '') AS transaction_voided,
        coalesce(cast(o.void_sale_period AS varchar), '') AS void_sale_period,
        coalesce(cast(o.created AS varchar), '') AS created,
        coalesce(cast(o.modified AS varchar), '') AS modified,
        coalesce(trim(o.sequence), '') AS sequence,
        coalesce(cast(o.sale_period AS varchar), '') AS sale_period,
        coalesce(cast(o.shift_number AS varchar), '') AS shift_number,
        cast(dto.top_sample_status AS varchar) AS top_sample_status,
        dto.top_sample_destination
    FROM duplicate_trace_orders dto
    JOIN orders_parquet o ON o.id = dto.order_id AND o.t_open_date = dto.t_open_date
    WHERE o.t_open_date BETWEEN date '%s' AND date '%s'
)
SELECT
    cast(trace_rank AS varchar) AS trace_rank,
    cast(raw_row_rank AS varchar) AS raw_row_rank,
    cast(duplicate_row_count AS varchar) AS duplicate_row_count,
    t_open_date,
    order_id,
    status,
    destination,
    branch_id,
    branch,
    cast(total_milli AS varchar) AS total_milli,
    cast(item_subtotal_milli AS varchar) AS item_subtotal_milli,
    cast(discount_subtotal_milli AS varchar) AS discount_subtotal_milli,
    cast(payment_subtotal_milli AS varchar) AS payment_subtotal_milli,
    cast(included_tax_subtotal_milli AS varchar) AS included_tax_subtotal_milli,
    cast(tax_subtotal_milli AS varchar) AS tax_subtotal_milli,
    cast(item_surcharge_subtotal_milli AS varchar) AS item_surcharge_subtotal_milli,
    cast(trans_surcharge_subtotal_milli AS varchar) AS trans_surcharge_subtotal_milli,
    transaction_created,
    transaction_submitted,
    transaction_voided,
    void_sale_period,
    created,
    modified,
    sequence,
    sale_period,
    shift_number,
    top_sample_status,
    top_sample_destination
FROM duplicate_order_rows
ORDER BY trace_rank, raw_row_rank`,
		buildPreviewCTE(window),
		topTaxDeltaTraceLimit,
		safeMilliBigintExpr(safeDoubleExpr("o.total")),
		safeMilliBigintExpr(safeDoubleExpr("o.item_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("o.discount_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("o.payment_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("o.included_tax_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("o.tax_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("o.item_surcharge_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("o.trans_surcharge_subtotal")),
		startDate,
		endDate,
	)
}

func BuildDuplicateOrderSummarySQL(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`
WITH order_duplicate_stats AS (
    SELECT
        t_open_date,
        id AS order_id,
        count(*) AS raw_row_count,
        count_if(coalesce(trim(destination), '') = '外送') AS delivery_row_count,
        count(distinct cast(coalesce(status, -1) AS varchar)) AS distinct_status_count,
        count(distinct %s) AS distinct_total_count,
        count(distinct %s) AS distinct_included_tax_count,
        count(distinct concat(
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar)
        )) AS distinct_financial_signature_count,
        count(distinct coalesce(cast(modified AS varchar), '')) AS distinct_modified_count,
        count(distinct coalesce(cast(created AS varchar), '')) AS distinct_created_count,
        count(distinct coalesce(cast(transaction_submitted AS varchar), '')) AS distinct_transaction_submitted_count
    FROM orders_parquet
    WHERE t_open_date BETWEEN date '%s' AND date '%s'
    GROUP BY 1, 2
),
duplicate_only AS (
    SELECT *
    FROM order_duplicate_stats
    WHERE raw_row_count > 1
),
duplicate_order_summary AS (
    SELECT 1 AS sort_order, 'order_id_open_date_total_count' AS metric, cast(count(*) AS bigint) AS metric_value, '日期區間內 orders_parquet 的 order_id + t_open_date 唯一鍵數' AS note
    FROM order_duplicate_stats
    UNION ALL
    SELECT 2, 'duplicate_order_id_open_date_count', cast(count(*) AS bigint), 'raw_row_count > 1 的 order_id + t_open_date 數量'
    FROM duplicate_only
    UNION ALL
    SELECT 3, 'duplicate_row_total_count', cast(coalesce(sum(raw_row_count), 0) AS bigint), '所有重複鍵對應的原始列總筆數'
    FROM duplicate_only
    UNION ALL
    SELECT 4, 'duplicate_extra_row_count', cast(coalesce(sum(raw_row_count - 1), 0) AS bigint), '扣掉每個唯一鍵第一筆後，真正多出來的 duplicate rows'
    FROM duplicate_only
    UNION ALL
    SELECT 5, 'duplicate_delivery_only_count', cast(count(*) AS bigint), '重複鍵中，全部 raw rows 都是 destination = 外送 的數量'
    FROM duplicate_only
    WHERE delivery_row_count = raw_row_count
    UNION ALL
    SELECT 6, 'duplicate_status_mismatch_count', cast(count(*) AS bigint), '重複鍵中，status 至少有兩種值'
    FROM duplicate_only
    WHERE distinct_status_count > 1
    UNION ALL
    SELECT 7, 'duplicate_financial_mismatch_count', cast(count(*) AS bigint), '重複鍵中，financial 欄位簽章不一致'
    FROM duplicate_only
    WHERE distinct_financial_signature_count > 1
    UNION ALL
    SELECT 8, 'duplicate_included_tax_mismatch_count', cast(count(*) AS bigint), '重複鍵中，included_tax_subtotal 不一致'
    FROM duplicate_only
    WHERE distinct_included_tax_count > 1
    UNION ALL
    SELECT 9, 'duplicate_total_mismatch_count', cast(count(*) AS bigint), '重複鍵中，total 不一致'
    FROM duplicate_only
    WHERE distinct_total_count > 1
    UNION ALL
    SELECT 10, 'duplicate_with_distinct_modified_count', cast(count(*) AS bigint), '重複鍵中，modified 存在不同值，可作為版本排序訊號'
    FROM duplicate_only
    WHERE distinct_modified_count > 1
    UNION ALL
    SELECT 11, 'duplicate_with_distinct_created_count', cast(count(*) AS bigint), '重複鍵中，created 存在不同值，可作為版本排序訊號'
    FROM duplicate_only
    WHERE distinct_created_count > 1
    UNION ALL
    SELECT 12, 'duplicate_with_distinct_transaction_submitted_count', cast(count(*) AS bigint), '重複鍵中，transaction_submitted 存在不同值，可作為版本排序訊號'
    FROM duplicate_only
    WHERE distinct_transaction_submitted_count > 1
    UNION ALL
    SELECT 13, 'duplicate_with_any_dedup_sort_signal_count', cast(count(*) AS bigint), '重複鍵中，modified / created / transaction_submitted 任一欄位可提供排序訊號'
    FROM duplicate_only
    WHERE distinct_modified_count > 1 OR distinct_created_count > 1 OR distinct_transaction_submitted_count > 1
)
SELECT
    metric,
    cast(metric_value AS varchar) AS metric_value,
    note
FROM duplicate_order_summary
ORDER BY sort_order`,
		safeMilliBigintExpr(safeDoubleExpr("total")),
		safeMilliBigintExpr(safeDoubleExpr("included_tax_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("total")),
		safeMilliBigintExpr(safeDoubleExpr("item_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("discount_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("payment_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("included_tax_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("tax_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("item_surcharge_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("trans_surcharge_subtotal")),
		startDate,
		endDate,
	)
}

func BuildStatusDedupCandidateSummarySQL(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`%s,
order_duplicate_stats AS (
    SELECT
        t_open_date AS business_date,
        id AS order_id,
        count(*) AS raw_row_count,
        count(distinct cast(coalesce(status, -1) AS varchar)) AS distinct_status_count,
        count(distinct %s) AS distinct_total_count,
        count(distinct %s) AS distinct_included_tax_count,
        count(distinct concat(
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar), '|',
            cast(%s AS varchar)
        )) AS distinct_financial_signature_count
    FROM orders_parquet
    WHERE t_open_date BETWEEN date '%s' AND date '%s'
    GROUP BY 1, 2
),
duplicate_only AS (
    SELECT *
    FROM order_duplicate_stats
    WHERE raw_row_count > 1
),
status_dedup_candidate_summary AS (
    SELECT 1 AS sort_order, 'raw_order_rows' AS metric, cast(count(*) AS bigint) AS metric_value, '日期區間內 orders_parquet 原始列數' AS note
    FROM orders_ranked_by_status
    UNION ALL
    SELECT 2, 'raw_order_keys', cast(count(*) AS bigint), '日期區間內 order_id + t_open_date 唯一鍵數'
    FROM order_duplicate_stats
    UNION ALL
    SELECT 3, 'duplicate_order_keys', cast(count(*) AS bigint), 'raw_row_count > 1 的 order_id + t_open_date 數量'
    FROM duplicate_only
    UNION ALL
    SELECT 4, 'duplicate_raw_rows', cast(coalesce(sum(raw_row_count), 0) AS bigint), 'duplicate keys 對應的原始列總筆數'
    FROM duplicate_only
    UNION ALL
    SELECT 5, 'status_mismatch_duplicate_keys', cast(count(*) AS bigint), 'duplicate keys 中 status 至少出現兩種值'
    FROM duplicate_only
    WHERE distinct_status_count > 1
    UNION ALL
    SELECT 6, 'financial_mismatch_duplicate_keys', cast(count(*) AS bigint), 'duplicate keys 中 financial signature 不一致'
    FROM duplicate_only
    WHERE distinct_financial_signature_count > 1
    UNION ALL
    SELECT 7, 'included_tax_mismatch_duplicate_keys', cast(count(*) AS bigint), 'duplicate keys 中 included_tax_subtotal 不一致'
    FROM duplicate_only
    WHERE distinct_included_tax_count > 1
    UNION ALL
    SELECT 8, 'total_mismatch_duplicate_keys', cast(count(*) AS bigint), 'duplicate keys 中 total 不一致'
    FROM duplicate_only
    WHERE distinct_total_count > 1
    UNION ALL
    SELECT 9, 'sales_candidate_order_keys', cast(count(*) AS bigint), 'status-aware latest candidate 中 status = 1 的 key 數；正式 preview sales path 僅吃這批來源'
    FROM orders_sales_candidate
    UNION ALL
    SELECT 10, 'void_candidate_order_keys', cast(count(*) AS bigint), 'status-aware latest candidate 中 status = -2 的 key 數；保留供 void preview/debug'
    FROM orders_void_candidate
    UNION ALL
    SELECT 11, 'excluded_status_minus_1_order_keys', cast(count(*) AS bigint), 'status-aware latest candidate 中 status = -1 的排除 key 數；只進 debug'
    FROM orders_excluded_candidate
    WHERE order_status = -1
    UNION ALL
    SELECT 12, 'excluded_status_2_order_keys', cast(count(*) AS bigint), 'status-aware latest candidate 中 status = 2 的排除 key 數；只進 debug'
    FROM orders_excluded_candidate
    WHERE order_status = 2
    UNION ALL
    SELECT 13, 'orders_with_both_status_1_and_2', cast(count(*) AS bigint), '同一個 order_id + t_open_date 同時存在 status = 1 與 2'
    FROM orders_status_presence
    WHERE has_status_1 = 1 AND has_status_2 = 1
    UNION ALL
    SELECT 14, 'orders_with_both_status_1_and_minus_2', cast(count(*) AS bigint), '同一個 order_id + t_open_date 同時存在 status = 1 與 -2'
    FROM orders_status_presence
    WHERE has_status_1 = 1 AND has_status_minus_2 = 1
    UNION ALL
    SELECT 15, 'orders_with_only_status_2', cast(count(*) AS bigint), '只存在 status = 2，且沒有 1 / -1 / -2'
    FROM orders_status_presence
    WHERE has_status_2 = 1 AND has_status_1 = 0 AND has_status_minus_1 = 0 AND has_status_minus_2 = 0
    UNION ALL
    SELECT 16, 'orders_with_only_status_minus_1', cast(count(*) AS bigint), '只存在 status = -1，且沒有 1 / 2 / -2'
    FROM orders_status_presence
    WHERE has_status_minus_1 = 1 AND has_status_1 = 0 AND has_status_2 = 0 AND has_status_minus_2 = 0
)
SELECT
    metric,
    cast(metric_value AS varchar) AS metric_value,
    note
FROM status_dedup_candidate_summary
ORDER BY sort_order`,
		buildPreviewCTE(window),
		safeMilliBigintExpr(safeDoubleExpr("total")),
		safeMilliBigintExpr(safeDoubleExpr("included_tax_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("total")),
		safeMilliBigintExpr(safeDoubleExpr("item_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("discount_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("payment_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("included_tax_subtotal")),
		safeMilliBigintExpr(safeDoubleExpr("tax_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("item_surcharge_subtotal")),
		safeMilliBigintExpr(positiveDoubleExpr("trans_surcharge_subtotal")),
		startDate,
		endDate,
	)
}

func BuildStatusDedupReconciliationComparisonSQL(window QueryWindow) string {
	return fmt.Sprintf(`%s,
current_preview_summary AS (
    SELECT
        coalesce(sum(tax_milli), 0) AS current_preview_tax_milli,
        coalesce(sum(net_sales_milli), 0) AS current_preview_net_milli,
        coalesce(sum(sales_ex_tax_milli), 0) AS current_preview_sales_ex_tax_milli
    FROM final_aggregation
),
status_dedup_preview_summary AS (
    SELECT
        coalesce(sum(tax_milli), 0) AS status_dedup_preview_tax_milli,
        coalesce(sum(net_sales_milli), 0) AS status_dedup_preview_net_milli,
        coalesce(sum(sales_ex_tax_milli), 0) AS status_dedup_preview_sales_ex_tax_milli
    FROM status_dedup_final_aggregation
),
source_status_1_summary AS (
    SELECT
        cast(round(coalesce(sum(order_tax_total), 0.0) * 1000) AS bigint) AS source_status_1_tax_milli,
        cast(round(coalesce(sum(order_net_sales), 0.0) * 1000) AS bigint) AS source_status_1_net_milli,
        cast(round(coalesce(sum(order_sales_ex_tax), 0.0) * 1000) AS bigint) AS source_status_1_sales_ex_tax_milli
    FROM status_dedup_order_financials
    WHERE item_count > 0
),
status_dedup_reconciliation_comparison AS (
    SELECT 1 AS sort_order, 'current_preview_tax_milli' AS metric, cps.current_preview_tax_milli AS metric_value, 'legacy 非 dedup preview tax 總額；保留作為 regression comparison' AS note
    FROM current_preview_summary cps
    UNION ALL
    SELECT 2, 'status_dedup_preview_tax_milli', sdps.status_dedup_preview_tax_milli, '正式 preview sales path tax 總額；已切到 status-aware dedup sales source'
    FROM status_dedup_preview_summary sdps
    UNION ALL
    SELECT 3, 'source_status_1_tax_milli', sss.source_status_1_tax_milli, 'source scope = status = 1 latest candidate 且有 item 的稅額總和'
    FROM source_status_1_summary sss
    UNION ALL
    SELECT 4, 'current_delta_milli', cps.current_preview_tax_milli - sss.source_status_1_tax_milli, 'current preview tax - source status = 1 tax'
    FROM current_preview_summary cps
    CROSS JOIN source_status_1_summary sss
    UNION ALL
    SELECT 5, 'status_dedup_delta_milli', sdps.status_dedup_preview_tax_milli - sss.source_status_1_tax_milli, 'status-aware preview tax - source status = 1 tax'
    FROM status_dedup_preview_summary sdps
    CROSS JOIN source_status_1_summary sss
    UNION ALL
    SELECT 6, 'current_preview_net_milli', cps.current_preview_net_milli, 'legacy 非 dedup preview net 總額；保留作為 regression comparison'
    FROM current_preview_summary cps
    UNION ALL
    SELECT 7, 'status_dedup_preview_net_milli', sdps.status_dedup_preview_net_milli, '正式 preview sales path net 總額；已切到 status-aware dedup sales source'
    FROM status_dedup_preview_summary sdps
    UNION ALL
    SELECT 8, 'source_status_1_net_milli', sss.source_status_1_net_milli, 'source scope = status = 1 latest candidate 且有 item 的 net 總和'
    FROM source_status_1_summary sss
    UNION ALL
    SELECT 9, 'current_net_delta_milli', cps.current_preview_net_milli - sss.source_status_1_net_milli, 'current preview net - source status = 1 net'
    FROM current_preview_summary cps
    CROSS JOIN source_status_1_summary sss
    UNION ALL
    SELECT 10, 'status_dedup_net_delta_milli', sdps.status_dedup_preview_net_milli - sss.source_status_1_net_milli, 'status-aware preview net - source status = 1 net'
    FROM status_dedup_preview_summary sdps
    CROSS JOIN source_status_1_summary sss
    UNION ALL
    SELECT 11, 'current_preview_sales_ex_tax_milli', cps.current_preview_sales_ex_tax_milli, 'legacy 非 dedup preview sales_ex_tax 總額；保留作為 regression comparison'
    FROM current_preview_summary cps
    UNION ALL
    SELECT 12, 'status_dedup_preview_sales_ex_tax_milli', sdps.status_dedup_preview_sales_ex_tax_milli, '正式 preview sales path sales_ex_tax 總額；已切到 status-aware dedup sales source'
    FROM status_dedup_preview_summary sdps
    UNION ALL
    SELECT 13, 'source_status_1_sales_ex_tax_milli', sss.source_status_1_sales_ex_tax_milli, 'source scope = status = 1 latest candidate 且有 item 的 sales_ex_tax 總和'
    FROM source_status_1_summary sss
    UNION ALL
    SELECT 14, 'current_sales_ex_tax_delta_milli', cps.current_preview_sales_ex_tax_milli - sss.source_status_1_sales_ex_tax_milli, 'current preview sales_ex_tax - source status = 1 sales_ex_tax'
    FROM current_preview_summary cps
    CROSS JOIN source_status_1_summary sss
    UNION ALL
    SELECT 15, 'status_dedup_sales_ex_tax_delta_milli', sdps.status_dedup_preview_sales_ex_tax_milli - sss.source_status_1_sales_ex_tax_milli, 'status-aware preview sales_ex_tax - source status = 1 sales_ex_tax'
    FROM status_dedup_preview_summary sdps
    CROSS JOIN source_status_1_summary sss
)
SELECT
    metric,
    cast(metric_value AS varchar) AS metric_value,
    note
FROM status_dedup_reconciliation_comparison
ORDER BY sort_order`, buildPreviewCTE(window))
}

func BuildTopTaxDeltaBeforeAfterStatusDedupSQL(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`%s,
current_top_tax_delta_orders AS (
    SELECT
        business_date AS t_open_date,
        order_id,
        max(abs(delta_milli)) AS current_max_abs_delta_milli
    FROM rounding_order_debug
    GROUP BY 1, 2
    ORDER BY current_max_abs_delta_milli DESC, t_open_date, order_id
    LIMIT %d
),
raw_order_rollup AS (
    SELECT
        o.t_open_date,
        o.id AS order_id,
        count(*) AS raw_order_row_count,
        array_join(array_sort(array_agg(distinct cast(coalesce(o.status, -1) AS varchar))), ',') AS raw_status_list
    FROM orders_parquet o
    JOIN current_top_tax_delta_orders t ON t.order_id = o.id AND t.t_open_date = o.t_open_date
    WHERE o.t_open_date BETWEEN date '%s' AND date '%s'
    GROUP BY 1, 2
),
current_order_allocated AS (
    SELECT
        business_date AS t_open_date,
        order_id,
        max(allocated_included_tax_milli) AS current_allocated_included_tax_milli,
        max(allocated_net_sales_milli) AS current_allocated_net_milli
    FROM order_scope
    GROUP BY 1, 2
),
status_dedup_before_after AS (
    SELECT
        t.order_id,
        cast(t.t_open_date AS varchar) AS t_open_date,
        coalesce(r.raw_order_row_count, 0) AS raw_order_row_count,
        coalesce(r.raw_status_list, '') AS raw_status_list,
        coalesce(cast(osc.order_status AS varchar), '') AS selected_sales_status,
        coalesce(cast(osc.transaction_submitted AS varchar), '') AS selected_sales_transaction_submitted,
        coalesce(cast(osc.modified AS varchar), '') AS selected_sales_modified,
        coalesce(osc.payment_subtotal_milli, 0) AS selected_sales_payment_subtotal_milli,
        coalesce(sdos.source_included_tax_milli, 0) AS source_status_1_included_tax_milli,
        coalesce(coa.current_allocated_included_tax_milli, 0) AS current_allocated_included_tax_milli,
        coalesce(sdos.allocated_included_tax_milli, 0) AS status_dedup_allocated_included_tax_milli,
        coalesce(coa.current_allocated_included_tax_milli, 0) - coalesce(sdos.source_included_tax_milli, 0) AS current_delta_milli,
        coalesce(sdos.allocated_included_tax_milli, 0) - coalesce(sdos.source_included_tax_milli, 0) AS status_dedup_delta_milli,
        coalesce(sdos.source_net_sales_milli, 0) AS source_status_1_net_milli,
        coalesce(coa.current_allocated_net_milli, 0) AS current_allocated_net_milli,
        coalesce(sdos.allocated_net_sales_milli, 0) AS status_dedup_allocated_net_milli,
        coalesce(coa.current_allocated_net_milli, 0) - coalesce(sdos.source_net_sales_milli, 0) AS current_net_delta_milli,
        coalesce(sdos.allocated_net_sales_milli, 0) - coalesce(sdos.source_net_sales_milli, 0) AS status_dedup_net_delta_milli,
        t.current_max_abs_delta_milli
    FROM current_top_tax_delta_orders t
    LEFT JOIN raw_order_rollup r ON r.order_id = t.order_id AND r.t_open_date = t.t_open_date
    LEFT JOIN orders_sales_candidate osc ON osc.order_id = t.order_id AND osc.business_date = t.t_open_date
    LEFT JOIN current_order_allocated coa ON coa.order_id = t.order_id AND coa.t_open_date = t.t_open_date
    LEFT JOIN status_dedup_order_scope sdos ON sdos.order_id = t.order_id AND sdos.business_date = t.t_open_date
)
SELECT
    order_id,
    t_open_date,
    cast(raw_order_row_count AS varchar) AS raw_order_row_count,
    raw_status_list,
    selected_sales_status,
    selected_sales_transaction_submitted,
    selected_sales_modified,
    cast(selected_sales_payment_subtotal_milli AS varchar) AS selected_sales_payment_subtotal_milli,
    cast(source_status_1_included_tax_milli AS varchar) AS source_status_1_included_tax_milli,
    cast(current_allocated_included_tax_milli AS varchar) AS current_allocated_included_tax_milli,
    cast(status_dedup_allocated_included_tax_milli AS varchar) AS status_dedup_allocated_included_tax_milli,
    cast(current_delta_milli AS varchar) AS current_delta_milli,
    cast(status_dedup_delta_milli AS varchar) AS status_dedup_delta_milli,
    cast(source_status_1_net_milli AS varchar) AS source_status_1_net_milli,
    cast(current_allocated_net_milli AS varchar) AS current_allocated_net_milli,
    cast(status_dedup_allocated_net_milli AS varchar) AS status_dedup_allocated_net_milli,
    cast(current_net_delta_milli AS varchar) AS current_net_delta_milli,
    cast(status_dedup_net_delta_milli AS varchar) AS status_dedup_net_delta_milli
FROM status_dedup_before_after
ORDER BY current_max_abs_delta_milli DESC, t_open_date, order_id`,
		buildPreviewCTE(window),
		topTaxDeltaTraceLimit,
		startDate,
		endDate,
	)
}

func BuildStatusExcludedSummarySQL(window QueryWindow) string {
	return fmt.Sprintf(`%s,
status_excluded_candidate_base AS (
    SELECT
        order_status,
        order_id,
        business_date,
        CASE WHEN trim(coalesce(destination_raw, '')) = '' THEN '(blank)' ELSE trim(destination_raw) END AS destination_label,
        transaction_voided,
        transaction_submitted,
        order_total,
        order_included_tax,
        payment_subtotal_milli
    FROM orders_excluded_candidate
),
status_excluded_raw_counts AS (
    SELECT
        order_status,
        count(*) AS raw_rows
    FROM orders_ranked_by_status
    WHERE order_status IN (-1, 2)
    GROUP BY 1
),
status_excluded_destination_counts AS (
    SELECT
        order_status,
        destination_label,
        count(*) AS destination_count
    FROM status_excluded_candidate_base
    GROUP BY 1, 2
),
status_excluded_destination_distribution AS (
    SELECT
        order_status,
        array_join(array_agg(concat(destination_label, ':', cast(destination_count AS varchar)) ORDER BY destination_count DESC, destination_label), ', ') AS destination_distribution
    FROM status_excluded_destination_counts
    GROUP BY 1
)
SELECT
    cast(b.order_status AS varchar) AS status,
    cast(count(*) AS varchar) AS order_keys,
    cast(coalesce(max(r.raw_rows), 0) AS varchar) AS raw_rows,
    cast(cast(round(coalesce(sum(b.order_total), 0.0) * 1000) AS bigint) AS varchar) AS total_milli,
    cast(coalesce(sum(b.payment_subtotal_milli), 0) AS varchar) AS payment_subtotal_milli,
    cast(cast(round(coalesce(sum(b.order_included_tax), 0.0) * 1000) AS bigint) AS varchar) AS included_tax_milli,
    coalesce(max(d.destination_distribution), '') AS destination_distribution,
    cast(cast(sum(CASE WHEN b.transaction_voided IS NOT NULL THEN 1 ELSE 0 END) AS bigint) AS varchar) AS voided_count,
    cast(cast(sum(CASE WHEN b.transaction_submitted IS NULL THEN 1 ELSE 0 END) AS bigint) AS varchar) AS submitted_null_count
FROM status_excluded_candidate_base b
LEFT JOIN status_excluded_raw_counts r ON r.order_status = b.order_status
LEFT JOIN status_excluded_destination_distribution d ON d.order_status = b.order_status
GROUP BY 1
ORDER BY cast(status AS integer)`, buildPreviewCTE(window))
}

func buildPreviewCTE(window QueryWindow) string {
	startDate := window.StartDate.Format(dateLayout)
	endDate := window.EndDate.Format(dateLayout)

	return fmt.Sprintf(`
WITH orders_base AS (
    SELECT
        id AS order_id,
        t_open_date AS business_date,
        greatest(least(coalesce(try_cast(nullif(trim(s_hour), '') AS integer), hour(transaction_created), 0), 23), 0) AS hour_of_day,
        coalesce(nullif(trim(branch_id), ''), 'UNKNOWN_BRANCH') AS branch_id,
        coalesce(trim(destination), '') AS destination_raw,
        coalesce(status, -1) AS order_status,
        transaction_voided,
        %s AS order_total,
        %s AS order_item_subtotal,
        %s AS order_discount_subtotal,
        %s AS order_surcharge_subtotal,
        %s AS order_included_tax,
        0.0 AS order_excluded_tax,
        CASE
            WHEN total IS NOT NULL AND NOT is_finite(CAST(total AS double)) THEN 1
            WHEN item_subtotal IS NOT NULL AND NOT is_finite(CAST(item_subtotal AS double)) THEN 1
            WHEN discount_subtotal IS NOT NULL AND NOT is_finite(CAST(discount_subtotal AS double)) THEN 1
            WHEN surcharge_subtotal IS NOT NULL AND NOT is_finite(CAST(surcharge_subtotal AS double)) THEN 1
            WHEN included_tax_subtotal IS NOT NULL AND NOT is_finite(CAST(included_tax_subtotal AS double)) THEN 1
            ELSE 0
        END AS has_invalid_amount,
        CASE
            WHEN greatest(abs(%s), abs(%s), abs(%s), abs(%s), abs(%s)) > %f THEN 1
            ELSE 0
        END AS is_amount_outlier,
        CASE WHEN coalesce(status, -1) = 1 THEN 1 ELSE 0 END AS is_status_completed,
        CASE WHEN transaction_voided IS NULL THEN 0 ELSE 1 END AS is_voided,
        CASE
            WHEN coalesce(status, -1) = 1 AND transaction_voided IS NULL THEN 1
            ELSE 0
        END AS is_completed_proxy
    FROM orders_parquet
    WHERE t_open_date BETWEEN date '%s' AND date '%s'
),
filtered_orders AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        destination_raw,
        order_status,
        transaction_voided,
        order_total,
        order_item_subtotal,
        order_discount_subtotal,
        order_surcharge_subtotal,
        order_included_tax,
        order_excluded_tax,
        has_invalid_amount,
        is_amount_outlier,
        is_status_completed,
        is_voided,
        is_completed_proxy
    FROM orders_base
),
order_destinations AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        destination_raw,
        order_total,
        order_item_subtotal,
        order_discount_subtotal,
        order_surcharge_subtotal,
        order_status,
        transaction_voided,
        has_invalid_amount,
        is_amount_outlier,
        is_status_completed,
        is_voided,
        is_completed_proxy,
        order_included_tax,
        order_excluded_tax,
        CASE
            WHEN destination_raw = '' THEN 0
            WHEN regexp_like(lower(destination_raw), '.*(掃碼|qr).*') THEN 8
            WHEN regexp_like(lower(destination_raw), '.*快一點自取.*') THEN 6
            WHEN regexp_like(lower(destination_raw), '.*快一點外送.*') THEN 7
            WHEN regexp_like(lower(destination_raw), '.*(熊貓|foodpanda).*') THEN 2
            WHEN regexp_like(lower(destination_raw), '.*(ubereats|uber eats).*') THEN 5
            WHEN regexp_like(lower(destination_raw), '.*外送.*') THEN 3
            WHEN regexp_like(lower(destination_raw), '.*自取.*') THEN 4
            WHEN regexp_like(lower(destination_raw), '.*(來店|內用|店內).*') THEN 1
            ELSE 9
        END AS order_type_id
    FROM filtered_orders
),
order_additions_totals AS (
    SELECT
        order_id,
        sum(%s) AS addition_discount_total,
        sum(%s) AS addition_surcharge_total,
        sum(%s) AS addition_include_tax_total
    FROM order_additions_parquet
    WHERE t_open_date BETWEEN date '%s' AND date '%s'
    GROUP BY 1
),
item_lines AS (
    SELECT
        od.order_id,
        od.business_date,
        od.hour_of_day,
        od.branch_id,
        coalesce(nullif(trim(oi.product_no), ''), 'UNKNOWN_PRODUCT') AS product_no,
        od.order_type_id,
        sum(%s) AS qty_milli,
        sum(%s) AS item_net_subtotal,
        sum(%s) AS item_discount_total,
        sum(%s) AS item_surcharge_total,
        sum(%s + %s - %s) AS item_gross_subtotal
    FROM order_destinations od
    JOIN order_items_parquet oi ON oi.order_id = od.order_id
    WHERE oi.t_open_date BETWEEN date '%s' AND date '%s'
      AND coalesce(nullif(trim(oi.product_no), ''), '') <> ''
    GROUP BY 1, 2, 3, 4, 5, 6
),
order_item_totals AS (
    SELECT
        order_id,
        count(*) AS item_count,
        sum(item_gross_subtotal) AS order_gross_sales,
        sum(item_discount_total) AS order_item_discount_total,
        sum(item_surcharge_total) AS order_item_surcharge_total,
        sum(item_net_subtotal) AS order_item_net_sales
    FROM item_lines
    GROUP BY 1
),
payment_raw_values AS (
    SELECT
        order_id,
        coalesce(trim(name), '') AS payment_name_raw,
        CASE
            WHEN trim(coalesce(name, '')) = '' THEN 0
            WHEN lower(trim(name)) = 'undefined_pay' THEN 0
            WHEN lower(trim(name)) = 'cash' THEN 1
            WHEN regexp_like(lower(trim(name)), 'credit_card|debit_card|national_credit_card|card') THEN 2
            WHEN lower(trim(name)) IN ('linepay', 'easycard', 'uupay') THEN 3
            WHEN regexp_like(lower(trim(name)), 'foodpanda|uber eats|ubereats') THEN 4
            WHEN regexp_like(lower(trim(name)), 'coupon|voucher|gift|禮券|折抵') THEN 5
            ELSE 9
        END AS canonical_payment_type_id,
        %s AS payment_amount
    FROM order_payments_parquet
    WHERE t_open_date BETWEEN date '%s' AND date '%s'
),
payment_groups AS (
    SELECT
        order_id,
        canonical_payment_type_id,
        sum(payment_amount) AS payment_amount
    FROM payment_raw_values
    GROUP BY 1, 2
),
payment_summary AS (
    SELECT
        order_id,
        CASE
            WHEN count_if(abs(payment_amount) > 0.0001 AND canonical_payment_type_id <> 0) = 0 THEN 0
            WHEN count_if(abs(payment_amount) > 0.0001 AND canonical_payment_type_id <> 0) = 1 THEN max_by(canonical_payment_type_id, abs(payment_amount))
            ELSE 8
        END AS payment_type_id
    FROM payment_groups
    GROUP BY 1
),
order_financials AS (
    SELECT
        od.order_id,
        od.business_date,
        od.hour_of_day,
        od.branch_id,
        od.destination_raw,
        od.order_status,
        od.transaction_voided,
        od.has_invalid_amount,
        od.is_amount_outlier,
        od.is_status_completed,
        od.is_voided,
        od.is_completed_proxy,
        od.order_type_id,
        od.order_included_tax,
        od.order_excluded_tax,
        od.order_total,
        od.order_item_subtotal,
        od.order_discount_subtotal,
        od.order_surcharge_subtotal,
        coalesce(oit.item_count, 0) AS item_count,
        coalesce(oit.order_gross_sales, 0.0) AS order_gross_sales,
        coalesce(oit.order_item_discount_total, 0.0) AS order_item_discount_total,
        coalesce(oit.order_item_surcharge_total, 0.0) AS order_item_surcharge_total,
        coalesce(oit.order_item_net_sales, 0.0) AS order_item_net_sales,
        coalesce(oat.addition_discount_total, 0.0) AS addition_discount_total,
        coalesce(oat.addition_surcharge_total, 0.0) AS addition_surcharge_total,
        coalesce(oat.addition_include_tax_total, 0.0) AS addition_include_tax_total,
        coalesce(oit.order_gross_sales, 0.0) AS order_gross_sales_base,
        coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0) AS order_discount_total,
        coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0) AS order_surcharge_total,
        coalesce(oit.order_gross_sales, 0.0) - (coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0)) + (coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0)) AS order_net_sales,
        (od.order_included_tax + od.order_excluded_tax) AS order_tax_total,
        coalesce(oit.order_gross_sales, 0.0) - (coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0)) + (coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0)) - (od.order_included_tax + od.order_excluded_tax) AS order_sales_ex_tax
    FROM order_destinations od
    LEFT JOIN order_item_totals oit ON oit.order_id = od.order_id
    LEFT JOIN order_additions_totals oat ON oat.order_id = od.order_id
),
line_enriched AS (
    SELECT
        il.order_id,
        il.business_date,
        il.hour_of_day,
        il.branch_id,
        il.product_no,
        il.order_type_id,
        coalesce(ps.payment_type_id, 0) AS payment_type_id,
        il.qty_milli,
        il.item_gross_subtotal,
        il.item_discount_total,
        il.item_surcharge_total,
        of.order_included_tax,
        of.order_excluded_tax,
        of.order_gross_sales,
        of.order_net_sales,
        CASE
            WHEN coalesce(of.order_gross_sales, 0.0) = 0.0 THEN 0.0
            ELSE coalesce(of.addition_discount_total, 0.0) * (il.item_gross_subtotal / of.order_gross_sales)
        END AS addition_discount_allocated,
        CASE
            WHEN coalesce(of.order_gross_sales, 0.0) = 0.0 THEN 0.0
            ELSE coalesce(of.addition_surcharge_total, 0.0) * (il.item_gross_subtotal / of.order_gross_sales)
        END AS addition_surcharge_allocated
    FROM item_lines il
    JOIN order_financials of ON of.order_id = il.order_id
    LEFT JOIN payment_summary ps ON ps.order_id = il.order_id
),
line_component_milli AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        qty_milli,
        cast(round(item_gross_subtotal * 1000) AS bigint) AS gross_sales_milli,
        cast(round((item_discount_total + addition_discount_allocated) * 1000) AS bigint) AS discount_milli,
        cast(round((item_surcharge_total + addition_surcharge_allocated) * 1000) AS bigint) AS surcharge_milli,
        cast(round(CASE WHEN coalesce(order_net_sales, 0.0) = 0.0 THEN 0.0 ELSE order_included_tax * ((item_gross_subtotal - (item_discount_total + addition_discount_allocated) + (item_surcharge_total + addition_surcharge_allocated)) / order_net_sales) END * 1000) AS bigint) AS included_tax_milli,
        cast(round(CASE WHEN coalesce(order_net_sales, 0.0) = 0.0 THEN 0.0 ELSE order_excluded_tax * ((item_gross_subtotal - (item_discount_total + addition_discount_allocated) + (item_surcharge_total + addition_surcharge_allocated)) / order_net_sales) END * 1000) AS bigint) AS excluded_tax_milli
    FROM line_enriched
),
line_allocations AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        qty_milli,
        gross_sales_milli,
        discount_milli,
        surcharge_milli,
        gross_sales_milli - discount_milli + surcharge_milli AS net_sales_milli,
        gross_sales_milli - discount_milli + surcharge_milli - included_tax_milli - excluded_tax_milli AS sales_ex_tax_milli,
        included_tax_milli,
        excluded_tax_milli,
        included_tax_milli + excluded_tax_milli AS tax_milli
    FROM line_component_milli
),
order_allocation_summary AS (
    SELECT
        order_id,
        max(business_date) AS business_date,
        cast(sum(gross_sales_milli) AS bigint) AS gross_sales_milli,
        cast(sum(discount_milli) AS bigint) AS discount_milli,
        cast(sum(surcharge_milli) AS bigint) AS surcharge_milli,
        cast(sum(net_sales_milli) AS bigint) AS net_sales_milli,
        cast(sum(sales_ex_tax_milli) AS bigint) AS sales_ex_tax_milli,
        cast(sum(included_tax_milli) AS bigint) AS included_tax_milli,
        cast(sum(excluded_tax_milli) AS bigint) AS excluded_tax_milli,
        cast(sum(tax_milli) AS bigint) AS tax_milli,
        cast(sum(qty_milli) AS bigint) AS qty_milli,
        count(*) AS allocation_row_count
    FROM line_allocations
    GROUP BY 1
),
order_scope AS (
    SELECT
        of.order_id,
        of.business_date,
        of.hour_of_day,
        of.branch_id,
        of.destination_raw,
        of.order_status,
        of.transaction_voided,
        of.has_invalid_amount,
        of.is_amount_outlier,
        of.is_status_completed,
        of.is_voided,
        of.is_completed_proxy,
        of.item_count,
        CASE WHEN of.item_count > 0 THEN 1 ELSE 0 END AS has_valid_items,
        CASE WHEN abs(of.order_net_sales) > 0.000001 THEN 1 ELSE 0 END AS has_tax_allocation_denominator,
        cast(round(of.order_gross_sales * 1000) AS bigint) AS source_gross_sales_milli,
        cast(round(of.order_discount_total * 1000) AS bigint) AS source_discount_milli,
        cast(round(of.order_surcharge_total * 1000) AS bigint) AS source_surcharge_milli,
        cast(round(of.order_net_sales * 1000) AS bigint) AS source_net_sales_milli,
        cast(round(of.order_included_tax * 1000) AS bigint) AS source_included_tax_milli,
        cast(round(of.order_sales_ex_tax * 1000) AS bigint) AS source_sales_ex_tax_milli,
        cast(round(of.addition_include_tax_total * 1000) AS bigint) AS addition_include_tax_milli,
        cast(round(of.order_net_sales * 1000) AS bigint) AS allocation_denominator_milli,
        coalesce(oas.gross_sales_milli, 0) AS allocated_gross_sales_milli,
        coalesce(oas.discount_milli, 0) AS allocated_discount_milli,
        coalesce(oas.surcharge_milli, 0) AS allocated_surcharge_milli,
        coalesce(oas.net_sales_milli, 0) AS allocated_net_sales_milli,
        coalesce(oas.sales_ex_tax_milli, 0) AS allocated_sales_ex_tax_milli,
        coalesce(oas.included_tax_milli, 0) AS allocated_included_tax_milli,
        coalesce(oas.excluded_tax_milli, 0) AS allocated_excluded_tax_milli,
        coalesce(oas.tax_milli, 0) AS allocated_tax_milli,
        CASE WHEN oas.order_id IS NULL THEN 0 ELSE 1 END AS appears_in_preview_allocation,
        CASE
            WHEN of.has_invalid_amount = 1 THEN 'invalid_non_finite_amount'
            WHEN of.is_amount_outlier = 1 THEN 'amount_outlier_over_threshold'
            WHEN of.is_completed_proxy = 0 THEN 'status_not_completed_or_voided'
            WHEN of.item_count <= 0 THEN 'no_valid_items'
            WHEN abs(of.order_net_sales) <= 0.000001 THEN 'zero_tax_allocation_denominator'
            ELSE 'allocatable'
        END AS reconciliation_bucket
    FROM order_financials of
    LEFT JOIN order_allocation_summary oas ON oas.order_id = of.order_id
),
tax_reconciliation_breakdown AS (
    SELECT
        1 AS sort_order,
        'source_orders_all' AS metric,
        count(*) AS order_count,
        coalesce(sum(source_gross_sales_milli), 0) AS gross_sales_milli,
        coalesce(sum(source_discount_milli), 0) AS discount_milli,
        coalesce(sum(source_surcharge_milli), 0) AS surcharge_milli,
        coalesce(sum(source_net_sales_milli), 0) AS net_sales_milli,
        coalesce(sum(source_included_tax_milli), 0) AS included_tax_milli,
        coalesce(sum(source_sales_ex_tax_milli), 0) AS sales_ex_tax_milli,
        'orders_base 篩日期後全部訂單；尚未排除 invalid / outlier / status / no-items' AS note
    FROM order_scope
    UNION ALL
    SELECT
        2,
        'source_orders_valid',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        '排除 invalid / Infinity，且 outlier 門檻為 abs(order_total) <= 100000 TWD' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0
    UNION ALL
    SELECT
        3,
        'source_completed_orders',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'completed proxy = status = 1 AND transaction_voided IS NULL；orders_parquet 無明確 cancelled/unresolved 欄位' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT
        4,
        'preview_allocatable_orders',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'completed + valid + has items + tax allocation denominator > 0；這批才會產生可比較的稅額分攤' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        5,
        'preview_excluded_zero_denominator',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'completed + valid + has items，但 order_net_sales denominator 近似 0；目前 preview tax allocation 會落成 0' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 0
    UNION ALL
    SELECT
        6,
        'preview_excluded_no_items',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'completed + valid，但無可用 item_lines；因此不會進 preview allocation' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 0
    UNION ALL
    SELECT
        7,
        'preview_excluded_status',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'valid orders 中 status != 1 或 transaction_voided 非空；用來檢查 source / preview status 邊界是否一致' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 0
    UNION ALL
    SELECT
        8,
        'preview_after_allocation',
        count(*),
        coalesce(sum(allocated_gross_sales_milli), 0),
        coalesce(sum(allocated_discount_milli), 0),
        coalesce(sum(allocated_surcharge_milli), 0),
        coalesce(sum(allocated_net_sales_milli), 0),
        coalesce(sum(allocated_included_tax_milli), 0),
        coalesce(sum(allocated_sales_ex_tax_milli), 0),
        'completed + valid + allocatable orders，彙總 allocation 後實際落到 preview 的 order-level totals' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        9,
        'rounding_delta',
        0,
        coalesce(sum(allocated_gross_sales_milli - source_gross_sales_milli), 0),
        coalesce(sum(allocated_discount_milli - source_discount_milli), 0),
        coalesce(sum(allocated_surcharge_milli - source_surcharge_milli), 0),
        coalesce(sum(allocated_net_sales_milli - source_net_sales_milli), 0),
        coalesce(sum(allocated_included_tax_milli - source_included_tax_milli), 0),
        coalesce(sum(allocated_sales_ex_tax_milli - source_sales_ex_tax_milli), 0),
        'source_completed_orders 與 preview_after_allocation 的 order-level delta；若數值很小，代表主差異不在 rounding' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        10,
        'preview_excluded_invalid_or_outlier',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'invalid 非有限值或 outlier；用來隔離資料品質造成的差異' 
    FROM order_scope
    WHERE has_invalid_amount = 1 OR is_amount_outlier = 1
),
additions_tax_debug AS (
    SELECT 1 AS sort_order, 'additions_include_tax_milli_total' AS metric, coalesce(sum(addition_include_tax_milli), 0) AS metric_value, 'completed + valid 訂單的 order_additions.include_tax 總額' AS note
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 2, 'orders_included_tax_milli_total', coalesce(sum(source_included_tax_milli), 0), 'completed + valid 訂單的 orders.included_tax_subtotal 總額'
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 3, 'items_included_tax_milli_total', coalesce(sum(allocated_included_tax_milli), 0), 'completed + valid + allocatable 訂單，從 orders tax 分攤到 item/preview 後的 included tax 總額'
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT 4, 'orders_minus_items_tax_milli', coalesce(sum(source_included_tax_milli - allocated_included_tax_milli), 0), 'orders tax 與 preview allocated tax 的差額；包含排除訂單與 rounding' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 5, 'orders_minus_items_minus_additions_tax_milli', coalesce(sum(source_included_tax_milli - allocated_included_tax_milli - addition_include_tax_milli), 0), '若把 additions.include_tax 視為額外稅額後的剩餘差額；可用來判斷是否接近 double count' 
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
),
rounding_order_debug AS (
    SELECT
        business_date,
        order_id,
        source_included_tax_milli,
        allocated_included_tax_milli,
        allocated_included_tax_milli - source_included_tax_milli AS delta_milli,
        item_count,
        allocation_denominator_milli,
        order_status AS status,
        destination_raw AS destination
    FROM order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
),
rounding_debug AS (
    SELECT 1 AS sort_order, 'source_order_level_included_tax_milli' AS metric, coalesce(sum(source_included_tax_milli), 0) AS metric_value, 'scope = completed + valid + allocatable orders' AS note
    FROM rounding_order_debug
    UNION ALL
    SELECT 2, 'allocated_preview_included_tax_milli', coalesce(sum(allocated_included_tax_milli), 0), 'scope = completed + valid + allocatable orders'
    FROM rounding_order_debug
    UNION ALL
    SELECT 3, 'rounding_delta_milli', coalesce(sum(delta_milli), 0), 'allocated preview tax - source order tax；若接近 0，差異主因不是 rounding'
    FROM rounding_order_debug
    UNION ALL
    SELECT 4, 'affected_order_count', count_if(delta_milli <> 0), '有非零 order-level rounding delta 的訂單數'
    FROM rounding_order_debug
    UNION ALL
    SELECT 5, 'max_abs_order_delta_milli', coalesce(max(abs(delta_milli)), 0), '單筆訂單最大絕對 delta'
    FROM rounding_order_debug
    UNION ALL
    SELECT 6, 'avg_abs_order_delta_milli', cast(round(coalesce(avg(abs(delta_milli)), 0.0)) AS bigint), '單筆訂單平均絕對 delta，已四捨五入到 milli bigint'
    FROM rounding_order_debug
),
top_tax_delta_sample AS (
    SELECT
        business_date,
        order_id,
        source_included_tax_milli,
        allocated_included_tax_milli,
        delta_milli,
        item_count,
        allocation_denominator_milli,
        status,
        destination
    FROM rounding_order_debug
    ORDER BY abs(delta_milli) DESC, business_date, order_id
    LIMIT %d
),
final_aggregation AS (
    SELECT
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        sum(qty_milli) AS qty_milli,
        sum(gross_sales_milli) AS gross_sales_milli,
        sum(discount_milli) AS discount_milli,
        sum(surcharge_milli) AS surcharge_milli,
        sum(net_sales_milli) AS net_sales_milli,
        sum(sales_ex_tax_milli) AS sales_ex_tax_milli,
        sum(included_tax_milli) AS included_tax_milli,
        sum(excluded_tax_milli) AS excluded_tax_milli,
        sum(tax_milli) AS tax_milli
    FROM line_allocations
    GROUP BY 1, 2, 3, 4, 5, 6
),
%s
    `,
		safeDoubleExpr("total"),
		safeDoubleExpr("item_subtotal"),
		positiveDoubleExpr("discount_subtotal"),
		positiveDoubleExpr("surcharge_subtotal"),
		safeDoubleExpr("included_tax_subtotal"),
		safeDoubleExpr("total"),
		safeDoubleExpr("item_subtotal"),
		positiveDoubleExpr("discount_subtotal"),
		positiveDoubleExpr("surcharge_subtotal"),
		safeDoubleExpr("included_tax_subtotal"),
		debugOrderTotalOutlierThresholdTWD,
		startDate,
		endDate,
		positiveDoubleExpr("current_discount"),
		positiveDoubleExpr("current_surcharge"),
		positiveDoubleExpr("include_tax"),
		startDate,
		endDate,
		qtyMilliExpr("oi.current_qty"),
		safeDoubleExpr("oi.current_subtotal"),
		positiveDoubleExpr("oi.current_discount"),
		positiveDoubleExpr("oi.current_surcharge"),
		safeDoubleExpr("oi.current_subtotal"),
		positiveDoubleExpr("oi.current_discount"),
		positiveDoubleExpr("oi.current_surcharge"),
		startDate,
		endDate,
		safeDoubleExpr("amount"),
		startDate,
		endDate,
		topTaxDeltaSampleLimit,
		buildStatusDedupCTEBlock(startDate, endDate),
	)
}

func buildStatusDedupCTEBlock(startDate string, endDate string) string {
	return fmt.Sprintf(`orders_ranked_by_status AS (
    SELECT
        o.id AS order_id,
        o.t_open_date AS business_date,
        greatest(least(coalesce(try_cast(nullif(trim(o.s_hour), '') AS integer), hour(o.transaction_created), 0), 23), 0) AS hour_of_day,
        coalesce(nullif(trim(o.branch_id), ''), 'UNKNOWN_BRANCH') AS branch_id,
        coalesce(trim(o.destination), '') AS destination_raw,
        CASE
            WHEN coalesce(trim(o.destination), '') = '' THEN 0
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*(掃碼|qr).*') THEN 8
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*快一點自取.*') THEN 6
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*快一點外送.*') THEN 7
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*(熊貓|foodpanda).*') THEN 2
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*(ubereats|uber eats).*') THEN 5
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*外送.*') THEN 3
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*自取.*') THEN 4
            WHEN regexp_like(lower(coalesce(trim(o.destination), '')), '.*(來店|內用|店內).*') THEN 1
            ELSE 9
        END AS order_type_id,
        coalesce(o.status, -1) AS order_status,
        o.transaction_voided,
        o.void_sale_period,
        o.transaction_submitted,
        o.transaction_created,
        o.modified,
        %s AS payment_subtotal_milli,
        %s AS order_total,
        %s AS order_item_subtotal,
        %s AS order_discount_subtotal,
        %s AS order_surcharge_subtotal,
        %s AS order_included_tax,
        %s AS order_tax_subtotal,
        0.0 AS order_excluded_tax,
        CASE
            WHEN o.total IS NOT NULL AND NOT is_finite(CAST(o.total AS double)) THEN 1
            WHEN o.item_subtotal IS NOT NULL AND NOT is_finite(CAST(o.item_subtotal AS double)) THEN 1
            WHEN o.discount_subtotal IS NOT NULL AND NOT is_finite(CAST(o.discount_subtotal AS double)) THEN 1
            WHEN o.surcharge_subtotal IS NOT NULL AND NOT is_finite(CAST(o.surcharge_subtotal AS double)) THEN 1
            WHEN o.included_tax_subtotal IS NOT NULL AND NOT is_finite(CAST(o.included_tax_subtotal AS double)) THEN 1
            ELSE 0
        END AS has_invalid_amount,
        CASE
            WHEN greatest(abs(%s), abs(%s), abs(%s), abs(%s), abs(%s)) > %f THEN 1
            ELSE 0
        END AS is_amount_outlier,
        CASE WHEN coalesce(o.status, -1) = 1 THEN 1 ELSE 0 END AS is_status_completed,
        CASE WHEN o.transaction_voided IS NULL THEN 0 ELSE 1 END AS is_voided,
        CASE
            WHEN coalesce(o.status, -1) = 1 AND o.transaction_voided IS NULL THEN 1
            ELSE 0
        END AS is_completed_proxy,
        row_number() OVER (
            PARTITION BY o.id, o.t_open_date, coalesce(o.status, -1)
            ORDER BY o.transaction_submitted DESC NULLS LAST,
                     o.modified DESC NULLS LAST,
                     o.transaction_created DESC NULLS LAST,
                     %s DESC,
                     %s DESC
        ) AS status_row_rank
    FROM orders_parquet o
    WHERE o.t_open_date BETWEEN date '%s' AND date '%s'
),
orders_status_latest AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        destination_raw,
        order_type_id,
        order_status,
        transaction_voided,
        void_sale_period,
        transaction_submitted,
        transaction_created,
        modified,
        payment_subtotal_milli,
        order_total,
        order_item_subtotal,
        order_discount_subtotal,
        order_surcharge_subtotal,
        order_included_tax,
        order_tax_subtotal,
        order_excluded_tax,
        has_invalid_amount,
        is_amount_outlier,
        is_status_completed,
        is_voided,
        is_completed_proxy
    FROM orders_ranked_by_status
    WHERE status_row_rank = 1
),
orders_sales_candidate AS (
    SELECT *
    FROM orders_status_latest
    WHERE order_status = 1
),
orders_void_candidate AS (
    SELECT *
    FROM orders_status_latest
    WHERE order_status = -2
),
orders_excluded_candidate AS (
    SELECT *
    FROM orders_status_latest
    WHERE order_status IN (-1, 2)
),
orders_status_presence AS (
    SELECT
        order_id,
        business_date,
        max(CASE WHEN order_status = 1 THEN 1 ELSE 0 END) AS has_status_1,
        max(CASE WHEN order_status = 2 THEN 1 ELSE 0 END) AS has_status_2,
        max(CASE WHEN order_status = -1 THEN 1 ELSE 0 END) AS has_status_minus_1,
        max(CASE WHEN order_status = -2 THEN 1 ELSE 0 END) AS has_status_minus_2
    FROM orders_status_latest
    GROUP BY 1, 2
),
status_dedup_order_destinations AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        destination_raw,
        order_type_id,
        order_total,
        order_item_subtotal,
        order_discount_subtotal,
        order_surcharge_subtotal,
        order_status,
        transaction_voided,
        void_sale_period,
        has_invalid_amount,
        is_amount_outlier,
        is_status_completed,
        is_voided,
        is_completed_proxy,
        order_included_tax,
        order_tax_subtotal,
        order_excluded_tax
    FROM orders_sales_candidate
),
status_dedup_item_lines AS (
    SELECT
        od.order_id,
        od.business_date,
        od.hour_of_day,
        od.branch_id,
        coalesce(nullif(trim(oi.product_no), ''), 'UNKNOWN_PRODUCT') AS product_no,
        od.order_type_id,
        sum(%s) AS qty_milli,
        sum(%s) AS item_net_subtotal,
        sum(%s) AS item_discount_total,
        sum(%s) AS item_surcharge_total,
        sum(%s + %s - %s) AS item_gross_subtotal
    FROM status_dedup_order_destinations od
    JOIN order_items_parquet oi ON oi.order_id = od.order_id
    WHERE oi.t_open_date BETWEEN date '%s' AND date '%s'
      AND coalesce(nullif(trim(oi.product_no), ''), '') <> ''
    GROUP BY 1, 2, 3, 4, 5, 6
),
status_dedup_order_item_totals AS (
    SELECT
        order_id,
        count(*) AS item_count,
        sum(item_gross_subtotal) AS order_gross_sales,
        sum(item_discount_total) AS order_item_discount_total,
        sum(item_surcharge_total) AS order_item_surcharge_total,
        sum(item_net_subtotal) AS order_item_net_sales
    FROM status_dedup_item_lines
    GROUP BY 1
),
status_dedup_order_financials AS (
    SELECT
        od.order_id,
        od.business_date,
        od.hour_of_day,
        od.branch_id,
        od.destination_raw,
        od.order_status,
        od.transaction_voided,
        od.has_invalid_amount,
        od.is_amount_outlier,
        od.is_status_completed,
        od.is_voided,
        od.is_completed_proxy,
        od.order_type_id,
        od.order_included_tax,
        od.order_excluded_tax,
        od.order_total,
        od.order_item_subtotal,
        od.order_discount_subtotal,
        od.order_surcharge_subtotal,
        coalesce(oit.item_count, 0) AS item_count,
        coalesce(oit.order_gross_sales, 0.0) AS order_gross_sales,
        coalesce(oit.order_item_discount_total, 0.0) AS order_item_discount_total,
        coalesce(oit.order_item_surcharge_total, 0.0) AS order_item_surcharge_total,
        coalesce(oit.order_item_net_sales, 0.0) AS order_item_net_sales,
        coalesce(oat.addition_discount_total, 0.0) AS addition_discount_total,
        coalesce(oat.addition_surcharge_total, 0.0) AS addition_surcharge_total,
        coalesce(oat.addition_include_tax_total, 0.0) AS addition_include_tax_total,
        coalesce(oit.order_gross_sales, 0.0) AS order_gross_sales_base,
        coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0) AS order_discount_total,
        coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0) AS order_surcharge_total,
        coalesce(oit.order_gross_sales, 0.0) - (coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0)) + (coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0)) AS order_net_sales,
        (od.order_included_tax + od.order_excluded_tax) AS order_tax_total,
        coalesce(oit.order_gross_sales, 0.0) - (coalesce(oit.order_item_discount_total, 0.0) + coalesce(oat.addition_discount_total, 0.0)) + (coalesce(oit.order_item_surcharge_total, 0.0) + coalesce(oat.addition_surcharge_total, 0.0)) - (od.order_included_tax + od.order_excluded_tax) AS order_sales_ex_tax
    FROM status_dedup_order_destinations od
    LEFT JOIN status_dedup_order_item_totals oit ON oit.order_id = od.order_id
    LEFT JOIN order_additions_totals oat ON oat.order_id = od.order_id
),
status_dedup_line_enriched AS (
    SELECT
        il.order_id,
        il.business_date,
        il.hour_of_day,
        il.branch_id,
        il.product_no,
        il.order_type_id,
        coalesce(ps.payment_type_id, 0) AS payment_type_id,
        il.qty_milli,
        il.item_gross_subtotal,
        il.item_discount_total,
        il.item_surcharge_total,
        sof.order_included_tax,
        sof.order_excluded_tax,
        sof.order_gross_sales,
        sof.order_net_sales,
        CASE
            WHEN coalesce(sof.order_gross_sales, 0.0) = 0.0 THEN 0.0
            ELSE coalesce(sof.addition_discount_total, 0.0) * (il.item_gross_subtotal / sof.order_gross_sales)
        END AS addition_discount_allocated,
        CASE
            WHEN coalesce(sof.order_gross_sales, 0.0) = 0.0 THEN 0.0
            ELSE coalesce(sof.addition_surcharge_total, 0.0) * (il.item_gross_subtotal / sof.order_gross_sales)
        END AS addition_surcharge_allocated
    FROM status_dedup_item_lines il
    JOIN status_dedup_order_financials sof ON sof.order_id = il.order_id
    LEFT JOIN payment_summary ps ON ps.order_id = il.order_id
),
status_dedup_line_component_milli AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        qty_milli,
        cast(round(item_gross_subtotal * 1000) AS bigint) AS gross_sales_milli,
        cast(round((item_discount_total + addition_discount_allocated) * 1000) AS bigint) AS discount_milli,
        cast(round((item_surcharge_total + addition_surcharge_allocated) * 1000) AS bigint) AS surcharge_milli,
        cast(round(CASE WHEN coalesce(order_net_sales, 0.0) = 0.0 THEN 0.0 ELSE order_included_tax * ((item_gross_subtotal - (item_discount_total + addition_discount_allocated) + (item_surcharge_total + addition_surcharge_allocated)) / order_net_sales) END * 1000) AS bigint) AS included_tax_milli,
        cast(round(CASE WHEN coalesce(order_net_sales, 0.0) = 0.0 THEN 0.0 ELSE order_excluded_tax * ((item_gross_subtotal - (item_discount_total + addition_discount_allocated) + (item_surcharge_total + addition_surcharge_allocated)) / order_net_sales) END * 1000) AS bigint) AS excluded_tax_milli
    FROM status_dedup_line_enriched
),
status_dedup_line_allocations AS (
    SELECT
        order_id,
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        qty_milli,
        gross_sales_milli,
        discount_milli,
        surcharge_milli,
        gross_sales_milli - discount_milli + surcharge_milli AS net_sales_milli,
        gross_sales_milli - discount_milli + surcharge_milli - included_tax_milli - excluded_tax_milli AS sales_ex_tax_milli,
        included_tax_milli,
        excluded_tax_milli,
        included_tax_milli + excluded_tax_milli AS tax_milli
    FROM status_dedup_line_component_milli
),
status_dedup_order_allocation_summary AS (
    SELECT
        order_id,
        max(business_date) AS business_date,
        cast(sum(gross_sales_milli) AS bigint) AS gross_sales_milli,
        cast(sum(discount_milli) AS bigint) AS discount_milli,
        cast(sum(surcharge_milli) AS bigint) AS surcharge_milli,
        cast(sum(net_sales_milli) AS bigint) AS net_sales_milli,
        cast(sum(sales_ex_tax_milli) AS bigint) AS sales_ex_tax_milli,
        cast(sum(included_tax_milli) AS bigint) AS included_tax_milli,
        cast(sum(excluded_tax_milli) AS bigint) AS excluded_tax_milli,
        cast(sum(tax_milli) AS bigint) AS tax_milli,
        cast(sum(qty_milli) AS bigint) AS qty_milli,
        count(*) AS allocation_row_count
    FROM status_dedup_line_allocations
    GROUP BY 1
),
status_dedup_order_scope AS (
    SELECT
        sof.order_id,
        sof.business_date,
        sof.hour_of_day,
        sof.branch_id,
        sof.destination_raw,
        sof.order_status,
        sof.transaction_voided,
        sof.has_invalid_amount,
        sof.is_amount_outlier,
        sof.is_status_completed,
        sof.is_voided,
        sof.is_completed_proxy,
        sof.item_count,
        CASE WHEN sof.item_count > 0 THEN 1 ELSE 0 END AS has_valid_items,
        CASE WHEN abs(sof.order_net_sales) > 0.000001 THEN 1 ELSE 0 END AS has_tax_allocation_denominator,
        cast(round(sof.order_gross_sales * 1000) AS bigint) AS source_gross_sales_milli,
        cast(round(sof.order_discount_total * 1000) AS bigint) AS source_discount_milli,
        cast(round(sof.order_surcharge_total * 1000) AS bigint) AS source_surcharge_milli,
        cast(round(sof.order_net_sales * 1000) AS bigint) AS source_net_sales_milli,
        cast(round(sof.order_sales_ex_tax * 1000) AS bigint) AS source_sales_ex_tax_milli,
        cast(round(sof.order_included_tax * 1000) AS bigint) AS source_included_tax_milli,
        cast(round(sof.order_tax_total * 1000) AS bigint) AS source_tax_milli,
        cast(round(sof.addition_include_tax_total * 1000) AS bigint) AS addition_include_tax_milli,
        cast(round(sof.order_net_sales * 1000) AS bigint) AS allocation_denominator_milli,
        coalesce(sdoas.gross_sales_milli, 0) AS allocated_gross_sales_milli,
        coalesce(sdoas.discount_milli, 0) AS allocated_discount_milli,
        coalesce(sdoas.surcharge_milli, 0) AS allocated_surcharge_milli,
        coalesce(sdoas.net_sales_milli, 0) AS allocated_net_sales_milli,
        coalesce(sdoas.sales_ex_tax_milli, 0) AS allocated_sales_ex_tax_milli,
        coalesce(sdoas.included_tax_milli, 0) AS allocated_included_tax_milli,
        coalesce(sdoas.excluded_tax_milli, 0) AS allocated_excluded_tax_milli,
        coalesce(sdoas.tax_milli, 0) AS allocated_tax_milli,
        CASE WHEN sdoas.order_id IS NULL THEN 0 ELSE 1 END AS appears_in_preview_allocation
    FROM status_dedup_order_financials sof
    LEFT JOIN status_dedup_order_allocation_summary sdoas ON sdoas.order_id = sof.order_id
),
status_dedup_tax_reconciliation_breakdown AS (
    SELECT
        1 AS sort_order,
        'source_orders_all' AS metric,
        count(*) AS order_count,
        coalesce(sum(source_gross_sales_milli), 0) AS gross_sales_milli,
        coalesce(sum(source_discount_milli), 0) AS discount_milli,
        coalesce(sum(source_surcharge_milli), 0) AS surcharge_milli,
        coalesce(sum(source_net_sales_milli), 0) AS net_sales_milli,
        coalesce(sum(source_included_tax_milli), 0) AS included_tax_milli,
        coalesce(sum(source_sales_ex_tax_milli), 0) AS sales_ex_tax_milli,
        'status-aware latest sales candidate 全部訂單；正式 preview sales path 的 source pool' AS note
    FROM status_dedup_order_scope
    UNION ALL
    SELECT
        2,
        'source_orders_valid',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        '正式 preview sales path，排除 invalid / Infinity 與 outlier'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0
    UNION ALL
    SELECT
        3,
        'source_completed_orders',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        '正式 preview sales path completed scope = status = 1 latest row'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT
        4,
        'preview_allocatable_orders',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        '正式 preview sales path = status-aware status = 1 latest + valid + has items + denominator > 0'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        5,
        'preview_excluded_zero_denominator',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'status-aware sales candidate 中 denominator 近似 0 的訂單'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 0
    UNION ALL
    SELECT
        6,
        'preview_excluded_no_items',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'status-aware sales candidate 中沒有可進 item_lines 的訂單'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 0
    UNION ALL
    SELECT
        7,
        'preview_excluded_status',
        cast((SELECT count(*) FROM orders_excluded_candidate) AS bigint),
        0,
        0,
        0,
        0,
        0,
        0,
        'status = -1 / 2 已自正式 sales preview 完全排除；詳見 status_excluded_summary'
    UNION ALL
    SELECT
        8,
        'preview_after_allocation',
        count(*),
        coalesce(sum(allocated_gross_sales_milli), 0),
        coalesce(sum(allocated_discount_milli), 0),
        coalesce(sum(allocated_surcharge_milli), 0),
        coalesce(sum(allocated_net_sales_milli), 0),
        coalesce(sum(allocated_included_tax_milli), 0),
        coalesce(sum(allocated_sales_ex_tax_milli), 0),
        '正式 preview sales path allocation 後的 order-level totals'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        9,
        'rounding_delta',
        0,
        coalesce(sum(allocated_gross_sales_milli - source_gross_sales_milli), 0),
        coalesce(sum(allocated_discount_milli - source_discount_milli), 0),
        coalesce(sum(allocated_surcharge_milli - source_surcharge_milli), 0),
        coalesce(sum(allocated_net_sales_milli - source_net_sales_milli), 0),
        coalesce(sum(allocated_included_tax_milli - source_included_tax_milli), 0),
        coalesce(sum(allocated_sales_ex_tax_milli - source_sales_ex_tax_milli), 0),
        '正式 preview sales path 與 source status = 1 latest row 的 order-level delta'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT
        10,
        'preview_excluded_invalid_or_outlier',
        count(*),
        coalesce(sum(source_gross_sales_milli), 0),
        coalesce(sum(source_discount_milli), 0),
        coalesce(sum(source_surcharge_milli), 0),
        coalesce(sum(source_net_sales_milli), 0),
        coalesce(sum(source_included_tax_milli), 0),
        coalesce(sum(source_sales_ex_tax_milli), 0),
        'status-aware sales candidate 中因 invalid / outlier 排除的訂單'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 1 OR is_amount_outlier = 1
),
status_dedup_additions_tax_debug AS (
    SELECT 1 AS sort_order, 'additions_include_tax_milli_total' AS metric, coalesce(sum(addition_include_tax_milli), 0) AS metric_value, '正式 preview sales path completed + valid 訂單的 order_additions.include_tax 總額' AS note
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 2, 'orders_included_tax_milli_total', coalesce(sum(source_included_tax_milli), 0), '正式 preview sales path completed + valid 訂單的 orders.included_tax_subtotal 總額'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 3, 'items_included_tax_milli_total', coalesce(sum(allocated_included_tax_milli), 0), '正式 preview sales path completed + valid + allocatable 訂單，從 orders tax 分攤到 preview 後的 included tax 總額'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
    UNION ALL
    SELECT 4, 'orders_minus_items_tax_milli', coalesce(sum(source_included_tax_milli - allocated_included_tax_milli), 0), '正式 preview sales path中 source tax 與 allocated tax 的差額'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
    UNION ALL
    SELECT 5, 'orders_minus_items_minus_additions_tax_milli', coalesce(sum(source_included_tax_milli - allocated_included_tax_milli - addition_include_tax_milli), 0), '正式 preview sales path若視 additions.include_tax 為額外稅額後的剩餘差額'
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1
),
status_dedup_rounding_order_debug AS (
    SELECT
        business_date,
        order_id,
        source_included_tax_milli,
        allocated_included_tax_milli,
        allocated_included_tax_milli - source_included_tax_milli AS delta_milli,
        item_count,
        allocation_denominator_milli,
        order_status AS status,
        destination_raw AS destination
    FROM status_dedup_order_scope
    WHERE has_invalid_amount = 0 AND is_amount_outlier = 0 AND is_completed_proxy = 1 AND has_valid_items = 1 AND has_tax_allocation_denominator = 1
),
status_dedup_rounding_debug AS (
    SELECT 1 AS sort_order, 'source_order_level_included_tax_milli' AS metric, coalesce(sum(source_included_tax_milli), 0) AS metric_value, '正式 preview sales path scope = completed + valid + allocatable orders' AS note
    FROM status_dedup_rounding_order_debug
    UNION ALL
    SELECT 2, 'allocated_preview_included_tax_milli', coalesce(sum(allocated_included_tax_milli), 0), '正式 preview sales path scope = completed + valid + allocatable orders'
    FROM status_dedup_rounding_order_debug
    UNION ALL
    SELECT 3, 'rounding_delta_milli', coalesce(sum(delta_milli), 0), '正式 preview allocated tax - source order tax；若接近 0，主差異不在 rounding'
    FROM status_dedup_rounding_order_debug
    UNION ALL
    SELECT 4, 'affected_order_count', count_if(delta_milli <> 0), '正式 preview sales path中有非零 order-level rounding delta 的訂單數'
    FROM status_dedup_rounding_order_debug
    UNION ALL
    SELECT 5, 'max_abs_order_delta_milli', coalesce(max(abs(delta_milli)), 0), '正式 preview sales path單筆訂單最大絕對 delta'
    FROM status_dedup_rounding_order_debug
    UNION ALL
    SELECT 6, 'avg_abs_order_delta_milli', cast(round(coalesce(avg(abs(delta_milli)), 0.0)) AS bigint), '正式 preview sales path單筆訂單平均絕對 delta，已四捨五入到 milli bigint'
    FROM status_dedup_rounding_order_debug
),
status_dedup_top_tax_delta_sample AS (
    SELECT
        business_date,
        order_id,
        source_included_tax_milli,
        allocated_included_tax_milli,
        delta_milli,
        item_count,
        allocation_denominator_milli,
        status,
        destination
    FROM status_dedup_rounding_order_debug
    ORDER BY abs(delta_milli) DESC, business_date, order_id
    LIMIT %d
),
status_dedup_final_aggregation AS (
    SELECT
        business_date,
        hour_of_day,
        branch_id,
        product_no,
        order_type_id,
        payment_type_id,
        sum(qty_milli) AS qty_milli,
        sum(gross_sales_milli) AS gross_sales_milli,
        sum(discount_milli) AS discount_milli,
        sum(surcharge_milli) AS surcharge_milli,
        sum(net_sales_milli) AS net_sales_milli,
        sum(sales_ex_tax_milli) AS sales_ex_tax_milli,
        sum(included_tax_milli) AS included_tax_milli,
        sum(excluded_tax_milli) AS excluded_tax_milli,
        sum(tax_milli) AS tax_milli
    FROM status_dedup_line_allocations
    GROUP BY 1, 2, 3, 4, 5, 6
)`,
		safeMilliBigintExpr(safeDoubleExpr("o.payment_subtotal")),
		safeDoubleExpr("o.total"),
		safeDoubleExpr("o.item_subtotal"),
		positiveDoubleExpr("o.discount_subtotal"),
		positiveDoubleExpr("o.surcharge_subtotal"),
		safeDoubleExpr("o.included_tax_subtotal"),
		safeDoubleExpr("o.tax_subtotal"),
		safeDoubleExpr("o.total"),
		safeDoubleExpr("o.item_subtotal"),
		positiveDoubleExpr("o.discount_subtotal"),
		positiveDoubleExpr("o.surcharge_subtotal"),
		safeDoubleExpr("o.included_tax_subtotal"),
		debugOrderTotalOutlierThresholdTWD,
		safeDoubleExpr("o.payment_subtotal"),
		safeDoubleExpr("o.total"),
		startDate,
		endDate,
		qtyMilliExpr("oi.current_qty"),
		safeDoubleExpr("oi.current_subtotal"),
		positiveDoubleExpr("oi.current_discount"),
		positiveDoubleExpr("oi.current_surcharge"),
		safeDoubleExpr("oi.current_subtotal"),
		positiveDoubleExpr("oi.current_discount"),
		positiveDoubleExpr("oi.current_surcharge"),
		startDate,
		endDate,
		topTaxDeltaSampleLimit,
	)
}

func resolvePreviewLimit(raw int) int {
	if raw <= 0 {
		return defaultPreviewLimit
	}

	return raw
}

func orderTypeCodeSQL(expression string) string {
	return fmt.Sprintf(`CASE
            WHEN %s = 0 THEN 'unknown'
            WHEN %s = 1 THEN 'in_store'
            WHEN %s = 2 THEN 'foodpanda'
            WHEN %s = 3 THEN 'delivery'
            WHEN %s = 4 THEN 'pickup'
            WHEN %s = 5 THEN 'ubereats'
            WHEN %s = 6 THEN 'quick_pickup'
            WHEN %s = 7 THEN 'quick_delivery'
            WHEN %s = 8 THEN 'qr_order'
            ELSE 'other'
        END`, expression, expression, expression, expression, expression, expression, expression, expression, expression)
}

func paymentTypeIDSQL(expression string) string {
	return fmt.Sprintf(`CASE
            WHEN trim(coalesce(%s, '')) = '' THEN 0
            WHEN lower(trim(%s)) = 'undefined_pay' THEN 0
            WHEN lower(trim(%s)) = 'cash' THEN 1
            WHEN regexp_like(lower(trim(%s)), 'credit_card|debit_card|national_credit_card|card') THEN 2
            WHEN lower(trim(%s)) IN ('linepay', 'easycard', 'uupay') THEN 3
            WHEN regexp_like(lower(trim(%s)), 'foodpanda|uber eats|ubereats') THEN 4
            WHEN regexp_like(lower(trim(%s)), 'coupon|voucher|gift|禮券|折抵') THEN 5
            ELSE 9
        END`, expression, expression, expression, expression, expression, expression, expression)
}

func paymentTypeCodeSQL(expression string) string {
	return fmt.Sprintf(`CASE
            WHEN %s = 0 THEN 'unknown_payment'
            WHEN %s = 1 THEN 'cash'
            WHEN %s = 2 THEN 'card'
            WHEN %s = 3 THEN 'e_wallet'
            WHEN %s = 4 THEN 'platform_payment'
            WHEN %s = 5 THEN 'coupon'
            WHEN %s = 8 THEN 'mixed'
            ELSE 'other'
        END`, expression, expression, expression, expression, expression, expression, expression)
}

func safeDoubleExpr(expression string) string {
	return fmt.Sprintf("CASE WHEN %s IS NOT NULL AND is_finite(CAST(%s AS double)) THEN CAST(%s AS double) ELSE 0.0 END", expression, expression, expression)
}

func safeMilliBigintExpr(expression string) string {
	return fmt.Sprintf("CAST(ROUND(CASE WHEN abs(%s) <= CAST(%d AS double) THEN %s ELSE 0.0 END * 1000) AS bigint)", expression, maxMilliBigintInputTWD, expression)
}

func positiveDoubleExpr(expression string) string {
	return fmt.Sprintf("abs(%s)", safeDoubleExpr(expression))
}

func qtyMilliExpr(expression string) string {
	return fmt.Sprintf("CAST(coalesce(try_cast(%s AS decimal(20,3)), decimal '0.000') * 1000 AS bigint)", expression)
}
