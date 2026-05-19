package salespipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ia-analyses-db/internal/athena"
	"ia-analyses-db/internal/config"
	"ia-analyses-db/internal/postgres"
	"ia-analyses-db/internal/sales"
)

const (
	stateFileName            = "sales_fact_pipe_state.json"
	defaultInternalChunkSize = 7
	defaultLongRunSafetyDays = 31
	dateLayout               = "2006-01-02"
	timestampLayout          = time.RFC3339
	reportFilePattern        = "phase2c_sales_fact_pipe_summary_%s.md"
	incidentReportFileName   = "phase2c_31day_execution_incident_report.md"
	statusRunning            = "running"
	statusSuccess            = "success"
	statusFailed             = "failed"
	statusInterrupted        = "interrupted"
	statusNotStarted         = "not_started"
)

type Mode string

const (
	ModeStatus   Mode = "status"
	ModePlan     Mode = "write-plan"
	ModeValidate Mode = "validate-only"
	ModeWrite    Mode = "write-local"
	ModeResume   Mode = "resume"
	ModeReport   Mode = "report"
)

type Request struct {
	OwnerUserKey   string
	OwnerUserID    int64
	StartDate      time.Time
	EndDate        time.Time
	Mode           Mode
	Force          bool
	ConfirmLongRun bool
}

type State struct {
	RunID                 string       `json:"run_id"`
	OwnerUserKey          string       `json:"owner_user_key"`
	OwnerUserID           int64        `json:"owner_user_id"`
	SourceSchema          string       `json:"source_schema"`
	StartDate             string       `json:"start_date"`
	EndDate               string       `json:"end_date"`
	RequestedRangeDays    int          `json:"requested_range_days"`
	InternalChunkSize     int          `json:"internal_chunk_size"`
	Mode                  Mode         `json:"mode"`
	Status                string       `json:"status"`
	CurrentBusinessDate   string       `json:"current_business_date"`
	CompletedDates        []string     `json:"completed_dates"`
	FailedDate            string       `json:"failed_date"`
	FailedStage           string       `json:"failed_stage"`
	FailedGate            string       `json:"failed_gate"`
	FailedMetric          string       `json:"failed_metric"`
	FailedExpectedValue   string       `json:"failed_expected_value"`
	FailedActualValue     string       `json:"failed_actual_value"`
	RollbackExecuted      bool         `json:"rollback_executed"`
	StartedAt             string       `json:"started_at"`
	UpdatedAt             string       `json:"updated_at"`
	FinishedAt            string       `json:"finished_at"`
	TotalRowsWrittenSoFar int64        `json:"total_rows_written_so_far"`
	LastErrorSummary      string       `json:"last_error_summary"`
	SummaryReportFile     string       `json:"summary_report_file"`
	SummaryReportFileSize string       `json:"summary_report_file_size"`
	PID                   int          `json:"pid"`
	ConfirmLongRun        bool         `json:"confirm_long_run,omitempty"`
	Days                  []DaySummary `json:"days,omitempty"`
}

type DaySummary struct {
	BusinessDate             string `json:"business_date"`
	Status                   string `json:"status"`
	RowCount                 int64  `json:"row_count"`
	ValidationResult         string `json:"validation_result,omitempty"`
	TransactionResult        string `json:"transaction_result,omitempty"`
	SourceCandidateDeltaZero bool   `json:"source_candidate_delta_zero"`
	HasPostInsertCompare     bool   `json:"has_post_insert_compare"`
	PostInsertDeltaZero      bool   `json:"post_insert_delta_zero"`
	ProductDimMissCount      int64  `json:"product_dim_miss_count"`
	BranchDimMissCount       int64  `json:"branch_dim_miss_count"`
	OrderTypeDimMissCount    int64  `json:"order_type_dim_miss_count"`
	PaymentTypeDimMissCount  int64  `json:"payment_type_dim_miss_count"`
	ForbiddenColumnCount     int64  `json:"forbidden_column_count"`
	HardGateFailed           bool   `json:"hard_gate_failed"`
	RollbackExecuted         bool   `json:"rollback_executed"`
	Skipped                  bool   `json:"skipped"`
	ErrorSummary             string `json:"error_summary,omitempty"`
	FailedStage              string `json:"failed_stage,omitempty"`
	FailedGate               string `json:"failed_gate,omitempty"`
	FailedMetric             string `json:"failed_metric,omitempty"`
	FailedExpectedValue      string `json:"failed_expected_value,omitempty"`
	FailedActualValue        string `json:"failed_actual_value,omitempty"`
}

type Result struct {
	StatePath              string
	State                  State
	ActivePipelineProcess  bool
	PersistedDailyRowCount []PersistedDayCount
	TableSize              StorageSummary
}

type PersistedDayCount struct {
	BusinessDate string
	RowCount     int64
}

type StorageSummary struct {
	PosSalesHourlyFactTotalSize   string
	PosSalesHourlyFactTableSize   string
	PosSalesHourlyFactIndexesSize string
	DatabaseSize                  string
}

type Controller struct {
	baseDir    string
	stateDir   string
	reportsDir string
	cfg        config.AppConfig
	db         *sql.DB
}

func NewController(baseDir string, cfg config.AppConfig) (*Controller, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("base directory is required")
	}

	if err := os.MkdirAll(filepath.Join(baseDir, "state"), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "reports"), 0o755); err != nil {
		return nil, fmt.Errorf("create reports directory: %w", err)
	}

	db, err := postgres.Open(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	return &Controller{
		baseDir:    baseDir,
		stateDir:   filepath.Join(baseDir, "state"),
		reportsDir: filepath.Join(baseDir, "reports"),
		cfg:        cfg,
		db:         db,
	}, nil
}

