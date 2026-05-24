package main

// Bridge copy only: future sales-pipe command edits should land in ia-analyses-go.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"ia-analyses-db/internal/config"
	"ia-analyses-db/internal/salespipe"
)

const dateLayout = "2006-01-02"

type options struct {
	ownerUserKey   string
	ownerUserID    int64
	startDate      string
	endDate        string
	mode           string
	force          bool
	confirmLongRun bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	request, err := buildRequest(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	controller, err := salespipe.NewController(baseDir, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	defer controller.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	result, execErr := controller.Execute(ctx, request)
	printResult(stdout, result)
	if execErr != nil {
		fmt.Fprintf(stderr, "error: %v\n", execErr)
		return 1
	}

	return 0
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var opts options

	fs := flag.NewFlagSet("sales-pipe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.ownerUserKey, "owner-user-key", "", "owner user key")
	fs.Int64Var(&opts.ownerUserID, "owner-user-id", 0, "owner user id")
	fs.StringVar(&opts.startDate, "start-date", "", "execution start date, format: YYYY-MM-DD")
	fs.StringVar(&opts.endDate, "end-date", "", "execution end date, format: YYYY-MM-DD")
	fs.StringVar(&opts.mode, "mode", string(salespipe.ModeStatus), "controller mode: status, write-plan, validate-only, write-local, resume, report")
	fs.BoolVar(&opts.force, "force", false, "force rerun already completed dates during resume or write-local")
	fs.BoolVar(&opts.confirmLongRun, "confirm-long-run", false, "confirm longer local-only writes; controller still chunks internally")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	return opts, nil
}

func buildRequest(opts options) (salespipe.Request, error) {
	mode := salespipe.Mode(strings.TrimSpace(opts.mode))
	request := salespipe.Request{
		OwnerUserKey:   strings.TrimSpace(opts.ownerUserKey),
		OwnerUserID:    opts.ownerUserID,
		Mode:           mode,
		Force:          opts.force,
		ConfirmLongRun: opts.confirmLongRun,
	}

	if opts.startDate != "" {
		startDate, err := time.Parse(dateLayout, opts.startDate)
		if err != nil {
			return salespipe.Request{}, fmt.Errorf("invalid --start-date: %w", err)
		}
		request.StartDate = startDate
	}
	if opts.endDate != "" {
		endDate, err := time.Parse(dateLayout, opts.endDate)
		if err != nil {
			return salespipe.Request{}, fmt.Errorf("invalid --end-date: %w", err)
		}
		request.EndDate = endDate
	}
	if !request.StartDate.IsZero() && request.EndDate.IsZero() {
		request.EndDate = request.StartDate
	}

	return request, nil
}

func printResult(stdout io.Writer, result salespipe.Result) {
	fmt.Fprintf(stdout, "sales_pipe_mode: %s\n", result.State.Mode)
	fmt.Fprintf(stdout, "controller_state_file: %s\n", result.StatePath)
	if result.State.SummaryReportFile != "" {
		fmt.Fprintf(stdout, "summary_report_file: %s\n", result.State.SummaryReportFile)
	}
	if result.State.SummaryReportFileSize != "" {
		fmt.Fprintf(stdout, "summary_report_file_size: %s\n", result.State.SummaryReportFileSize)
	}
	fmt.Fprintf(stdout, "controller_status: %s\n", result.State.Status)
	fmt.Fprintf(stdout, "active_pipeline_process: %t\n", result.ActivePipelineProcess)
	if result.State.RunID != "" {
		fmt.Fprintf(stdout, "run_id: %s\n", result.State.RunID)
	}
	if result.State.OwnerUserKey != "" {
		fmt.Fprintf(stdout, "owner_user_key: %s\n", result.State.OwnerUserKey)
	}
	if result.State.OwnerUserID > 0 {
		fmt.Fprintf(stdout, "owner_user_id: %d\n", result.State.OwnerUserID)
	}
	if result.State.StartDate != "" || result.State.EndDate != "" {
		fmt.Fprintf(stdout, "date_range: %s -> %s\n", result.State.StartDate, result.State.EndDate)
		fmt.Fprintf(stdout, "requested_range_days: %d\n", result.State.RequestedRangeDays)
		fmt.Fprintf(stdout, "internal_chunk_size: %d\n", result.State.InternalChunkSize)
	}
	if result.State.CurrentBusinessDate != "" {
		fmt.Fprintf(stdout, "current_business_date: %s\n", result.State.CurrentBusinessDate)
	}
	if result.State.StartedAt != "" {
		fmt.Fprintf(stdout, "started_at: %s\n", result.State.StartedAt)
	}
	if result.State.FinishedAt != "" {
		fmt.Fprintf(stdout, "finished_at: %s\n", result.State.FinishedAt)
	}
	if result.State.LastErrorSummary != "" {
		fmt.Fprintf(stdout, "last_error_summary: %s\n", result.State.LastErrorSummary)
	}
	fmt.Fprintf(stdout, "total_rows_written_so_far: %d\n", result.State.TotalRowsWrittenSoFar)

	if result.TableSize.DatabaseSize != "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "postgres_storage_summary:")
		fmt.Fprintf(stdout, "  pos_sales_hourly_fact_total_size: %s\n", result.TableSize.PosSalesHourlyFactTotalSize)
		fmt.Fprintf(stdout, "  pos_sales_hourly_fact_table_size: %s\n", result.TableSize.PosSalesHourlyFactTableSize)
		fmt.Fprintf(stdout, "  pos_sales_hourly_fact_indexes_size: %s\n", result.TableSize.PosSalesHourlyFactIndexesSize)
		fmt.Fprintf(stdout, "  database_size: %s\n", result.TableSize.DatabaseSize)
	}

	if len(result.State.Days) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "daily_summary:")
		writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "business_date\tstatus\trow_count\tvalidation_result\ttransaction_result")
		for _, day := range result.State.Days {
			fmt.Fprintf(writer, "%s\t%s\t%d\t%s\t%s\n", day.BusinessDate, day.Status, day.RowCount, day.ValidationResult, day.TransactionResult)
		}
		_ = writer.Flush()
	}

	if len(result.PersistedDailyRowCount) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "persisted_daily_row_count:")
		writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "business_date\trow_count")
		for _, count := range result.PersistedDailyRowCount {
			fmt.Fprintf(writer, "%s\t%d\n", count.BusinessDate, count.RowCount)
		}
		_ = writer.Flush()
	}
}
