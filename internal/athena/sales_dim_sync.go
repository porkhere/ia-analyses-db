package athena

import (
	"context"
	"fmt"
	"strings"
	"time"

	appconfig "ia-analyses-db/internal/config"
	"ia-analyses-db/internal/salesdims"
)

type SalesDimensionSyncService struct {
	service             *Service
	conflictSampleLimit int
}

func NewSalesDimensionSyncService(ctx context.Context, cfg appconfig.AthenaConfig) (*SalesDimensionSyncService, error) {
	service, err := NewService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &SalesDimensionSyncService{
		service:             service,
		conflictSampleLimit: defaultConflictSampleLimit,
	}, nil
}

func (service *SalesDimensionSyncService) CollectPlan(ctx context.Context, request salesdims.SyncRequest) (salesdims.PlanResult, error) {
	window := QueryWindow{
		OwnerUserKey: request.OwnerUserKey,
		StartDate:    request.StartDate,
		EndDate:      request.EndDate,
		PreviewLimit: defaultPreviewLimit,
	}

	productCandidates, err := service.loadProductCandidates(ctx, window, request.OwnerUserID)
	if err != nil {
		return salesdims.PlanResult{}, err
	}

	branchCandidates, err := service.loadBranchCandidates(ctx, window, request.OwnerUserID)
	if err != nil {
		return salesdims.PlanResult{}, err
	}

	productConflictCount, err := service.loadConflictCount(ctx, BuildSalesProductDimConflictCountSQL(window, request.OwnerUserID))
	if err != nil {
		return salesdims.PlanResult{}, fmt.Errorf("load product conflict count: %w", err)
	}

	branchConflictCount, err := service.loadConflictCount(ctx, BuildSalesBranchDimConflictCountSQL(window, request.OwnerUserID))
	if err != nil {
		return salesdims.PlanResult{}, fmt.Errorf("load branch conflict count: %w", err)
	}

	productConflictSamples, err := service.loadProductConflictSamples(ctx, window, request.OwnerUserID)
	if err != nil {
		return salesdims.PlanResult{}, err
	}

	branchConflictSamples, err := service.loadBranchConflictSamples(ctx, window, request.OwnerUserID)
	if err != nil {
		return salesdims.PlanResult{}, err
	}

	return salesdims.PlanResult{
		Request:                request,
		ProductCandidates:      productCandidates,
		BranchCandidates:       branchCandidates,
		ProductConflictCount:   productConflictCount,
		BranchConflictCount:    branchConflictCount,
		ProductConflictSamples: productConflictSamples,
		BranchConflictSamples:  branchConflictSamples,
	}, nil
}

func (service *SalesDimensionSyncService) loadProductCandidates(ctx context.Context, window QueryWindow, ownerUserID int64) ([]salesdims.ProductCandidate, error) {
	rows, err := service.runRowsQuery(ctx, BuildSalesProductDimCandidateSQL(window, ownerUserID), 0)
	if err != nil {
		return nil, fmt.Errorf("load product dim candidates: %w", err)
	}

	result := make([]salesdims.ProductCandidate, 0, len(rows))
	for _, row := range rows {
		lastSeenAt, err := parseAthenaTimestamp(row["last_seen_at"])
		if err != nil {
			return nil, fmt.Errorf("parse product last_seen_at: %w", err)
		}

		result = append(result, salesdims.ProductCandidate{
			OwnerUserID:    mustParseInt64(row["owner_user_id"]),
			ProductNo:      row["product_no"],
			ProductName:    row["product_name"],
			CateNo:         row["cate_no"],
			CateName:       row["cate_name"],
			LastSeenAt:     lastSeenAt,
			SourceRowCount: mustParseInt64(row["source_row_count"]),
		})
	}

	return result, nil
}

