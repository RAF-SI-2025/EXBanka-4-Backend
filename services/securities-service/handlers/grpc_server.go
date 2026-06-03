package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const listingCacheTTL = 15 * time.Minute

type SecuritiesServer struct {
	pb.UnimplementedSecuritiesServiceServer
	DB    *sql.DB
	Redis *redis.Client
}

func listingCacheKey(id int64) string {
	return fmt.Sprintf("listing:%d", id)
}

func (s *SecuritiesServer) loadCachedListing(ctx context.Context, id int64) *pb.GetListingByIdResponse {
	if s.Redis == nil {
		return nil
	}
	data, err := s.Redis.Get(ctx, listingCacheKey(id)).Bytes()
	if err != nil {
		return nil
	}
	var resp pb.GetListingByIdResponse
	if err := protojson.Unmarshal(data, &resp); err != nil {
		return nil
	}
	return &resp
}

func (s *SecuritiesServer) storeCachedListing(ctx context.Context, id int64, resp *pb.GetListingByIdResponse) {
	if s.Redis == nil {
		return
	}
	data, err := protojson.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.Redis.Set(ctx, listingCacheKey(id), data, listingCacheTTL).Err()
}

// InvalidateListing deletes the cached listing from Redis. Called by the price refresh scheduler.
func (s *SecuritiesServer) InvalidateListing(ctx context.Context, id int64) {
	if s.Redis == nil {
		return
	}
	_ = s.Redis.Del(ctx, listingCacheKey(id)).Err()
}

// ── Ping ──────────────────────────────────────────────────────────────────────

func (s *SecuritiesServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "securities-service ok"}, nil
}

// ── Stock Exchanges ───────────────────────────────────────────────────────────

func (s *SecuritiesServer) GetStockExchanges(ctx context.Context, req *pb.GetStockExchangesRequest) (*pb.GetStockExchangesResponse, error) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var total int32
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_exchanges`).Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "count query failed: %v", err)
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, acronym, mic_code, polity, currency, timezone
		FROM stock_exchanges
		ORDER BY name
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var exchanges []*pb.StockExchange
	for rows.Next() {
		e := &pb.StockExchange{}
		if err := rows.Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		exchanges = append(exchanges, e)
	}
	return &pb.GetStockExchangesResponse{Exchanges: exchanges, TotalCount: total}, nil
}

func (s *SecuritiesServer) GetStockExchangeById(ctx context.Context, req *pb.GetStockExchangeByIdRequest) (*pb.GetStockExchangeByIdResponse, error) {
	e := &pb.StockExchange{}
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, name, acronym, mic_code, polity, currency, timezone
		FROM stock_exchanges
		WHERE id = $1`, req.Id).
		Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "exchange with ID %d not found", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	return &pb.GetStockExchangeByIdResponse{Exchange: e}, nil
}

func (s *SecuritiesServer) GetStockExchangeByMIC(ctx context.Context, req *pb.GetStockExchangeByMICRequest) (*pb.GetStockExchangeByMICResponse, error) {
	e := &pb.StockExchange{}
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, name, acronym, mic_code, polity, currency, timezone
		FROM stock_exchanges
		WHERE mic_code = $1`, req.MicCode).
		Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "exchange with MIC %q not found", req.MicCode)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	return &pb.GetStockExchangeByMICResponse{Exchange: e}, nil
}

func (s *SecuritiesServer) CreateStockExchange(ctx context.Context, req *pb.CreateStockExchangeRequest) (*pb.CreateStockExchangeResponse, error) {
	e := &pb.StockExchange{}
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO stock_exchanges (name, acronym, mic_code, polity, currency, timezone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, acronym, mic_code, polity, currency, timezone`,
		req.Name, req.Acronym, req.MicCode, req.Polity, req.Currency, req.Timezone).
		Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, status.Errorf(codes.AlreadyExists, "exchange with MIC %q already exists", req.MicCode)
		}
		return nil, status.Errorf(codes.Internal, "insert failed: %v", err)
	}
	return &pb.CreateStockExchangeResponse{Exchange: e}, nil
}

func (s *SecuritiesServer) UpdateStockExchange(ctx context.Context, req *pb.UpdateStockExchangeRequest) (*pb.UpdateStockExchangeResponse, error) {
	e := &pb.StockExchange{}
	err := s.DB.QueryRowContext(ctx, `
		UPDATE stock_exchanges
		SET name=$1, acronym=$2, polity=$3, currency=$4, timezone=$5
		WHERE mic_code=$6
		RETURNING id, name, acronym, mic_code, polity, currency, timezone`,
		req.Name, req.Acronym, req.Polity, req.Currency, req.Timezone, req.MicCode).
		Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "exchange with MIC %q not found", req.MicCode)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}
	return &pb.UpdateStockExchangeResponse{Exchange: e}, nil
}

