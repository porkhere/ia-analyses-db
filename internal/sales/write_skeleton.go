package sales

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"ia-analyses-db/internal/validation"
)

const syncDateLayout = "2006-01-02"

const MaxLocalWriteWindowDays = 31

var (
	ErrSourceCandidateProviderPending   = errors.New("phase 2C-5 source candidate builder placeholder not implemented")
	ErrPostInsertMetricsProviderPending = errors.New("phase 2C-5 post-insert target metrics placeholder not implemented")
	ErrActualWriteRangeTooLarge         = errors.New("phase 2C-5.X local-only actual write requires a window of at most 31 days")
)

type ExecutionMode string

const (
	ExecutionModeWritePG      ExecutionMode = "write-pg"
	ExecutionModeValidateOnly ExecutionMode = "validate-only"
)

type SyncRequest struct {
	OwnerUserKey       string
	OwnerUserID        int64
	StartDate          time.Time
	EndDate            time.Time
	Mode               ExecutionMode
	ActualWriteEnabled bool
}

type DayScope struct {
	OwnerUserKey string
	OwnerUserID  int64
	BusinessDate time.Time
}

type FactRow struct {
	OwnerUserID      int64
	BusinessDate     time.Time
	HourOfDay        int16
	BranchID         string
	ProductNo        string
	OrderTypeID      int16
	PaymentTypeID    int16
	QtyMilli         int64
	GrossSalesMilli  int64
	DiscountMilli    int64
	SurchargeMilli   int64
	NetSalesMilli    int64
	SalesExTaxMilli  int64
	IncludedTaxMilli int64
	ExcludedTaxMilli int64
	TaxMilli         int64
}

type CandidateBuildResult struct {
	HasSourceMetrics    bool
	SourceMetrics       validation.MetricsSnapshot
	HasCandidateMetrics bool
	CandidateMetrics    validation.MetricsSnapshot
	Rows                []FactRow
	DimensionGate       validation.DimensionGateResult
	NegativeSchemaGate  validation.NegativeSchemaGateResult
	Blockers            []string
}

type CandidateProvider interface {
	BuildDayCandidate(ctx context.Context, scope DayScope) (CandidateBuildResult, error)
	LoadPostInsertTargetMetrics(ctx context.Context, scope DayScope) (validation.MetricsSnapshot, error)
}

type PlaceholderCandidateProvider struct{}

func (PlaceholderCandidateProvider) BuildDayCandidate(_ context.Context, scope DayScope) (CandidateBuildResult, error) {
	return CandidateBuildResult{
		Blockers: []string{
			ErrSourceCandidateProviderPending.Error(),
			fmt.Sprintf("owner_user_id=%d business_date=%s 尚未接上 Phase 2B status-aware sales candidate -> pre-insert candidate metrics/data flow", scope.OwnerUserID, scope.BusinessDate.Format(syncDateLayout)),
		},
	}, nil
}

func (PlaceholderCandidateProvider) LoadPostInsertTargetMetrics(_ context.Context, _ DayScope) (validation.MetricsSnapshot, error) {
	return validation.MetricsSnapshot{}, ErrPostInsertMetricsProviderPending
}

type DayReplaceWriter interface {
	BeginDayReplace(ctx context.Context, ownerUserID int64, businessDate time.Time) (DayReplaceTx, error)
}

