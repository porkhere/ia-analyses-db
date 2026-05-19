package athena

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
)

func RenderPreviewTable(rows []PreviewRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\thour_of_day\tbranch_id\tproduct_no\torder_type_id\tpayment_type_id\tqty_milli\tgross_sales_milli\tdiscount_milli\tsurcharge_milli\tnet_sales_milli\tsales_ex_tax_milli\ttax_milli")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.BusinessDate,
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
			row.TaxMilli,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderQueryMetricTable(metrics []QueryMetric) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "dataset\trow_count\tquery_scanned_mb\tengine_execution_sec\tquery_execution_id")

	for _, metric := range metrics {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%.2f\t%.2f\t%s\n",
			metric.Label,
			groupInt64(metric.RowCount),
			metric.DataScannedMB,
			metric.EngineExecutionSec,
			metric.QueryExecutionID,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderMappingSummaryTable(rows []MappingSummaryRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "canonical_id\tcanonical_code\tsource_rows\tdistinct_raw_values")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			row.CanonicalID,
			row.CanonicalCode,
			groupInt64(row.SourceRows),
			groupInt64(row.DistinctRawValues),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderReconciliationSummaryTable(rows []ReconciliationSummaryRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "metric\tsource_value\tpreview_value\tdelta_value")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			row.Metric,
			row.SourceValue,
			row.PreviewValue,
			row.DeltaValue,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderTaxReconciliationBreakdownTable(rows []TaxReconciliationBreakdownRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "metric\torder_count\tgross_sales_milli\tdiscount_milli\tsurcharge_milli\tnet_sales_milli\tincluded_tax_milli\tsales_ex_tax_milli\tnote")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Metric,
			groupInt64(row.OrderCount),
			groupInt64(row.GrossSalesMilli),
			groupInt64(row.DiscountMilli),
			groupInt64(row.SurchargeMilli),
			groupInt64(row.NetSalesMilli),
			groupInt64(row.IncludedTaxMilli),
			groupInt64(row.SalesExTaxMilli),
			row.Note,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderDebugMetricTable(rows []DebugMetricRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "metric\tvalue\tnote")

	for _, row := range rows {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", row.Metric, groupInt64(row.Value), row.Note)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderTaxDeltaSampleTable(rows []TaxDeltaSampleRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\torder_id\tsource_included_tax_milli\tallocated_included_tax_milli\tdelta_milli\titem_count\tallocation_denominator_milli\tstatus\tdestination")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.BusinessDate,
			row.OrderID,
			groupInt64(row.SourceIncludedTaxMilli),
			groupInt64(row.AllocatedIncludedTaxMilli),
			groupInt64(row.DeltaMilli),
			groupInt64(row.ItemCount),
			groupInt64(row.AllocationDenominatorMilli),
			row.Status,
			row.Destination,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderTopTaxDeltaOrderTrace(rows []OrderTraceRow) string {
	if len(rows) == 0 {
		return ""
	}

	type traceGroup struct {
		TraceRank string
		OrderID   string
		Rows      []OrderTraceRow
	}

	groups := make([]traceGroup, 0)
	for _, row := range rows {
		if len(groups) == 0 || groups[len(groups)-1].OrderID != row.OrderID {
			groups = append(groups, traceGroup{TraceRank: row.TraceRank, OrderID: row.OrderID, Rows: []OrderTraceRow{row}})
			continue
		}
		groups[len(groups)-1].Rows = append(groups[len(groups)-1].Rows, row)
	}

	var builder strings.Builder
	for index, group := range groups {
		if index > 0 {
			builder.WriteByte('\n')
		}

		fmt.Fprintf(&builder, "order_rank: %s\n", displayTraceValue(group.TraceRank))
		fmt.Fprintf(&builder, "order_id: %s\n", displayTraceValue(group.OrderID))

		if header := firstTraceRow(group.Rows, "order_header"); header != nil {
			renderTraceKeyValueSection(&builder, "order_header", [][2]string{
				{"business_date", displayTraceValue(header.BusinessDate)},
				{"status", displayTraceValue(header.Status)},
				{"destination", displayTraceValue(header.Destination)},
				{"normalized_order_type_id", displayTraceValue(header.NormalizedOrderTypeID)},
				{"branch_id", displayTraceValue(header.BranchID)},
				{"total_milli", displayTraceValue(header.TotalMilli)},
				{"item_subtotal_milli", displayTraceValue(header.ItemSubtotalMilli)},
				{"discount_milli", displayTraceValue(header.DiscountMilli)},
				{"surcharge_milli", displayTraceValue(header.SurchargeMilli)},
				{"included_tax_milli", displayTraceValue(header.IncludedTaxMilli)},
				{"tax_subtotal_milli", displayTraceValue(header.TaxSubtotalMilli)},
				{"transaction_voided", displayTraceValue(header.TransactionVoided)},
				{"void_sale_period", displayTraceValue(header.VoidSalePeriod)},
			})
		}

		paymentRows := filterTraceRows(group.Rows, "payment_trace")
		if len(paymentRows) > 0 {
			builder.WriteString("payment_trace:\n")
			writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "payment_row_id\tpayment_name\tpayment_amount_milli\tnormalized_payment_type_id\tfinal_payment_type_id\tis_mixed\thas_offsetting_payments\tnote")
			for _, row := range paymentRows {
				fmt.Fprintf(
					writer,
					"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					displayTraceValue(row.PaymentRowID),
					displayTraceValue(row.PaymentName),
					displayTraceValue(row.PaymentAmountMilli),
					displayTraceValue(row.NormalizedPaymentTypeID),
					displayTraceValue(row.FinalPaymentTypeID),
					displayTraceFlag(row.IsMixed),
					displayTraceFlag(row.HasOffsettingPayments),
					displayTraceValue(row.Note),
				)
			}
			_ = writer.Flush()
		}

		itemRows := filterTraceRows(group.Rows, "item_allocation_trace")
		if len(itemRows) > 0 {
			builder.WriteString("item_allocation_trace:\n")
			writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "item_id\tproduct_no\tproduct_name\tcurrent_qty_milli\tcurrent_subtotal_milli\tcurrent_discount_milli\tcurrent_surcharge_milli\traw_item_included_tax_milli\tallocation_denominator_milli\tallocation_ratio\tallocated_included_tax_milli\tallocated_discount_milli\tallocated_surcharge_milli\tnote")
			for _, row := range itemRows {
				fmt.Fprintf(
					writer,
					"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					displayTraceValue(row.ItemID),
					displayTraceValue(row.ProductNo),
					displayTraceValue(row.ProductName),
					displayTraceValue(row.CurrentQtyMilli),
					displayTraceValue(row.CurrentSubtotalMilli),
					displayTraceValue(row.CurrentDiscountMilli),
					displayTraceValue(row.CurrentSurchargeMilli),
					displayTraceValue(row.RawItemIncludedTaxMilli),
					displayTraceValue(row.AllocationDenominatorMilli),
					displayTraceValue(row.AllocationRatio),
					displayTraceValue(row.AllocatedIncludedTaxMilli),
					displayTraceValue(row.AllocatedDiscountMilli),
					displayTraceValue(row.AllocatedSurchargeMilli),
					displayTraceValue(row.Note),
				)
			}
			_ = writer.Flush()
		}

		if summary := firstTraceRow(group.Rows, "allocation_summary"); summary != nil {
			renderTraceKeyValueSection(&builder, "allocation_summary", [][2]string{
				{"source_order_included_tax_milli", displayTraceValue(summary.SourceOrderIncludedTaxMilli)},
				{"sum_allocated_included_tax_milli", displayTraceValue(summary.SumAllocatedIncludedTaxMilli)},
				{"delta_milli", displayTraceValue(summary.DeltaMilli)},
				{"source_order_net_milli", displayTraceValue(summary.SourceOrderNetMilli)},
				{"sum_allocated_net_milli", displayTraceValue(summary.SumAllocatedNetMilli)},
				{"net_delta_milli", displayTraceValue(summary.NetDeltaMilli)},
				{"item_line_count", displayTraceValue(summary.ItemLineCount)},
				{"product_group_count", displayTraceValue(summary.ProductGroupCount)},
				{"payment_row_count", displayTraceValue(summary.PaymentRowCount)},
				{"normalized_payment_type_count", displayTraceValue(summary.NormalizedPaymentTypeCount)},
				{"final_payment_type_id", displayTraceValue(summary.FinalPaymentTypeID)},
				{"is_mixed", displayTraceFlag(summary.IsMixed)},
				{"has_offsetting_payments", displayTraceFlag(summary.HasOffsettingPayments)},
				{"note", displayTraceValue(summary.Note)},
			})
		}

		if joinCheck := firstTraceRow(group.Rows, "join_duplication_check"); joinCheck != nil {
			renderTraceKeyValueSection(&builder, "join_duplication_check", [][2]string{
				{"raw_item_row_count", displayTraceValue(joinCheck.RawItemRowCount)},
				{"grouped_item_row_count", displayTraceValue(joinCheck.GroupedItemRowCount)},
				{"raw_payment_row_count", displayTraceValue(joinCheck.RawPaymentRowCount)},
				{"payment_after_order_aggregation_row_count", displayTraceValue(joinCheck.PaymentAfterOrderAggregationRowCount)},
				{"final_joined_row_count", displayTraceValue(joinCheck.FinalJoinedRowCount)},
				{"raw_addition_row_count", displayTraceValue(joinCheck.RawAdditionRowCount)},
				{"additions_after_order_aggregation_row_count", displayTraceValue(joinCheck.AdditionsAfterOrderAggregationRowCount)},
				{"note", displayTraceValue(joinCheck.Note)},
			})
		}
	}

	return builder.String()
}