func (s *SecuritiesServer) DeleteStockExchange(ctx context.Context, req *pb.DeleteStockExchangeRequest) (*pb.DeleteStockExchangeResponse, error) {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM stock_exchanges WHERE mic_code = $1`, req.MicCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, status.Errorf(codes.NotFound, "exchange with MIC %q not found", req.MicCode)
	}
	return &pb.DeleteStockExchangeResponse{}, nil
}

// ── Working Hours ─────────────────────────────────────────────────────────────

func (s *SecuritiesServer) GetWorkingHours(ctx context.Context, req *pb.GetWorkingHoursRequest) (*pb.GetWorkingHoursResponse, error) {
	var polity string
	err := s.DB.QueryRowContext(ctx, `SELECT polity FROM stock_exchanges WHERE mic_code = $1`, req.MicCode).
		Scan(&polity)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "exchange with MIC %q not found", req.MicCode)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, polity, segment, open_time, close_time
		FROM exchange_working_hours
		WHERE polity = $1
		ORDER BY segment`, polity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var hours []*pb.ExchangeWorkingHours
	for rows.Next() {
		h := &pb.ExchangeWorkingHours{}
		if err := rows.Scan(&h.Id, &h.Polity, &h.Segment, &h.OpenTime, &h.CloseTime); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		hours = append(hours, h)
	}
	return &pb.GetWorkingHoursResponse{Hours: hours}, nil
}

func (s *SecuritiesServer) SetWorkingHours(ctx context.Context, req *pb.SetWorkingHoursRequest) (*pb.SetWorkingHoursResponse, error) {
	h := &pb.ExchangeWorkingHours{}
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO exchange_working_hours (polity, segment, open_time, close_time)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (polity, segment) DO UPDATE
		  SET open_time = EXCLUDED.open_time, close_time = EXCLUDED.close_time
		RETURNING id, polity, segment, open_time, close_time`,
		req.Polity, req.Segment, req.OpenTime, req.CloseTime).
		Scan(&h.Id, &h.Polity, &h.Segment, &h.OpenTime, &h.CloseTime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert failed: %v", err)
	}
	return &pb.SetWorkingHoursResponse{Hours: h}, nil
}

// ── Holidays ──────────────────────────────────────────────────────────────────

func (s *SecuritiesServer) GetHolidays(ctx context.Context, req *pb.GetHolidaysRequest) (*pb.GetHolidaysResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, polity, holiday_date, COALESCE(description, '')
		FROM exchange_holidays
		WHERE polity = $1
		ORDER BY holiday_date`, req.Polity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var holidays []*pb.ExchangeHoliday
	for rows.Next() {
		h := &pb.ExchangeHoliday{}
		var d time.Time
		if err := rows.Scan(&h.Id, &h.Polity, &d, &h.Description); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		h.HolidayDate = d.Format("2006-01-02")
		holidays = append(holidays, h)
	}
	return &pb.GetHolidaysResponse{Holidays: holidays}, nil
}

