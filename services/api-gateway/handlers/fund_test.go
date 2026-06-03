package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- makeSupervisorToken helper ----

func makeSupervisorToken() string {
	claims := jwt.MapClaims{
		"user_id": float64(5),
		"role":    "EMPLOYEE",
		"dozvole": []interface{}{"SUPERVISOR"},
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(""))
	return "Bearer " + tok
}

// ---- stub fund client ----

type stubFundClient struct {
	pingFn                      func(context.Context, *pb.PingRequest, ...grpc.CallOption) (*pb.PingResponse, error)
	createFundFn                func(context.Context, *pb.CreateFundRequest, ...grpc.CallOption) (*pb.FundResponse, error)
	listFundsFn                 func(context.Context, *pb.ListFundsRequest, ...grpc.CallOption) (*pb.ListFundsResponse, error)
	getFundFn                   func(context.Context, *pb.GetFundRequest, ...grpc.CallOption) (*pb.FundResponse, error)
	updateFundFn                func(context.Context, *pb.UpdateFundRequest, ...grpc.CallOption) (*pb.FundResponse, error)
	deleteFundFn                func(context.Context, *pb.DeleteFundRequest, ...grpc.CallOption) (*pb.DeleteFundResponse, error)
	investFundFn                func(context.Context, *pb.InvestFundRequest, ...grpc.CallOption) (*pb.FundResponse, error)
	withdrawFundFn              func(context.Context, *pb.WithdrawFundRequest, ...grpc.CallOption) (*pb.WithdrawFundResponse, error)
	checkPendingWithdrawalsFn   func(context.Context, *pb.CheckPendingWithdrawalsRequest, ...grpc.CallOption) (*pb.CheckPendingWithdrawalsResponse, error)
	getBankPositionsFn          func(context.Context, *pb.GetBankPositionsRequest, ...grpc.CallOption) (*pb.GetBankPositionsResponse, error)
	validateFundAccountFn       func(context.Context, *pb.ValidateFundAccountRequest, ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error)
	updateFundHoldingFn         func(context.Context, *pb.UpdateFundHoldingRequest, ...grpc.CallOption) (*pb.UpdateFundHoldingResponse, error)
	getMyPositionsFn            func(context.Context, *pb.GetMyPositionsRequest, ...grpc.CallOption) (*pb.GetMyPositionsResponse, error)
	transferFundsByManagerFn    func(context.Context, *pb.TransferFundsByManagerRequest, ...grpc.CallOption) (*pb.TransferFundsByManagerResponse, error)
	getFundPortfolioFn          func(context.Context, *pb.GetFundPortfolioRequest, ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error)
	getFundPerformanceHistoryFn func(context.Context, *pb.GetFundPerformanceRequest, ...grpc.CallOption) (*pb.GetFundPerformanceResponse, error)
}

