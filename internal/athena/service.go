package athena

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	awsathena "github.com/aws/aws-sdk-go-v2/service/athena"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"

	appconfig "ia-analyses-db/internal/config"
)

type QueryMetric struct {
	Label              string
	RowCount           int64
	DataScannedMB      float64
	EngineExecutionSec float64
	QueryExecutionID   string
}

type PreviewRow struct {
	BusinessDate    string
	HourOfDay       string
	BranchID        string
	ProductNo       string
	OrderTypeID     string
	PaymentTypeID   string
	QtyMilli        string
	GrossSalesMilli string
	DiscountMilli   string
	SurchargeMilli  string
	NetSalesMilli   string
	SalesExTaxMilli string
	TaxMilli        string
}

type MappingSummaryRow struct {
	CanonicalID       string
	CanonicalCode     string
	SourceRows        int64
	DistinctRawValues int64
}

type ReconciliationSummaryRow struct {
	Metric       string
	SourceValue  string
	PreviewValue string
	DeltaValue   string
}

type TaxReconciliationBreakdownRow struct {
	Metric           string
	OrderCount       int64
	GrossSalesMilli  int64
	DiscountMilli    int64
	SurchargeMilli   int64
	NetSalesMilli    int64
	IncludedTaxMilli int64
	SalesExTaxMilli  int64
	Note             string
}

type DebugMetricRow struct {
	Metric string
	Value  int64
	Note   string
}

type TaxDeltaSampleRow struct {
	BusinessDate               string
	OrderID                    string
	SourceIncludedTaxMilli     int64
	AllocatedIncludedTaxMilli  int64
	DeltaMilli                 int64
	ItemCount                  int64
	AllocationDenominatorMilli int64
	Status                     string
	Destination                string
}

type OrderTraceRow struct {
	TraceRank                              string
	Section                                string
	RowOrder                               string
	BusinessDate                           string
	OrderID                                string
	Status                                 string
	Destination                            string
	NormalizedOrderTypeID                  string
	BranchID                               string
	TotalMilli                             string
	ItemSubtotalMilli                      string
	DiscountMilli                          string
	SurchargeMilli                         string
	IncludedTaxMilli                       string
	TaxSubtotalMilli                       string
	TransactionVoided                      string
	VoidSalePeriod                         string
	PaymentRowID                           string
	PaymentName                            string
	PaymentAmountMilli                     string
	NormalizedPaymentTypeID                string
	FinalPaymentTypeID                     string
	IsMixed                                string
	HasOffsettingPayments                  string
	ItemID                                 string
	ProductNo                              string
	ProductName                            string
	CurrentQtyMilli                        string
	CurrentSubtotalMilli                   string
	CurrentDiscountMilli                   string
	CurrentSurchargeMilli                  string
	RawItemIncludedTaxMilli                string
	AllocationDenominatorMilli             string
	AllocationRatio                        string
	AllocatedIncludedTaxMilli              string
	AllocatedDiscountMilli                 string
	AllocatedSurchargeMilli                string
	SourceOrderIncludedTaxMilli            string
	SumAllocatedIncludedTaxMilli           string
	DeltaMilli                             string
	SourceOrderNetMilli                    string
	SumAllocatedNetMilli                   string
	NetDeltaMilli                          string
	ItemLineCount                          string
	ProductGroupCount                      string
	PaymentRowCount                        string
	NormalizedPaymentTypeCount             string
	RawItemRowCount                        string
	GroupedItemRowCount                    string
	RawPaymentRowCount                     string
	PaymentAfterOrderAggregationRowCount   string
	FinalJoinedRowCount                    string
	RawAdditionRowCount                    string
	AdditionsAfterOrderAggregationRowCount string
	Note                                   string
}

type DuplicateOrderTraceRow struct {
	TraceRank                   string
	RawRowRank                  string
	DuplicateRowCount           string
	OpenDate                    string
	OrderID                     string
	Status                      string
	Destination                 string
	BranchID                    string
	Branch                      string
	TotalMilli                  string
	ItemSubtotalMilli           string
	DiscountSubtotalMilli       string
	PaymentSubtotalMilli        string
	IncludedTaxSubtotalMilli    string
	TaxSubtotalMilli            string
	ItemSurchargeSubtotalMilli  string
	TransSurchargeSubtotalMilli string
	TransactionCreated          string
	TransactionSubmitted        string
	TransactionVoided           string
	VoidSalePeriod              string
	Created                     string
	Modified                    string
	Sequence                    string
	SalePeriod                  string
	ShiftNumber                 string
	TopSampleStatus             string
	TopSampleDestination        string
}

