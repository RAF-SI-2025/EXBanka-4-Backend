package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- stub securities service client ----

type stubSecuritiesClient struct {
	getExchangesFn   func(context.Context, *pb.GetStockExchangesRequest, ...grpc.CallOption) (*pb.GetStockExchangesResponse, error)
	getByMICFn       func(context.Context, *pb.GetStockExchangeByMICRequest, ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error)
	getByIdFn        func(context.Context, *pb.GetStockExchangeByIdRequest, ...grpc.CallOption) (*pb.GetStockExchangeByIdResponse, error)
	createExchangeFn func(context.Context, *pb.CreateStockExchangeRequest, ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error)
	updateExchangeFn func(context.Context, *pb.UpdateStockExchangeRequest, ...grpc.CallOption) (*pb.UpdateStockExchangeResponse, error)
	deleteExchangeFn func(context.Context, *pb.DeleteStockExchangeRequest, ...grpc.CallOption) (*pb.DeleteStockExchangeResponse, error)
	getHoursFn       func(context.Context, *pb.GetWorkingHoursRequest, ...grpc.CallOption) (*pb.GetWorkingHoursResponse, error)
	setHoursFn       func(context.Context, *pb.SetWorkingHoursRequest, ...grpc.CallOption) (*pb.SetWorkingHoursResponse, error)
	getHolidaysFn    func(context.Context, *pb.GetHolidaysRequest, ...grpc.CallOption) (*pb.GetHolidaysResponse, error)
	addHolidayFn     func(context.Context, *pb.AddHolidayRequest, ...grpc.CallOption) (*pb.AddHolidayResponse, error)
	deleteHolidayFn  func(context.Context, *pb.DeleteHolidayRequest, ...grpc.CallOption) (*pb.DeleteHolidayResponse, error)
	isOpenFn         func(context.Context, *pb.IsExchangeOpenRequest, ...grpc.CallOption) (*pb.IsExchangeOpenResponse, error)
	getTestModeFn    func(context.Context, *pb.GetTestModeRequest, ...grpc.CallOption) (*pb.GetTestModeResponse, error)
	setTestModeFn    func(context.Context, *pb.SetTestModeRequest, ...grpc.CallOption) (*pb.SetTestModeResponse, error)
	getListingsFn    func(context.Context, *pb.GetListingsRequest, ...grpc.CallOption) (*pb.GetListingsResponse, error)
	getListingByIdFn func(context.Context, *pb.GetListingByIdRequest, ...grpc.CallOption) (*pb.GetListingByIdResponse, error)
	getListingHistFn func(context.Context, *pb.GetListingHistoryRequest, ...grpc.CallOption) (*pb.GetListingHistoryResponse, error)
}

