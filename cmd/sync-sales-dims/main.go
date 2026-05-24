package main

// Bridge copy only: future sync-sales-dims command edits should land in ia-analyses-go.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ia-analyses-db/internal/athena"
	"ia-analyses-db/internal/config"
	"ia-analyses-db/internal/postgres"
	"ia-analyses-db/internal/salesdims"
)

const dateLayout = "2006-01-02"

type options struct {
	ownerUserKey string
	ownerUserID  int64
	startDate    string
	endDate      string
	plan         bool
	apply        bool
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

	startDate, err := time.Parse(dateLayout, opts.startDate)
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid --start-date: %v\n", err)
		return 2
	}

	endDateText := opts.endDate
	if endDateText == "" {
		endDateText = opts.startDate
	}

	endDate, err := time.Parse(dateLayout, endDateText)
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid --end-date: %v\n", err)
		return 2
	}

	if endDate.Before(startDate) {
		fmt.Fprintln(stderr, "error: --end-date must be on or after --start-date")
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	request := salesdims.SyncRequest{
		OwnerUserKey: opts.ownerUserKey,
		OwnerUserID:  opts.ownerUserID,
		StartDate:    startDate,
		EndDate:      endDate,
		Mode:         syncMode(opts),
	}

	service, err := athena.NewSalesDimensionSyncService(context.Background(), cfg.Athena)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	plan, err := service.CollectPlan(context.Background(), request)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	renderHeader(stdout, request, cfg)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "dimension_plan_summary:")
	fmt.Fprintln(stdout, salesdims.RenderPlanSummaryTable(plan))
	if len(plan.ProductConflictSamples) > 0 {
		fmt.Fprintln(stdout, "product_conflict_summary:")
		fmt.Fprintln(stdout, salesdims.RenderProductConflictTable(plan.ProductConflictSamples))
	}
	if len(plan.BranchConflictSamples) > 0 {
		fmt.Fprintln(stdout, "branch_conflict_summary:")
		fmt.Fprintln(stdout, salesdims.RenderBranchConflictTable(plan.BranchConflictSamples))
	}

	if request.Mode == salesdims.SyncModePlan {
		fmt.Fprintf(stdout, "writes_performed: false\n")
		fmt.Fprintf(stdout, "sales_fact_written: false\n")
		return 0
	}

	db, err := postgres.Open(cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	defer db.Close()

	writer := postgres.NewSalesDimensionSyncWriter(db)
	applyResult, err := writer.Apply(context.Background(), plan)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "dimension_apply_summary:")
	fmt.Fprintln(stdout, salesdims.RenderApplySummaryTable(applyResult))
	fmt.Fprintf(stdout, "written_tables: %s\n", strings.Join(applyResult.WrittenTables, ", "))
	fmt.Fprintf(stdout, "sales_fact_written: %t\n", applyResult.SalesFactWritten)

	return 0
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var opts options

	fs := flag.NewFlagSet("sync-sales-dims", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.ownerUserKey, "owner-user-key", "", "owner user key")
	fs.Int64Var(&opts.ownerUserID, "owner-user-id", 0, "owner user id for local PostgreSQL product/branch dim sync")
	fs.StringVar(&opts.startDate, "start-date", "", "sync start date, format: YYYY-MM-DD")
	fs.StringVar(&opts.endDate, "end-date", "", "sync end date, format: YYYY-MM-DD")
	fs.BoolVar(&opts.plan, "plan", false, "plan-only Athena read for product/branch dims")
	fs.BoolVar(&opts.apply, "apply", false, "Athena read + PostgreSQL upsert for product/branch dims")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	if opts.ownerUserKey == "" {
		return options{}, fmt.Errorf("--owner-user-key is required")
	}
	if opts.ownerUserID <= 0 {
		return options{}, fmt.Errorf("--owner-user-id is required")
	}
	if opts.startDate == "" {
		return options{}, fmt.Errorf("--start-date is required")
	}
	if opts.plan == opts.apply {
		return options{}, fmt.Errorf("exactly one of --plan or --apply is required")
	}

	return opts, nil
}

func syncMode(opts options) salesdims.SyncMode {
	if opts.apply {
		return salesdims.SyncModeApply
	}

	return salesdims.SyncModePlan
}

func renderHeader(stdout io.Writer, request salesdims.SyncRequest, cfg config.AppConfig) {
	fmt.Fprintf(stdout, "sync-sales-dims phase 2C-5.2 %s\n", request.Mode)
	fmt.Fprintf(stdout, "owner_user_key: %s\n", request.OwnerUserKey)
	fmt.Fprintf(stdout, "owner_user_id: %d\n", request.OwnerUserID)
	fmt.Fprintf(stdout, "date_range: %s -> %s\n", request.StartDate.Format(dateLayout), request.EndDate.Format(dateLayout))
	fmt.Fprintf(stdout, "athena_database: %s\n", cfg.Athena.Database)
	fmt.Fprintf(stdout, "athena_workgroup: %s\n", cfg.Athena.Workgroup)
	fmt.Fprintf(stdout, "athena_output: %s\n", cfg.Athena.OutputLocation)
	fmt.Fprintf(stdout, "postgres_target: %s\n", postgres.RedactedDSN(cfg.Database))
	fmt.Fprintln(stdout, "allowed_pg_writes: pos_product_dim, pos_branch_dim")
	fmt.Fprintln(stdout, "forbidden_pg_writes: pos_sales_hourly_fact, pos_order_daily_fact, pos_payment_daily_fact, pos_condiment_hourly_fact, pos_branch_opening_daily_fact")
	fmt.Fprintf(stdout, "sales_fact_write_enabled: %t\n", false)
	fmt.Fprintf(stdout, "conflict_selection_rule: %s\n", salesdims.ConflictSelectionRule)
	fmt.Fprintf(stdout, "group_code_policy: %s\n", salesdims.GroupCodePolicy)
	fmt.Fprintln(stdout, "notes:")
	fmt.Fprintln(stdout, "  - product source = status-aware sales candidate key space joined with order_items_parquet")
	fmt.Fprintln(stdout, "  - branch source = status-aware sales candidate key space joined with orders_parquet branch name")
	fmt.Fprintln(stdout, "  - sync-sales-dims does not write pos_sales_hourly_fact")
}