func (s *stubFundClient) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	if s.pingFn != nil {
		return s.pingFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) CreateFund(ctx context.Context, in *pb.CreateFundRequest, opts ...grpc.CallOption) (*pb.FundResponse, error) {
	if s.createFundFn != nil {
		return s.createFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) ListFunds(ctx context.Context, in *pb.ListFundsRequest, opts ...grpc.CallOption) (*pb.ListFundsResponse, error) {
	if s.listFundsFn != nil {
		return s.listFundsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) GetFund(ctx context.Context, in *pb.GetFundRequest, opts ...grpc.CallOption) (*pb.FundResponse, error) {
	if s.getFundFn != nil {
		return s.getFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) UpdateFund(ctx context.Context, in *pb.UpdateFundRequest, opts ...grpc.CallOption) (*pb.FundResponse, error) {
	if s.updateFundFn != nil {
		return s.updateFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) DeleteFund(ctx context.Context, in *pb.DeleteFundRequest, opts ...grpc.CallOption) (*pb.DeleteFundResponse, error) {
	if s.deleteFundFn != nil {
		return s.deleteFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) InvestFund(ctx context.Context, in *pb.InvestFundRequest, opts ...grpc.CallOption) (*pb.FundResponse, error) {
	if s.investFundFn != nil {
		return s.investFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) WithdrawFund(ctx context.Context, in *pb.WithdrawFundRequest, opts ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
	if s.withdrawFundFn != nil {
		return s.withdrawFundFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) CheckPendingWithdrawals(ctx context.Context, in *pb.CheckPendingWithdrawalsRequest, opts ...grpc.CallOption) (*pb.CheckPendingWithdrawalsResponse, error) {
	if s.checkPendingWithdrawalsFn != nil {
		return s.checkPendingWithdrawalsFn(ctx, in, opts...)
	}
	return &pb.CheckPendingWithdrawalsResponse{}, nil
}
func (s *stubFundClient) GetBankPositions(ctx context.Context, in *pb.GetBankPositionsRequest, opts ...grpc.CallOption) (*pb.GetBankPositionsResponse, error) {
	if s.getBankPositionsFn != nil {
		return s.getBankPositionsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) ValidateFundAccount(ctx context.Context, in *pb.ValidateFundAccountRequest, opts ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
	if s.validateFundAccountFn != nil {
		return s.validateFundAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) UpdateFundHolding(ctx context.Context, in *pb.UpdateFundHoldingRequest, opts ...grpc.CallOption) (*pb.UpdateFundHoldingResponse, error) {
	if s.updateFundHoldingFn != nil {
		return s.updateFundHoldingFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) GetMyPositions(ctx context.Context, in *pb.GetMyPositionsRequest, opts ...grpc.CallOption) (*pb.GetMyPositionsResponse, error) {
	if s.getMyPositionsFn != nil {
		return s.getMyPositionsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFundClient) TransferFundsByManager(ctx context.Context, in *pb.TransferFundsByManagerRequest, opts ...grpc.CallOption) (*pb.TransferFundsByManagerResponse, error) {
	if s.transferFundsByManagerFn != nil {
		return s.transferFundsByManagerFn(ctx, in, opts...)
	}
	return &pb.TransferFundsByManagerResponse{}, nil
}
func (s *stubFundClient) GetFundPortfolio(ctx context.Context, in *pb.GetFundPortfolioRequest, opts ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
	if s.getFundPortfolioFn != nil {
		return s.getFundPortfolioFn(ctx, in, opts...)
	}
	return &pb.GetFundPortfolioResponse{}, nil
}
func (s *stubFundClient) GetFundPerformanceHistory(ctx context.Context, in *pb.GetFundPerformanceRequest, opts ...grpc.CallOption) (*pb.GetFundPerformanceResponse, error) {
	if s.getFundPerformanceHistoryFn != nil {
		return s.getFundPerformanceHistoryFn(ctx, in, opts...)
	}
	return &pb.GetFundPerformanceResponse{}, nil
}

// sampleFund returns a sample FundResponse.
func sampleFund() *pb.FundResponse {
	return &pb.FundResponse{
		Id:                  1,
		Name:                "Test Fund",
		Description:         "A test investment fund",
		MinimumContribution: 1000.0,
		ManagerId:           5,
		ManagerName:         "Jane Manager",
		LiquidAssets:        500000.0,
		FundValue:           500000.0,
		Profit:              0.0,
		AccountNumber:       "123-456-78",
		AccountId:           100,
		CreatedAt:           time.Now().Format(time.RFC3339),
		Active:              true,
	}
}

var validCreateFundBody = `{
	"name": "Test Fund",
	"description": "A test fund",
	"minimumContribution": 1000.0,
	"managerId": 5
}`

// ---- CreateFund tests ----

func TestCreateFund_NoToken(t *testing.T) {
	w := serveHandler(CreateFund(&stubFundClient{}), "POST", "/investment/funds", "/investment/funds", validCreateFundBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestCreateFund_BadBody(t *testing.T) {
	// Missing name
	body := `{"description": "no name fund"}`
	w := serveHandlerFull(CreateFund(&stubFundClient{}), "POST", "/investment/funds", "/investment/funds", body, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateFund_Duplicate(t *testing.T) {
	svc := &stubFundClient{
		createFundFn: func(_ context.Context, _ *pb.CreateFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "fund with name already exists")
		},
	}
	w := serveHandlerFull(CreateFund(svc), "POST", "/investment/funds", "/investment/funds", validCreateFundBody, makeSupervisorToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestCreateFund_Happy(t *testing.T) {
	svc := &stubFundClient{
		createFundFn: func(_ context.Context, _ *pb.CreateFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return sampleFund(), nil
		},
	}
	w := serveHandlerFull(CreateFund(svc), "POST", "/investment/funds", "/investment/funds", validCreateFundBody, makeSupervisorToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["name"] != "Test Fund" {
		t.Fatalf("expected name 'Test Fund' got %v", resp["name"])
	}
}

// ---- ListFunds tests ----

func TestListFunds_NoToken(t *testing.T) {
	w := serveHandler(ListFunds(&stubFundClient{}), "GET", "/investment/funds", "/investment/funds", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestListFunds_Empty(t *testing.T) {
	svc := &stubFundClient{
		listFundsFn: func(_ context.Context, _ *pb.ListFundsRequest, _ ...grpc.CallOption) (*pb.ListFundsResponse, error) {
			return &pb.ListFundsResponse{Funds: []*pb.FundResponse{}}, nil
		},
	}
	w := serveHandlerFull(ListFunds(svc), "GET", "/investment/funds", "/investment/funds", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array got len %d", len(resp))
	}
}

func TestListFunds_WithManagerFilter(t *testing.T) {
	var capturedManagerID int64
	svc := &stubFundClient{
		listFundsFn: func(_ context.Context, req *pb.ListFundsRequest, _ ...grpc.CallOption) (*pb.ListFundsResponse, error) {
			capturedManagerID = req.ManagerIdFilter
			return &pb.ListFundsResponse{Funds: []*pb.FundResponse{sampleFund()}}, nil
		},
	}
	w := serveHandlerFull(ListFunds(svc), "GET", "/investment/funds", "/investment/funds?managerId=1", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if capturedManagerID != 1 {
		t.Fatalf("expected managerId filter 1 got %d", capturedManagerID)
	}
}

func TestListFunds_Happy(t *testing.T) {
	svc := &stubFundClient{
		listFundsFn: func(_ context.Context, _ *pb.ListFundsRequest, _ ...grpc.CallOption) (*pb.ListFundsResponse, error) {
			f2 := sampleFund()
			f2.Id = 2
			f2.Name = "Second Fund"
			return &pb.ListFundsResponse{Funds: []*pb.FundResponse{sampleFund(), f2}}, nil
		},
	}
	w := serveHandlerFull(ListFunds(svc), "GET", "/investment/funds", "/investment/funds", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 funds got %d", len(resp))
	}
}

// ---- GetFund tests ----

func TestGetFund_NoToken(t *testing.T) {
	w := serveHandler(GetFund(&stubFundClient{}), "GET", "/investment/funds/:id", "/investment/funds/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetFund_BadID(t *testing.T) {
	w := serveHandlerFull(GetFund(&stubFundClient{}), "GET", "/investment/funds/:id", "/investment/funds/abc", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestGetFund_NotFound(t *testing.T) {
	svc := &stubFundClient{
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(GetFund(svc), "GET", "/investment/funds/:id", "/investment/funds/99", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetFund_Happy(t *testing.T) {
	svc := &stubFundClient{
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return sampleFund(), nil
		},
	}
	w := serveHandlerFull(GetFund(svc), "GET", "/investment/funds/:id", "/investment/funds/1", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["fundValue"]; !ok {
		t.Fatalf("expected fundValue in response, got keys: %v", resp)
	}
}

// ---- UpdateFund tests ----

var validUpdateFundBody = `{
	"name": "Updated Fund",
	"description": "Updated description",
	"minimumContribution": 2000.0,
	"managerId": 5
}`

func TestUpdateFund_NoToken(t *testing.T) {
	w := serveHandler(UpdateFund(&stubFundClient{}), "PUT", "/investment/funds/:id", "/investment/funds/1", validUpdateFundBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestUpdateFund_BadID(t *testing.T) {
	w := serveHandlerFull(UpdateFund(&stubFundClient{}), "PUT", "/investment/funds/:id", "/investment/funds/abc", validUpdateFundBody, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUpdateFund_NotFound(t *testing.T) {
	svc := &stubFundClient{
		updateFundFn: func(_ context.Context, _ *pb.UpdateFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(UpdateFund(svc), "PUT", "/investment/funds/:id", "/investment/funds/99", validUpdateFundBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestUpdateFund_Happy(t *testing.T) {
	svc := &stubFundClient{
		updateFundFn: func(_ context.Context, _ *pb.UpdateFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			f := sampleFund()
			f.Name = "Updated Fund"
			return f, nil
		},
	}
	w := serveHandlerFull(UpdateFund(svc), "PUT", "/investment/funds/:id", "/investment/funds/1", validUpdateFundBody, makeSupervisorToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- DeleteFund tests ----

func TestDeleteFund_NoToken(t *testing.T) {
	w := serveHandler(DeleteFund(&stubFundClient{}), "DELETE", "/investment/funds/:id", "/investment/funds/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestDeleteFund_HasPositions(t *testing.T) {
	svc := &stubFundClient{
		deleteFundFn: func(_ context.Context, _ *pb.DeleteFundRequest, _ ...grpc.CallOption) (*pb.DeleteFundResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "cannot delete fund with active client positions")
		},
	}
	w := serveHandlerFull(DeleteFund(svc), "DELETE", "/investment/funds/:id", "/investment/funds/1", "", makeSupervisorToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestDeleteFund_NotFound(t *testing.T) {
	svc := &stubFundClient{
		deleteFundFn: func(_ context.Context, _ *pb.DeleteFundRequest, _ ...grpc.CallOption) (*pb.DeleteFundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(DeleteFund(svc), "DELETE", "/investment/funds/:id", "/investment/funds/99", "", makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestDeleteFund_Happy(t *testing.T) {
	svc := &stubFundClient{
		deleteFundFn: func(_ context.Context, _ *pb.DeleteFundRequest, _ ...grpc.CallOption) (*pb.DeleteFundResponse, error) {
			return &pb.DeleteFundResponse{}, nil
		},
	}
	w := serveHandlerFull(DeleteFund(svc), "DELETE", "/investment/funds/:id", "/investment/funds/1", "", makeSupervisorToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["message"] != "fund deleted" {
		t.Fatalf("expected message 'fund deleted' got %v", resp["message"])
	}
}

func TestGetMyPositions_NoToken(t *testing.T) {
	w := serveHandlerFull(GetMyPositions(&stubFundClient{}), "GET", "/client/funds/positions", "/client/funds/positions", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetMyPositions_Empty(t *testing.T) {
	svc := &stubFundClient{
		getMyPositionsFn: func(_ context.Context, _ *pb.GetMyPositionsRequest, _ ...grpc.CallOption) (*pb.GetMyPositionsResponse, error) {
			return &pb.GetMyPositionsResponse{Positions: []*pb.ClientFundPosition{}}, nil
		},
	}
	w := serveHandlerFull(GetMyPositions(svc), "GET", "/client/funds/positions", "/client/funds/positions", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestGetMyPositions_Happy(t *testing.T) {
	svc := &stubFundClient{
		getMyPositionsFn: func(_ context.Context, req *pb.GetMyPositionsRequest, _ ...grpc.CallOption) (*pb.GetMyPositionsResponse, error) {
			return &pb.GetMyPositionsResponse{
				Positions: []*pb.ClientFundPosition{
					{
						FundId:               7,
						FundName:             "RAF Growth Fund",
						Description:          "A growth fund",
						FundValue:            50000,
						FundPercentage:       100,
						CurrentPositionValue: 50000,
						TotalInvestedAmount:  10000,
						Profit:               40000,
						MinimumContribution:  1000,
					},
				},
			}, nil
		},
	}
	w := serveHandlerFull(GetMyPositions(svc), "GET", "/client/funds/positions", "/client/funds/positions", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 position, got %d", len(resp))
	}
	if resp[0]["fundName"] != "RAF Growth Fund" {
		t.Fatalf("unexpected fundName: %v", resp[0]["fundName"])
	}
	if resp[0]["profit"] != float64(40000) {
		t.Fatalf("unexpected profit: %v", resp[0]["profit"])
	}
}

// ---- GetFundSecurities tests ----

func TestGetFundSecurities_Empty(t *testing.T) {
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return &pb.GetFundPortfolioResponse{Positions: []*pb.FundPortfolioPosition{}}, nil
		},
	}
	w := serveHandlerFull(GetFundSecurities(fund, &stubSecuritiesClient{}), "GET", "/investment/funds/:id/securities", "/investment/funds/1/securities", "", makeSupervisorToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestGetFundSecurities_Happy(t *testing.T) {
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return &pb.GetFundPortfolioResponse{
				Positions: []*pb.FundPortfolioPosition{
					{ListingId: 10, Quantity: 50, AverageCost: 125.0, AcquisitionDate: "2026-04-01"},
				},
			}, nil
		},
	}
	sec := &stubSecuritiesClient{
		getListingByIdFn: func(_ context.Context, req *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{
					Id: 10, Ticker: "AAPL", Name: "Apple Inc.", Price: 175.0,
					ChangePercent: 1.25, Volume: 50000, InitialMarginCost: 3500.0,
				},
			}, nil
		},
	}
	w := serveHandlerFull(GetFundSecurities(fund, sec), "GET", "/investment/funds/:id/securities", "/investment/funds/1/securities", "", makeSupervisorToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp))
	}
	if resp[0]["ticker"] != "AAPL" {
		t.Fatalf("unexpected ticker: %v", resp[0]["ticker"])
	}
	if resp[0]["acquisitionDate"] != "2026-04-01" {
		t.Fatalf("unexpected acquisitionDate: %v", resp[0]["acquisitionDate"])
	}
}

// ---- BuyFundSecurities tests ----

var buyBody = `{"ticker":"AAPL","amount":10}`

func TestBuyFundSecurities_TickerNotFound(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{}}, nil
		},
	}
	w := serveHandlerFull(BuyFundSecurities(&stubFundClient{}, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuyFundSecurities_InsufficientLiquidity(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return &pb.ValidateFundAccountResponse{AccountId: 100, IsLiquid: false, LiquidAssets: 100.0}, nil
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuyFundSecurities_NotManager(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "not the fund manager")
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuyFundSecurities_Success(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return &pb.ValidateFundAccountResponse{AccountId: 100, IsLiquid: true, LiquidAssets: 99999.0}, nil
		},
	}
	order := &stubOrderClient{
		createOrderFn: func(_ context.Context, _ *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			return &pb_order.CreateOrderResponse{OrderId: 42, OrderType: "MARKET", Status: "APPROVED"}, nil
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, order, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp["orderId"] != float64(42) {
		t.Fatalf("unexpected orderId: %v", resp["orderId"])
	}
}

// ---- SellFundSecurities tests ----

var sellBody = `{"ticker":"AAPL","amount":5}`

func TestSellFundSecurities_NoPosition(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{} // getFundPortfolioFn returns empty by default
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestSellFundSecurities_InsufficientAmount(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return &pb.GetFundPortfolioResponse{
				Positions: []*pb.FundPortfolioPosition{{ListingId: 10, Quantity: 3}}, // only 3, selling 5
			}, nil
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestSellFundSecurities_Success(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return &pb.GetFundPortfolioResponse{
				Positions: []*pb.FundPortfolioPosition{{ListingId: 10, Quantity: 50}},
			}, nil
		},
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return &pb.ValidateFundAccountResponse{AccountId: 100, IsLiquid: true}, nil
		},
	}
	order := &stubOrderClient{
		createOrderFn: func(_ context.Context, _ *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			return &pb_order.CreateOrderResponse{OrderId: 55, OrderType: "MARKET", Status: "APPROVED"}, nil
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, order), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp["orderId"] != float64(55) {
		t.Fatalf("unexpected orderId: %v", resp["orderId"])
	}
}

// ---- WithdrawFund gateway tests ----

var withdrawBody = `{"destinationAccountId":99,"amount":1000}`
var withdrawAllBody = `{"destinationAccountId":99,"withdrawAll":true}`

func TestWithdrawFund_ImmediateSuccess(t *testing.T) {
	svc := &stubFundClient{
		withdrawFundFn: func(_ context.Context, req *pb.WithdrawFundRequest, _ ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
			return &pb.WithdrawFundResponse{Pending: false, Fund: sampleFund()}, nil
		},
	}
	w := serveHandlerFull(WithdrawFund(svc), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawBody, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp["name"] != "Test Fund" {
		t.Fatalf("unexpected name: %v", resp["name"])
	}
}

func TestWithdrawFund_Pending202(t *testing.T) {
	svc := &stubFundClient{
		withdrawFundFn: func(_ context.Context, req *pb.WithdrawFundRequest, _ ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
			return &pb.WithdrawFundResponse{Pending: true, Message: "Payment will arrive once orders are executed"}, nil
		},
	}
	w := serveHandlerFull(WithdrawFund(svc), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawBody, makeClientToken())
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp["message"] != "Payment will arrive once orders are executed" {
		t.Fatalf("unexpected message: %v", resp["message"])
	}
}

func TestWithdrawFund_WithdrawAll_SendsZeroAmount(t *testing.T) {
	var capturedAmount float64
	svc := &stubFundClient{
		withdrawFundFn: func(_ context.Context, req *pb.WithdrawFundRequest, _ ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
			capturedAmount = req.Amount
			return &pb.WithdrawFundResponse{Pending: false, Fund: sampleFund()}, nil
		},
	}
	w := serveHandlerFull(WithdrawFund(svc), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawAllBody, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	if capturedAmount != 0 {
		t.Fatalf("expected amount=0 for withdrawAll, got %v", capturedAmount)
	}
}

// ── GetFundPerformanceHistory ──────────────────────────────────────────────────

func TestGetFundPerformanceHistory_Happy(t *testing.T) {
	svc := &stubFundClient{
		getFundPerformanceHistoryFn: func(_ context.Context, _ *pb.GetFundPerformanceRequest, _ ...grpc.CallOption) (*pb.GetFundPerformanceResponse, error) {
			return &pb.GetFundPerformanceResponse{
				Records: []*pb.PerformanceRecord{
					{Date: "2025-01-01", FundValue: 100000.0, Profit: 5000.0},
					{Date: "2025-01-02", FundValue: 102000.0, Profit: 7000.0},
				},
			}, nil
		},
	}
	w := serveHandlerFull(GetFundPerformanceHistory(svc), "GET", "/investment/funds/:id/performance", "/investment/funds/1/performance?from=2025-01-01&to=2025-01-02", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}
	if result[0]["date"] != "2025-01-01" {
		t.Errorf("expected date 2025-01-01, got %v", result[0]["date"])
	}
	if result[0]["fundValue"].(float64) != 100000.0 {
		t.Errorf("unexpected fundValue: %v", result[0]["fundValue"])
	}
	if result[0]["profit"].(float64) != 5000.0 {
		t.Errorf("unexpected profit: %v", result[0]["profit"])
	}
}

func TestGetFundPerformanceHistory_Empty(t *testing.T) {
	svc := &stubFundClient{
		getFundPerformanceHistoryFn: func(_ context.Context, _ *pb.GetFundPerformanceRequest, _ ...grpc.CallOption) (*pb.GetFundPerformanceResponse, error) {
			return &pb.GetFundPerformanceResponse{Records: []*pb.PerformanceRecord{}}, nil
		},
	}
	w := serveHandlerFull(GetFundPerformanceHistory(svc), "GET", "/investment/funds/:id/performance", "/investment/funds/1/performance", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestGetFundPerformanceHistory_BadId(t *testing.T) {
	w := serveHandlerFull(GetFundPerformanceHistory(&stubFundClient{}), "GET", "/investment/funds/:id/performance", "/investment/funds/abc/performance", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}
