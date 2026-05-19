package athena

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	appconfig "ia-analyses-db/internal/config"
	"ia-analyses-db/internal/postgres"
	"ia-analyses-db/internal/sales"
	"ia-analyses-db/internal/validation"
)

type SalesCandidateProvider struct {
	service          *Service
	validationReader *postgres.SalesValidationReader
}

func NewSalesCandidateProvider(ctx context.Context, athenaCfg appconfig.AthenaConfig, db *sql.DB) (*SalesCandidateProvider, error) {
	service, err := NewService(ctx, athenaCfg)
	if err != nil {
		return nil, err
	}

	return &SalesCandidateProvider{
		service:          service,
		validationReader: postgres.NewSalesValidationReader(db),
	}, nil
}

func (provider *SalesCandidateProvider) BuildDayCandidate(ctx context.Context, scope sales.DayScope) (sales.CandidateBuildResult, error) {
	window := QueryWindow{
		OwnerUserKey: scope.OwnerUserKey,
		StartDate:    scope.BusinessDate,
		EndDate:      scope.BusinessDate,
		PreviewLimit: defaultPreviewLimit,
	}

	rows, err := provider.loadCandidateRows(ctx, window, scope.OwnerUserID)
	if err != nil {
		return sales.CandidateBuildResult{}, err
	}

	sourceMetrics, err := provider.loadSourceMetrics(ctx, window, scope.OwnerUserID)
	if err != nil {
		return sales.CandidateBuildResult{}, err
	}

	candidateMetrics := sales.BuildCandidateMetricsFromRows(scope, rows, sourceMetrics.ItemCount)

	dimensionGate, err := provider.validationReader.EvaluateDayCandidate(ctx, scope, rows, sourceMetrics)
	if err != nil {
		return sales.CandidateBuildResult{}, err
	}

	negativeSchemaGate, err := provider.validationReader.LoadNegativeSchemaGate(ctx)
	if err != nil {
		return sales.CandidateBuildResult{}, err
	}

	return sales.CandidateBuildResult{
		HasSourceMetrics:    true,
		SourceMetrics:       sourceMetrics,
		HasCandidateMetrics: true,
		CandidateMetrics:    candidateMetrics,
		Rows:                rows,
		DimensionGate:       dimensionGate,
		NegativeSchemaGate:  negativeSchemaGate,
	}, nil
}

func (provider *SalesCandidateProvider) LoadPostInsertTargetMetrics(ctx context.Context, scope sales.DayScope) (validation.MetricsSnapshot, error) {
	return provider.validationReader.LoadPersistedTargetMetrics(ctx, scope)
}

func (provider *SalesCandidateProvider) loadCandidateRows(ctx context.Context, window QueryWindow, ownerUserID int64) ([]sales.FactRow, error) {
	queryExecutionID, _, err := provider.service.runQuery(ctx, BuildSalesCandidateRowsSQL(window, ownerUserID))
	if err != nil {
		return nil, fmt.Errorf("run sales candidate query: %w", err)
	}

	rows, err := provider.service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read sales candidate rows: %w", err)
	}

	result := make([]sales.FactRow, 0, len(rows))
	for _, row := range rows {
		businessDate, err := time.Parse(dateLayout, row["business_date"])
		if err != nil {
			return nil, fmt.Errorf("parse candidate business_date: %w", err)
		}

		result = append(result, sales.FactRow{
			OwnerUserID:      mustParseInt64(row["owner_user_id"]),
			BusinessDate:     businessDate,
			HourOfDay:        mustParseInt16(row["hour_of_day"]),
			BranchID:         row["branch_id"],
			ProductNo:        row["product_no"],
			OrderTypeID:      mustParseInt16(row["order_type_id"]),
			PaymentTypeID:    mustParseInt16(row["payment_type_id"]),
			QtyMilli:         mustParseInt64(row["qty_milli"]),
			GrossSalesMilli:  mustParseInt64(row["gross_sales_milli"]),
			DiscountMilli:    mustParseInt64(row["discount_milli"]),
			SurchargeMilli:   mustParseInt64(row["surcharge_milli"]),
			NetSalesMilli:    mustParseInt64(row["net_sales_milli"]),
			SalesExTaxMilli:  mustParseInt64(row["sales_ex_tax_milli"]),
			IncludedTaxMilli: mustParseInt64(row["included_tax_milli"]),
			ExcludedTaxMilli: mustParseInt64(row["excluded_tax_milli"]),
			TaxMilli:         mustParseInt64(row["tax_milli"]),
		})
	}

	return result, nil
}

func (provider *SalesCandidateProvider) loadSourceMetrics(ctx context.Context, window QueryWindow, ownerUserID int64) (validation.MetricsSnapshot, error) {
	queryExecutionID, _, err := provider.service.runQuery(ctx, BuildSalesSourceMetricsSQL(window, ownerUserID))
	if err != nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("run sales source metrics query: %w", err)
	}

	rows, err := provider.service.readRows(ctx, queryExecutionID, 1)
	if err != nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("read sales source metrics rows: %w", err)
	}
	if len(rows) == 0 {
		return validation.MetricsSnapshot{}, fmt.Errorf("sales source metrics returned no rows")
	}

	row := rows[0]
	salePeriod, err := time.Parse(dateLayout, row["sale_period"])
	if err != nil {
		return validation.MetricsSnapshot{}, fmt.Errorf("parse sales source sale_period: %w", err)
	}

	itemCount := mustParseInt64(row["item_count"])
	status1Rows := mustParseInt64(row["status_1_rows"])
	nonStatus1Rows := mustParseInt64(row["non_status_1_rows"])
	latestStatusRows := mustParseInt64(row["latest_status_rows"])

	return validation.MetricsSnapshot{
		OwnerUserID:      mustParseInt64(row["owner_user_id"]),
		SalePeriod:       salePeriod,
		RowCount:         mustParseInt64(row["row_count"]),
		GrossSalesMilli:  mustParseInt64(row["gross_sales_milli"]),
		DiscountMilli:    mustParseInt64(row["discount_milli"]),
		SurchargeMilli:   mustParseInt64(row["surcharge_milli"]),
		NetSalesMilli:    mustParseInt64(row["net_sales_milli"]),
		SalesExTaxMilli:  mustParseInt64(row["sales_ex_tax_milli"]),
		TaxMilli:         mustParseInt64(row["tax_milli"]),
		IncludedTaxMilli: mustParseInt64(row["included_tax_milli"]),
		ExcludedTaxMilli: mustParseInt64(row["excluded_tax_milli"]),
		QtyMilli:         mustParseInt64(row["qty_milli"]),
		ItemCount:        &itemCount,
		Status1Rows:      &status1Rows,
		NonStatus1Rows:   &nonStatus1Rows,
		LatestStatusRows: &latestStatusRows,
	}, nil
}

func mustParseInt16(raw string) int16 {
	parsed, err := strconv.ParseInt(raw, 10, 16)
	if err != nil {
		panic(fmt.Sprintf("parse int16 from %q: %v", raw, err))
	}

	return int16(parsed)
}