func (controller *Controller) Close() error {
	if controller == nil || controller.db == nil {
		return nil
	}

	return controller.db.Close()
}

func (controller *Controller) Execute(ctx context.Context, request Request) (Result, error) {
	switch request.Mode {
	case ModeStatus:
		return controller.status(ctx)
	case ModePlan:
		return controller.plan(ctx, request)
	case ModeValidate:
		return controller.executeRange(ctx, request, false)
	case ModeWrite:
		return controller.executeRange(ctx, request, false)
	case ModeResume:
		return controller.resume(ctx, request)
	case ModeReport:
		return controller.report(ctx)
	default:
		return Result{}, fmt.Errorf("unsupported mode: %s", request.Mode)
	}
}

func (controller *Controller) status(ctx context.Context) (Result, error) {
	state, statePath, err := controller.loadState()
	if err != nil {
		return Result{}, err
	}

	result := Result{StatePath: statePath}
	if state == nil {
		result.State = State{Status: statusNotStarted}
		storage, storageErr := controller.loadStorageSummary(ctx)
		if storageErr == nil {
			result.TableSize = storage
		}
		return result, nil
	}

	result.State = *state
	result.ActivePipelineProcess = isProcessAlive(state.PID)
	storage, storageErr := controller.loadStorageSummary(ctx)
	if storageErr == nil {
		result.TableSize = storage
	}

	if state.OwnerUserID > 0 && state.StartDate != "" && state.EndDate != "" {
		startDate, startErr := time.Parse(dateLayout, state.StartDate)
		endDate, endErr := time.Parse(dateLayout, state.EndDate)
		if startErr == nil && endErr == nil {
			counts, countErr := controller.loadPersistedDailyCounts(ctx, state.OwnerUserID, startDate, endDate)
			if countErr == nil {
				result.PersistedDailyRowCount = counts
			}
		}
	}

	return result, nil
}

func (controller *Controller) plan(ctx context.Context, request Request) (Result, error) {
	if err := validateRangeRequest(request, true); err != nil {
		return Result{}, err
	}

	previous, _, err := controller.loadState()
	if err != nil {
		return Result{}, err
	}
	persistedCounts, err := controller.loadPersistedDailyCounts(ctx, request.OwnerUserID, request.StartDate, request.EndDate)
	if err != nil {
		return Result{}, err
	}
	persistedMap := persistedCountMap(persistedCounts)
	previousCompleted := completedDayCountMap(previous, ModeWrite)
	activePipelineProcess := false
	if previous != nil {
		activePipelineProcess = isProcessAlive(previous.PID)
	}

	now := time.Now().Format(timestampLayout)
	state := State{
		RunID:              newRunID(),
		OwnerUserKey:       request.OwnerUserKey,
		OwnerUserID:        request.OwnerUserID,
		SourceSchema:       controller.cfg.Athena.Database,
		StartDate:          request.StartDate.Format(dateLayout),
		EndDate:            request.EndDate.Format(dateLayout),
		RequestedRangeDays: sales.LocalWriteWindowDays(request.StartDate, request.EndDate),
		InternalChunkSize:  defaultInternalChunkSize,
		Mode:               ModePlan,
		Status:             statusSuccess,
		StartedAt:          now,
		UpdatedAt:          now,
		FinishedAt:         now,
		PID:                os.Getpid(),
	}
	state.SummaryReportFile = filepath.Join(controller.reportsDir, fmt.Sprintf(reportFilePattern, state.RunID))

	for _, businessDate := range expandDates(request.StartDate, request.EndDate) {
		dateText := businessDate.Format(dateLayout)
		if activePipelineProcess && previous != nil && previous.Status == statusRunning && previous.CurrentBusinessDate == dateText {
			state.Days = append(state.Days, DaySummary{
				BusinessDate: dateText,
				Status:       "running",
				RowCount:     loadPersistedRowCount(persistedMap, dateText),
			})
			continue
		}

		if rowCount, ok := persistedMap[dateText]; ok {
			state.Days = append(state.Days, DaySummary{
				BusinessDate: dateText,
				Status:       "completed",
				RowCount:     rowCount,
			})
			continue
		}

		if rowCount, ok := previousCompleted[dateText]; ok {
			state.Days = append(state.Days, DaySummary{
				BusinessDate: dateText,
				Status:       "completed",
				RowCount:     rowCount,
			})
			continue
		}

		state.Days = append(state.Days, DaySummary{
			BusinessDate: dateText,
			Status:       "pending",
		})
	}
	normalizeState(&state)

	if err := controller.writeSummaryReport(ctx, &state); err != nil {
		return Result{}, err
	}

	storage, _ := controller.loadStorageSummary(ctx)
	return Result{
		StatePath:              controller.statePath(),
		State:                  state,
		ActivePipelineProcess:  activePipelineProcess,
		PersistedDailyRowCount: persistedCounts,
		TableSize:              storage,
	}, nil
}