func (s *stubSecuritiesClient) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetStockExchanges(ctx context.Context, in *pb.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb.GetStockExchangesResponse, error) {
	if s.getExchangesFn != nil {
		return s.getExchangesFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetStockExchangeByMIC(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
	if s.getByMICFn != nil {
		return s.getByMICFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetStockExchangeById(ctx context.Context, in *pb.GetStockExchangeByIdRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByIdResponse, error) {
	if s.getByIdFn != nil {
		return s.getByIdFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) CreateStockExchange(ctx context.Context, in *pb.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error) {
	if s.createExchangeFn != nil {
		return s.createExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) UpdateStockExchange(ctx context.Context, in *pb.UpdateStockExchangeRequest, opts ...grpc.CallOption) (*pb.UpdateStockExchangeResponse, error) {
	if s.updateExchangeFn != nil {
		return s.updateExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) DeleteStockExchange(ctx context.Context, in *pb.DeleteStockExchangeRequest, opts ...grpc.CallOption) (*pb.DeleteStockExchangeResponse, error) {
	if s.deleteExchangeFn != nil {
		return s.deleteExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetWorkingHours(ctx context.Context, in *pb.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb.GetWorkingHoursResponse, error) {
	if s.getHoursFn != nil {
		return s.getHoursFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) SetWorkingHours(ctx context.Context, in *pb.SetWorkingHoursRequest, opts ...grpc.CallOption) (*pb.SetWorkingHoursResponse, error) {
	if s.setHoursFn != nil {
		return s.setHoursFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetHolidays(ctx context.Context, in *pb.GetHolidaysRequest, opts ...grpc.CallOption) (*pb.GetHolidaysResponse, error) {
	if s.getHolidaysFn != nil {
		return s.getHolidaysFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) AddHoliday(ctx context.Context, in *pb.AddHolidayRequest, opts ...grpc.CallOption) (*pb.AddHolidayResponse, error) {
	if s.addHolidayFn != nil {
		return s.addHolidayFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) DeleteHoliday(ctx context.Context, in *pb.DeleteHolidayRequest, opts ...grpc.CallOption) (*pb.DeleteHolidayResponse, error) {
	if s.deleteHolidayFn != nil {
		return s.deleteHolidayFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) IsExchangeOpen(ctx context.Context, in *pb.IsExchangeOpenRequest, opts ...grpc.CallOption) (*pb.IsExchangeOpenResponse, error) {
	if s.isOpenFn != nil {
		return s.isOpenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetTestMode(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
	if s.getTestModeFn != nil {
		return s.getTestModeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) SetTestMode(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
	if s.setTestModeFn != nil {
		return s.setTestModeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetListings(ctx context.Context, in *pb.GetListingsRequest, opts ...grpc.CallOption) (*pb.GetListingsResponse, error) {
	if s.getListingsFn != nil {
		return s.getListingsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetListingById(ctx context.Context, in *pb.GetListingByIdRequest, opts ...grpc.CallOption) (*pb.GetListingByIdResponse, error) {
	if s.getListingByIdFn != nil {
		return s.getListingByIdFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetListingHistory(ctx context.Context, in *pb.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb.GetListingHistoryResponse, error) {
	if s.getListingHistFn != nil {
		return s.getListingHistFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) CreatePriceAlert(ctx context.Context, in *pb.CreatePriceAlertRequest, opts ...grpc.CallOption) (*pb.CreatePriceAlertResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) ListPriceAlerts(ctx context.Context, in *pb.ListPriceAlertsRequest, opts ...grpc.CallOption) (*pb.ListPriceAlertsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) DeletePriceAlert(ctx context.Context, in *pb.DeletePriceAlertRequest, opts ...grpc.CallOption) (*pb.DeletePriceAlertResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- GetTestMode ----

func TestGetTestMode_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getTestModeFn: func(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
			return &pb.GetTestModeResponse{Enabled: true}, nil
		},
	}
	w := serveHandler(GetTestMode(client), "GET", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestGetTestMode_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		getTestModeFn: func(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(GetTestMode(client), "GET", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- SetTestMode ----

func TestSetTestMode_Enable(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return &pb.SetTestModeResponse{Enabled: in.Enabled}, nil
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestSetTestMode_Disable(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return &pb.SetTestModeResponse{Enabled: in.Enabled}, nil
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":false}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":false`)
}

func TestSetTestMode_InvalidBody(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetTestMode_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":true}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- helpers ----

func sampleExchange() *pb.StockExchange {
	return &pb.StockExchange{
		Id: 1, Name: "New York Stock Exchange", Acronym: "NYSE",
		MicCode: "XNYS", Polity: "United States", Currency: "USD", Timezone: "America/New_York",
	}
}

// ---- GetStockExchanges ----

func TestGetStockExchanges_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getExchangesFn: func(ctx context.Context, in *pb.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb.GetStockExchangesResponse, error) {
			return &pb.GetStockExchangesResponse{Exchanges: []*pb.StockExchange{sampleExchange()}, TotalCount: 1}, nil
		},
	}
	w := serveHandler(GetStockExchanges(client), "GET", "/stock-exchanges", "/stock-exchanges?page=1&pageSize=10", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"totalCount":1`)
	assert.Contains(t, w.Body.String(), "XNYS")
}

func TestGetStockExchanges_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		getExchangesFn: func(ctx context.Context, in *pb.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb.GetStockExchangesResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(GetStockExchanges(client), "GET", "/stock-exchanges", "/stock-exchanges", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- GetStockExchange (by MIC or ID) ----

func TestGetStockExchange_ByMIC_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return &pb.GetStockExchangeByMICResponse{Exchange: sampleExchange()}, nil
		},
	}
	w := serveHandler(GetStockExchange(client), "GET", "/stock-exchanges/:id", "/stock-exchanges/XNYS", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "XNYS")
}

func TestGetStockExchange_ByID_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getByIdFn: func(ctx context.Context, in *pb.GetStockExchangeByIdRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByIdResponse, error) {
			return &pb.GetStockExchangeByIdResponse{Exchange: sampleExchange()}, nil
		},
	}
	w := serveHandler(GetStockExchange(client), "GET", "/stock-exchanges/:id", "/stock-exchanges/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "XNYS")
}

func TestGetStockExchange_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(GetStockExchange(client), "GET", "/stock-exchanges/:id", "/stock-exchanges/XXXX", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetStockExchange_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(GetStockExchange(client), "GET", "/stock-exchanges/:id", "/stock-exchanges/XNYS", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- CreateStockExchange ----

func TestCreateStockExchange_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		createExchangeFn: func(ctx context.Context, in *pb.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error) {
			return &pb.CreateStockExchangeResponse{Exchange: sampleExchange()}, nil
		},
	}
	body := `{"name":"NYSE","acronym":"NYSE","micCode":"XNYS","polity":"United States","currency":"USD","timezone":"America/New_York"}`
	w := serveHandler(CreateStockExchange(client), "POST", "/stock-exchanges", "/stock-exchanges", body)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "XNYS")
}

func TestCreateStockExchange_Conflict(t *testing.T) {
	client := &stubSecuritiesClient{
		createExchangeFn: func(ctx context.Context, in *pb.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "already exists")
		},
	}
	body := `{"name":"NYSE","acronym":"NYSE","micCode":"XNYS","polity":"United States","currency":"USD","timezone":"America/New_York"}`
	w := serveHandler(CreateStockExchange(client), "POST", "/stock-exchanges", "/stock-exchanges", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateStockExchange_BadBody(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandler(CreateStockExchange(client), "POST", "/stock-exchanges", "/stock-exchanges", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStockExchange_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		createExchangeFn: func(ctx context.Context, in *pb.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	body := `{"name":"NYSE","acronym":"NYSE","micCode":"XNYS","polity":"United States","currency":"USD","timezone":"America/New_York"}`
	w := serveHandler(CreateStockExchange(client), "POST", "/stock-exchanges", "/stock-exchanges", body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- UpdateStockExchange ----

func TestUpdateStockExchange_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return &pb.GetStockExchangeByMICResponse{Exchange: sampleExchange()}, nil
		},
		updateExchangeFn: func(ctx context.Context, in *pb.UpdateStockExchangeRequest, opts ...grpc.CallOption) (*pb.UpdateStockExchangeResponse, error) {
			return &pb.UpdateStockExchangeResponse{Exchange: sampleExchange()}, nil
		},
	}
	body := `{"name":"NYSE","acronym":"NYSE","polity":"United States","currency":"USD","timezone":"America/New_York"}`
	w := serveHandler(UpdateStockExchange(client), "PUT", "/stock-exchanges/:id", "/stock-exchanges/XNYS", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateStockExchange_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	body := `{"name":"NYSE","acronym":"NYSE","polity":"United States","currency":"USD","timezone":"America/New_York"}`
	w := serveHandler(UpdateStockExchange(client), "PUT", "/stock-exchanges/:id", "/stock-exchanges/XXXX", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateStockExchange_BadBody(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return &pb.GetStockExchangeByMICResponse{Exchange: sampleExchange()}, nil
		},
	}
	w := serveHandler(UpdateStockExchange(client), "PUT", "/stock-exchanges/:id", "/stock-exchanges/XNYS", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- DeleteStockExchange ----

func TestDeleteStockExchange_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return &pb.GetStockExchangeByMICResponse{Exchange: sampleExchange()}, nil
		},
		deleteExchangeFn: func(ctx context.Context, in *pb.DeleteStockExchangeRequest, opts ...grpc.CallOption) (*pb.DeleteStockExchangeResponse, error) {
			return &pb.DeleteStockExchangeResponse{}, nil
		},
	}
	w := serveHandler(DeleteStockExchange(client), "DELETE", "/stock-exchanges/:id", "/stock-exchanges/XNYS", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteStockExchange_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(DeleteStockExchange(client), "DELETE", "/stock-exchanges/:id", "/stock-exchanges/XXXX", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- GetWorkingHours ----

func TestGetWorkingHours_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return &pb.GetStockExchangeByMICResponse{Exchange: sampleExchange()}, nil
		},
		getHoursFn: func(ctx context.Context, in *pb.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb.GetWorkingHoursResponse, error) {
			return &pb.GetWorkingHoursResponse{Hours: []*pb.ExchangeWorkingHours{
				{Id: 1, Polity: "United States", Segment: "regular", OpenTime: "09:30", CloseTime: "16:00"},
			}}, nil
		},
	}
	w := serveHandler(GetWorkingHours(client), "GET", "/stock-exchanges/:id/hours", "/stock-exchanges/XNYS/hours", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "regular")
}

func TestGetWorkingHours_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getByMICFn: func(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(GetWorkingHours(client), "GET", "/stock-exchanges/:id/hours", "/stock-exchanges/XXXX/hours", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── GetSecurities ─────────────────────────────────────────────────────────────

// makeAgentToken produces a Bearer token with AGENT permission (employee).
func makeAgentToken() string {
	claims := jwt.MapClaims{
		"user_id": float64(10),
		"dozvole": []interface{}{"AGENT"},
		"exp":     time.Now().Add(time.Hour).Unix(),
		"type":    "access",
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(""))
	return "Bearer " + tok
}

func sampleListingSummary() *pb.ListingSummary {
	return &pb.ListingSummary{
		Id: 1, Ticker: "AAPL", Name: "Apple Inc", Type: "STOCK",
		ExchangeAcronym: "NASDAQ", Price: 150.0, Ask: 151.0, Bid: 149.0,
		Volume: 500000, ChangePercent: 1.5, MaintenanceMargin: 75.0,
		InitialMarginCost: 82.5, NominalValue: 150.0,
	}
}

func TestGetSecurities_NoToken(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetSecurities_EmployeeHappyPath(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingsFn: func(ctx context.Context, in *pb.GetListingsRequest, opts ...grpc.CallOption) (*pb.GetListingsResponse, error) {
			return &pb.GetListingsResponse{
				Listings:      []*pb.ListingSummary{sampleListingSummary()},
				TotalPages:    1,
				TotalElements: 1,
			}, nil
		},
	}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities?type=STOCK", "", makeAgentToken())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
	assert.Contains(t, w.Body.String(), `"totalElements":1`)
}

func TestGetSecurities_ClientSeesStockAndFutures(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingsFn: func(ctx context.Context, in *pb.GetListingsRequest, opts ...grpc.CallOption) (*pb.GetListingsResponse, error) {
			// Returns mixed types — gateway must filter out FOREX/OPTION for clients
			return &pb.GetListingsResponse{
				Listings: []*pb.ListingSummary{
					{Id: 1, Ticker: "AAPL", Type: "STOCK"},
					{Id: 2, Ticker: "EUR/USD", Type: "FOREX_PAIR"},
					{Id: 3, Ticker: "CLJ25", Type: "FUTURES_CONTRACT"},
					{Id: 4, Ticker: "AAPLC", Type: "OPTION"},
				},
				TotalPages: 1, TotalElements: 4,
			}, nil
		},
	}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "AAPL")
	assert.Contains(t, body, "CLJ25")
	assert.NotContains(t, body, "EUR/USD")
	assert.NotContains(t, body, "AAPLC")
}

func TestGetSecurities_ClientRequestsForexReturnsEmpty(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities?type=FOREX_PAIR", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"totalElements":0`)
}

func TestGetSecurities_ClientRequestsOptionReturnsEmpty(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities?type=OPTION", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"totalElements":0`)
}

func TestGetSecurities_GRPCError(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingsFn: func(ctx context.Context, in *pb.GetListingsRequest, opts ...grpc.CallOption) (*pb.GetListingsResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(GetSecurities(client), "GET", "/securities", "/securities", "", makeAgentToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetSecurityById ───────────────────────────────────────────────────────────

func TestGetSecurityById_NoToken(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurityById(client), "GET", "/securities/:id", "/securities/1", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetSecurityById_InvalidId(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurityById(client), "GET", "/securities/:id", "/securities/abc", "", makeAgentToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSecurityById_StockHappyPath(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingByIdFn: func(ctx context.Context, in *pb.GetListingByIdRequest, opts ...grpc.CallOption) (*pb.GetListingByIdResponse, error) {
			return &pb.GetListingByIdResponse{
				Summary: sampleListingSummary(),
				PriceHistory: []*pb.DailyPriceInfo{
					{Date: "2025-01-01", Price: 148.0, Ask: 149.0, Bid: 147.0, Change: 2.0, Volume: 400000},
				},
				Detail: &pb.GetListingByIdResponse_Stock{
					Stock: &pb.StockDetail{OutstandingShares: 1000000, DividendYield: 0.02, MarketCap: 150_000_000},
				},
			}, nil
		},
	}
	w := serveHandlerFull(GetSecurityById(client), "GET", "/securities/:id", "/securities/1", "", makeAgentToken())
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "AAPL")
	assert.Contains(t, body, "priceHistory")
	assert.Contains(t, body, "outstandingShares")
}

func TestGetSecurityById_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingByIdFn: func(ctx context.Context, in *pb.GetListingByIdRequest, opts ...grpc.CallOption) (*pb.GetListingByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(GetSecurityById(client), "GET", "/securities/:id", "/securities/999", "", makeAgentToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSecurityById_GRPCError(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingByIdFn: func(ctx context.Context, in *pb.GetListingByIdRequest, opts ...grpc.CallOption) (*pb.GetListingByIdResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(GetSecurityById(client), "GET", "/securities/:id", "/securities/1", "", makeAgentToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetSecurityHistory ────────────────────────────────────────────────────────

func TestGetSecurityHistory_NoToken(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurityHistory(client), "GET", "/securities/:id/history", "/securities/1/history", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetSecurityHistory_InvalidId(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandlerFull(GetSecurityHistory(client), "GET", "/securities/:id/history", "/securities/abc/history", "", makeAgentToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSecurityHistory_HappyPath(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingHistFn: func(ctx context.Context, in *pb.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb.GetListingHistoryResponse, error) {
			assert.Equal(t, int64(1), in.Id)
			assert.Equal(t, "2025-01-01", in.FromDate)
			assert.Equal(t, "2025-01-31", in.ToDate)
			return &pb.GetListingHistoryResponse{
				History: []*pb.DailyPriceInfo{
					{Date: "2025-01-01", Price: 148.0, Ask: 149.0, Bid: 147.0, Change: 2.0, Volume: 400000},
					{Date: "2025-01-02", Price: 150.0, Ask: 151.0, Bid: 149.0, Change: 2.0, Volume: 500000},
				},
			}, nil
		},
	}
	w := serveHandlerFull(GetSecurityHistory(client), "GET", "/securities/:id/history",
		"/securities/1/history?from=2025-01-01&to=2025-01-31", "", makeAgentToken())
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "2025-01-01")
	assert.Contains(t, body, "2025-01-02")
}

func TestGetSecurityHistory_NotFound(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingHistFn: func(ctx context.Context, in *pb.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb.GetListingHistoryResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(GetSecurityHistory(client), "GET", "/securities/:id/history",
		"/securities/999/history?from=2025-01-01&to=2025-01-31", "", makeAgentToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSecurityHistory_GRPCError(t *testing.T) {
	client := &stubSecuritiesClient{
		getListingHistFn: func(ctx context.Context, in *pb.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb.GetListingHistoryResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(GetSecurityHistory(client), "GET", "/securities/:id/history",
		"/securities/1/history?from=2025-01-01&to=2025-01-31", "", makeAgentToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
