package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- InvestFund tests (0% → covered) ----

var investBody = `{"sourceAccountId":10,"amount":5000}`

func TestInvestFund_NoToken(t *testing.T) {
	w := serveHandler(InvestFund(&stubFundClient{}), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestInvestFund_BadID(t *testing.T) {
	w := serveHandlerFull(InvestFund(&stubFundClient{}), "POST", "/investment/funds/:id/invest", "/investment/funds/abc/invest", investBody, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInvestFund_BadBody(t *testing.T) {
	w := serveHandlerFull(InvestFund(&stubFundClient{}), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", `{}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInvestFund_NotFound(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestInvestFund_InsufficientFunds(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "insufficient balance")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestInvestFund_PermissionDenied(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "fund is closed")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestInvestFund_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestInvestFund_Happy_Client(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, req *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			if req.ClientType != "CLIENT" {
				return nil, status.Error(codes.InvalidArgument, "unexpected client type")
			}
			return sampleFund(), nil
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp["name"] == nil {
		t.Fatalf("expected fund name in response")
	}
}

func TestInvestFund_Happy_Employee_UsesBANK(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, req *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			if req.ClientType != "BANK" {
				return nil, status.Errorf(codes.InvalidArgument, "expected BANK got %s", req.ClientType)
			}
			return sampleFund(), nil
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeEmployeeToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
}

// ---- mapFundError remaining branches ----

func TestMapFundError_AlreadyExists(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "position already exists")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestMapFundError_FailedPrecondition(t *testing.T) {
	svc := &stubFundClient{
		investFundFn: func(_ context.Context, _ *pb.InvestFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "fund not accepting investments")
		},
	}
	w := serveHandlerFull(InvestFund(svc), "POST", "/investment/funds/:id/invest", "/investment/funds/1/invest", investBody, makeClientToken())
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 got %d", w.Code)
	}
}

// ---- WithdrawFund edge cases ----

func TestWithdrawFund_NoToken(t *testing.T) {
	w := serveHandler(WithdrawFund(&stubFundClient{}), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestWithdrawFund_BadID(t *testing.T) {
	w := serveHandlerFull(WithdrawFund(&stubFundClient{}), "POST", "/investment/funds/:id/redeem", "/investment/funds/abc/redeem", withdrawBody, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestWithdrawFund_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		withdrawFundFn: func(_ context.Context, _ *pb.WithdrawFundRequest, _ ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "insufficient position")
		},
	}
	w := serveHandlerFull(WithdrawFund(svc), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawBody, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// ---- GetFundSecurities edge cases ----

func TestGetFundSecurities_BadID(t *testing.T) {
	w := serveHandlerFull(GetFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}), "GET", "/investment/funds/:id/securities", "/investment/funds/abc/securities", "", makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestGetFundSecurities_GrpcError(t *testing.T) {
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(GetFundSecurities(fund, &stubSecuritiesClient{}), "GET", "/investment/funds/:id/securities", "/investment/funds/1/securities", "", makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetFundSecurities_SkipsFailedSecLookup(t *testing.T) {
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return &pb.GetFundPortfolioResponse{
				Positions: []*pb.FundPortfolioPosition{
					{ListingId: 99, Quantity: 10, AverageCost: 100.0, AcquisitionDate: "2026-01-01"},
				},
			}, nil
		},
	}
	sec := &stubSecuritiesClient{
		getListingByIdFn: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "listing not found")
		},
	}
	w := serveHandlerFull(GetFundSecurities(fund, sec), "GET", "/investment/funds/:id/securities", "/investment/funds/1/securities", "", makeSupervisorToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array (failed lookup skipped), got %d", len(resp))
	}
}

// ---- BuyFundSecurities edge cases ----

func TestBuyFundSecurities_NoToken(t *testing.T) {
	w := serveHandler(BuyFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestBuyFundSecurities_BadID(t *testing.T) {
	w := serveHandlerFull(BuyFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/abc/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestBuyFundSecurities_BadBody(t *testing.T) {
	w := serveHandlerFull(BuyFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", `{}`, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestBuyFundSecurities_OrderError(t *testing.T) {
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
			return nil, status.Error(codes.Internal, "order service error")
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, order, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestBuyFundSecurities_AdminPath(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 10.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			f := sampleFund()
			f.LiquidAssets = 99999
			f.AccountId = 200
			return f, nil
		},
	}
	order := &stubOrderClient{
		createOrderFn: func(_ context.Context, req *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			return &pb_order.CreateOrderResponse{OrderId: 77, OrderType: "MARKET", Status: "APPROVED"}, nil
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, order, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeAdminToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuyFundSecurities_AdminPath_InsufficientLiquidity(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 1000.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			f := sampleFund()
			f.LiquidAssets = 1.0 // not enough for 10 * 1000
			return f, nil
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeAdminToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// ---- SellFundSecurities edge cases ----

func TestSellFundSecurities_NoToken(t *testing.T) {
	w := serveHandler(SellFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestSellFundSecurities_BadID(t *testing.T) {
	w := serveHandlerFull(SellFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/abc/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestSellFundSecurities_BadBody(t *testing.T) {
	w := serveHandlerFull(SellFundSecurities(&stubFundClient{}, &stubSecuritiesClient{}, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", `{}`, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestSellFundSecurities_SecError(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return nil, status.Error(codes.Internal, "sec service down")
		},
	}
	w := serveHandlerFull(SellFundSecurities(&stubFundClient{}, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestSellFundSecurities_NotManager(t *testing.T) {
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
			return nil, status.Error(codes.PermissionDenied, "not the fund manager")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestSellFundSecurities_OrderError(t *testing.T) {
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
			return nil, status.Error(codes.Internal, "order service error")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, order), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestSellFundSecurities_AdminPath(t *testing.T) {
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
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			f := sampleFund()
			f.AccountId = 200
			return f, nil
		},
	}
	order := &stubOrderClient{
		createOrderFn: func(_ context.Context, _ *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			return &pb_order.CreateOrderResponse{OrderId: 88, OrderType: "MARKET", Status: "APPROVED"}, nil
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, order), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeAdminToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
}

// ---- DeleteFund missing branches ----

func TestDeleteFund_BadID(t *testing.T) {
	w := serveHandlerFull(DeleteFund(&stubFundClient{}), "DELETE", "/investment/funds/:id", "/investment/funds/abc", "", makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// ---- ListFunds missing branches ----

func TestListFunds_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		listFundsFn: func(_ context.Context, _ *pb.ListFundsRequest, _ ...grpc.CallOption) (*pb.ListFundsResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(ListFunds(svc), "GET", "/investment/funds", "/investment/funds", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

// ---- UpdateFund missing branches ----

func TestUpdateFund_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		updateFundFn: func(_ context.Context, _ *pb.UpdateFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(UpdateFund(svc), "PUT", "/investment/funds/:id", "/investment/funds/1", validUpdateFundBody, makeSupervisorToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

// ---- WithdrawFund missing branches ----

func TestWithdrawFund_BadBody(t *testing.T) {
	w := serveHandlerFull(WithdrawFund(&stubFundClient{}), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", `{}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// ---- GetMyPositions missing branches ----

func TestGetMyPositions_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		getMyPositionsFn: func(_ context.Context, _ *pb.GetMyPositionsRequest, _ ...grpc.CallOption) (*pb.GetMyPositionsResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(GetMyPositions(svc), "GET", "/client/funds/positions", "/client/funds/positions", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

// ---- GetFundPerformanceHistory missing branches ----

func TestGetFundPerformanceHistory_GrpcError(t *testing.T) {
	svc := &stubFundClient{
		getFundPerformanceHistoryFn: func(_ context.Context, _ *pb.GetFundPerformanceRequest, _ ...grpc.CallOption) (*pb.GetFundPerformanceResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(GetFundPerformanceHistory(svc), "GET", "/investment/funds/:id/performance", "/investment/funds/1/performance", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

// ---- BuyFundSecurities — ValidateFundAccount NotFound path ----

func TestBuyFundSecurities_FundNotFound(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

// ---- SellFundSecurities — portfolio error and ValidateFundAccount NotFound ----

func TestSellFundSecurities_PortfolioError(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundPortfolioFn: func(_ context.Context, _ *pb.GetFundPortfolioRequest, _ ...grpc.CallOption) (*pb.GetFundPortfolioResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

// ---- UpdateFund missing branches ----

func TestUpdateFund_BadBody(t *testing.T) {
	w := serveHandlerFull(UpdateFund(&stubFundClient{}), "PUT", "/investment/funds/:id", "/investment/funds/1", `{bad}`, makeSupervisorToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// ---- WithdrawFund — employee (BANK) clientType path ----

func TestWithdrawFund_EmployeeUsesBANK(t *testing.T) {
	var capturedClientType string
	svc := &stubFundClient{
		withdrawFundFn: func(_ context.Context, req *pb.WithdrawFundRequest, _ ...grpc.CallOption) (*pb.WithdrawFundResponse, error) {
			capturedClientType = req.ClientType
			return &pb.WithdrawFundResponse{Pending: false, Fund: sampleFund()}, nil
		},
	}
	w := serveHandlerFull(WithdrawFund(svc), "POST", "/investment/funds/:id/redeem", "/investment/funds/1/redeem", withdrawBody, makeEmployeeToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if capturedClientType != "BANK" {
		t.Fatalf("expected ClientType=BANK for employee caller, got %q", capturedClientType)
	}
}

// ---- BuyFundSecurities — ValidateFundAccount default error path ----

func TestBuyFundSecurities_ValidateInternalError(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 175.0}}}, nil
		},
	}
	fund := &stubFundClient{
		validateFundAccountFn: func(_ context.Context, _ *pb.ValidateFundAccountRequest, _ ...grpc.CallOption) (*pb.ValidateFundAccountResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeSupervisorToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

// ---- BuyFundSecurities — admin path GetFund error ----

func TestBuyFundSecurities_AdminPath_GetFundError(t *testing.T) {
	sec := &stubSecuritiesClient{
		getListingsFn: func(_ context.Context, _ *pb_sec.GetListingsRequest, _ ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
			return &pb_sec.GetListingsResponse{Listings: []*pb_sec.ListingSummary{{Id: 10, Price: 10.0}}}, nil
		},
	}
	fund := &stubFundClient{
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(BuyFundSecurities(fund, sec, &stubOrderClient{}, &stubExchangeClient{}), "POST", "/investment/funds/:id/securities/buy", "/investment/funds/1/securities/buy", buyBody, makeAdminToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

// ---- SellFundSecurities — admin path GetFund error ----

func TestSellFundSecurities_AdminPath_GetFundError(t *testing.T) {
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
		getFundFn: func(_ context.Context, _ *pb.GetFundRequest, _ ...grpc.CallOption) (*pb.FundResponse, error) {
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeAdminToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

// ---- SellFundSecurities — ValidateFundAccount default error path ----

func TestSellFundSecurities_ValidateInternalError(t *testing.T) {
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
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestSellFundSecurities_FundNotFound(t *testing.T) {
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
			return nil, status.Error(codes.NotFound, "fund not found")
		},
	}
	w := serveHandlerFull(SellFundSecurities(fund, sec, &stubOrderClient{}), "POST", "/investment/funds/:id/securities/sell", "/investment/funds/1/securities/sell", sellBody, makeSupervisorToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}