func (s *SecuritiesServer) AddHoliday(ctx context.Context, req *pb.AddHolidayRequest) (*pb.AddHolidayResponse, error) {
	h := &pb.ExchangeHoliday{}
	var d time.Time
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO exchange_holidays (polity, holiday_date, description)
		VALUES ($1, $2, $3)
		RETURNING id, polity, holiday_date, COALESCE(description, '')`,
		req.Polity, req.HolidayDate, nullableString(req.Description)).
		Scan(&h.Id, &h.Polity, &d, &h.Description)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, status.Errorf(codes.AlreadyExists, "holiday on %s for %q already exists", req.HolidayDate, req.Polity)
		}
		return nil, status.Errorf(codes.Internal, "insert failed: %v", err)
	}
	h.HolidayDate = d.Format("2006-01-02")
	return &pb.AddHolidayResponse{Holiday: h}, nil
}

func (s *SecuritiesServer) DeleteHoliday(ctx context.Context, req *pb.DeleteHolidayRequest) (*pb.DeleteHolidayResponse, error) {
	res, err := s.DB.ExecContext(ctx, `
		DELETE FROM exchange_holidays WHERE polity = $1 AND holiday_date = $2`,
		req.Polity, req.HolidayDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, status.Errorf(codes.NotFound, "holiday on %s for %q not found", req.HolidayDate, req.Polity)
	}
	return &pb.DeleteHolidayResponse{}, nil
}

// ── Test Mode ─────────────────────────────────────────────────────────────────

func (s *SecuritiesServer) GetTestMode(ctx context.Context, _ *pb.GetTestModeRequest) (*pb.GetTestModeResponse, error) {
	var enabled bool
	err := s.DB.QueryRowContext(ctx, `SELECT test_mode_enabled FROM settings WHERE id = TRUE`).Scan(&enabled)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	return &pb.GetTestModeResponse{Enabled: enabled}, nil
}

func (s *SecuritiesServer) SetTestMode(ctx context.Context, req *pb.SetTestModeRequest) (*pb.SetTestModeResponse, error) {
	_, err := s.DB.ExecContext(ctx, `UPDATE settings SET test_mode_enabled = $1 WHERE id = TRUE`, req.Enabled)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}
	return &pb.SetTestModeResponse{Enabled: req.Enabled}, nil
}

// ── Exchange Status ───────────────────────────────────────────────────────────

func (s *SecuritiesServer) IsExchangeOpen(ctx context.Context, req *pb.IsExchangeOpenRequest) (*pb.IsExchangeOpenResponse, error) {
	// 1. Check test mode — if enabled, all exchanges are treated as open
	var testMode bool
	if err := s.DB.QueryRowContext(ctx, `SELECT test_mode_enabled FROM settings WHERE id = TRUE`).Scan(&testMode); err != nil {
		return nil, status.Errorf(codes.Internal, "test mode check failed: %v", err)
	}
	if testMode {
		return &pb.IsExchangeOpenResponse{
			MicCode: req.MicCode, IsOpen: true, Segment: "test_mode",
		}, nil
	}

	// 2. Fetch timezone and polity for the exchange
	var timezone, polity string
	err := s.DB.QueryRowContext(ctx,
		`SELECT timezone, polity FROM stock_exchanges WHERE mic_code = $1`, req.MicCode).
		Scan(&timezone, &polity)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "exchange with MIC %q not found", req.MicCode)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}

	// 3. Get current time in the exchange's local timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid timezone %q: %v", timezone, err)
	}
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	currentTime := now.Format("15:04")

	// 4. Check if today is a holiday
	var holidayExists bool
	err = s.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM exchange_holidays WHERE polity = $1 AND holiday_date = $2)`,
		polity, today).Scan(&holidayExists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "holiday check failed: %v", err)
	}
	if holidayExists {
		return &pb.IsExchangeOpenResponse{
			MicCode:          req.MicCode,
			IsOpen:           false,
			Segment:          "closed",
			CurrentTimeLocal: currentTime,
		}, nil
	}

	// 5. Load working hours for this polity
	rows, err := s.DB.QueryContext(ctx,
		`SELECT segment, TO_CHAR(open_time, 'HH24:MI'), TO_CHAR(close_time, 'HH24:MI') FROM exchange_working_hours WHERE polity = $1`,
		polity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	// 6. Check which segment (if any) the current time falls in; also track the
	//    regular session hours for pre/post-market calculation below.
	var regularOpen, regularClose string
	for rows.Next() {
		var segment, openStr, closeStr string
		if err := rows.Scan(&segment, &openStr, &closeStr); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		if segment == "regular" {
			regularOpen, regularClose = openStr, closeStr
		}
		if timeInRange(currentTime, openStr, closeStr) {
			return &pb.IsExchangeOpenResponse{
				MicCode:          req.MicCode,
				IsOpen:           true,
				Segment:          segment,
				CurrentTimeLocal: currentTime,
			}, nil
		}
	}

	// 7. No explicit segment matched — check pre/post-market windows (±4 h of regular session).
	if regularOpen != "" && regularClose != "" {
		const window = 4 * time.Hour
		if seg := prePostMarketSegment(currentTime, regularOpen, regularClose, window); seg != "" {
			return &pb.IsExchangeOpenResponse{
				MicCode:          req.MicCode,
				IsOpen:           false,
				Segment:          seg,
				CurrentTimeLocal: currentTime,
			}, nil
		}
	}

	return &pb.IsExchangeOpenResponse{
		MicCode:          req.MicCode,
		IsOpen:           false,
		Segment:          "closed",
		CurrentTimeLocal: currentTime,
	}, nil
}