type StatusDedupTopTaxDeltaRow struct {
	OrderID                              string
	BusinessDate                         string
	RawOrderRowCount                     int64
	RawStatusList                        string
	SelectedSalesStatus                  string
	SelectedSalesTransactionSubmitted    string
	SelectedSalesModified                string
	SelectedSalesPaymentSubtotalMilli    int64
	SourceStatus1IncludedTaxMilli        int64
	CurrentAllocatedIncludedTaxMilli     int64
	StatusDedupAllocatedIncludedTaxMilli int64
	CurrentDeltaMilli                    int64
	StatusDedupDeltaMilli                int64
	SourceStatus1NetMilli                int64
	CurrentAllocatedNetMilli             int64
	StatusDedupAllocatedNetMilli         int64
	CurrentNetDeltaMilli                 int64
	StatusDedupNetDeltaMilli             int64
}

type StatusExcludedSummaryRow struct {
	Status                  string
	OrderKeys               int64
	RawRows                 int64
	TotalMilli              int64
	PaymentSubtotalMilli    int64
	IncludedTaxMilli        int64
	DestinationDistribution string
	VoidedCount             int64
	SubmittedNullCount      int64
}

type DryRunResult struct {
	SourceMetrics                       []QueryMetric
	ResultMetric                        QueryMetric
	PreviewMetric                       QueryMetric
	PreviewRows                         []PreviewRow
	OrderMappingSummary                 []MappingSummaryRow
	PaymentMappingSummary               []MappingSummaryRow
	ReconciliationSummary               []ReconciliationSummaryRow
	TaxReconciliationBreakdown          []TaxReconciliationBreakdownRow
	AdditionsTaxDebug                   []DebugMetricRow
	RoundingDebug                       []DebugMetricRow
	TopTaxDeltaSample                   []TaxDeltaSampleRow
	TopTaxDeltaOrderTrace               []OrderTraceRow
	DuplicateOrderSummary               []DebugMetricRow
	DuplicateOrderTrace                 []DuplicateOrderTraceRow
	StatusDedupCandidateSummary         []DebugMetricRow
	StatusDedupReconciliationComparison []DebugMetricRow
	TopTaxDeltaBeforeAfterStatusDedup   []StatusDedupTopTaxDeltaRow
	StatusExcludedSummary               []StatusExcludedSummaryRow
}

type Service struct {
	client         *awsathena.Client
	database       string
	workgroup      string
	outputLocation string
}

func NewService(ctx context.Context, cfg appconfig.AthenaConfig) (*Service, error) {
	if strings.TrimSpace(cfg.OutputLocation) == "" || strings.Contains(cfg.OutputLocation, "replace-me") {
		return nil, fmt.Errorf("ATHENA_OUTPUT_LOCATION 尚未設定為可用的 S3 output location")
	}

	loadOptions := []func(*awscfg.LoadOptions) error{}
	if strings.TrimSpace(cfg.AWSProfile) != "" {
		loadOptions = append(loadOptions, awscfg.WithSharedConfigProfile(cfg.AWSProfile))
	}
	if strings.TrimSpace(cfg.Region) != "" {
		loadOptions = append(loadOptions, awscfg.WithRegion(cfg.Region))
	}

	awsConfig, err := awscfg.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &Service{
		client:         awsathena.NewFromConfig(awsConfig),
		database:       cfg.Database,
		workgroup:      cfg.Workgroup,
		outputLocation: cfg.OutputLocation,
	}, nil
}