func (controller *Controller) executeRange(ctx context.Context, request Request, resume bool) (result Result, resultErr error) {
	if err := validateRangeRequest(request, request.Mode != ModeResume); err != nil {
		return Result{}, err
	}

	if request.Mode == ModeWrite && sales.LocalWriteWindowDays(request.StartDate, request.EndDate) > defaultLongRunSafetyDays && !request.ConfirmLongRun {
		return Result{}, fmt.Errorf("write-local over %d requested days requires --confirm-long-run; controller will chunk internally and resume automatically", defaultLongRunSafetyDays)
	}

	provider, err := athena.NewSalesCandidateProvider(ctx, controller.cfg.Athena, controller.db)
	if err != nil {
		return Result{}, fmt.Errorf("create sales candidate provider: %w", err)
	}

	writer := sales.DayReplaceWriter(postgres.DisabledSalesFactWriter{})
	if request.Mode == ModeWrite {
		writer = postgres.NewSQLSalesFactWriter(controller.db)
	}

	service := sales.NewWriteService(provider, writer)
	state, skipDates, err := controller.prepareStateForExecution(ctx, request, resume)
	if err != nil {
		return Result{}, err
	}

	result = Result{StatePath: controller.statePath(), State: *state}

	defer func() {
		controller.finalizeState(ctx, state, resultErr)
		_ = controller.writeSummaryReport(context.Background(), state)
		_ = controller.persistState(state)
		storage, storageErr := controller.loadStorageSummary(context.Background())
		if storageErr == nil {
			result.TableSize = storage
		}
		counts, countErr := controller.loadPersistedDailyCountsForState(context.Background(), state)
		if countErr == nil {
			result.PersistedDailyRowCount = counts
		}
		result.State = *state
		result.ActivePipelineProcess = isProcessAlive(state.PID)
	}()

	if err := controller.persistState(state); err != nil {
		return result, err
	}
	if err := controller.writeSummaryReport(ctx, state); err != nil {
		return result, err
	}

	chunks := chunkDates(expandDates(request.StartDate, request.EndDate), state.InternalChunkSize)
	for _, chunk := range chunks {
		for _, businessDate := range chunk {
			if ctx.Err() != nil {
				resultErr = ctx.Err()
				return result, resultErr
			}

			dateText := businessDate.Format(dateLayout)
			if _, ok := skipDates[dateText]; ok && !request.Force {
				controller.upsertDaySummary(state, DaySummary{
					BusinessDate: dateText,
					Status:       "skipped",
					RowCount:     loadPersistedRowCount(skipDates, dateText),
					Skipped:      true,
				})
				state.CurrentBusinessDate = dateText
				state.UpdatedAt = time.Now().Format(timestampLayout)
				normalizeState(state)
				if err := controller.persistState(state); err != nil {
					resultErr = err
					return result, resultErr
				}
				if err := controller.writeSummaryReport(ctx, state); err != nil {
					resultErr = err
					return result, resultErr
				}
				continue
			}

			state.CurrentBusinessDate = dateText
			state.UpdatedAt = time.Now().Format(timestampLayout)
			if err := controller.persistState(state); err != nil {
				resultErr = err
				return result, resultErr
			}

			executionMode := sales.ExecutionModeWritePG
			actualWrite := false
			if request.Mode == ModeValidate {
				executionMode = sales.ExecutionModeValidateOnly
			}
			if request.Mode == ModeWrite {
				actualWrite = true
			}

			syncResult, dayErr := service.Run(ctx, sales.SyncRequest{
				OwnerUserKey:       request.OwnerUserKey,
				OwnerUserID:        request.OwnerUserID,
				StartDate:          businessDate,
				EndDate:            businessDate,
				Mode:               executionMode,
				ActualWriteEnabled: actualWrite,
			})

			daySummary := summarizeDay(request.Mode, businessDate, syncResult, dayErr)
			controller.upsertDaySummary(state, daySummary)
			normalizeState(state)
			state.UpdatedAt = time.Now().Format(timestampLayout)
			if daySummary.Status == "failed" {
				state.FailedDate = daySummary.BusinessDate
				state.FailedStage = daySummary.FailedStage
				state.FailedGate = daySummary.FailedGate
				state.FailedMetric = daySummary.FailedMetric
				state.FailedExpectedValue = daySummary.FailedExpectedValue
				state.FailedActualValue = daySummary.FailedActualValue
				state.RollbackExecuted = daySummary.RollbackExecuted
				state.LastErrorSummary = firstNonEmpty(daySummary.ErrorSummary, dayErrString(dayErr))
			}

			if err := controller.persistState(state); err != nil {
				resultErr = err
				return result, resultErr
			}
			if err := controller.writeSummaryReport(ctx, state); err != nil {
				resultErr = err
				return result, resultErr
			}

			if dayErr != nil || daySummary.Status == "failed" {
				if dayErr != nil {
					resultErr = dayErr
				} else {
					resultErr = errors.New(firstNonEmpty(daySummary.ErrorSummary, "sales fact pipe stopped on failed day"))
				}
				return result, resultErr
			}
		}
	}

	return result, nil
}

