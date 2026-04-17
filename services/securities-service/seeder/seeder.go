package seeder

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

// wellKnownStocks is a curated list of popular S&P 500 stocks.
// OutstandingShares and DividendYield are hardcoded public data.
var wellKnownStocks = []struct {
	Symbol            string
	Name              string
	OutstandingShares int64
	DividendYield     float64 // decimal, e.g. 0.03 = 3%
}{
	{"AAPL", "Apple Inc.", 15_408_756_000, 0.0051},
	{"MSFT", "Microsoft Corporation", 7_433_088_000, 0.0082},
	{"GOOGL", "Alphabet Inc.", 12_155_016_000, 0.0},
	{"AMZN", "Amazon.com Inc.", 10_598_400_000, 0.0},
	{"NVDA", "NVIDIA Corporation", 24_387_000_000, 0.0003},
	{"META", "Meta Platforms Inc.", 2_531_000_000, 0.0035},
	{"TSLA", "Tesla Inc.", 3_210_000_000, 0.0},
	{"JPM", "JPMorgan Chase & Co.", 2_849_000_000, 0.0230},
	{"V", "Visa Inc.", 2_049_000_000, 0.0085},
	{"JNJ", "Johnson & Johnson", 2_401_000_000, 0.0320},
	{"WMT", "Walmart Inc.", 8_048_000_000, 0.0099},
	{"XOM", "Exxon Mobil Corporation", 3_979_000_000, 0.0380},
	{"MA", "Mastercard Inc.", 929_000_000, 0.0066},
	{"UNH", "UnitedHealth Group Inc.", 924_000_000, 0.0165},
	{"HD", "The Home Depot Inc.", 993_000_000, 0.0240},
	{"PG", "Procter & Gamble Co.", 2_356_000_000, 0.0240},
	{"BAC", "Bank of America Corp.", 7_790_000_000, 0.0260},
	{"KO", "The Coca-Cola Company", 4_289_000_000, 0.0300},
	{"NFLX", "Netflix Inc.", 429_000_000, 0.0},
	{"DIS", "The Walt Disney Company", 1_828_000_000, 0.0},
}

// forexPairs is the fixed list of pairs to seed. Liquidity is assigned statically.
var forexPairs = []struct {
	From      string
	To        string
	Liquidity string
}{
	{"EUR", "USD", "HIGH"},
	{"GBP", "USD", "HIGH"},
	{"USD", "JPY", "HIGH"},
	{"USD", "CHF", "MEDIUM"},
	{"AUD", "USD", "MEDIUM"},
	{"USD", "CAD", "MEDIUM"},
	{"NZD", "USD", "MEDIUM"},
	{"EUR", "GBP", "MEDIUM"},
	{"EUR", "JPY", "LOW"},
	{"GBP", "JPY", "LOW"},
}