// prePostMarketSegment returns "pre_market" if t falls within [regularOpen-window, regularOpen),
// "post_market" if within [regularClose, regularClose+window), or "" otherwise.
// All time strings are "HH:MM".
func prePostMarketSegment(t, regularOpen, regularClose string, window time.Duration) string {
	parse := func(s string) (time.Duration, error) {
		var h, m int
		if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
			return 0, err
		}
		return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, nil
	}
	tDur, err := parse(t)
	if err != nil {
		return ""
	}
	openDur, err := parse(regularOpen)
	if err != nil {
		return ""
	}
	closeDur, err := parse(regularClose)
	if err != nil {
		return ""
	}
	if tDur >= openDur-window && tDur < openDur {
		return "pre_market"
	}
	if tDur >= closeDur && tDur < closeDur+window {
		return "post_market"
	}
	return ""
}

// timeInRange returns true if t is within [open, close) (all "HH:MM" strings).
func timeInRange(t, open, close string) bool {
	return t >= open && t < close
}

// nullableString converts an empty string to nil for optional DB fields.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ── Listings ──────────────────────────────────────────────────────────────────

const listingBaseFrom = `
	FROM listing l
	JOIN stock_exchanges se ON l.exchange_id = se.id
	LEFT JOIN listing_stock ls ON l.id = ls.listing_id
	LEFT JOIN listing_futures_contract lfc ON l.id = lfc.listing_id
	LEFT JOIN listing_option lo ON l.id = lo.listing_id
	LEFT JOIN listing stock_ul ON lo.stock_listing_id = stock_ul.id`

const listingBaseWhere = `
	WHERE ($1 = '' OR l.type = $1::listing_type)
	  AND ($2 = '' OR se.acronym ILIKE $2||'%')
	  AND ($3 = '' OR l.ticker ILIKE $3||'%')
	  AND ($4 = '' OR l.name   ILIKE '%'||$4||'%')`

