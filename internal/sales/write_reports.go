package sales

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"ia-analyses-db/internal/validation"
)

type DayValidationDetail struct {
	BusinessDate        string
	SourceMetrics       validation.MetricsSnapshot
	CandidateMetrics    validation.MetricsSnapshot
	PreInsertReport     validation.PreInsertReport
	PostInsertReport    validation.PostInsertReport
	HasPostInsertReport bool
	CandidateRowSample  []FactRow
	CandidateRowCount   int
}

func CloneFactRowSample(rows []FactRow, limit int) []FactRow {
	if limit <= 0 || len(rows) <= limit {
		return append([]FactRow(nil), rows...)
	}

	return append([]FactRow(nil), rows[:limit]...)
}

func BuildCandidateMetricsFromRows(scope DayScope, rows []FactRow, itemCount *int64) validation.MetricsSnapshot {
	result := validation.MetricsSnapshot{
		OwnerUserID: scope.OwnerUserID,
		SalePeriod:  scope.BusinessDate,
		ItemCount:   itemCount,
	}

	for _, row := range rows {
		result.RowCount++
		result.GrossSalesMilli += row.GrossSalesMilli
		result.DiscountMilli += row.DiscountMilli
		result.SurchargeMilli += row.SurchargeMilli
		result.NetSalesMilli += row.NetSalesMilli
		result.SalesExTaxMilli += row.SalesExTaxMilli
		result.TaxMilli += row.TaxMilli
		result.IncludedTaxMilli += row.IncludedTaxMilli
		result.ExcludedTaxMilli += row.ExcludedTaxMilli
		result.QtyMilli += row.QtyMilli
	}

	return result
}

func RenderMetricsSnapshotTable(details []DayValidationDetail, useCandidate bool) string {
	return renderMetricsSnapshotTable(details, func(detail DayValidationDetail) (validation.MetricsSnapshot, bool) {
		if useCandidate {
			return detail.CandidateMetrics, true
		}

		return detail.SourceMetrics, true
	})
}

func RenderPersistedTargetMetricsTable(details []DayValidationDetail) string {
	return renderMetricsSnapshotTable(details, func(detail DayValidationDetail) (validation.MetricsSnapshot, bool) {
		if !detail.HasPostInsertReport {
			return validation.MetricsSnapshot{}, false
		}

		return detail.PostInsertReport.PersistedTargetMetrics, true
	})
}

