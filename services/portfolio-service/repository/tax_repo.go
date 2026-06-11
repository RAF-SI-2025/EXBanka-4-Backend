package repository

import (
	"context"
	"database/sql"
	"time"
)

type TaxRecord struct {
	ID        int64
	UserID    int64
	UserType  string
	AmountRSD float64
	Month     int
	Year      int
	IsPaid    bool
	PaidAt    *time.Time
}

type TaxDebt struct {
	UserID   int64
	UserType string
	DebtRSD  float64
}

func InsertTaxRecord(ctx context.Context, db *sql.DB, userID int64, userType string, amountRSD float64, month, year int) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO tax_record (user_id, user_type, amount_rsd, month, year)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, userType, amountRSD, month, year,
	)
	return err
}

func GetUnpaidRecords(ctx context.Context, db *sql.DB) ([]TaxRecord, error) {
	return queryTaxRecords(ctx, db, `
		SELECT id, user_id, user_type, amount_rsd, month, year, is_paid, paid_at
		FROM tax_record WHERE is_paid = FALSE`)
}

func GetUnpaidRecordsForUser(ctx context.Context, db *sql.DB, userID int64, userType string) ([]TaxRecord, error) {
	return queryTaxRecords(ctx, db, `
		SELECT id, user_id, user_type, amount_rsd, month, year, is_paid, paid_at
		FROM tax_record WHERE is_paid = FALSE AND user_id = $1 AND user_type = $2`,
		userID, userType,
	)
}

func MarkTaxPaid(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE tax_record SET is_paid = TRUE, paid_at = NOW() WHERE id = $1`, id,
	)
	return err
}

// GetMyTax returns amounts paid in the given year and unpaid in the current month.
func GetMyTax(ctx context.Context, db *sql.DB, userID int64, userType string, year, month int) (paidThisYear, unpaidThisMonth float64, err error) {
	row := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount_rsd) FILTER (WHERE is_paid = TRUE AND year = $3), 0),
			COALESCE(SUM(amount_rsd) FILTER (WHERE is_paid = FALSE AND year = $3 AND month = $4), 0)
		FROM tax_record
		WHERE user_id = $1 AND user_type = $2`,
		userID, userType, year, month,
	)
	err = row.Scan(&paidThisYear, &unpaidThisMonth)
	return
}

// GetTaxDebtList returns all users who have any tax record, with their total unpaid
// debt in RSD. Pass userTypeFilter="" to return all user types.
func GetTaxDebtList(ctx context.Context, db *sql.DB, userTypeFilter string) ([]TaxDebt, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT user_id, user_type,
		       SUM(CASE WHEN is_paid = FALSE THEN amount_rsd ELSE 0 END) AS debt_rsd
		FROM tax_record
		WHERE ($1 = '' OR user_type = $1)
		GROUP BY user_id, user_type
		HAVING SUM(CASE WHEN is_paid = FALSE THEN amount_rsd ELSE 0 END) >= 0
		ORDER BY user_id`,
		userTypeFilter)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var debts []TaxDebt
	for rows.Next() {
		var d TaxDebt
		if err := rows.Scan(&d.UserID, &d.UserType, &d.DebtRSD); err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	return debts, rows.Err()
}

func queryTaxRecords(ctx context.Context, db *sql.DB, query string, args ...interface{}) ([]TaxRecord, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []TaxRecord
	for rows.Next() {
		var r TaxRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.UserType, &r.AmountRSD, &r.Month, &r.Year, &r.IsPaid, &r.PaidAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