func (service *SalesDimensionSyncService) loadBranchCandidates(ctx context.Context, window QueryWindow, ownerUserID int64) ([]salesdims.BranchCandidate, error) {
	rows, err := service.runRowsQuery(ctx, BuildSalesBranchDimCandidateSQL(window, ownerUserID), 0)
	if err != nil {
		return nil, fmt.Errorf("load branch dim candidates: %w", err)
	}

	result := make([]salesdims.BranchCandidate, 0, len(rows))
	for _, row := range rows {
		lastSeenAt, err := parseAthenaTimestamp(row["last_seen_at"])
		if err != nil {
			return nil, fmt.Errorf("parse branch last_seen_at: %w", err)
		}

		result = append(result, salesdims.BranchCandidate{
			OwnerUserID:    mustParseInt64(row["owner_user_id"]),
			BranchID:       row["branch_id"],
			BranchName:     row["branch_name"],
			GroupCode:      row["group_code"],
			LastSeenAt:     lastSeenAt,
			SourceRowCount: mustParseInt64(row["source_row_count"]),
		})
	}

	return result, nil
}

func (service *SalesDimensionSyncService) loadProductConflictSamples(ctx context.Context, window QueryWindow, ownerUserID int64) ([]salesdims.ProductConflictSample, error) {
	rows, err := service.runRowsQuery(ctx, BuildSalesProductDimConflictSampleSQL(window, ownerUserID, service.conflictSampleLimit), 0)
	if err != nil {
		return nil, fmt.Errorf("load product conflict samples: %w", err)
	}

	result := make([]salesdims.ProductConflictSample, 0, len(rows))
	for _, row := range rows {
		result = append(result, salesdims.ProductConflictSample{
			ProductNo:            row["product_no"],
			VariantCount:         mustParseInt64(row["variant_count"]),
			ChosenProductName:    row["chosen_product_name"],
			ChosenCateNo:         row["chosen_cate_no"],
			ChosenCateName:       row["chosen_cate_name"],
			ChosenSourceRowCount: mustParseInt64(row["chosen_source_row_count"]),
			ChosenLastSeenAt:     row["chosen_last_seen_at"],
			SampleVariants:       row["sample_variants"],
		})
	}

	return result, nil
}

func (service *SalesDimensionSyncService) loadBranchConflictSamples(ctx context.Context, window QueryWindow, ownerUserID int64) ([]salesdims.BranchConflictSample, error) {
	rows, err := service.runRowsQuery(ctx, BuildSalesBranchDimConflictSampleSQL(window, ownerUserID, service.conflictSampleLimit), 0)
	if err != nil {
		return nil, fmt.Errorf("load branch conflict samples: %w", err)
	}

	result := make([]salesdims.BranchConflictSample, 0, len(rows))
	for _, row := range rows {
		result = append(result, salesdims.BranchConflictSample{
			BranchID:             row["branch_id"],
			VariantCount:         mustParseInt64(row["variant_count"]),
			ChosenBranchName:     row["chosen_branch_name"],
			ChosenGroupCode:      row["chosen_group_code"],
			ChosenSourceRowCount: mustParseInt64(row["chosen_source_row_count"]),
			ChosenLastSeenAt:     row["chosen_last_seen_at"],
			SampleVariants:       row["sample_variants"],
		})
	}

	return result, nil
}

func (service *SalesDimensionSyncService) loadConflictCount(ctx context.Context, sql string) (int64, error) {
	rows, err := service.runRowsQuery(ctx, sql, 1)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	return mustParseInt64(rows[0]["conflict_key_count"]), nil
}

func (service *SalesDimensionSyncService) runRowsQuery(ctx context.Context, sql string, limit int) ([]map[string]string, error) {
	queryExecutionID, _, err := service.service.runQuery(ctx, sql)
	if err != nil {
		return nil, err
	}

	return service.service.readRows(ctx, queryExecutionID, limit)
}

func parseAthenaTimestamp(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000000",
		time.DateTime,
		time.RFC3339,
		dateLayout,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported Athena timestamp %q", raw)
}