// Seed populates the database with exchanges, stocks, forex pairs, futures, and options.
// exchangeCSVData and futureCSVData are the raw bytes of exchange_1.csv and future_data.csv.
// It is idempotent: if listings are already present it returns immediately.
// Intended to be called in a goroutine so it does not block the gRPC server.
func Seed(db *sql.DB, alpacaKey, alpacaSecret, avKey string, exchangeCSV, futureDataCSV []byte) {
	log.Println("seeder: checking if seed is needed")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM listing`).Scan(&count); err != nil {
		log.Printf("seeder: count check failed: %v", err)
		return
	}
	if count > 0 {
		// Listings exist — check if options are also present.
		var optCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM listing WHERE type = 'OPTION'`).Scan(&optCount); err == nil && optCount == 0 {
			log.Println("seeder: listings present but no options — seeding options only")
			seedOptionsForAllStocks(db)
		} else {
			log.Printf("seeder: %d listings already present, skipping", count)
		}
		return
	}

	log.Println("seeder: starting full data import")

	// ── 1. Exchanges ─────────────────────────────────────────────────────────────
	exchanges, err := ParseExchanges(exchangeCSV)
	if err != nil {
		log.Printf("seeder: parse exchanges: %v", err)
		return
	}
	log.Printf("seeder: inserting %d exchanges", len(exchanges))
	for _, ex := range exchanges {
		_, err := db.Exec(`
			INSERT INTO stock_exchanges (name, acronym, mic_code, polity, currency, timezone)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (mic_code) DO NOTHING`,
			ex.Name, ex.Acronym, ex.MICCode, ex.Country, ex.Currency, ex.Timezone,
		)
		if err != nil {
			log.Printf("seeder: insert exchange %s: %v", ex.MICCode, err)
		}
	}

	// Get a default exchange ID (NYSE) to associate listings that don't have a better match.
	var defaultExchangeID int64
	if err := db.QueryRow(`SELECT id FROM stock_exchanges WHERE mic_code = 'XNYS' LIMIT 1`).Scan(&defaultExchangeID); err != nil {
		// Fall back to the first available exchange.
		if err2 := db.QueryRow(`SELECT id FROM stock_exchanges ORDER BY id LIMIT 1`).Scan(&defaultExchangeID); err2 != nil {
			log.Printf("seeder: no exchanges found after import: %v", err2)
			return
		}
	}

	// ── 2. Stocks ─────────────────────────────────────────────────────────────────
	log.Printf("seeder: seeding %d stocks", len(wellKnownStocks))
	for _, s := range wellKnownStocks {
		seedStock(db, s.Symbol, s.Name, s.OutstandingShares, s.DividendYield, defaultExchangeID, alpacaKey, alpacaSecret, avKey)
	}

	// ── 3. Forex pairs ────────────────────────────────────────────────────────────
	// Reuse the first exchange that uses USD as a proxy for forex listings.
	var forexExchangeID int64
	if err := db.QueryRow(`SELECT id FROM stock_exchanges WHERE currency = 'USD' ORDER BY id LIMIT 1`).Scan(&forexExchangeID); err != nil {
		forexExchangeID = defaultExchangeID
	}

	log.Printf("seeder: seeding %d forex pairs", len(forexPairs))
	for _, fp := range forexPairs {
		seedForex(db, fp.From, fp.To, fp.Liquidity, forexExchangeID)
	}

	// ── 4. Futures ────────────────────────────────────────────────────────────────
	futures, err := ParseFutures(futureDataCSV)
	if err != nil {
		log.Printf("seeder: parse futures: %v", err)
	} else {
		log.Printf("seeder: seeding %d futures contracts", len(futures))
		settlement := lastBusinessDayOfMonth(time.Now())
		for _, f := range futures {
			seedFuture(db, f, defaultExchangeID, settlement)
		}
	}

	// ── 5. Options ────────────────────────────────────────────────────────────────
	seedOptionsForAllStocks(db)

	log.Println("seeder: data import complete")
}

// seedStock fetches metadata + history and inserts one stock.
func seedStock(db *sql.DB, ticker, fallbackName string, outstandingShares int64, dividendYield float64, exchangeID int64, alpacaKey, alpacaSecret, avKey string) {
	name := fallbackName
	if name == "" {
		name = ticker
	}
	exID := exchangeID

	var listingID int64
	err := db.QueryRow(`
		INSERT INTO listing (ticker, name, exchange_id, type, price, ask, bid, volume, change)
		VALUES ($1, $2, $3, 'STOCK', 0, 0, 0, 0, 0)
		ON CONFLICT (ticker) DO NOTHING
		RETURNING id`, ticker, name, exID).Scan(&listingID)
	if err == sql.ErrNoRows {
		if err2 := db.QueryRow(`SELECT id FROM listing WHERE ticker = $1`, ticker).Scan(&listingID); err2 != nil {
			log.Printf("seeder: lookup existing listing %s: %v", ticker, err2)
			return
		}
	} else if err != nil {
		log.Printf("seeder: insert listing %s: %v", ticker, err)
		return
	}

	if _, err := db.Exec(`
		INSERT INTO listing_stock (listing_id, outstanding_shares, dividend_yield)
		VALUES ($1, $2, $3)
		ON CONFLICT (listing_id) DO NOTHING`,
		listingID, outstandingShares, dividendYield); err != nil {
		log.Printf("seeder: insert listing_stock %s: %v", ticker, err)
	}

	// Historical prices from Yahoo Finance (free, no key, 30-day daily bars).
	bars, err := FetchStockBarsYahoo(ticker)
	if err != nil {
		log.Printf("seeder: yahoo bars %s: %v", ticker, err)
	} else {
		insertDailyBars(db, listingID, bars)
	}

	// Current price snapshot from Alpaca.
	q, err := FetchStockSnapshot(ticker, alpacaKey, alpacaSecret)
	if err != nil {
		log.Printf("seeder: stock snapshot %s: %v", ticker, err)
	} else if q != nil {
		_, err = db.Exec(`
			UPDATE listing SET price=$2, ask=$3, bid=$4, change=$5, volume=$6, last_refresh=$7
			WHERE id=$1`,
			listingID, q.Price, q.Ask, q.Bid, q.Change, q.Volume, time.Now())
		if err != nil {
			log.Printf("seeder: update price %s: %v", ticker, err)
		}
	}
}