func RenderDuplicateOrderTrace(rows []DuplicateOrderTraceRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "trace_rank\traw_row_rank\tduplicate_row_count\tt_open_date\torder_id\tstatus\tdestination\tbranch_id\tbranch\ttotal_milli\titem_subtotal_milli\tdiscount_subtotal_milli\tpayment_subtotal_milli\tincluded_tax_subtotal_milli\ttax_subtotal_milli\titem_surcharge_subtotal_milli\ttrans_surcharge_subtotal_milli\ttransaction_created\ttransaction_submitted\ttransaction_voided\tvoid_sale_period\tcreated\tmodified\tsequence\tsale_period\tshift_number\ttop_sample_status\ttop_sample_destination")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			displayTraceValue(row.TraceRank),
			displayTraceValue(row.RawRowRank),
			displayTraceValue(row.DuplicateRowCount),
			displayTraceValue(row.OpenDate),
			displayTraceValue(row.OrderID),
			displayTraceValue(row.Status),
			displayTraceValue(row.Destination),
			displayTraceValue(row.BranchID),
			displayTraceValue(row.Branch),
			displayTraceValue(row.TotalMilli),
			displayTraceValue(row.ItemSubtotalMilli),
			displayTraceValue(row.DiscountSubtotalMilli),
			displayTraceValue(row.PaymentSubtotalMilli),
			displayTraceValue(row.IncludedTaxSubtotalMilli),
			displayTraceValue(row.TaxSubtotalMilli),
			displayTraceValue(row.ItemSurchargeSubtotalMilli),
			displayTraceValue(row.TransSurchargeSubtotalMilli),
			displayTraceValue(row.TransactionCreated),
			displayTraceValue(row.TransactionSubmitted),
			displayTraceValue(row.TransactionVoided),
			displayTraceValue(row.VoidSalePeriod),
			displayTraceValue(row.Created),
			displayTraceValue(row.Modified),
			displayTraceValue(row.Sequence),
			displayTraceValue(row.SalePeriod),
			displayTraceValue(row.ShiftNumber),
			displayTraceValue(row.TopSampleStatus),
			displayTraceValue(row.TopSampleDestination),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderStatusDedupTopTaxDeltaTable(rows []StatusDedupTopTaxDeltaRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "order_id\tt_open_date\traw_order_row_count\traw_status_list\tselected_sales_status\tselected_sales_transaction_submitted\tselected_sales_modified\tselected_sales_payment_subtotal_milli\tsource_status_1_included_tax_milli\tcurrent_allocated_included_tax_milli\tstatus_dedup_allocated_included_tax_milli\tcurrent_delta_milli\tstatus_dedup_delta_milli\tsource_status_1_net_milli\tcurrent_allocated_net_milli\tstatus_dedup_allocated_net_milli\tcurrent_net_delta_milli\tstatus_dedup_net_delta_milli")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			displayTraceValue(row.OrderID),
			displayTraceValue(row.BusinessDate),
			groupInt64(row.RawOrderRowCount),
			displayTraceValue(row.RawStatusList),
			displayTraceValue(row.SelectedSalesStatus),
			displayTraceValue(row.SelectedSalesTransactionSubmitted),
			displayTraceValue(row.SelectedSalesModified),
			groupInt64(row.SelectedSalesPaymentSubtotalMilli),
			groupInt64(row.SourceStatus1IncludedTaxMilli),
			groupInt64(row.CurrentAllocatedIncludedTaxMilli),
			groupInt64(row.StatusDedupAllocatedIncludedTaxMilli),
			groupInt64(row.CurrentDeltaMilli),
			groupInt64(row.StatusDedupDeltaMilli),
			groupInt64(row.SourceStatus1NetMilli),
			groupInt64(row.CurrentAllocatedNetMilli),
			groupInt64(row.StatusDedupAllocatedNetMilli),
			groupInt64(row.CurrentNetDeltaMilli),
			groupInt64(row.StatusDedupNetDeltaMilli),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func RenderStatusExcludedSummaryTable(rows []StatusExcludedSummaryRow) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "status\torder_keys\traw_rows\ttotal_milli\tpayment_subtotal_milli\tincluded_tax_milli\tdestination_distribution\tvoided_count\tsubmitted_null_count")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			displayTraceValue(row.Status),
			groupInt64(row.OrderKeys),
			groupInt64(row.RawRows),
			groupInt64(row.TotalMilli),
			groupInt64(row.PaymentSubtotalMilli),
			groupInt64(row.IncludedTaxMilli),
			displayTraceValue(row.DestinationDistribution),
			groupInt64(row.VoidedCount),
			groupInt64(row.SubmittedNullCount),
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func filterTraceRows(rows []OrderTraceRow, section string) []OrderTraceRow {
	result := make([]OrderTraceRow, 0)
	for _, row := range rows {
		if row.Section == section {
			result = append(result, row)
		}
	}
	return result
}

