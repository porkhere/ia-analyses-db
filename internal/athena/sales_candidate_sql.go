package athena

import "fmt"

func BuildSalesCandidateRowsSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s
SELECT
	cast(%d AS bigint) AS owner_user_id,
	cast(business_date AS varchar) AS business_date,
	cast(hour_of_day AS varchar) AS hour_of_day,
	branch_id,
	product_no,
	cast(order_type_id AS varchar) AS order_type_id,
	cast(payment_type_id AS varchar) AS payment_type_id,
	cast(qty_milli AS varchar) AS qty_milli,
	cast(gross_sales_milli AS varchar) AS gross_sales_milli,
	cast(discount_milli AS varchar) AS discount_milli,
	cast(surcharge_milli AS varchar) AS surcharge_milli,
	cast(net_sales_milli AS varchar) AS net_sales_milli,
	cast(sales_ex_tax_milli AS varchar) AS sales_ex_tax_milli,
	cast(included_tax_milli AS varchar) AS included_tax_milli,
	cast(excluded_tax_milli AS varchar) AS excluded_tax_milli,
	cast(tax_milli AS varchar) AS tax_milli
FROM status_dedup_final_aggregation
ORDER BY business_date, hour_of_day, branch_id, product_no, order_type_id, payment_type_id`, buildPreviewCTE(window), ownerUserID)
}

func BuildSalesSourceMetricsSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s,
source_metrics AS (
	SELECT
		cast(count(*) AS bigint) AS row_count,
		cast(coalesce(sum(gross_sales_milli), 0) AS bigint) AS gross_sales_milli,
		cast(coalesce(sum(discount_milli), 0) AS bigint) AS discount_milli,
		cast(coalesce(sum(surcharge_milli), 0) AS bigint) AS surcharge_milli,
		cast(coalesce(sum(net_sales_milli), 0) AS bigint) AS net_sales_milli,
		cast(coalesce(sum(sales_ex_tax_milli), 0) AS bigint) AS sales_ex_tax_milli,
		cast(coalesce(sum(tax_milli), 0) AS bigint) AS tax_milli,
		cast(coalesce(sum(included_tax_milli), 0) AS bigint) AS included_tax_milli,
		cast(coalesce(sum(excluded_tax_milli), 0) AS bigint) AS excluded_tax_milli,
		cast(coalesce(sum(qty_milli), 0) AS bigint) AS qty_milli
	FROM status_dedup_final_aggregation
),
source_item_control AS (
	SELECT cast(count(*) AS bigint) AS item_count
	FROM status_dedup_line_allocations
),
source_status_gate AS (
	SELECT cast(count(*) AS bigint) AS status_1_rows
	FROM orders_sales_candidate
)
SELECT
	cast(%d AS varchar) AS owner_user_id,
	cast(date '%s' AS varchar) AS sale_period,
	cast(sm.row_count AS varchar) AS row_count,
	cast(sm.gross_sales_milli AS varchar) AS gross_sales_milli,
	cast(sm.discount_milli AS varchar) AS discount_milli,
	cast(sm.surcharge_milli AS varchar) AS surcharge_milli,
	cast(sm.net_sales_milli AS varchar) AS net_sales_milli,
	cast(sm.sales_ex_tax_milli AS varchar) AS sales_ex_tax_milli,
	cast(sm.tax_milli AS varchar) AS tax_milli,
	cast(sm.included_tax_milli AS varchar) AS included_tax_milli,
	cast(sm.excluded_tax_milli AS varchar) AS excluded_tax_milli,
	cast(sm.qty_milli AS varchar) AS qty_milli,
	cast(sic.item_count AS varchar) AS item_count,
	cast(ssg.status_1_rows AS varchar) AS status_1_rows,
	cast(0 AS varchar) AS non_status_1_rows,
	cast(ssg.status_1_rows AS varchar) AS latest_status_rows
FROM source_metrics sm
CROSS JOIN source_item_control sic
CROSS JOIN source_status_gate ssg`, buildPreviewCTE(window), ownerUserID, window.StartDate.Format(dateLayout))
}
