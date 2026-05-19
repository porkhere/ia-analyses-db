package athena

import "time"

const dateLayout = "2006-01-02"

type DryRunMode string

const (
	DryRunModeFast DryRunMode = "fast"
	DryRunModeFull DryRunMode = "full"
)

type QueryWindow struct {
	OwnerUserKey string
	StartDate    time.Time
	EndDate      time.Time
	PreviewLimit int
	DryRun       bool
	DryRunMode   DryRunMode
}

type Plan struct {
	Database       string
	Workgroup      string
	OutputLocation string
	WindowDays     int
	SourceTables   []string
	Grain          []string
	Notes          []string
}

func BuildPlan(window QueryWindow, database string, workgroup string, outputLocation string) Plan {
	windowDays := inclusiveDays(window.StartDate, window.EndDate)
	dryRunMode := normalizeDryRunMode(window.DryRunMode)
	var notes []string

	if window.DryRun {
		notes = []string{
			"phase 2A dry-run 會真的送出 Athena 查詢，但不會寫 PostgreSQL",
			"owner_user_key 目前只作執行標示，不參與 Athena 過濾",
			"付款型態預期先做 order-level 聚合，再映射到 payment_type_id",
			"order_additions 目前會以 order-level discount/surcharge 比例分攤進 preview aggregation",
			"金額統一使用 milli-TWD，避免後續浮點誤差",
			"source metric 與 preview metric 皆來自 Athena 真實 query execution 統計",
		}

		if dryRunMode == DryRunModeFast {
			notes = append(notes, "dry-run fast mode 只跑 source metrics、result metrics、reconciliation summary、top tax delta sample、status excluded summary")
		} else {
			notes = append(notes, "dry-run full mode 會額外輸出 mapping、preview sample 與完整 debug 區塊；適合短日期窗深挖")
		}
	} else {
		notes = []string{
			"phase 2C-5.1 已接上 sales fact Athena status-aware source candidate provider + validation gate integration",
			"validate-only / write-plan 會真的查 Athena，並只讀 PostgreSQL 執行 dimension gate / negative schema gate；仍不會真正寫入 PostgreSQL",
			"write skeleton 仍要求 day-level replace、transaction boundary、validation first 與 hard gate fail stop / rollback",
			"owner_user_id 目前必須由 CLI 顯式提供；owner_user_key -> owner_user_id resolution 尚未實作",
		}
	}

	return Plan{
		Database:       database,
		Workgroup:      workgroup,
		OutputLocation: outputLocation,
		WindowDays:     windowDays,
		SourceTables: []string{
			"orders_parquet",
			"order_items_parquet",
			"order_additions_parquet",
			"order_payments_parquet",
		},
		Grain: []string{
			"owner_user_id",
			"business_date",
			"hour_of_day",
			"branch_id",
			"product_no",
			"order_type_id",
			"payment_type_id",
		},
		Notes: notes,
	}
}

func normalizeDryRunMode(mode DryRunMode) DryRunMode {
	if mode == DryRunModeFull {
		return DryRunModeFull
	}

	return DryRunModeFast
}

func inclusiveDays(startDate time.Time, endDate time.Time) int {
	return int(endDate.Sub(startDate).Hours()/24) + 1
}