func (controller *Controller) resume(ctx context.Context, request Request) (Result, error) {
	state, _, err := controller.loadState()
	if err != nil {
		return Result{}, err
	}
	if state == nil && (request.StartDate.IsZero() || request.EndDate.IsZero() || request.OwnerUserID <= 0 || strings.TrimSpace(request.OwnerUserKey) == "") {
		return Result{}, fmt.Errorf("no saved controller state found; resume requires either saved state or explicit owner/date range")
	}

	resumeMode := ModeWrite
	ownerUserKey := request.OwnerUserKey
	ownerUserID := request.OwnerUserID
	startDate := request.StartDate
	endDate := request.EndDate
	confirmLongRun := request.ConfirmLongRun

	if state != nil {
		if state.Mode == ModeWrite || state.Mode == ModeValidate {
			resumeMode = state.Mode
		}
		if strings.TrimSpace(ownerUserKey) == "" {
			ownerUserKey = state.OwnerUserKey
		}
		if ownerUserID <= 0 {
			ownerUserID = state.OwnerUserID
		}
		if startDate.IsZero() && state.StartDate != "" {
			parsedStartDate, parseErr := time.Parse(dateLayout, state.StartDate)
			if parseErr != nil {
				return Result{}, fmt.Errorf("parse state start_date: %w", parseErr)
			}
			startDate = parsedStartDate
		}
		if endDate.IsZero() && state.EndDate != "" {
			parsedEndDate, parseErr := time.Parse(dateLayout, state.EndDate)
			if parseErr != nil {
				return Result{}, fmt.Errorf("parse state end_date: %w", parseErr)
			}
			endDate = parsedEndDate
		}
		confirmLongRun = request.ConfirmLongRun || state.ConfirmLongRun
	}

	resumeRequest := Request{
		OwnerUserKey:   ownerUserKey,
		OwnerUserID:    ownerUserID,
		StartDate:      startDate,
		EndDate:        endDate,
		Mode:           resumeMode,
		Force:          request.Force,
		ConfirmLongRun: confirmLongRun,
	}
	if err := validateRangeRequest(resumeRequest, true); err != nil {
		return Result{}, err
	}

	return controller.executeRange(ctx, resumeRequest, true)
}

func (controller *Controller) report(ctx context.Context) (Result, error) {
	state, statePath, err := controller.loadState()
	if err != nil {
		return Result{}, err
	}
	if state == nil {
		return Result{}, fmt.Errorf("no saved controller state found")
	}
	if err := controller.writeSummaryReport(ctx, state); err != nil {
		return Result{}, err
	}
	if err := controller.persistState(state); err != nil {
		return Result{}, err
	}
	storage, _ := controller.loadStorageSummary(ctx)
	counts, _ := controller.loadPersistedDailyCountsForState(ctx, state)
	return Result{
		StatePath:              statePath,
		State:                  *state,
		ActivePipelineProcess:  isProcessAlive(state.PID),
		PersistedDailyRowCount: counts,
		TableSize:              storage,
	}, nil
}

func (controller *Controller) prepareStateForExecution(ctx context.Context, request Request, resume bool) (*State, map[string]int64, error) {
	now := time.Now().Format(timestampLayout)
	state := &State{
		RunID:              newRunID(),
		OwnerUserKey:       request.OwnerUserKey,
		OwnerUserID:        request.OwnerUserID,
		SourceSchema:       controller.cfg.Athena.Database,
		StartDate:          request.StartDate.Format(dateLayout),
		EndDate:            request.EndDate.Format(dateLayout),
		RequestedRangeDays: sales.LocalWriteWindowDays(request.StartDate, request.EndDate),
		InternalChunkSize:  defaultInternalChunkSize,
		Mode:               request.Mode,
		Status:             statusRunning,
		StartedAt:          now,
		UpdatedAt:          now,
		PID:                os.Getpid(),
		ConfirmLongRun:     request.ConfirmLongRun,
	}
	state.SummaryReportFile = filepath.Join(controller.reportsDir, fmt.Sprintf(reportFilePattern, state.RunID))

	skipDates := map[string]int64{}
	if !resume {
		return state, skipDates, nil
	}

	previous, _, err := controller.loadState()
	if err != nil {
		return nil, nil, err
	}

	counts, err := controller.loadPersistedDailyCounts(ctx, request.OwnerUserID, request.StartDate, request.EndDate)
	if err != nil {
		return nil, nil, err
	}
	for _, count := range counts {
		skipDates[count.BusinessDate] = count.RowCount
	}
	for businessDate, rowCount := range completedDayCountMap(previous, request.Mode) {
		if _, ok := skipDates[businessDate]; !ok {
			skipDates[businessDate] = rowCount
		}
	}

	return state, skipDates, nil
}

func (controller *Controller) finalizeState(ctx context.Context, state *State, executionErr error) {
	if state == nil {
		return
	}

	state.UpdatedAt = time.Now().Format(timestampLayout)
	if executionErr == nil {
		state.Status = statusSuccess
		state.FinishedAt = time.Now().Format(timestampLayout)
		return
	}

	if errors.Is(executionErr, context.Canceled) || errors.Is(executionErr, context.DeadlineExceeded) {
		state.Status = statusInterrupted
		state.LastErrorSummary = firstNonEmpty(state.LastErrorSummary, executionErr.Error())
		state.FinishedAt = time.Now().Format(timestampLayout)
		return
	}

	state.Status = statusFailed
	state.LastErrorSummary = firstNonEmpty(state.LastErrorSummary, executionErr.Error())
	state.FinishedAt = time.Now().Format(timestampLayout)

	if state.FailedDate == "" {
		state.FailedDate = state.CurrentBusinessDate
	}
	if state.FailedStage == "" {
		state.FailedStage = "controller execution"
	}
	_ = ctx
}