func renderMetricsSnapshotTable(details []DayValidationDetail, selectMetrics func(detail DayValidationDetail) (validation.MetricsSnapshot, bool)) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\trow_count\tgross_sales_milli\tdiscount_milli\tsurcharge_milli\tnet_sales_milli\tsales_ex_tax_milli\ttax_milli\tincluded_tax_milli\texcluded_tax_milli\tqty_milli\titem_count")

	for _, detail := range details {
		metrics, ok := selectMetrics(detail)
		if !ok {
			continue
		}

		fmt.Fprintf(
			writer,
			"%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			detail.BusinessDate,
			metrics.RowCount,
			metrics.GrossSalesMilli,
			metrics.DiscountMilli,
			metrics.SurchargeMilli,
			metrics.NetSalesMilli,
			metrics.SalesExTaxMilli,
			metrics.TaxMilli,
			metrics.IncludedTaxMilli,
			metrics.ExcludedTaxMilli,
			metrics.QtyMilli,
			optionalInt64(metrics.ItemCount),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderPostInsertCompareTable(details []DayValidationDetail) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\trow_count_delta\tgross_sales_milli_delta\tdiscount_milli_delta\tsurcharge_milli_delta\tnet_sales_milli_delta\tsales_ex_tax_milli_delta\ttax_milli_delta\tincluded_tax_milli_delta\texcluded_tax_milli_delta\tqty_milli_delta\thard_gate_failed")

	for _, detail := range details {
		if !detail.HasPostInsertReport {
			continue
		}

		compare := detail.PostInsertReport.MetricsComparison
		fmt.Fprintf(
			writer,
			"%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%t\n",
			detail.BusinessDate,
			compare.RowCountDelta,
			compare.GrossSalesMilliDelta,
			compare.DiscountMilliDelta,
			compare.SurchargeMilliDelta,
			compare.NetSalesMilliDelta,
			compare.SalesExTaxMilliDelta,
			compare.TaxMilliDelta,
			compare.IncludedTaxMilliDelta,
			compare.ExcludedTaxMilliDelta,
			compare.QtyMilliDelta,
			compare.HardGateFailed,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderPreInsertCompareTable(details []DayValidationDetail) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\trow_count_delta\tgross_sales_milli_delta\tdiscount_milli_delta\tsurcharge_milli_delta\tnet_sales_milli_delta\tsales_ex_tax_milli_delta\ttax_milli_delta\tincluded_tax_milli_delta\texcluded_tax_milli_delta\tqty_milli_delta\titem_count_delta\thard_gate_failed")

	for _, detail := range details {
		compare := detail.PreInsertReport.MetricsComparison
		fmt.Fprintf(
			writer,
			"%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%t\n",
			detail.BusinessDate,
			compare.RowCountDelta,
			compare.GrossSalesMilliDelta,
			compare.DiscountMilliDelta,
			compare.SurchargeMilliDelta,
			compare.NetSalesMilliDelta,
			compare.SalesExTaxMilliDelta,
			compare.TaxMilliDelta,
			compare.IncludedTaxMilliDelta,
			compare.ExcludedTaxMilliDelta,
			compare.QtyMilliDelta,
			optionalInt64(compare.ItemCountDelta),
			compare.HardGateFailed,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderPreInsertGateTable(details []DayValidationDetail) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\tproduct_dim_miss_count\tbranch_dim_miss_count\torder_type_dim_miss_count\tpayment_type_dim_miss_count\tbusiness_date_not_equal_sale_period_count\tnon_status_1_count\tnot_latest_status_count\tforbidden_column_count\thard_gate_failed\tfailure_reasons")

	for _, detail := range details {
		dimensionGate := detail.PreInsertReport.DimensionGate
		negativeGate := detail.PreInsertReport.NegativeSchemaGate
		fmt.Fprintf(
			writer,
			"%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%t\t%s\n",
			detail.BusinessDate,
			dimensionGate.ProductDimMissCount,
			dimensionGate.BranchDimMissCount,
			dimensionGate.OrderTypeDimMissCount,
			dimensionGate.PaymentTypeDimMissCount,
			dimensionGate.BusinessDateNotEqualSalePeriodCount,
			dimensionGate.NonStatus1Count,
			dimensionGate.NotLatestStatusCount,
			negativeGate.ForbiddenColumnCount,
			detail.PreInsertReport.HardGateFailed,
			strings.Join(detail.PreInsertReport.FailureReasons, "; "),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderCandidateRowSampleTable(details []DayValidationDetail, limit int) string {
	if limit <= 0 {
		limit = 10
	}

	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\towner_user_id\thour_of_day\tbranch_id\tproduct_no\torder_type_id\tpayment_type_id\tqty_milli\tgross_sales_milli\tdiscount_milli\tsurcharge_milli\tnet_sales_milli\tsales_ex_tax_milli\tincluded_tax_milli\texcluded_tax_milli\ttax_milli")

	for _, detail := range details {
		rows := detail.CandidateRowSample
		if len(rows) > limit {
			rows = rows[:limit]
		}
		for _, row := range rows {
			fmt.Fprintf(
				writer,
				"%s\t%d\t%02d\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
				row.BusinessDate.Format(syncDateLayout),
				row.OwnerUserID,
				row.HourOfDay,
				row.BranchID,
				row.ProductNo,
				row.OrderTypeID,
				row.PaymentTypeID,
				row.QtyMilli,
				row.GrossSalesMilli,
				row.DiscountMilli,
				row.SurchargeMilli,
				row.NetSalesMilli,
				row.SalesExTaxMilli,
				row.IncludedTaxMilli,
				row.ExcludedTaxMilli,
				row.TaxMilli,
			)
		}
	}

	_ = writer.Flush()

	return builder.String()
}

func optionalInt64(value *int64) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%d", *value)
}