// seedForex inserts a forex pair listing with history and current rate.
// Uses exchangerate-api.com for current rates and Frankfurter for history (both free, no key needed).
func seedForex(db *sql.DB, from, to, liquidity string, exchangeID int64) {
	ticker := from + to
	name := from + "/" + to

	var listingID int64
	err := db.QueryRow(`
		INSERT INTO listing (ticker, name, exchange_id, type, price, ask, bid, volume, change)
		VALUES ($1, $2, $3, 'FOREX_PAIR', 0, 0, 0, 0, 0)
		ON CONFLICT (ticker) DO NOTHING
		RETURNING id`, ticker, name, exchangeID).Scan(&listingID)
	if err == sql.ErrNoRows {
		if err2 := db.QueryRow(`SELECT id FROM listing WHERE ticker = $1`, ticker).Scan(&listingID); err2 != nil {
			log.Printf("seeder: lookup forex %s: %v", ticker, err2)
			return
		}
	} else if err != nil {
		log.Printf("seeder: insert forex listing %s: %v", ticker, err)
		return
	}

	if _, err := db.Exec(`
		INSERT INTO listing_forex_pair (listing_id, base_currency, quote_currency, liquidity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (listing_id) DO NOTHING`,
		listingID, from, to, liquidity); err != nil {
		log.Printf("seeder: insert forex pair %s: %v", ticker, err)
	}

	// Historical prices from Frankfurter (ECB rates, free, no key).
	bars, err := FetchFXDailyFree(from, to)
	if err != nil {
		log.Printf("seeder: FX daily %s: %v", ticker, err)
	} else {
		insertDailyBars(db, listingID, bars)
	}

	// Current rate from exchangerate-api.com (free, no key).
	rate, err := FetchFXRateFree(from, to)
	if err != nil {
		log.Printf("seeder: FX rate %s: %v", ticker, err)
	} else if rate != nil {
		_, err = db.Exec(`
			UPDATE listing SET price=$2, ask=$3, bid=$4, last_refresh=$5
			WHERE id=$1`,
			listingID, rate.Price, rate.Ask, rate.Bid, time.Now())
		if err != nil {
			log.Printf("seeder: update forex price %s: %v", ticker, err)
		}
	}
}

// seedFuture inserts one futures contract listing.
func seedFuture(db *sql.DB, f FutureRow, exchangeID int64, settlement time.Time) {
	ticker := sanitizeTicker(f.ContractName)

	// Derive initial price from the CSV maintenance margin.
	// The handler computes: maintenanceMargin = contractSize * price * 0.10
	// So: price = maintenanceMargin / (contractSize * 0.10)
	price := 0.0
	if f.ContractSize > 0 && f.MaintenanceMargin > 0 {
		price = f.MaintenanceMargin / (f.ContractSize * 0.10)
	}
	ask := price * 1.001
	bid := price * 0.999

	var listingID int64
	err := db.QueryRow(`
		INSERT INTO listing (ticker, name, exchange_id, type, price, ask, bid, volume, change)
		VALUES ($1, $2, $3, 'FUTURES_CONTRACT', $4, $5, $6, 1000, 0)
		ON CONFLICT (ticker) DO NOTHING
		RETURNING id`, ticker, f.ContractName, exchangeID, price, ask, bid).Scan(&listingID)
	if err == sql.ErrNoRows {
		if err2 := db.QueryRow(`SELECT id FROM listing WHERE ticker = $1`, ticker).Scan(&listingID); err2 != nil {
			log.Printf("seeder: lookup future %s: %v", ticker, err2)
			return
		}
	} else if err != nil {
		log.Printf("seeder: insert future listing %s: %v", ticker, err)
		return
	}

	if _, err := db.Exec(`
		INSERT INTO listing_futures_contract (listing_id, contract_size, contract_unit, settlement_date)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (listing_id) DO NOTHING`,
		listingID, f.ContractSize, f.ContractUnit, settlement); err != nil {
		log.Printf("seeder: insert futures contract %s: %v", ticker, err)
	}
}