func summarizeDay(mode Mode, businessDate time.Time, syncResult sales.SyncResult, dayErr error) DaySummary {
	dateText := businessDate.Format(dateLayout)
	summary := DaySummary{BusinessDate: dateText}

	if len(syncResult.Days) > 0 {
		log := syncResult.Days[0]
		summary.ValidationResult = log.ValidationResult
		summary.TransactionResult = log.TransactionResult
		summary.ErrorSummary = strings.TrimSpace(log.Note)
		summary.RollbackExecuted = log.TransactionResult == "rollback"
	}

	if len(syncResult.DayDetails) > 0 {
		detail := syncResult.DayDetails[0]
		summary.RowCount = detail.CandidateMetrics.RowCount
		summary.SourceCandidateDeltaZero = !detail.PreInsertReport.MetricsComparison.HardGateFailed
		summary.ProductDimMissCount = detail.PreInsertReport.DimensionGate.ProductDimMissCount
		summary.BranchDimMissCount = detail.PreInsertReport.DimensionGate.BranchDimMissCount
		summary.OrderTypeDimMissCount = detail.PreInsertReport.DimensionGate.OrderTypeDimMissCount
		summary.PaymentTypeDimMissCount = detail.PreInsertReport.DimensionGate.PaymentTypeDimMissCount
		summary.ForbiddenColumnCount = detail.PreInsertReport.NegativeSchemaGate.ForbiddenColumnCount
		summary.HardGateFailed = detail.PreInsertReport.HardGateFailed

		if detail.HasPostInsertReport {
			summary.HasPostInsertCompare = true
			summary.PostInsertDeltaZero = !detail.PostInsertReport.MetricsComparison.HardGateFailed
			summary.RowCount = detail.PostInsertReport.PersistedTargetMetrics.RowCount
			if detail.PostInsertReport.HardGateFailed {
				summary.HardGateFailed = true
			}
		}

		if detail.PreInsertReport.HardGateFailed {
			reason := firstFailureReason(detail.PreInsertReport.FailureReasons)
			summary.FailedStage = "pre-insert validation"
			summary.FailedGate, summary.FailedMetric, summary.FailedExpectedValue, summary.FailedActualValue = parseFailure(reason)
			summary.ErrorSummary = firstNonEmpty(summary.ErrorSummary, reason)
		}
		if detail.HasPostInsertReport && detail.PostInsertReport.HardGateFailed {
			reason := firstFailureReason(detail.PostInsertReport.FailureReasons)
			summary.FailedStage = "post-insert compare"
			summary.FailedGate, summary.FailedMetric, summary.FailedExpectedValue, summary.FailedActualValue = parseFailure(reason)
			summary.ErrorSummary = firstNonEmpty(summary.ErrorSummary, reason)
		}
	}

	if dayErr != nil {
		summary.Status = "failed"
		summary.FailedStage = firstNonEmpty(summary.FailedStage, "controller execution")
		summary.ErrorSummary = firstNonEmpty(summary.ErrorSummary, dayErr.Error())
		if summary.FailedMetric == "" {
			summary.FailedGate, summary.FailedMetric, summary.FailedExpectedValue, summary.FailedActualValue = parseFailure(summary.ErrorSummary)
		}
		return summary
	}

	if summary.HardGateFailed || summary.RollbackExecuted {
		summary.Status = "failed"
		if summary.FailedStage == "" {
			summary.FailedStage = "pre-insert validation"
		}
		return summary
	}

	switch mode {
	case ModeValidate:
		summary.Status = "validated"
	case ModeWrite:
		summary.Status = "success"
	case ModePlan:
		summary.Status = "planned"
	default:
		summary.Status = "success"
	}

	return summary
}

func (controller *Controller) upsertDaySummary(state *State, summary DaySummary) {
	for index := range state.Days {
		if state.Days[index].BusinessDate == summary.BusinessDate {
			state.Days[index] = summary
			return
		}
	}
	state.Days = append(state.Days, summary)
}

func normalizeState(state *State) {
	sort.Slice(state.Days, func(i int, j int) bool {
		return state.Days[i].BusinessDate < state.Days[j].BusinessDate
	})

	completed := make([]string, 0)
	state.TotalRowsWrittenSoFar = 0
	for _, day := range state.Days {
		switch day.Status {
		case "success", "validated", "completed", "skipped":
			completed = append(completed, day.BusinessDate)
		}
		if day.Status == "success" || day.Status == "completed" || day.Status == "skipped" {
			state.TotalRowsWrittenSoFar += day.RowCount
		}
	}
	state.CompletedDates = uniqueSortedStrings(completed)
}

func validateRangeRequest(request Request, requireRange bool) error {
	if request.Mode == "" {
		return fmt.Errorf("mode is required")
	}
	if request.Mode == ModeResume || request.Mode == ModeStatus || request.Mode == ModeReport {
		return nil
	}
	if strings.TrimSpace(request.OwnerUserKey) == "" {
		return fmt.Errorf("owner_user_key is required")
	}
	if request.OwnerUserID <= 0 {
		return fmt.Errorf("owner_user_id is required")
	}
	if requireRange {
		if request.StartDate.IsZero() || request.EndDate.IsZero() {
			return fmt.Errorf("start_date and end_date are required")
		}
		if request.EndDate.Before(request.StartDate) {
			return fmt.Errorf("end_date must be on or after start_date")
		}
	}
	return nil
}

func expandDates(startDate time.Time, endDate time.Time) []time.Time {
	dates := make([]time.Time, 0, sales.LocalWriteWindowDays(startDate, endDate))
	for businessDate := startDate; !businessDate.After(endDate); businessDate = businessDate.AddDate(0, 0, 1) {
		dates = append(dates, businessDate)
	}
	return dates
}

func chunkDates(dates []time.Time, chunkSize int) [][]time.Time {
	if chunkSize <= 0 {
		chunkSize = defaultInternalChunkSize
	}

	var chunks [][]time.Time
	for start := 0; start < len(dates); start += chunkSize {
		end := start + chunkSize
		if end > len(dates) {
			end = len(dates)
		}
		chunks = append(chunks, dates[start:end])
	}

	return chunks
}

