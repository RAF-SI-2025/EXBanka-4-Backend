package repository

import (
	"context"
	"database/sql"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/models"
)

// UpsertHolding updates portfolio holdings on each order fill.
// BUY: creates or updates entry with weighted average buy price.
// SELL: decrements amount; deletes entry if amount reaches zero.
func UpsertHolding(ctx context.Context, db *sql.DB, userID int64, userType string, listingID, accountID int64, qty int32, price float64, direction string) error {
	if direction == "BUY" {
		_, err := db.ExecContext(ctx, `
			INSERT INTO portfolio_entry (user_id, user_type, listing_id, amount, buy_price, account_id, last_modified)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (user_id, listing_id) DO UPDATE SET
				buy_price     = (portfolio_entry.amount * portfolio_entry.buy_price + $4 * $5) / (portfolio_entry.amount + $4),
				amount        = portfolio_entry.amount + $4,
				last_modified = NOW()`,
			userID, userType, listingID, qty, price, accountID,
		)
		return err
	}

	// SELL: decrement amount
	_, err := db.ExecContext(ctx, `
		UPDATE portfolio_entry
		SET amount = amount - $1, last_modified = NOW()
		WHERE user_id = $2 AND listing_id = $3`,
		qty, userID, listingID,
	)
	if err != nil {
		return err
	}

	// Remove entry if fully sold
	_, err = db.ExecContext(ctx, `
		DELETE FROM portfolio_entry
		WHERE user_id = $1 AND listing_id = $2 AND amount <= 0`,
		userID, listingID,
	)
	return err
}

// GetHoldings returns all portfolio entries for a user.
func GetHoldings(ctx context.Context, db *sql.DB, userID int64) ([]models.PortfolioEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, user_type, listing_id, amount, buy_price, last_modified, is_public, public_amount, account_id
		FROM portfolio_entry
		WHERE user_id = $1 AND amount > 0
		ORDER BY last_modified DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.PortfolioEntry
	for rows.Next() {
		var e models.PortfolioEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserType, &e.ListingID, &e.Amount, &e.BuyPrice, &e.LastModified, &e.IsPublic, &e.PublicAmount, &e.AccountID); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
