package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ia-analyses-db/internal/salesdims"
)

type SalesDimensionSyncWriter struct {
	db *sql.DB
}

func NewSalesDimensionSyncWriter(db *sql.DB) *SalesDimensionSyncWriter {
	return &SalesDimensionSyncWriter{db: db}
}

func (writer *SalesDimensionSyncWriter) Apply(ctx context.Context, plan salesdims.PlanResult) (salesdims.ApplyResult, error) {
	if writer == nil || writer.db == nil {
		return salesdims.ApplyResult{}, fmt.Errorf("postgres DB handle is required")
	}

	tx, err := writer.db.BeginTx(ctx, nil)
	if err != nil {
		return salesdims.ApplyResult{}, fmt.Errorf("begin sales dim sync tx: %w", err)
	}

	productStmt, err := tx.PrepareContext(ctx, `INSERT INTO public.pos_product_dim (
		owner_user_id,
		product_no,
		product_name,
		product_name_normalized,
		cate_no,
		cate_name,
		is_active,
		last_seen_at,
		updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, NOW())
	ON CONFLICT (owner_user_id, product_no) DO UPDATE SET
		product_name = EXCLUDED.product_name,
		product_name_normalized = EXCLUDED.product_name_normalized,
		cate_no = EXCLUDED.cate_no,
		cate_name = EXCLUDED.cate_name,
		is_active = TRUE,
		last_seen_at = COALESCE(EXCLUDED.last_seen_at, public.pos_product_dim.last_seen_at),
		updated_at = NOW()`)
	if err != nil {
		_ = tx.Rollback()
		return salesdims.ApplyResult{}, fmt.Errorf("prepare product dim upsert: %w", err)
	}
	defer productStmt.Close()

	branchStmt, err := tx.PrepareContext(ctx, `INSERT INTO public.pos_branch_dim (
		owner_user_id,
		branch_id,
		branch_name,
		branch_name_normalized,
		group_code,
		is_active,
		last_seen_at,
		updated_at
	) VALUES ($1, $2, $3, $4, $5, TRUE, $6, NOW())
	ON CONFLICT (owner_user_id, branch_id) DO UPDATE SET
		branch_name = EXCLUDED.branch_name,
		branch_name_normalized = EXCLUDED.branch_name_normalized,
		group_code = EXCLUDED.group_code,
		is_active = TRUE,
		last_seen_at = COALESCE(EXCLUDED.last_seen_at, public.pos_branch_dim.last_seen_at),
		updated_at = NOW()`)
	if err != nil {
		_ = tx.Rollback()
		return salesdims.ApplyResult{}, fmt.Errorf("prepare branch dim upsert: %w", err)
	}
	defer branchStmt.Close()

	for _, candidate := range plan.ProductCandidates {
		if _, err := productStmt.ExecContext(
			ctx,
			candidate.OwnerUserID,
			candidate.ProductNo,
			candidate.ProductName,
			normalizeDimName(candidate.ProductName),
			nullIfBlank(candidate.CateNo),
			nullIfBlank(candidate.CateName),
			nullTime(candidate.LastSeenAt),
		); err != nil {
			_ = tx.Rollback()
			return salesdims.ApplyResult{}, fmt.Errorf("upsert pos_product_dim product_no=%s: %w", candidate.ProductNo, err)
		}
	}

	for _, candidate := range plan.BranchCandidates {
		if _, err := branchStmt.ExecContext(
			ctx,
			candidate.OwnerUserID,
			candidate.BranchID,
			candidate.BranchName,
			normalizeDimName(candidate.BranchName),
			nullIfBlank(candidate.GroupCode),
			nullTime(candidate.LastSeenAt),
		); err != nil {
			_ = tx.Rollback()
			return salesdims.ApplyResult{}, fmt.Errorf("upsert pos_branch_dim branch_id=%s: %w", candidate.BranchID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return salesdims.ApplyResult{}, fmt.Errorf("commit sales dim sync tx: %w", err)
	}

	return salesdims.ApplyResult{
		Plan:               plan,
		ProductUpsertCount: int64(len(plan.ProductCandidates)),
		BranchUpsertCount:  int64(len(plan.BranchCandidates)),
		WrittenTables:      []string{"pos_product_dim", "pos_branch_dim"},
		SalesFactWritten:   false,
	}, nil
}

func normalizeDimName(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}

	return strings.Join(strings.Fields(trimmed), " ")
}

func nullIfBlank(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	return trimmed
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value
}