func (controller *Controller) loadPersistedDailyCountsForState(ctx context.Context, state *State) ([]PersistedDayCount, error) {
	if state == nil || state.OwnerUserID <= 0 || state.StartDate == "" || state.EndDate == "" {
		return nil, nil
	}
	startDate, err := time.Parse(dateLayout, state.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := time.Parse(dateLayout, state.EndDate)
	if err != nil {
		return nil, err
	}
	return controller.loadPersistedDailyCounts(ctx, state.OwnerUserID, startDate, endDate)
}

func (controller *Controller) loadPersistedDailyCounts(ctx context.Context, ownerUserID int64, startDate time.Time, endDate time.Time) ([]PersistedDayCount, error) {
	rows, err := controller.db.QueryContext(
		ctx,
		`SELECT business_date::text, COUNT(*) AS row_count
		 FROM public.pos_sales_hourly_fact
		 WHERE owner_user_id = $1
		   AND business_date BETWEEN $2 AND $3
		 GROUP BY business_date
		 ORDER BY business_date`,
		ownerUserID,
		startDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("load persisted daily counts: %w", err)
	}
	defer rows.Close()

	counts := make([]PersistedDayCount, 0)
	for rows.Next() {
		var count PersistedDayCount
		if err := rows.Scan(&count.BusinessDate, &count.RowCount); err != nil {
			return nil, fmt.Errorf("scan persisted daily count: %w", err)
		}
		counts = append(counts, count)
	}

	return counts, rows.Err()
}

func (controller *Controller) loadStorageSummary(ctx context.Context) (StorageSummary, error) {
	var summary StorageSummary
	if err := controller.db.QueryRowContext(
		ctx,
		`SELECT
			pg_size_pretty(pg_total_relation_size('public.pos_sales_hourly_fact')) AS total_size,
			pg_size_pretty(pg_relation_size('public.pos_sales_hourly_fact')) AS table_size,
			pg_size_pretty(pg_indexes_size('public.pos_sales_hourly_fact')) AS indexes_size`,
	).Scan(&summary.PosSalesHourlyFactTotalSize, &summary.PosSalesHourlyFactTableSize, &summary.PosSalesHourlyFactIndexesSize); err != nil {
		return StorageSummary{}, fmt.Errorf("load pos_sales_hourly_fact size: %w", err)
	}

	if err := controller.db.QueryRowContext(ctx, `SELECT pg_size_pretty(pg_database_size(current_database()))`).Scan(&summary.DatabaseSize); err != nil {
		return StorageSummary{}, fmt.Errorf("load database size: %w", err)
	}

	return summary, nil
}

func (controller *Controller) persistState(state *State) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}

	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tempPath := controller.statePath() + ".tmp"
	if err := os.WriteFile(tempPath, append(body, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}

	if err := os.Rename(tempPath, controller.statePath()); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}

	return nil
}

func (controller *Controller) loadState() (*State, string, error) {
	path := controller.statePath()
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, path, nil
		}
		return nil, path, fmt.Errorf("read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, path, fmt.Errorf("unmarshal state file: %w", err)
	}

	return &state, path, nil
}

func (controller *Controller) statePath() string {
	return filepath.Join(controller.stateDir, stateFileName)
}

func (controller *Controller) writeSummaryReport(ctx context.Context, state *State) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}
	if state.SummaryReportFile == "" {
		state.SummaryReportFile = filepath.Join(controller.reportsDir, fmt.Sprintf(reportFilePattern, state.RunID))
	}

	storage, _ := controller.loadStorageSummary(ctx)
	persistedCounts, _ := controller.loadPersistedDailyCountsForState(ctx, state)
	persistedMap := map[string]int64{}
	for _, count := range persistedCounts {
		persistedMap[count.BusinessDate] = count.RowCount
	}

	for attempt := 0; attempt < 3; attempt++ {
		content := renderSummaryReport(state, storage, persistedMap)
		if err := os.WriteFile(state.SummaryReportFile, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write summary report: %w", err)
		}

		info, err := os.Stat(state.SummaryReportFile)
		if err != nil {
			return fmt.Errorf("stat summary report: %w", err)
		}

		sizeText := humanFileSize(info.Size())
		if sizeText == state.SummaryReportFileSize {
			return nil
		}
		state.SummaryReportFileSize = sizeText
	}

	return nil
}