func (service *Service) CollectDryRun(ctx context.Context, window QueryWindow) (DryRunResult, error) {
	sourceTables := []string{"orders_parquet", "order_items_parquet", "order_additions_parquet", "order_payments_parquet"}
	result := DryRunResult{
		SourceMetrics: make([]QueryMetric, 0, len(sourceTables)),
	}
	mode := normalizeDryRunMode(window.DryRunMode)

	for _, tableName := range sourceTables {
		metric, err := service.runCountMetric(ctx, tableName, BuildSourceCountSQL(tableName, window))
		if err != nil {
			return DryRunResult{}, err
		}
		result.SourceMetrics = append(result.SourceMetrics, metric)
	}

	resultMetric, err := service.runCountMetric(ctx, "result_aggregation", BuildPreviewCountSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.ResultMetric = resultMetric

	reconciliationSummary, err := service.runReconciliationSummary(ctx, BuildReconciliationSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.ReconciliationSummary = reconciliationSummary

	topTaxDeltaSample, err := service.runTopTaxDeltaSample(ctx, BuildTopTaxDeltaSampleSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.TopTaxDeltaSample = topTaxDeltaSample

	statusExcludedSummary, err := service.runStatusExcludedSummary(ctx, BuildStatusExcludedSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.StatusExcludedSummary = statusExcludedSummary

	if mode != DryRunModeFull {
		return result, nil
	}

	previewMetric, rows, err := service.runPreviewMetric(ctx, "preview_sample", BuildPreviewSelectSQL(window), window.PreviewLimit)
	if err != nil {
		return DryRunResult{}, err
	}
	result.PreviewMetric = previewMetric
	result.PreviewRows = rows

	orderMappingSummary, err := service.runMappingSummary(ctx, BuildOrderMappingSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.OrderMappingSummary = orderMappingSummary

	paymentMappingSummary, err := service.runMappingSummary(ctx, BuildPaymentMappingSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.PaymentMappingSummary = paymentMappingSummary

	taxReconciliationBreakdown, err := service.runTaxReconciliationBreakdown(ctx, BuildTaxReconciliationBreakdownSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.TaxReconciliationBreakdown = taxReconciliationBreakdown

	additionsTaxDebug, err := service.runDebugMetricRows(ctx, BuildAdditionsTaxDebugSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.AdditionsTaxDebug = additionsTaxDebug

	roundingDebug, err := service.runDebugMetricRows(ctx, BuildRoundingDebugSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.RoundingDebug = roundingDebug

	topTaxDeltaOrderTrace, err := service.runTopTaxDeltaOrderTrace(ctx, BuildTopTaxDeltaOrderTraceSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.TopTaxDeltaOrderTrace = topTaxDeltaOrderTrace

	duplicateOrderSummary, err := service.runDebugMetricRows(ctx, BuildDuplicateOrderSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.DuplicateOrderSummary = duplicateOrderSummary

	duplicateOrderTrace, err := service.runDuplicateOrderTrace(ctx, BuildDuplicateOrderTraceSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.DuplicateOrderTrace = duplicateOrderTrace

	statusDedupCandidateSummary, err := service.runDebugMetricRows(ctx, BuildStatusDedupCandidateSummarySQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.StatusDedupCandidateSummary = statusDedupCandidateSummary

	statusDedupReconciliationComparison, err := service.runDebugMetricRows(ctx, BuildStatusDedupReconciliationComparisonSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.StatusDedupReconciliationComparison = statusDedupReconciliationComparison

	topTaxDeltaBeforeAfterStatusDedup, err := service.runStatusDedupTopTaxDelta(ctx, BuildTopTaxDeltaBeforeAfterStatusDedupSQL(window))
	if err != nil {
		return DryRunResult{}, err
	}
	result.TopTaxDeltaBeforeAfterStatusDedup = topTaxDeltaBeforeAfterStatusDedup

	return result, nil
}

func (service *Service) runCountMetric(ctx context.Context, label string, sql string) (QueryMetric, error) {
	queryExecutionID, execution, err := service.runQuery(ctx, sql)
	if err != nil {
		return QueryMetric{}, fmt.Errorf("run count query for %s: %w", label, err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 1)
	if err != nil {
		return QueryMetric{}, fmt.Errorf("read count results for %s: %w", label, err)
	}

	if len(rows) == 0 {
		return QueryMetric{}, fmt.Errorf("count query for %s returned no rows", label)
	}

	rowCount, err := strconv.ParseInt(strings.TrimSpace(rows[0]["row_count"]), 10, 64)
	if err != nil {
		return QueryMetric{}, fmt.Errorf("parse row_count for %s: %w", label, err)
	}

	metric := buildMetric(label, queryExecutionID, execution)
	metric.RowCount = rowCount

	return metric, nil
}

func (service *Service) runPreviewMetric(ctx context.Context, label string, sql string, previewLimit ...int) (QueryMetric, []PreviewRow, error) {
	limit := defaultPreviewLimit
	if len(previewLimit) > 0 && previewLimit[0] > 0 {
		limit = previewLimit[0]
	}

	queryExecutionID, execution, err := service.runQuery(ctx, sql)
	if err != nil {
		return QueryMetric{}, nil, fmt.Errorf("run preview query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, limit)
	if err != nil {
		return QueryMetric{}, nil, fmt.Errorf("read preview results: %w", err)
	}

	previewRows := make([]PreviewRow, 0, len(rows))
	for _, row := range rows {
		previewRows = append(previewRows, PreviewRow{
			BusinessDate:    row["business_date"],
			HourOfDay:       leftPadHour(row["hour_of_day"]),
			BranchID:        row["branch_id"],
			ProductNo:       row["product_no"],
			OrderTypeID:     row["order_type_id"],
			PaymentTypeID:   row["payment_type_id"],
			QtyMilli:        row["qty_milli"],
			GrossSalesMilli: row["gross_sales_milli"],
			DiscountMilli:   row["discount_milli"],
			SurchargeMilli:  row["surcharge_milli"],
			NetSalesMilli:   row["net_sales_milli"],
			SalesExTaxMilli: row["sales_ex_tax_milli"],
			TaxMilli:        row["tax_milli"],
		})
	}

	metric := buildMetric(label, queryExecutionID, execution)
	metric.RowCount = int64(len(previewRows))

	return metric, previewRows, nil
}

func (service *Service) runMappingSummary(ctx context.Context, sql string) ([]MappingSummaryRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run mapping summary query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read mapping summary rows: %w", err)
	}

	result := make([]MappingSummaryRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, MappingSummaryRow{
			CanonicalID:       row["canonical_id"],
			CanonicalCode:     row["canonical_code"],
			SourceRows:        mustParseInt64(row["source_rows"]),
			DistinctRawValues: mustParseInt64(row["distinct_raw_values"]),
		})
	}

	return result, nil
}

func (service *Service) runReconciliationSummary(ctx context.Context, sql string) ([]ReconciliationSummaryRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run reconciliation query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 1)
	if err != nil {
		return nil, fmt.Errorf("read reconciliation summary row: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("reconciliation summary returned no rows")
	}

	row := rows[0]
	sourceGross := mustParseInt64(row["source_gross_sales_milli"])
	sourceDiscount := mustParseInt64(row["source_discount_milli"])
	sourceSurcharge := mustParseInt64(row["source_surcharge_milli"])
	sourceNet := mustParseInt64(row["source_net_sales_milli"])
	sourceSalesExTax := mustParseInt64(row["source_sales_ex_tax_milli"])
	sourceIncludedTax := mustParseInt64(row["source_included_tax_milli"])
	sourceExcludedTax := mustParseInt64(row["source_excluded_tax_milli"])
	sourceTax := mustParseInt64(row["source_tax_milli"])
	previewGross := mustParseInt64(row["preview_gross_sales_milli"])
	previewDiscount := mustParseInt64(row["preview_discount_milli"])
	previewSurcharge := mustParseInt64(row["preview_surcharge_milli"])
	previewNet := mustParseInt64(row["preview_net_sales_milli"])
	previewSalesExTax := mustParseInt64(row["preview_sales_ex_tax_milli"])
	previewIncludedTax := mustParseInt64(row["preview_included_tax_milli"])
	previewExcludedTax := mustParseInt64(row["preview_excluded_tax_milli"])
	previewTax := mustParseInt64(row["preview_tax_milli"])

	return []ReconciliationSummaryRow{
		{Metric: "source_order_count", SourceValue: formatInt64(row["source_order_count"]), PreviewValue: "", DeltaValue: ""},
		{Metric: "preview_group_count", SourceValue: "", PreviewValue: formatInt64(row["preview_group_count"]), DeltaValue: ""},
		{Metric: "source_addition_include_tax_milli", SourceValue: formatInt64(row["source_addition_include_tax_milli"]), PreviewValue: "", DeltaValue: ""},
		{Metric: "gross_sales_milli", SourceValue: formatInt64(row["source_gross_sales_milli"]), PreviewValue: formatInt64(row["preview_gross_sales_milli"]), DeltaValue: formatSignedInt64(previewGross - sourceGross)},
		{Metric: "discount_milli", SourceValue: formatInt64(row["source_discount_milli"]), PreviewValue: formatInt64(row["preview_discount_milli"]), DeltaValue: formatSignedInt64(previewDiscount - sourceDiscount)},
		{Metric: "surcharge_milli", SourceValue: formatInt64(row["source_surcharge_milli"]), PreviewValue: formatInt64(row["preview_surcharge_milli"]), DeltaValue: formatSignedInt64(previewSurcharge - sourceSurcharge)},
		{Metric: "net_sales_milli", SourceValue: formatInt64(row["source_net_sales_milli"]), PreviewValue: formatInt64(row["preview_net_sales_milli"]), DeltaValue: formatSignedInt64(previewNet - sourceNet)},
		{Metric: "sales_ex_tax_milli", SourceValue: formatInt64(row["source_sales_ex_tax_milli"]), PreviewValue: formatInt64(row["preview_sales_ex_tax_milli"]), DeltaValue: formatSignedInt64(previewSalesExTax - sourceSalesExTax)},
		{Metric: "included_tax_milli", SourceValue: formatInt64(row["source_included_tax_milli"]), PreviewValue: formatInt64(row["preview_included_tax_milli"]), DeltaValue: formatSignedInt64(previewIncludedTax - sourceIncludedTax)},
		{Metric: "excluded_tax_milli", SourceValue: formatInt64(row["source_excluded_tax_milli"]), PreviewValue: formatInt64(row["preview_excluded_tax_milli"]), DeltaValue: formatSignedInt64(previewExcludedTax - sourceExcludedTax)},
		{Metric: "tax_milli", SourceValue: formatInt64(row["source_tax_milli"]), PreviewValue: formatInt64(row["preview_tax_milli"]), DeltaValue: formatSignedInt64(previewTax - sourceTax)},
		{Metric: "source_addition_discount_milli", SourceValue: formatInt64(row["source_addition_discount_milli"]), PreviewValue: "", DeltaValue: ""},
		{Metric: "source_addition_surcharge_milli", SourceValue: formatInt64(row["source_addition_surcharge_milli"]), PreviewValue: "", DeltaValue: ""},
		{Metric: "check_preview_net_formula", SourceValue: formatSignedInt64(previewGross - previewDiscount + previewSurcharge), PreviewValue: formatSignedInt64(previewNet), DeltaValue: formatSignedInt64(previewNet - (previewGross - previewDiscount + previewSurcharge))},
		{Metric: "check_preview_sales_ex_tax_formula", SourceValue: formatSignedInt64(previewNet - previewIncludedTax - previewExcludedTax), PreviewValue: formatSignedInt64(previewSalesExTax), DeltaValue: formatSignedInt64(previewSalesExTax - (previewNet - previewIncludedTax - previewExcludedTax))},
		{Metric: "check_preview_tax_formula", SourceValue: formatSignedInt64(previewIncludedTax + previewExcludedTax), PreviewValue: formatSignedInt64(previewTax), DeltaValue: formatSignedInt64(previewTax - (previewIncludedTax + previewExcludedTax))},
	}, nil
}

func (service *Service) runTaxReconciliationBreakdown(ctx context.Context, sql string) ([]TaxReconciliationBreakdownRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run tax reconciliation breakdown query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read tax reconciliation breakdown rows: %w", err)
	}

	result := make([]TaxReconciliationBreakdownRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, TaxReconciliationBreakdownRow{
			Metric:           row["metric"],
			OrderCount:       mustParseInt64(row["order_count"]),
			GrossSalesMilli:  mustParseInt64(row["gross_sales_milli"]),
			DiscountMilli:    mustParseInt64(row["discount_milli"]),
			SurchargeMilli:   mustParseInt64(row["surcharge_milli"]),
			NetSalesMilli:    mustParseInt64(row["net_sales_milli"]),
			IncludedTaxMilli: mustParseInt64(row["included_tax_milli"]),
			SalesExTaxMilli:  mustParseInt64(row["sales_ex_tax_milli"]),
			Note:             row["note"],
		})
	}

	return result, nil
}

func (service *Service) runDebugMetricRows(ctx context.Context, sql string) ([]DebugMetricRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run debug metric query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read debug metric rows: %w", err)
	}

	result := make([]DebugMetricRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, DebugMetricRow{
			Metric: row["metric"],
			Value:  mustParseInt64(row["metric_value"]),
			Note:   row["note"],
		})
	}

	return result, nil
}

func (service *Service) runTopTaxDeltaSample(ctx context.Context, sql string) ([]TaxDeltaSampleRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run top tax delta sample query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read top tax delta sample rows: %w", err)
	}

	result := make([]TaxDeltaSampleRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, TaxDeltaSampleRow{
			BusinessDate:               row["business_date"],
			OrderID:                    row["order_id"],
			SourceIncludedTaxMilli:     mustParseInt64(row["source_included_tax_milli"]),
			AllocatedIncludedTaxMilli:  mustParseInt64(row["allocated_included_tax_milli"]),
			DeltaMilli:                 mustParseInt64(row["delta_milli"]),
			ItemCount:                  mustParseInt64(row["item_count"]),
			AllocationDenominatorMilli: mustParseInt64(row["allocation_denominator_milli"]),
			Status:                     row["status"],
			Destination:                row["destination"],
		})
	}

	return result, nil
}

func (service *Service) runTopTaxDeltaOrderTrace(ctx context.Context, sql string) ([]OrderTraceRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run top tax delta order trace query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read top tax delta order trace rows: %w", err)
	}

	result := make([]OrderTraceRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, OrderTraceRow{
			TraceRank:                              row["trace_rank"],
			Section:                                row["section"],
			RowOrder:                               row["row_order"],
			BusinessDate:                           row["business_date"],
			OrderID:                                row["order_id"],
			Status:                                 row["status"],
			Destination:                            row["destination"],
			NormalizedOrderTypeID:                  row["normalized_order_type_id"],
			BranchID:                               row["branch_id"],
			TotalMilli:                             row["total_milli"],
			ItemSubtotalMilli:                      row["item_subtotal_milli"],
			DiscountMilli:                          row["discount_milli"],
			SurchargeMilli:                         row["surcharge_milli"],
			IncludedTaxMilli:                       row["included_tax_milli"],
			TaxSubtotalMilli:                       row["tax_subtotal_milli"],
			TransactionVoided:                      row["transaction_voided"],
			VoidSalePeriod:                         row["void_sale_period"],
			PaymentRowID:                           row["payment_row_id"],
			PaymentName:                            row["payment_name"],
			PaymentAmountMilli:                     row["payment_amount_milli"],
			NormalizedPaymentTypeID:                row["normalized_payment_type_id"],
			FinalPaymentTypeID:                     row["final_payment_type_id"],
			IsMixed:                                row["is_mixed"],
			HasOffsettingPayments:                  row["has_offsetting_payments"],
			ItemID:                                 row["item_id"],
			ProductNo:                              row["product_no"],
			ProductName:                            row["product_name"],
			CurrentQtyMilli:                        row["current_qty_milli"],
			CurrentSubtotalMilli:                   row["current_subtotal_milli"],
			CurrentDiscountMilli:                   row["current_discount_milli"],
			CurrentSurchargeMilli:                  row["current_surcharge_milli"],
			RawItemIncludedTaxMilli:                row["raw_item_included_tax_milli"],
			AllocationDenominatorMilli:             row["allocation_denominator_milli"],
			AllocationRatio:                        row["allocation_ratio"],
			AllocatedIncludedTaxMilli:              row["allocated_included_tax_milli"],
			AllocatedDiscountMilli:                 row["allocated_discount_milli"],
			AllocatedSurchargeMilli:                row["allocated_surcharge_milli"],
			SourceOrderIncludedTaxMilli:            row["source_order_included_tax_milli"],
			SumAllocatedIncludedTaxMilli:           row["sum_allocated_included_tax_milli"],
			DeltaMilli:                             row["delta_milli"],
			SourceOrderNetMilli:                    row["source_order_net_milli"],
			SumAllocatedNetMilli:                   row["sum_allocated_net_milli"],
			NetDeltaMilli:                          row["net_delta_milli"],
			ItemLineCount:                          row["item_line_count"],
			ProductGroupCount:                      row["product_group_count"],
			PaymentRowCount:                        row["payment_row_count"],
			NormalizedPaymentTypeCount:             row["normalized_payment_type_count"],
			RawItemRowCount:                        row["raw_item_row_count"],
			GroupedItemRowCount:                    row["grouped_item_row_count"],
			RawPaymentRowCount:                     row["raw_payment_row_count"],
			PaymentAfterOrderAggregationRowCount:   row["payment_after_order_aggregation_row_count"],
			FinalJoinedRowCount:                    row["final_joined_row_count"],
			RawAdditionRowCount:                    row["raw_addition_row_count"],
			AdditionsAfterOrderAggregationRowCount: row["additions_after_order_aggregation_row_count"],
			Note:                                   row["note"],
		})
	}

	return result, nil
}

func (service *Service) runDuplicateOrderTrace(ctx context.Context, sql string) ([]DuplicateOrderTraceRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run duplicate order trace query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read duplicate order trace rows: %w", err)
	}

	result := make([]DuplicateOrderTraceRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, DuplicateOrderTraceRow{
			TraceRank:                   row["trace_rank"],
			RawRowRank:                  row["raw_row_rank"],
			DuplicateRowCount:           row["duplicate_row_count"],
			OpenDate:                    row["t_open_date"],
			OrderID:                     row["order_id"],
			Status:                      row["status"],
			Destination:                 row["destination"],
			BranchID:                    row["branch_id"],
			Branch:                      row["branch"],
			TotalMilli:                  row["total_milli"],
			ItemSubtotalMilli:           row["item_subtotal_milli"],
			DiscountSubtotalMilli:       row["discount_subtotal_milli"],
			PaymentSubtotalMilli:        row["payment_subtotal_milli"],
			IncludedTaxSubtotalMilli:    row["included_tax_subtotal_milli"],
			TaxSubtotalMilli:            row["tax_subtotal_milli"],
			ItemSurchargeSubtotalMilli:  row["item_surcharge_subtotal_milli"],
			TransSurchargeSubtotalMilli: row["trans_surcharge_subtotal_milli"],
			TransactionCreated:          row["transaction_created"],
			TransactionSubmitted:        row["transaction_submitted"],
			TransactionVoided:           row["transaction_voided"],
			VoidSalePeriod:              row["void_sale_period"],
			Created:                     row["created"],
			Modified:                    row["modified"],
			Sequence:                    row["sequence"],
			SalePeriod:                  row["sale_period"],
			ShiftNumber:                 row["shift_number"],
			TopSampleStatus:             row["top_sample_status"],
			TopSampleDestination:        row["top_sample_destination"],
		})
	}

	return result, nil
}

func (service *Service) runStatusDedupTopTaxDelta(ctx context.Context, sql string) ([]StatusDedupTopTaxDeltaRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run status dedup top tax delta query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read status dedup top tax delta rows: %w", err)
	}

	result := make([]StatusDedupTopTaxDeltaRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, StatusDedupTopTaxDeltaRow{
			OrderID:                              row["order_id"],
			BusinessDate:                         row["t_open_date"],
			RawOrderRowCount:                     mustParseInt64(row["raw_order_row_count"]),
			RawStatusList:                        row["raw_status_list"],
			SelectedSalesStatus:                  row["selected_sales_status"],
			SelectedSalesTransactionSubmitted:    row["selected_sales_transaction_submitted"],
			SelectedSalesModified:                row["selected_sales_modified"],
			SelectedSalesPaymentSubtotalMilli:    mustParseInt64(row["selected_sales_payment_subtotal_milli"]),
			SourceStatus1IncludedTaxMilli:        mustParseInt64(row["source_status_1_included_tax_milli"]),
			CurrentAllocatedIncludedTaxMilli:     mustParseInt64(row["current_allocated_included_tax_milli"]),
			StatusDedupAllocatedIncludedTaxMilli: mustParseInt64(row["status_dedup_allocated_included_tax_milli"]),
			CurrentDeltaMilli:                    mustParseInt64(row["current_delta_milli"]),
			StatusDedupDeltaMilli:                mustParseInt64(row["status_dedup_delta_milli"]),
			SourceStatus1NetMilli:                mustParseInt64(row["source_status_1_net_milli"]),
			CurrentAllocatedNetMilli:             mustParseInt64(row["current_allocated_net_milli"]),
			StatusDedupAllocatedNetMilli:         mustParseInt64(row["status_dedup_allocated_net_milli"]),
			CurrentNetDeltaMilli:                 mustParseInt64(row["current_net_delta_milli"]),
			StatusDedupNetDeltaMilli:             mustParseInt64(row["status_dedup_net_delta_milli"]),
		})
	}

	return result, nil
}

func (service *Service) runStatusExcludedSummary(ctx context.Context, sql string) ([]StatusExcludedSummaryRow, error) {
	queryExecutionID, _, err := service.runQuery(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("run status excluded summary query: %w", err)
	}

	rows, err := service.readRows(ctx, queryExecutionID, 0)
	if err != nil {
		return nil, fmt.Errorf("read status excluded summary rows: %w", err)
	}

	result := make([]StatusExcludedSummaryRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, StatusExcludedSummaryRow{
			Status:                  row["status"],
			OrderKeys:               mustParseInt64(row["order_keys"]),
			RawRows:                 mustParseInt64(row["raw_rows"]),
			TotalMilli:              mustParseInt64(row["total_milli"]),
			PaymentSubtotalMilli:    mustParseInt64(row["payment_subtotal_milli"]),
			IncludedTaxMilli:        mustParseInt64(row["included_tax_milli"]),
			DestinationDistribution: row["destination_distribution"],
			VoidedCount:             mustParseInt64(row["voided_count"]),
			SubmittedNullCount:      mustParseInt64(row["submitted_null_count"]),
		})
	}

	return result, nil
}

func (service *Service) runQuery(ctx context.Context, sql string) (string, *athenatypes.QueryExecution, error) {
	response, err := service.client.StartQueryExecution(ctx, &awsathena.StartQueryExecutionInput{
		QueryString: &sql,
		QueryExecutionContext: &athenatypes.QueryExecutionContext{
			Database: &service.database,
		},
		ResultConfiguration: &athenatypes.ResultConfiguration{
			OutputLocation: &service.outputLocation,
		},
		WorkGroup: &service.workgroup,
	})
	if err != nil {
		return "", nil, err
	}

	queryExecutionID := strings.TrimSpace(*response.QueryExecutionId)
	if queryExecutionID == "" {
		return "", nil, fmt.Errorf("Athena returned empty QueryExecutionId")
	}

	for {
		executionResponse, err := service.client.GetQueryExecution(ctx, &awsathena.GetQueryExecutionInput{
			QueryExecutionId: &queryExecutionID,
		})
		if err != nil {
			return "", nil, err
		}

		queryExecution := executionResponse.QueryExecution
		if queryExecution == nil || queryExecution.Status == nil {
			return "", nil, fmt.Errorf("Athena returned empty query execution state")
		}

		switch queryExecution.Status.State {
		case athenatypes.QueryExecutionStateSucceeded:
			return queryExecutionID, queryExecution, nil
		case athenatypes.QueryExecutionStateFailed, athenatypes.QueryExecutionStateCancelled:
			reason := ""
			if queryExecution.Status.StateChangeReason != nil {
				reason = *queryExecution.Status.StateChangeReason
			}
			return "", nil, fmt.Errorf("Athena query %s: %s (qid=%s)", queryExecution.Status.State, reason, queryExecutionID)
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (service *Service) readRows(ctx context.Context, queryExecutionID string, limit int) ([]map[string]string, error) {
	pageSize := int32(1000)
	if limit > 0 && limit+1 < int(pageSize) {
		pageSize = int32(limit + 1)
	}

	paginator := awsathena.NewGetQueryResultsPaginator(service.client, &awsathena.GetQueryResultsInput{
		QueryExecutionId: &queryExecutionID,
		MaxResults:       &pageSize,
	})

	headers := []string{}
	rows := []map[string]string{}
	isFirstPage := true

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for index, row := range page.ResultSet.Rows {
			if isFirstPage && index == 0 {
				headers = extractHeaders(row)
				continue
			}

			rows = append(rows, rowToMap(headers, row))
			if limit > 0 && len(rows) >= limit {
				return rows, nil
			}
		}

		isFirstPage = false
	}

	return rows, nil
}

func extractHeaders(row athenatypes.Row) []string {
	headers := make([]string, 0, len(row.Data))
	for _, cell := range row.Data {
		headers = append(headers, cellValue(cell))
	}
	return headers
}

func rowToMap(headers []string, row athenatypes.Row) map[string]string {
	result := make(map[string]string, len(headers))
	for index, header := range headers {
		value := ""
		if index < len(row.Data) {
			value = cellValue(row.Data[index])
		}
		result[header] = value
	}
	return result
}

func cellValue(cell athenatypes.Datum) string {
	if cell.VarCharValue == nil {
		return ""
	}
	return strings.TrimSpace(*cell.VarCharValue)
}

func buildMetric(label string, queryExecutionID string, execution *athenatypes.QueryExecution) QueryMetric {
	metric := QueryMetric{
		Label:            label,
		QueryExecutionID: queryExecutionID,
	}

	if execution == nil || execution.Statistics == nil {
		return metric
	}

	stats := execution.Statistics
	if stats.DataScannedInBytes != nil {
		metric.DataScannedMB = float64(*stats.DataScannedInBytes) / 1024 / 1024
	}
	if stats.EngineExecutionTimeInMillis != nil {
		metric.EngineExecutionSec = float64(*stats.EngineExecutionTimeInMillis) / 1000
	}

	return metric
}

func mustParseInt64(raw string) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0
	}

	return value
}

func formatInt64(raw string) string {
	return formatSignedInt64(mustParseInt64(raw))
}

func formatSignedInt64(value int64) string {
	text := strconv.FormatInt(value, 10)
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

func leftPadHour(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 1 {
		return "0" + trimmed
	}
	if trimmed == "" {
		return "00"
	}
	return trimmed
}