func (s *SecuritiesServer) GetListings(ctx context.Context, req *pb.GetListingsRequest) (*pb.GetListingsResponse, error) {
	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Whitelisted ORDER BY — safe for fmt.Sprintf because values come from a switch
	col := "l.id"
	switch req.SortBy {
	case "price":
		col = "l.price"
	case "volume":
		col = "l.volume"
	case "change_percent", "change":
		col = "l.change"
	}
	ord := "ASC"
	if req.SortOrder == "DESC" {
		ord = "DESC"
	}

	var total int64
	if err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) `+listingBaseFrom+listingBaseWhere,
		req.Type, req.ExchangeAcronymPrefix, req.TickerPrefix, req.NameSearch,
	).Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "count failed: %v", err)
	}
	totalPages := int32((total + int64(pageSize) - 1) / int64(pageSize))

	query := fmt.Sprintf(`
		SELECT l.id, l.ticker, l.name, l.type::text, se.acronym, se.currency,
		       l.price, l.ask, l.bid, l.volume, l.change,
		       COALESCE(ls.outstanding_shares, 0),
		       COALESCE(lfc.contract_size, 1),
		       lo.stock_listing_id,
		       COALESCE(stock_ul.price, 0),
		       lo.option_type::text, lo.strike_price, lo.settlement_date, lo.open_interest
		%s%s
		ORDER BY %s %s
		LIMIT $5 OFFSET $6`, listingBaseFrom, listingBaseWhere, col, ord)

	rows, err := s.DB.QueryContext(ctx, query,
		req.Type, req.ExchangeAcronymPrefix, req.TickerPrefix, req.NameSearch, pageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var listings []*pb.ListingSummary
	for rows.Next() {
		var (
			id                   int64
			ticker, name         string
			lType, acronym, curr string
			price, ask, bid      float64
			volume               int64
			change               float64
			outshares            int64
			contractSize         float64
			stockListingID       sql.NullInt64
			stockPrice           float64
			optionType           sql.NullString
			strikePrice          sql.NullFloat64
			settlementDate       sql.NullTime
			openInterest         sql.NullInt64
		)
		if err := rows.Scan(
			&id, &ticker, &name, &lType, &acronym, &curr,
			&price, &ask, &bid, &volume, &change,
			&outshares, &contractSize, &stockListingID, &stockPrice,
			&optionType, &strikePrice, &settlementDate, &openInterest,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		mm := computeMaintenanceMargin(lType, price, outshares, contractSize, stockPrice)
		cp := listingChangePercent(price, change)
		summary := &pb.ListingSummary{
			Id:                id,
			Ticker:            ticker,
			Name:              name,
			Type:              lType,
			ExchangeAcronym:   acronym,
			Currency:          curr,
			Price:             price,
			Ask:               ask,
			Bid:               bid,
			Volume:            volume,
			ChangePercent:     cp,
			MaintenanceMargin: mm,
			InitialMarginCost: mm * 1.1,
			NominalValue:      listingNominalValue(lType, price, contractSize),
		}
		if optionType.Valid {
			summary.OptionType = optionType.String
			summary.StrikePrice = strikePrice.Float64
			summary.OpenInterest = openInterest.Int64
			if settlementDate.Valid {
				summary.SettlementDate = settlementDate.Time.Format("2006-01-02")
			}
		}
		listings = append(listings, summary)
	}
	return &pb.GetListingsResponse{
		Listings:      listings,
		TotalPages:    totalPages,
		TotalElements: total,
	}, nil
}

func (s *SecuritiesServer) GetListingById(ctx context.Context, req *pb.GetListingByIdRequest) (*pb.GetListingByIdResponse, error) {
	if cached := s.loadCachedListing(ctx, req.Id); cached != nil {
		return cached, nil
	}

	var (
		id                                 int64
		ticker, name, lType, acronym, curr string
		price, ask, bid                    float64
		volume                             int64
		change                             float64
		// stock
		outshares     sql.NullInt64
		dividendYield sql.NullFloat64
		// forex
		baseCurrency, quoteCurrency, liquidity sql.NullString
		// futures
		contractSize      sql.NullFloat64
		contractUnit      sql.NullString
		futuresSettlement sql.NullTime
		// option
		stockListingID   sql.NullInt64
		optionType       sql.NullString
		strikePrice      sql.NullFloat64
		impliedVol       sql.NullFloat64
		openInterest     sql.NullInt64
		optionSettlement sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
		SELECT l.id, l.ticker, l.name, l.type::text, se.acronym, se.currency,
		       l.price, l.ask, l.bid, l.volume, l.change,
		       ls.outstanding_shares, ls.dividend_yield,
		       lfp.base_currency, lfp.quote_currency, lfp.liquidity::text,
		       lfc.contract_size, lfc.contract_unit, lfc.settlement_date,
		       lo.stock_listing_id, lo.option_type::text, lo.strike_price,
		       lo.implied_volatility, lo.open_interest, lo.settlement_date
		FROM listing l
		JOIN stock_exchanges se ON l.exchange_id = se.id
		LEFT JOIN listing_stock ls ON l.id = ls.listing_id
		LEFT JOIN listing_forex_pair lfp ON l.id = lfp.listing_id
		LEFT JOIN listing_futures_contract lfc ON l.id = lfc.listing_id
		LEFT JOIN listing_option lo ON l.id = lo.listing_id
		WHERE l.id = $1`, req.Id).Scan(
		&id, &ticker, &name, &lType, &acronym, &curr,
		&price, &ask, &bid, &volume, &change,
		&outshares, &dividendYield,
		&baseCurrency, &quoteCurrency, &liquidity,
		&contractSize, &contractUnit, &futuresSettlement,
		&stockListingID, &optionType, &strikePrice,
		&impliedVol, &openInterest, &optionSettlement,
	)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "listing with ID %d not found", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}

	// Fetch underlying stock price for options
	var stockPrice float64
	if stockListingID.Valid {
		_ = s.DB.QueryRowContext(ctx, `SELECT price FROM listing WHERE id = $1`, stockListingID.Int64).Scan(&stockPrice)
	}

	csVal := 1.0
	if contractSize.Valid {
		csVal = contractSize.Float64
	}
	mm := computeMaintenanceMargin(lType, price, outshares.Int64, csVal, stockPrice)
	cp := listingChangePercent(price, change)

	summary := &pb.ListingSummary{
		Id:                id,
		Ticker:            ticker,
		Name:              name,
		Type:              lType,
		ExchangeAcronym:   acronym,
		Currency:          curr,
		Price:             price,
		Ask:               ask,
		Bid:               bid,
		Volume:            volume,
		ChangePercent:     cp,
		MaintenanceMargin: mm,
		InitialMarginCost: mm * 1.1,
		NominalValue:      listingNominalValue(lType, price, csVal),
	}

	// Fetch last 30 days price history (descending)
	histRows, err := s.DB.QueryContext(ctx, `
		SELECT date, price, ask, bid, change, volume
		FROM listing_daily_price_info
		WHERE listing_id = $1
		ORDER BY date DESC
		LIMIT 30`, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "history query failed: %v", err)
	}
	defer func() { _ = histRows.Close() }()

	var history []*pb.DailyPriceInfo
	for histRows.Next() {
		var d time.Time
		p := &pb.DailyPriceInfo{}
		if err := histRows.Scan(&d, &p.Price, &p.Ask, &p.Bid, &p.Change, &p.Volume); err != nil {
			return nil, status.Errorf(codes.Internal, "history scan failed: %v", err)
		}
		p.Date = d.Format("2006-01-02")
		history = append(history, p)
	}

	resp := &pb.GetListingByIdResponse{Summary: summary, PriceHistory: history}

	switch lType {
	case "STOCK":
		resp.Detail = &pb.GetListingByIdResponse_Stock{
			Stock: &pb.StockDetail{
				OutstandingShares: outshares.Int64,
				DividendYield:     dividendYield.Float64,
				MarketCap:         float64(outshares.Int64) * price,
			},
		}
	case "FOREX_PAIR":
		resp.Detail = &pb.GetListingByIdResponse_Forex{
			Forex: &pb.ForexDetail{
				BaseCurrency:  baseCurrency.String,
				QuoteCurrency: quoteCurrency.String,
				Liquidity:     liquidity.String,
				NominalValue:  1000 * price,
			},
		}
	case "FUTURES_CONTRACT":
		sd := ""
		if futuresSettlement.Valid {
			sd = futuresSettlement.Time.Format("2006-01-02")
		}
		resp.Detail = &pb.GetListingByIdResponse_Futures{
			Futures: &pb.FuturesDetail{
				ContractSize:   csVal,
				ContractUnit:   contractUnit.String,
				SettlementDate: sd,
			},
		}
	case "OPTION":
		sd := ""
		if optionSettlement.Valid {
			sd = optionSettlement.Time.Format("2006-01-02")
		}
		resp.Detail = &pb.GetListingByIdResponse_Option{
			Option: &pb.OptionDetail{
				StockListingId:    stockListingID.Int64,
				OptionType:        optionType.String,
				StrikePrice:       strikePrice.Float64,
				ImpliedVolatility: impliedVol.Float64,
				OpenInterest:      openInterest.Int64,
				SettlementDate:    sd,
			},
		}
	}

	s.storeCachedListing(ctx, req.Id, resp)
	return resp, nil
}