func renderSummaryReport(state *State, storage StorageSummary, persistedMap map[string]int64) string {
	startedAt, _ := time.Parse(timestampLayout, state.StartedAt)
	finishedAt, _ := time.Parse(timestampLayout, state.FinishedAt)
	elapsedSeconds := 0.0
	elapsedHuman := ""
	if !startedAt.IsZero() {
		endTime := finishedAt
		if endTime.IsZero() {
			endTime = time.Now()
		}
		elapsed := endTime.Sub(startedAt)
		elapsedSeconds = elapsed.Seconds()
		elapsedHuman = elapsed.Truncate(time.Second).String()
	}

	processedDays, succeededDays, failedDays, rollbackDays, skippedDays := summarizeDayCounts(state.Days)
	sourceCandidateZero, postInsertZero, productMissTotal, branchMissTotal, orderTypeMissTotal, paymentTypeMissTotal, forbiddenTotal, hardGateFailedCount := summarizeValidation(state.Days)
	committedBeforeFailure, unprocessedAfterFailure := summarizeFailureProgress(state)

	var builder strings.Builder
	builder.WriteString("# Phase 2C Sales Fact Pipe Summary\n\n")
	builder.WriteString("## Basic Information\n\n")
	fmt.Fprintf(&builder, "- report_file: %s\n", state.SummaryReportFile)
	fmt.Fprintf(&builder, "- run_id: %s\n", state.RunID)
	fmt.Fprintf(&builder, "- owner_user_key: %s\n", state.OwnerUserKey)
	fmt.Fprintf(&builder, "- owner_user_id: %d\n", state.OwnerUserID)
	fmt.Fprintf(&builder, "- source_schema: %s\n", state.SourceSchema)
	fmt.Fprintf(&builder, "- start_date: %s\n", state.StartDate)
	fmt.Fprintf(&builder, "- end_date: %s\n", state.EndDate)
	fmt.Fprintf(&builder, "- requested_days: %d\n", state.RequestedRangeDays)
	fmt.Fprintf(&builder, "- internal_chunk_size: %d\n", state.InternalChunkSize)
	fmt.Fprintf(&builder, "- execution_mode: %s\n", state.Mode)
	fmt.Fprintf(&builder, "- actual_write_enabled: %t\n", state.Mode == ModeWrite)
	fmt.Fprintf(&builder, "- status: %s\n", state.Status)
	if state.CurrentBusinessDate != "" {
		fmt.Fprintf(&builder, "- current_business_date: %s\n", state.CurrentBusinessDate)
	}
	builder.WriteString("\n## Timing\n\n")
	fmt.Fprintf(&builder, "- started_at: %s\n", state.StartedAt)
	fmt.Fprintf(&builder, "- finished_at: %s\n", state.FinishedAt)
	fmt.Fprintf(&builder, "- elapsed_seconds: %.3f\n", elapsedSeconds)
	fmt.Fprintf(&builder, "- elapsed_human_readable: %s\n", elapsedHuman)

	builder.WriteString("\n## Write Summary\n\n")
	fmt.Fprintf(&builder, "- total_rows_written: %d\n", state.TotalRowsWrittenSoFar)
	fmt.Fprintf(&builder, "- processed_days: %d\n", processedDays)
	fmt.Fprintf(&builder, "- succeeded_days: %d\n", succeededDays)
	fmt.Fprintf(&builder, "- failed_days: %d\n", failedDays)
	fmt.Fprintf(&builder, "- rollback_days: %d\n", rollbackDays)
	fmt.Fprintf(&builder, "- skipped_days: %d\n", skippedDays)
	builder.WriteString("\n| business_date | row_count | status |\n")
	builder.WriteString("|---|---:|---|\n")
	for _, day := range state.Days {
		rowCount := day.RowCount
		if persistedRowCount, ok := persistedMap[day.BusinessDate]; ok && (day.Status == "success" || day.Status == "skipped") {
			rowCount = persistedRowCount
		}
		fmt.Fprintf(&builder, "| %s | %d | %s |\n", day.BusinessDate, rowCount, day.Status)
	}

	builder.WriteString("\n## Validation Summary\n\n")
	fmt.Fprintf(&builder, "- source_candidate_delta_all_zero: %t\n", sourceCandidateZero)
	fmt.Fprintf(&builder, "- post_insert_delta_all_zero: %t\n", postInsertZero)
	fmt.Fprintf(&builder, "- product_dim_miss_total: %d\n", productMissTotal)
	fmt.Fprintf(&builder, "- branch_dim_miss_total: %d\n", branchMissTotal)
	fmt.Fprintf(&builder, "- order_type_dim_miss_total: %d\n", orderTypeMissTotal)
	fmt.Fprintf(&builder, "- payment_type_dim_miss_total: %d\n", paymentTypeMissTotal)
	fmt.Fprintf(&builder, "- forbidden_column_count: %d\n", forbiddenTotal)
	fmt.Fprintf(&builder, "- hard_gate_failed_count: %d\n", hardGateFailedCount)

	if state.Status == statusFailed || state.Status == statusInterrupted {
		failedDay := findFailedDay(state.Days, state.FailedDate)
		builder.WriteString("\n## Failure Summary\n\n")
		fmt.Fprintf(&builder, "- failed_date: %s\n", state.FailedDate)
		fmt.Fprintf(&builder, "- failed_stage: %s\n", firstNonEmpty(state.FailedStage, failedDay.FailedStage))
		fmt.Fprintf(&builder, "- failed_gate: %s\n", firstNonEmpty(state.FailedGate, failedDay.FailedGate))
		fmt.Fprintf(&builder, "- failed_metric: %s\n", firstNonEmpty(state.FailedMetric, failedDay.FailedMetric))
		fmt.Fprintf(&builder, "- expected_value: %s\n", firstNonEmpty(state.FailedExpectedValue, failedDay.FailedExpectedValue))
		fmt.Fprintf(&builder, "- actual_value: %s\n", firstNonEmpty(state.FailedActualValue, failedDay.FailedActualValue))
		fmt.Fprintf(&builder, "- rollback_executed: %t\n", state.RollbackExecuted || failedDay.RollbackExecuted)
		fmt.Fprintf(&builder, "- committed_days_before_failure: %d\n", committedBeforeFailure)
		fmt.Fprintf(&builder, "- unprocessed_days_after_failure: %d\n", unprocessedAfterFailure)
		fmt.Fprintf(&builder, "- error_summary: %s\n", firstNonEmpty(state.LastErrorSummary, failedDay.ErrorSummary))
		fmt.Fprintf(&builder, "- next_recommended_action: %s\n", nextRecommendedAction(state, failedDay))
	}

	builder.WriteString("\n## PostgreSQL Size Summary\n\n")
	fmt.Fprintf(&builder, "- pos_sales_hourly_fact_total_size: %s\n", storage.PosSalesHourlyFactTotalSize)
	fmt.Fprintf(&builder, "- pos_sales_hourly_fact_table_size: %s\n", storage.PosSalesHourlyFactTableSize)
	fmt.Fprintf(&builder, "- pos_sales_hourly_fact_indexes_size: %s\n", storage.PosSalesHourlyFactIndexesSize)
	fmt.Fprintf(&builder, "- database_size: %s\n", storage.DatabaseSize)

	builder.WriteString("\n## Report Summary\n\n")
	fmt.Fprintf(&builder, "- summary_report_file: %s\n", state.SummaryReportFile)
	fmt.Fprintf(&builder, "- summary_report_file_size: %s\n", state.SummaryReportFileSize)
	builder.WriteString("- whether_raw_row_log_saved: no\n")

	return builder.String()
}

