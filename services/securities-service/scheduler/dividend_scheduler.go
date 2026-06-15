package scheduler

import (
	"context"
	"database/sql"
	"log"
	"time"

	pb_ex "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
)

// StartDividendScheduler fires at 09:00 each day and processes dividends on
// the last workday of each quarter (March, June, September, December).
func StartDividendScheduler(db *sql.DB, accountDB *sql.DB,
	portfolioClient pb_portfolio.PortfolioServiceClient,
	exchangeClient pb_ex.ExchangeServiceClient,
) {
	scheduleDividendNext(db, accountDB, portfolioClient, exchangeClient)
	log.Println("dividend-scheduler: scheduled daily at 09:00")
}

func scheduleDividendNext(db *sql.DB, accountDB *sql.DB,
	portfolioClient pb_portfolio.PortfolioServiceClient,
	exchangeClient pb_ex.ExchangeServiceClient,
) {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	time.AfterFunc(time.Until(next), func() {
		if isLastWorkdayOfQuarter(time.Now()) {
			processDividends(db, accountDB, portfolioClient, exchangeClient)
		}
		scheduleDividendNext(db, accountDB, portfolioClient, exchangeClient)
	})
}

func isLastWorkdayOfQuarter(t time.Time) bool {
	month := t.Month()
	if month != time.March && month != time.June && month != time.September && month != time.December {
		return false
	}
	// Check the next 3 days — if all are in the next month or weekend, today is last workday
	for i := 1; i <= 3; i++ {
		next := t.AddDate(0, 0, i)
		if next.Month() != month {
			break
		}
		if next.Weekday() != time.Saturday && next.Weekday() != time.Sunday {
			return false
		}
	}
	// Verify today itself is a workday
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

func processDividends(db *sql.DB, accountDB *sql.DB,
	portfolioClient pb_portfolio.PortfolioServiceClient,
	exchangeClient pb_ex.ExchangeServiceClient,
) {
	ctx := context.Background()
	log.Println("dividend-scheduler: processing quarterly dividends")

	// Build exchange rate map for currency conversions to RSD
	rateMap := map[string]float64{"RSD": 1.0}
	if exchangeClient != nil {
		if resp, err := exchangeClient.GetExchangeRates(ctx, &pb_ex.GetExchangeRatesRequest{}); err == nil {
			for _, r := range resp.Rates {
				rateMap[r.CurrencyCode] = r.MiddleRate
			}
		}
	}

	// Fetch all stocks with dividend yield
	rows, err := db.QueryContext(ctx, `
		SELECT ls.listing_id, l.price, ls.dividend_yield, e.currency
		FROM listing_stock ls
		JOIN listing l ON ls.listing_id = l.id
		JOIN stock_exchanges e ON l.exchange_id = e.id
		WHERE ls.dividend_yield > 0 AND l.price > 0`)
	if err != nil {
		log.Printf("dividend-scheduler: query stocks: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	type stockRow struct {
		listingID     int64
		price         float64
		dividendYield float64
		currency      string // from stock_exchanges.currency
	}
	var stocks []stockRow
	for rows.Next() {
		var s stockRow
		if err := rows.Scan(&s.listingID, &s.price, &s.dividendYield, &s.currency); err == nil {
			stocks = append(stocks, s)
		}
	}
	_ = rows.Close()

	today := time.Now().Format("2006-01-02")

	for _, s := range stocks {
		if portfolioClient == nil {
			continue
		}
		holdersResp, err := portfolioClient.GetHoldersByListing(ctx, &pb_portfolio.GetHoldersByListingRequest{
			ListingId: s.listingID,
		})
		if err != nil {
			log.Printf("dividend-scheduler: GetHoldersByListing %d: %v", s.listingID, err)
			continue
		}
		for _, h := range holdersResp.Holders {
			if h.Amount <= 0 {
				continue
			}
			gross := h.Amount * s.price * (s.dividendYield / 4.0)
			var taxRSD, net float64
			if h.UserType == "CLIENT" {
				taxRSD = gross * 0.15 * rateMap[s.currency]
				net = gross * 0.85
				// Record tax in portfolio-service
				_, _ = portfolioClient.CollectTaxForUser(ctx, &pb_portfolio.CollectTaxForUserRequest{
					UserId:   h.UserId,
					UserType: "CLIENT",
				})
			} else {
				// EMPLOYEE: no tax
				taxRSD = 0
				net = gross
			}

			// Credit account directly
			accountID := h.AccountId
			if accountID == 0 {
				// Fallback: find an active account for this user in the right currency
				_ = accountDB.QueryRowContext(ctx, `
					SELECT id FROM accounts
					WHERE owner_id = $1
					  AND currency_id = (SELECT id FROM currencies WHERE code = $2)
					  AND account_type NOT IN ('BANK','STATE')
					  AND status = 'ACTIVE'
					ORDER BY id LIMIT 1`, h.UserId, s.currency,
				).Scan(&accountID)
			}
			if accountID != 0 {
				_, _ = accountDB.ExecContext(ctx,
					`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
					net, accountID,
				)
			}

			// Record payout in portfolio-service
			_, err := portfolioClient.CreateDividendPayout(ctx, &pb_portfolio.CreateDividendPayoutRequest{
				UserId:         h.UserId,
				UserType:       h.UserType,
				StockListingId: s.listingID,
				Quantity:       h.Amount,
				GrossAmount:    gross,
				Currency:       s.currency,
				TaxRsd:         taxRSD,
				NetAmount:      net,
				AccountId:      accountID,
				PaymentDate:    today,
			})
			if err != nil {
				log.Printf("dividend-scheduler: CreateDividendPayout user %d listing %d: %v", h.UserId, s.listingID, err)
			}
		}
	}
	log.Println("dividend-scheduler: quarterly dividends processed")
}
