package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SecuritiesServer struct {
	pb.UnimplementedSecuritiesServiceServer
	DB *sql.DB
}

// ── Ping ──────────────────────────────────────────────────────────────────────

func (s *SecuritiesServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "securities-service ok"}, nil
}

// ── Stock Exchanges ───────────────────────────────────────────────────────────

func (s *SecuritiesServer) GetStockExchanges(ctx context.Context, _ *pb.GetStockExchangesRequest) (*pb.GetStockExchangesResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, acronym, mic_code, polity, currency, timezone
		FROM stock_exchanges
		ORDER BY name`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()

	var exchanges []*pb.StockExchange
	for rows.Next() {
		e := &pb.StockExchange{}
		if err := rows.Scan(&e.Id, &e.Name, &e.Acronym, &e.MicCode, &e.Polity, &e.Currency, &e.Timezone); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		exchanges = append(exchanges, e)
	}
	return &pb.GetStockExchangesResponse{Exchanges: exchanges}, nil
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
	defer rows.Close()

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
	defer rows.Close()

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
		`SELECT segment, open_time, close_time FROM exchange_working_hours WHERE polity = $1`,
		polity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()

	// 6. Check which segment (if any) the current time falls in
	for rows.Next() {
		var segment, openStr, closeStr string
		if err := rows.Scan(&segment, &openStr, &closeStr); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
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

	// 7. No segment matched — exchange is closed
	return &pb.IsExchangeOpenResponse{
		MicCode:          req.MicCode,
		IsOpen:           false,
		Segment:          "closed",
		CurrentTimeLocal: currentTime,
	}, nil
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