func summarizeDayCounts(days []DaySummary) (processed int, succeeded int, failed int, rollback int, skipped int) {
	for _, day := range days {
		if day.Skipped || day.Status == "skipped" {
			skipped++
			continue
		}
		if day.Status == "pending" {
			continue
		}
		processed++
		switch day.Status {
		case "success", "validated", "planned", "completed":
			succeeded++
		case "failed":
			failed++
		}
		if day.RollbackExecuted {
			rollback++
		}
	}
	return processed, succeeded, failed, rollback, skipped
}

func summarizeValidation(days []DaySummary) (bool, bool, int64, int64, int64, int64, int64, int) {
	sourceCandidateAllZero := true
	postInsertAllZero := true
	hasPostInsert := false
	var productMissTotal int64
	var branchMissTotal int64
	var orderTypeMissTotal int64
	var paymentTypeMissTotal int64
	var forbiddenTotal int64
	hardGateFailedCount := 0

	for _, day := range days {
		if day.Status == "planned" || day.Status == "pending" || day.Status == "completed" || day.Status == "skipped" {
			continue
		}
		if !day.SourceCandidateDeltaZero {
			sourceCandidateAllZero = false
		}
		if day.HasPostInsertCompare {
			hasPostInsert = true
			if !day.PostInsertDeltaZero {
				postInsertAllZero = false
			}
		}
		productMissTotal += day.ProductDimMissCount
		branchMissTotal += day.BranchDimMissCount
		orderTypeMissTotal += day.OrderTypeDimMissCount
		paymentTypeMissTotal += day.PaymentTypeDimMissCount
		forbiddenTotal += day.ForbiddenColumnCount
		if day.HardGateFailed {
			hardGateFailedCount++
		}
	}

	if !hasPostInsert {
		postInsertAllZero = false
	}

	return sourceCandidateAllZero, postInsertAllZero, productMissTotal, branchMissTotal, orderTypeMissTotal, paymentTypeMissTotal, forbiddenTotal, hardGateFailedCount
}

func summarizeFailureProgress(state *State) (int, int) {
	if state == nil {
		return 0, 0
	}
	committed := 0
	processed := map[string]struct{}{}
	for _, day := range state.Days {
		processed[day.BusinessDate] = struct{}{}
		if day.Status == "success" || day.Status == "skipped" {
			committed++
		}
	}
	return committed, maxInt(0, state.RequestedRangeDays-len(processed))
}

func findFailedDay(days []DaySummary, failedDate string) DaySummary {
	for _, day := range days {
		if day.BusinessDate == failedDate {
			return day
		}
	}
	return DaySummary{}
}

func nextRecommendedAction(state *State, failedDay DaySummary) string {
	if state == nil {
		return "inspect controller state before retrying"
	}
	if state.Status == statusInterrupted {
		return "run make sales-pipe-resume after confirming the last committed date"
	}
	if strings.Contains(firstNonEmpty(state.FailedStage, failedDay.FailedStage), "pre-insert") {
		return "inspect validation gate summary and fix the failed gate before resuming"
	}
	if strings.Contains(firstNonEmpty(state.FailedStage, failedDay.FailedStage), "post-insert") {
		return "inspect post-insert compare mismatch before resuming"
	}
	return "inspect the controller summary report and retry with make sales-pipe-resume when safe"
}

func firstFailureReason(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func parseFailure(reason string) (failedGate string, failedMetric string, expectedValue string, actualValue string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", "", "", ""
	}
	parts := strings.SplitN(reason, "=", 2)
	if len(parts) != 2 {
		return reason, reason, "", ""
	}
	name := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	return name, name, "0", value
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func newRunID() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func loadPersistedRowCount(counts map[string]int64, businessDate string) int64 {
	if value, ok := counts[businessDate]; ok {
		return value
	}
	return 0
}

func persistedCountMap(counts []PersistedDayCount) map[string]int64 {
	result := make(map[string]int64, len(counts))
	for _, count := range counts {
		result[count.BusinessDate] = count.RowCount
	}
	return result
}

func completedDayCountMap(state *State, mode Mode) map[string]int64 {
	result := map[string]int64{}
	if state == nil {
		return result
	}

	for _, day := range state.Days {
		switch mode {
		case ModeWrite:
			if day.Status == "success" || day.Status == "skipped" || day.Status == "completed" {
				result[day.BusinessDate] = day.RowCount
			}
		case ModeValidate:
			if day.Status == "validated" || day.Status == "skipped" || day.Status == "completed" {
				result[day.BusinessDate] = day.RowCount
			}
		default:
			if day.Status == "success" || day.Status == "validated" || day.Status == "skipped" || day.Status == "completed" {
				result[day.BusinessDate] = day.RowCount
			}
		}
	}

	return result
}

func humanFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func dayErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func (controller *Controller) IncidentReportPath() string {
	return filepath.Join(controller.reportsDir, incidentReportFileName)
}

func ParseStateDate(text string) (time.Time, error) {
	return time.Parse(dateLayout, text)
}

func ParseInt64(text string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	return value
}