// seedOptionsForAllStocks fetches all stock listings and seeds options for each.
func seedOptionsForAllStocks(db *sql.DB) {
	rows, err := db.Query(`SELECT l.id, l.ticker, l.price FROM listing l JOIN listing_stock ls ON ls.listing_id = l.id`)
	if err != nil {
		log.Printf("seeder: query stocks for options: %v", err)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("seeder: rows close: %v", err)
		}
	}()
	for rows.Next() {
		var id int64
		var ticker string
		var price float64
		if err := rows.Scan(&id, &ticker, &price); err != nil {
			continue
		}
		seedOptions(db, id, ticker, price)
	}
}

// seedOptions generates or fetches options for a stock and inserts them.
func seedOptions(db *sql.DB, stockListingID int64, ticker string, price float64) {
	opts, err := FetchOptions(ticker, stockListingID)
	if err != nil {
		log.Printf("seeder: yahoo options %s: %v — using generated options", ticker, err)
	}
	if len(opts) == 0 {
		opts = GenerateOptions(stockListingID, price)
	}

	for _, opt := range opts {
		// Build a synthetic ticker: e.g. AAPL240119C00150000
		optTicker := buildOptionTicker(ticker, opt)
		var optListingID int64
		err := db.QueryRow(`
			INSERT INTO listing (ticker, name, exchange_id, type, price, ask, bid, volume, change)
			SELECT $1, $2, exchange_id, 'OPTION', $3, $3, $3, 0, 0 FROM listing WHERE id = $4
			ON CONFLICT (ticker) DO NOTHING
			RETURNING id`, optTicker, optTicker, opt.StrikePrice, stockListingID).Scan(&optListingID)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			log.Printf("seeder: insert option listing %s: %v", optTicker, err)
			continue
		}

		if _, err := db.Exec(`
			INSERT INTO listing_option (listing_id, stock_listing_id, option_type, strike_price, implied_volatility, open_interest, settlement_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (listing_id) DO NOTHING`,
			optListingID, stockListingID, opt.OptionType, opt.StrikePrice,
			opt.ImpliedVolatility, opt.OpenInterest, opt.SettlementDate); err != nil {
			log.Printf("seeder: insert listing_option %s: %v", optTicker, err)
		}
	}
}

// insertDailyBars bulk-inserts OHLCV history rows for a listing.
func insertDailyBars(db *sql.DB, listingID int64, bars []DailyBar) {
	for _, bar := range bars {
		_, err := db.Exec(`
			INSERT INTO listing_daily_price_info (listing_id, date, price, ask, bid, change, volume)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (listing_id, date) DO NOTHING`,
			listingID, bar.Date, bar.Price, bar.Ask, bar.Bid, bar.Change, bar.Volume)
		if err != nil {
			log.Printf("seeder: insert daily bar listing %d date %s: %v", listingID, bar.Date.Format("2006-01-02"), err)
		}
	}
}

// lastBusinessDayOfMonth returns the last Mon–Fri of the given month.
func lastBusinessDayOfMonth(t time.Time) time.Time {
	// Start from last day of month.
	d := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// sanitizeTicker converts a contract name to an uppercase alphanumeric ticker.
func sanitizeTicker(name string) string {
	b := make([]byte, 0, len(name))
	for i := 0; i < len(name) && len(b) < 20; i++ {
		c := name[i]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b = append(b, c)
		} else if c >= 'a' && c <= 'z' {
			b = append(b, c-32)
		}
	}
	return string(b)
}

// buildOptionTicker creates a unique ticker in OCC format, e.g. MSFT220404C00180000.
// Format: <underlying><YYMMDD><C/P><strike in cents, 8 digits zero-padded>
func buildOptionTicker(underlying string, opt OptionRow) string {
	strikeInCents := int64(math.Round(opt.StrikePrice * 100))
	return fmt.Sprintf("%s%s%s%08d",
		underlying,
		opt.SettlementDate.Format("060102"),
		opt.OptionType[:1],
		strikeInCents,
	)
}