type DayReplaceTx interface {
	DeleteExistingDay(ctx context.Context) error
	InsertRows(ctx context.Context, rows []FactRow) error
	LoadPersistedTargetMetrics(ctx context.Context) (validation.MetricsSnapshot, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Service struct {
	provider CandidateProvider
	writer   DayReplaceWriter
}

func NewWriteService(provider CandidateProvider, writer DayReplaceWriter) *Service {
	return &Service{
		provider: provider,
		writer:   writer,
	}
}

type DayExecutionLog struct {
	BusinessDate      string
	SourceRowCount    string
	CandidateRowCount string
	TargetRowCount    string
	ValidationResult  string
	TransactionResult string
	Note              string
}

type SyncResult struct {
	Mode               ExecutionMode
	ActualWriteEnabled bool
	Days               []DayExecutionLog
	DayDetails         []DayValidationDetail
	SafetyMessage      string
}

func (service *Service) Run(ctx context.Context, request SyncRequest) (SyncResult, error) {
	if service.provider == nil {
		return SyncResult{}, fmt.Errorf("candidate provider is required")
	}

	if service.writer == nil {
		return SyncResult{}, fmt.Errorf("day replace writer is required")
	}

	result := SyncResult{
		Mode:               request.Mode,
		ActualWriteEnabled: request.ActualWriteEnabled,
		SafetyMessage:      buildSafetyMessage(request),
	}

	if request.ActualWriteEnabled && LocalWriteWindowDays(request.StartDate, request.EndDate) > MaxLocalWriteWindowDays {
		return result, ErrActualWriteRangeTooLarge
	}

	for businessDate := request.StartDate; !businessDate.After(request.EndDate); businessDate = businessDate.AddDate(0, 0, 1) {
		scope := DayScope{
			OwnerUserKey: request.OwnerUserKey,
			OwnerUserID:  request.OwnerUserID,
			BusinessDate: businessDate,
		}

		dayLog, dayDetail, stopProcessing, err := service.processDate(ctx, request, scope)
		result.Days = append(result.Days, dayLog)
		if dayDetail != nil {
			result.DayDetails = append(result.DayDetails, *dayDetail)
		}
		if err != nil {
			return result, err
		}
		if stopProcessing {
			break
		}
	}

	return result, nil
}

func (service *Service) processDate(ctx context.Context, request SyncRequest, scope DayScope) (DayExecutionLog, *DayValidationDetail, bool, error) {
	log := DayExecutionLog{
		BusinessDate:      scope.BusinessDate.Format(syncDateLayout),
		SourceRowCount:    "-",
		CandidateRowCount: "-",
		TargetRowCount:    "-",
		ValidationResult:  "not started",
		TransactionResult: "not started",
	}

	buildResult, err := service.provider.BuildDayCandidate(ctx, scope)
	if err != nil {
		log.ValidationResult = "error"
		log.Note = err.Error()
		return log, nil, true, err
	}

	var dayDetail *DayValidationDetail
	if buildResult.HasSourceMetrics {
		log.SourceRowCount = formatInt(buildResult.SourceMetrics.RowCount)
	}
	if buildResult.HasCandidateMetrics {
		log.CandidateRowCount = formatInt(buildResult.CandidateMetrics.RowCount)
		dayDetail = &DayValidationDetail{
			BusinessDate:       log.BusinessDate,
			SourceMetrics:      buildResult.SourceMetrics,
			CandidateMetrics:   buildResult.CandidateMetrics,
			CandidateRowCount:  len(buildResult.Rows),
			CandidateRowSample: CloneFactRowSample(buildResult.Rows, 10),
		}
	}

	if len(buildResult.Blockers) > 0 {
		log.ValidationResult = "blocked"
		log.TransactionResult = "not started"
		log.Note = strings.Join(buildResult.Blockers, "; ")
		return log, dayDetail, true, nil
	}

	preInsertReport := validation.BuildPreInsertReport(
		buildResult.SourceMetrics,
		buildResult.CandidateMetrics,
		buildResult.DimensionGate,
		buildResult.NegativeSchemaGate,
	)
	if dayDetail == nil {
		dayDetail = &DayValidationDetail{
			BusinessDate:       log.BusinessDate,
			SourceMetrics:      buildResult.SourceMetrics,
			CandidateMetrics:   buildResult.CandidateMetrics,
			CandidateRowCount:  len(buildResult.Rows),
			CandidateRowSample: CloneFactRowSample(buildResult.Rows, 10),
		}
	}
	dayDetail.PreInsertReport = preInsertReport

	if preInsertReport.HardGateFailed {
		log.ValidationResult = "pre-insert hard gate failed"
		log.TransactionResult = "not started"
		log.Note = strings.Join(preInsertReport.FailureReasons, "; ")
		return log, dayDetail, true, nil
	}

	log.ValidationResult = "pre-insert hard gate passed"

	if request.Mode == ExecutionModeValidateOnly {
		log.TransactionResult = "validate-only stop"
		log.Note = "validation gate skeleton ready; write path intentionally skipped"
		return log, dayDetail, false, nil
	}

	if !request.ActualWriteEnabled {
		log.TransactionResult = "write disabled"
		log.Note = "transaction skeleton 已建立，但本輪 actual insert disabled"
		return log, dayDetail, false, nil
	}

	postInsertReport, transactionResult, err := service.executeDayReplace(ctx, scope, buildResult)
	if err != nil {
		log.TransactionResult = transactionResult
		log.Note = err.Error()
		return log, dayDetail, true, err
	}
	if dayDetail != nil {
		dayDetail.PostInsertReport = postInsertReport
		dayDetail.HasPostInsertReport = true
	}

	log.TransactionResult = transactionResult
	log.TargetRowCount = formatInt(postInsertReport.PersistedTargetMetrics.RowCount)
	if postInsertReport.HardGateFailed {
		log.ValidationResult = "post-insert hard gate failed"
		log.Note = strings.Join(postInsertReport.FailureReasons, "; ")
		return log, dayDetail, true, nil
	}

	log.ValidationResult = "post-insert hard gate passed"
	log.Note = "commit"

	return log, dayDetail, false, nil
}

func (service *Service) executeDayReplace(ctx context.Context, scope DayScope, buildResult CandidateBuildResult) (validation.PostInsertReport, string, error) {
	tx, err := service.writer.BeginDayReplace(ctx, scope.OwnerUserID, scope.BusinessDate)
	if err != nil {
		return validation.PostInsertReport{}, "not started", err
	}

	if err := tx.DeleteExistingDay(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return validation.PostInsertReport{}, "rollback", err
	}

	if err := tx.InsertRows(ctx, buildResult.Rows); err != nil {
		_ = tx.Rollback(ctx)
		return validation.PostInsertReport{}, "rollback", err
	}

	targetMetrics, err := tx.LoadPersistedTargetMetrics(ctx)
	if err != nil {
		_ = tx.Rollback(ctx)
		return validation.PostInsertReport{}, "rollback", err
	}

	postInsertReport := validation.BuildPostInsertReport(buildResult.SourceMetrics, targetMetrics)
	if postInsertReport.HardGateFailed {
		_ = tx.Rollback(ctx)
		return postInsertReport, "rollback", nil
	}

	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return validation.PostInsertReport{}, "rollback", err
	}

	return postInsertReport, "commit", nil
}

func buildSafetyMessage(request SyncRequest) string {
	if request.Mode == ExecutionModeValidateOnly {
		return "phase 2C-5.X validate-only mode; PostgreSQL insert remains disabled"
	}

	if request.ActualWriteEnabled {
		return "phase 2C-5.X local-only actual write enabled; restricted to at most 31 days with independent day-level replace transactions and post-insert compare before each commit"
	}

	return "phase 2C-5.X write-plan mode; PostgreSQL insert remains disabled"
}

func LocalWriteWindowDays(startDate time.Time, endDate time.Time) int {
	return int(endDate.Sub(startDate).Hours()/24) + 1
}

func RenderWriteExecutionTable(rows []DayExecutionLog) string {
	var builder strings.Builder

	writer := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "business_date\tsource_row_count\tcandidate_row_count\ttarget_row_count\tvalidation_result\ttransaction_result\tnote")

	for _, row := range rows {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.BusinessDate,
			row.SourceRowCount,
			row.CandidateRowCount,
			row.TargetRowCount,
			row.ValidationResult,
			row.TransactionResult,
			row.Note,
		)
	}

	_ = writer.Flush()

	return builder.String()
}

func formatInt(value int64) string {
	return fmt.Sprintf("%d", value)
}
