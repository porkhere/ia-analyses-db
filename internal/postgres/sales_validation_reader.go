package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/lib/pq"

	"ia-analyses-db/internal/sales"
	"ia-analyses-db/internal/validation"
)

var forbiddenSalesFactColumns = []string{
	"raw_payment_name",
	"raw_payment_memo1",
	"item_count",
	"void_milli",
	"refund_milli",
	"order_count",
	"completed_order_count",
	"void_order_count",
	"refund_order_count",
	"cancelled_order_count",
	"tr_date",
	"t_open_date",
	"void_sale_period",
	"order_num",
}

type SalesValidationReader struct {
	db *sql.DB
}

type metricsRowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewSalesValidationReader(db *sql.DB) *SalesValidationReader {
	return &SalesValidationReader{db: db}
}

func (reader *SalesValidationReader) EvaluateDayCandidate(ctx context.Context, scope sales.DayScope, rows []sales.FactRow, sourceMetrics validation.MetricsSnapshot) (validation.DimensionGateResult, error) {
	if reader == nil || reader.db == nil {
		return validation.DimensionGateResult{}, fmt.Errorf("postgres validation reader requires DB handle")
	}

	result := validation.DimensionGateResult{
		OwnerUserID: scope.OwnerUserID,
		SalePeriod:  scope.BusinessDate,
	}

	for _, row := range rows {
		if !row.BusinessDate.Equal(scope.BusinessDate) {
			result.BusinessDateNotEqualSalePeriodCount++
		}
	}

	if sourceMetrics.NonStatus1Rows != nil {
		result.NonStatus1Count = *sourceMetrics.NonStatus1Rows
	}

	if sourceMetrics.Status1Rows != nil && sourceMetrics.LatestStatusRows != nil && *sourceMetrics.LatestStatusRows < *sourceMetrics.Status1Rows {
		result.NotLatestStatusCount = *sourceMetrics.Status1Rows - *sourceMetrics.LatestStatusRows
	}

	productNos := uniqueStrings(func(row sales.FactRow) string { return row.ProductNo }, rows)
	branchIDs := uniqueStrings(func(row sales.FactRow) string { return row.BranchID }, rows)
	orderTypeIDs := uniqueInt16(func(row sales.FactRow) int16 { return row.OrderTypeID }, rows)
	paymentTypeIDs := uniqueInt16(func(row sales.FactRow) int16 { return row.PaymentTypeID }, rows)

	existingProductNos, err := reader.loadExistingStringKeys(ctx, `SELECT product_no FROM public.pos_product_dim WHERE owner_user_id = $1 AND product_no = ANY($2)`, scope.OwnerUserID, productNos)
	if err != nil {
		return validation.DimensionGateResult{}, fmt.Errorf("load product dim keys: %w", err)
	}

	existingBranchIDs, err := reader.loadExistingStringKeys(ctx, `SELECT branch_id FROM public.pos_branch_dim WHERE owner_user_id = $1 AND branch_id = ANY($2)`, scope.OwnerUserID, branchIDs)
	if err != nil {
		return validation.DimensionGateResult{}, fmt.Errorf("load branch dim keys: %w", err)
	}

	existingOrderTypeIDs, err := reader.loadExistingSmallintKeys(ctx, `SELECT id FROM public.pos_order_type_dim WHERE id = ANY($1)`, orderTypeIDs)
	if err != nil {
		return validation.DimensionGateResult{}, fmt.Errorf("load order type dim keys: %w", err)
	}

	existingPaymentTypeIDs, err := reader.loadExistingSmallintKeys(ctx, `SELECT id FROM public.pos_payment_type_dim WHERE id = ANY($1)`, paymentTypeIDs)
	if err != nil {
		return validation.DimensionGateResult{}, fmt.Errorf("load payment type dim keys: %w", err)
	}

	for _, row := range rows {
		if _, ok := existingProductNos[row.ProductNo]; !ok {
			result.ProductDimMissCount++
		}
		if _, ok := existingBranchIDs[row.BranchID]; !ok {
			result.BranchDimMissCount++
		}
		if _, ok := existingOrderTypeIDs[row.OrderTypeID]; !ok {
			result.OrderTypeDimMissCount++
		}
		if _, ok := existingPaymentTypeIDs[row.PaymentTypeID]; !ok {
			result.PaymentTypeDimMissCount++
		}
	}

	return result, nil
}