func (s *SecuritiesServer) GetListingHistory(ctx context.Context, req *pb.GetListingHistoryRequest) (*pb.GetListingHistoryResponse, error) {
	var exists bool
	if err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM listing WHERE id = $1)`, req.Id,
	).Scan(&exists); err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "listing with ID %d not found", req.Id)
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT date, price, ask, bid, change, volume
		FROM listing_daily_price_info
		WHERE listing_id = $1
		  AND date BETWEEN $2::date AND $3::date
		ORDER BY date ASC`, req.Id, req.FromDate, req.ToDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var history []*pb.DailyPriceInfo
	for rows.Next() {
		var d time.Time
		p := &pb.DailyPriceInfo{}
		if err := rows.Scan(&d, &p.Price, &p.Ask, &p.Bid, &p.Change, &p.Volume); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		p.Date = d.Format("2006-01-02")
		history = append(history, p)
	}
	return &pb.GetListingHistoryResponse{History: history}, nil
}

// ── Derived field helpers ──────────────────────────────────────────────────────

func computeMaintenanceMargin(lType string, price float64, outshares int64, contractSize, stockPrice float64) float64 {
	switch lType {
	case "STOCK":
		return 0.5 * price
	case "FOREX_PAIR":
		return 1000 * price * 0.10
	case "FUTURES_CONTRACT":
		return contractSize * price * 0.10
	case "OPTION":
		return 100 * 0.5 * stockPrice
	}
	return 0
}

func listingChangePercent(price, change float64) float64 {
	prev := price - change
	if prev == 0 {
		return 0
	}
	return (100 * change) / prev
}

func listingNominalValue(lType string, price, contractSize float64) float64 {
	switch lType {
	case "FOREX_PAIR":
		return 1000 * price
	case "FUTURES_CONTRACT":
		return contractSize * price
	case "OPTION":
		return 100 * price
	}
	return price // STOCK: contractSize = 1
}
