package salesdims

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

func RenderPlanSummaryTable(plan PlanResult) string {
	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "dataset\tplanned_upsert_count\tconflict_count")
	fmt.Fprintf(writer, "pos_product_dim\t%d\t%d\n", len(plan.ProductCandidates), plan.ProductConflictCount)
	fmt.Fprintf(writer, "pos_branch_dim\t%d\t%d\n", len(plan.BranchCandidates), plan.BranchConflictCount)
	_ = writer.Flush()
	return builder.String()
}

func RenderProductConflictTable(rows []ProductConflictSample) string {
	if len(rows) == 0 {
		return ""
	}

	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "product_no\tvariant_count\tchosen_product_name\tchosen_cate_no\tchosen_cate_name\tchosen_source_row_count\tchosen_last_seen_at\tsample_variants")
	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			row.ProductNo,
			row.VariantCount,
			row.ChosenProductName,
			row.ChosenCateNo,
			row.ChosenCateName,
			row.ChosenSourceRowCount,
			row.ChosenLastSeenAt,
			row.SampleVariants,
		)
	}
	_ = writer.Flush()
	return builder.String()
}

func RenderBranchConflictTable(rows []BranchConflictSample) string {
	if len(rows) == 0 {
		return ""
	}

	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "branch_id\tvariant_count\tchosen_branch_name\tchosen_group_code\tchosen_source_row_count\tchosen_last_seen_at\tsample_variants")
	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%d\t%s\t%s\t%d\t%s\t%s\n",
			row.BranchID,
			row.VariantCount,
			row.ChosenBranchName,
			row.ChosenGroupCode,
			row.ChosenSourceRowCount,
			row.ChosenLastSeenAt,
			row.SampleVariants,
		)
	}
	_ = writer.Flush()
	return builder.String()
}

func RenderApplySummaryTable(result ApplyResult) string {
	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "table\tapplied_upsert_count")
	fmt.Fprintf(writer, "pos_product_dim\t%d\n", result.ProductUpsertCount)
	fmt.Fprintf(writer, "pos_branch_dim\t%d\n", result.BranchUpsertCount)
	_ = writer.Flush()
	return builder.String()
}
