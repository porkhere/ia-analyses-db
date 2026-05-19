package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ia-analyses-db/internal/sales"
	"ia-analyses-db/internal/validation"
)

var ErrActualWriteDisabled = errors.New("phase 2C-5 write skeleton ready but actual insert disabled")

type DisabledSalesFactWriter struct{}

func (DisabledSalesFactWriter) BeginDayReplace(_ context.Context, ownerUserID int64, businessDate time.Time) (sales.DayReplaceTx, error) {
	return nil, fmt.Errorf("%w: owner_user_id=%d business_date=%s", ErrActualWriteDisabled, ownerUserID, businessDate.Format("2006-01-02"))
}

type SQLSalesFactWriter struct {
	db *sql.DB
}

func NewSQLSalesFactWriter(db *sql.DB) *SQLSalesFactWriter {
	return &SQLSalesFactWriter{db: db}
}

func (writer *SQLSalesFactWriter) BeginDayReplace(ctx context.Context, ownerUserID int64, businessDate time.Time) (sales.DayReplaceTx, error) {
	if writer == nil || writer.db == nil {
		return nil, fmt.Errorf("postgres DB handle is required")
	}

	tx, err := writer.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	return &sqlSalesFactTx{
		tx:           tx,
		ownerUserID:  ownerUserID,
		businessDate: businessDate,
	}, nil
}

type sqlSalesFactTx struct {
	tx           *sql.Tx
	ownerUserID  int64
	businessDate time.Time
}

func (tx *sqlSalesFactTx) DeleteExistingDay(ctx context.Context) error {
	if _, err := tx.tx.ExecContext(
		ctx,
		`DELETE FROM public.pos_sales_hourly_fact WHERE owner_user_id = $1 AND business_date = $2`,
		tx.ownerUserID,
		tx.businessDate,
	); err != nil {
		return fmt.Errorf("delete existing day rows: %w", err)
	}

	return nil
}

func (tx *sqlSalesFactTx) InsertRows(ctx context.Context, rows []sales.FactRow) error {
	for _, row := range rows {
		if _, err := tx.tx.ExecContext(
			ctx,
			`INSERT INTO public.pos_sales_hourly_fact (
				owner_user_id,
				business_date,
				hour_of_day,
				branch_id,
				product_no,
				order_type_id,
				payment_type_id,
				qty_milli,
				gross_sales_milli,
				discount_milli,
				surcharge_milli,
				net_sales_milli,
				sales_ex_tax_milli,
				tax_milli,
				included_tax_milli,
				excluded_tax_milli,
				updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12, $13, $14, $15, $16,
				NOW()
			)`,
			row.OwnerUserID,
			row.BusinessDate,
			row.HourOfDay,
			row.BranchID,
			row.ProductNo,
			row.OrderTypeID,
			row.PaymentTypeID,
			row.QtyMilli,
			row.GrossSalesMilli,
			row.DiscountMilli,
			row.SurchargeMilli,
			row.NetSalesMilli,
			row.SalesExTaxMilli,
			row.TaxMilli,
			row.IncludedTaxMilli,
			row.ExcludedTaxMilli,
		); err != nil {
			return fmt.Errorf("insert sales fact row: %w", err)
		}
	}

	return nil
}

func (tx *sqlSalesFactTx) LoadPersistedTargetMetrics(ctx context.Context) (validation.MetricsSnapshot, error) {
	return loadPersistedTargetMetrics(ctx, tx.tx, sales.DayScope{
		OwnerUserID:  tx.ownerUserID,
		BusinessDate: tx.businessDate,
	})
}

func (tx *sqlSalesFactTx) Commit(_ context.Context) error {
	if err := tx.tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (tx *sqlSalesFactTx) Rollback(_ context.Context) error {
	if err := tx.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("rollback transaction: %w", err)
	}

	return nil
}