func firstTraceRow(rows []OrderTraceRow, section string) *OrderTraceRow {
	for index := range rows {
		if rows[index].Section == section {
			return &rows[index]
		}
	}
	return nil
}

func renderTraceKeyValueSection(builder *strings.Builder, title string, pairs [][2]string) {
	builder.WriteString(title)
	builder.WriteString(":\n")
	writer := tabwriter.NewWriter(builder, 0, 0, 2, ' ', 0)
	for _, pair := range pairs {
		fmt.Fprintf(writer, "%s\t%s\n", pair[0], pair[1])
	}
	_ = writer.Flush()
}

func displayTraceValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return groupInt64(parsed)
	}
	return trimmed
}

func displayTraceFlag(value string) string {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case "1":
		return "yes"
	case "0":
		return "no"
	case "":
		return "-"
	default:
		return trimmed
	}
}

func groupInt64(value int64) string {
	text := fmt.Sprintf("%d", value)
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign = "-"
		text = strings.TrimPrefix(text, "-")
	}

	if len(text) <= 3 {
		return sign + text
	}

	var builder strings.Builder
	builder.WriteString(sign)
	prefix := len(text) % 3
	if prefix > 0 {
		builder.WriteString(text[:prefix])
		if len(text) > prefix {
			builder.WriteByte(',')
		}
	}

	for index := prefix; index < len(text); index += 3 {
		builder.WriteString(text[index : index+3])
		if index+3 < len(text) {
			builder.WriteByte(',')
		}
	}

	return builder.String()
}