func (reader *SalesValidationReader) LoadNegativeSchemaGate(ctx context.Context) (validation.NegativeSchemaGateResult, error) {
	if reader == nil || reader.db == nil {
		return validation.NegativeSchemaGateResult{}, fmt.Errorf("postgres validation reader requires DB handle")
	}

	var result validation.NegativeSchemaGateResult
	if err := reader.db.QueryRowContext(
		ctx,
		`SELECT
			COUNT(*) AS forbidden_column_count,
			COALESCE(string_agg(column_name, ',' ORDER BY column_name), '') AS forbidden_column_names
		 FROM information_schema.columns
		 WHERE table_schema = 'public'
		   AND table_name = 'pos_sales_hourly_fact'
		   AND column_name = ANY($1)`,
		pq.Array(forbiddenSalesFactColumns),
	).Scan(&result.ForbiddenColumnCount, &result.ForbiddenColumnNames); err != nil {
		return validation.NegativeSchemaGateResult{}, fmt.Errorf("load negative schema gate: %w", err)
	}

	return result, nil
}

func (reader *SalesValidationReader) LoadPersistedTargetMetrics(ctx context.Context, scope sales.DayScope) (validation.MetricsSnapshot, error) {
	if reader == nil || reader.db == nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("postgres validation reader requires DB handle")
	}

	return loadPersistedTargetMetrics(ctx, reader.db, scope)
}

func loadPersistedTargetMetrics(ctx context.Context, queryer metricsRowQuerier, scope sales.DayScope) (validation.MetricsSnapshot, error) {
	if queryer == nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("persisted target metrics query handle is required")
	}

	result := validation.MetricsSnapshot{
		OwnerUserID: scope.OwnerUserID,
		SalePeriod:  scope.BusinessDate,
	}

	if err := queryer.QueryRowContext(
		ctx,
		`SELECT
			COUNT(*) AS row_count,
			COALESCE(SUM(gross_sales_milli), 0) AS gross_sales_milli,
			COALESCE(SUM(discount_milli), 0) AS discount_milli,
			COALESCE(SUM(surcharge_milli), 0) AS surcharge_milli,
			COALESCE(SUM(net_sales_milli), 0) AS net_sales_milli,
			COALESCE(SUM(sales_ex_tax_milli), 0) AS sales_ex_tax_milli,
			COALESCE(SUM(tax_milli), 0) AS tax_milli,
			COALESCE(SUM(included_tax_milli), 0) AS included_tax_milli,
			COALESCE(SUM(excluded_tax_milli), 0) AS excluded_tax_milli,
			COALESCE(SUM(qty_milli), 0) AS qty_milli
		 FROM public.pos_sales_hourly_fact
		 WHERE owner_user_id = $1
		   AND business_date = $2`,
		scope.OwnerUserID,
		scope.BusinessDate,
	).Scan(
		&result.RowCount,
		&result.GrossSalesMilli,
		&result.DiscountMilli,
		&result.SurchargeMilli,
		&result.NetSalesMilli,
		&result.SalesExTaxMilli,
		&result.TaxMilli,
		&result.IncludedTaxMilli,
		&result.ExcludedTaxMilli,
		&result.QtyMilli,
	); err != nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("load persisted target metrics: %w", err)
	}

	return result, nil
}

func (reader *SalesValidationReader) loadExistingStringKeys(ctx context.Context, sql string, ownerUserID int64, values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return map[string]struct{}{}, nil
	}

	rows, err := reader.db.QueryContext(ctx, sql, ownerUserID, pq.Array(values))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{}, len(values))
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		result[value] = struct{}{}
	}

	return result, rows.Err()
}

func (reader *SalesValidationReader) loadExistingSmallintKeys(ctx context.Context, sql string, values []int16) (map[int16]struct{}, error) {
	if len(values) == 0 {
		return map[int16]struct{}{}, nil
	}

	intValues := make([]int64, 0, len(values))
	for _, value := range values {
		intValues = append(intValues, int64(value))
	}

	rows, err := reader.db.QueryContext(ctx, sql, pq.Array(intValues))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int16]struct{}, len(values))
	for rows.Next() {
		var value int16
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		result[value] = struct{}{}
	}

	return result, rows.Err()
}

func uniqueStrings(selector func(sales.FactRow) string, rows []sales.FactRow) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	for _, row := range rows {
		value := selector(row)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func uniqueInt16(selector func(sales.FactRow) int16, rows []sales.FactRow) []int16 {
	seen := map[int16]struct{}{}
	result := make([]int16, 0)
	for _, row := range rows {
		value := selector(row)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
