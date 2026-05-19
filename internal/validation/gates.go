package validation

import (
	"fmt"
	"time"
)

type TargetScope string

const (
	TargetScopePersistedFact      TargetScope = "persisted_fact"
	TargetScopePreInsertCandidate TargetScope = "pre_insert_candidate"
)

type MetricsSnapshot struct {
	OwnerUserID               int64
	SalePeriod                time.Time
	RowCount                  int64
	GrossSalesMilli           int64
	DiscountMilli             int64
	SurchargeMilli            int64
	NetSalesMilli             int64
	SalesExTaxMilli           int64
	TaxMilli                  int64
	IncludedTaxMilli          int64
	ExcludedTaxMilli          int64
	QtyMilli                  int64
	ItemCount                 *int64
	Status1Rows               *int64
	NonStatus1Rows            *int64
	LatestStatusRows          *int64
	WarningRoundingDeltaMilli *int64
	WarningGateNote           string
}

type MetricsComparison struct {
	OwnerUserID               int64
	SalePeriod                time.Time
	TargetScope               TargetScope
	RowCountDelta             int64
	GrossSalesMilliDelta      int64
	DiscountMilliDelta        int64
	SurchargeMilliDelta       int64
	NetSalesMilliDelta        int64
	SalesExTaxMilliDelta      int64
	TaxMilliDelta             int64
	IncludedTaxMilliDelta     int64
	ExcludedTaxMilliDelta     int64
	QtyMilliDelta             int64
	ItemCountDelta            *int64
	WarningRoundingDeltaMilli *int64
	WarningGateNote           string
	WarningGateFailed         bool
	HardGateFailed            bool
	FailureReasons            []string
}

type DimensionGateResult struct {
	OwnerUserID                         int64
	SalePeriod                          time.Time
	ProductDimMissCount                 int64
	BranchDimMissCount                  int64
	OrderTypeDimMissCount               int64
	PaymentTypeDimMissCount             int64
	BusinessDateNotEqualSalePeriodCount int64
	NonStatus1Count                     int64
	NotLatestStatusCount                int64
	WarningGateNote                     string
}

type NegativeSchemaGateResult struct {
	ForbiddenColumnCount int64
	ForbiddenColumnNames string
	WarningGateNote      string
}

type PreInsertReport struct {
	SourceMetrics      MetricsSnapshot
	CandidateMetrics   MetricsSnapshot
	MetricsComparison  MetricsComparison
	DimensionGate      DimensionGateResult
	NegativeSchemaGate NegativeSchemaGateResult
	HardGateFailed     bool
	FailureReasons     []string
}

type PostInsertReport struct {
	SourceMetrics          MetricsSnapshot
	PersistedTargetMetrics MetricsSnapshot
	MetricsComparison      MetricsComparison
	HardGateFailed         bool
	FailureReasons         []string
}

func CompareMetrics(source MetricsSnapshot, target MetricsSnapshot, scope TargetScope) MetricsComparison {
	comparison := MetricsComparison{
		OwnerUserID:           source.OwnerUserID,
		SalePeriod:            source.SalePeriod,
		TargetScope:           scope,
		RowCountDelta:         source.RowCount - target.RowCount,
		GrossSalesMilliDelta:  source.GrossSalesMilli - target.GrossSalesMilli,
		DiscountMilliDelta:    source.DiscountMilli - target.DiscountMilli,
		SurchargeMilliDelta:   source.SurchargeMilli - target.SurchargeMilli,
		NetSalesMilliDelta:    source.NetSalesMilli - target.NetSalesMilli,
		SalesExTaxMilliDelta:  source.SalesExTaxMilli - target.SalesExTaxMilli,
		TaxMilliDelta:         source.TaxMilli - target.TaxMilli,
		IncludedTaxMilliDelta: source.IncludedTaxMilli - target.IncludedTaxMilli,
		ExcludedTaxMilliDelta: source.ExcludedTaxMilli - target.ExcludedTaxMilli,
		QtyMilliDelta:         source.QtyMilli - target.QtyMilli,
	}

	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("row_count_delta", comparison.RowCountDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("gross_sales_milli_delta", comparison.GrossSalesMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("discount_milli_delta", comparison.DiscountMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("surcharge_milli_delta", comparison.SurchargeMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("net_sales_milli_delta", comparison.NetSalesMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("sales_ex_tax_milli_delta", comparison.SalesExTaxMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("tax_milli_delta", comparison.TaxMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("included_tax_milli_delta", comparison.IncludedTaxMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("excluded_tax_milli_delta", comparison.ExcludedTaxMilliDelta)...)
	comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("qty_milli_delta", comparison.QtyMilliDelta)...)

	if scope == TargetScopePreInsertCandidate {
		switch {
		case source.ItemCount == nil || target.ItemCount == nil:
			comparison.FailureReasons = append(comparison.FailureReasons, "item_count_delta is required for pre_insert_candidate compare")
		case source.ItemCount != nil && target.ItemCount != nil:
			delta := *source.ItemCount - *target.ItemCount
			comparison.ItemCountDelta = &delta
			comparison.FailureReasons = append(comparison.FailureReasons, exactMatchFailure("item_count_delta", delta)...)
		}
	}

	comparison.WarningRoundingDeltaMilli = firstNonNilInt64(source.WarningRoundingDeltaMilli, target.WarningRoundingDeltaMilli)
	comparison.WarningGateNote = firstNonEmpty(source.WarningGateNote, target.WarningGateNote)
	if comparison.WarningRoundingDeltaMilli != nil && abs64(*comparison.WarningRoundingDeltaMilli) > 1 {
		comparison.WarningGateFailed = true
	}

	comparison.HardGateFailed = len(comparison.FailureReasons) > 0

	return comparison
}

func BuildPreInsertReport(source MetricsSnapshot, candidate MetricsSnapshot, dimensionGate DimensionGateResult, negativeSchemaGate NegativeSchemaGateResult) PreInsertReport {
	report := PreInsertReport{
		SourceMetrics:      source,
		CandidateMetrics:   candidate,
		MetricsComparison:  CompareMetrics(source, candidate, TargetScopePreInsertCandidate),
		DimensionGate:      dimensionGate,
		NegativeSchemaGate: negativeSchemaGate,
	}

	report.FailureReasons = append(report.FailureReasons, report.MetricsComparison.FailureReasons...)
	report.FailureReasons = append(report.FailureReasons, dimensionGateFailures(dimensionGate)...)
	report.FailureReasons = append(report.FailureReasons, negativeSchemaGateFailures(negativeSchemaGate)...)
	report.HardGateFailed = len(report.FailureReasons) > 0

	return report
}

func BuildPostInsertReport(source MetricsSnapshot, target MetricsSnapshot) PostInsertReport {
	report := PostInsertReport{
		SourceMetrics:          source,
		PersistedTargetMetrics: target,
		MetricsComparison:      CompareMetrics(source, target, TargetScopePersistedFact),
	}

	report.FailureReasons = append(report.FailureReasons, report.MetricsComparison.FailureReasons...)
	report.HardGateFailed = len(report.FailureReasons) > 0

	return report
}

func exactMatchFailure(name string, delta int64) []string {
	if delta == 0 {
		return nil
	}

	return []string{fmt.Sprintf("%s=%d", name, delta)}
}

func dimensionGateFailures(result DimensionGateResult) []string {
	var failures []string

	failures = appendCountFailure(failures, "product_dim_miss_count", result.ProductDimMissCount)
	failures = appendCountFailure(failures, "branch_dim_miss_count", result.BranchDimMissCount)
	failures = appendCountFailure(failures, "order_type_dim_miss_count", result.OrderTypeDimMissCount)
	failures = appendCountFailure(failures, "payment_type_dim_miss_count", result.PaymentTypeDimMissCount)
	failures = appendCountFailure(failures, "business_date_not_equal_sale_period_count", result.BusinessDateNotEqualSalePeriodCount)
	failures = appendCountFailure(failures, "non_status_1_count", result.NonStatus1Count)
	failures = appendCountFailure(failures, "not_latest_status_count", result.NotLatestStatusCount)

	return failures
}

func negativeSchemaGateFailures(result NegativeSchemaGateResult) []string {
	var failures []string

	failures = appendCountFailure(failures, "forbidden_column_count", result.ForbiddenColumnCount)
	if result.ForbiddenColumnCount > 0 && result.ForbiddenColumnNames != "" {
		failures = append(failures, fmt.Sprintf("forbidden_column_names=%s", result.ForbiddenColumnNames))
	}

	return failures
}

func appendCountFailure(failures []string, name string, value int64) []string {
	if value == 0 {
		return failures
	}

	return append(failures, fmt.Sprintf("%s=%d", name, value))
}

func firstNonNilInt64(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func abs64(value int64) int64 {
	if value < 0 {
		return -value
	}

	return value
}
