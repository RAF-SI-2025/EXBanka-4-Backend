package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// ---- stub client ----

type stubPortfolioClient struct {
	getPortfolioFn      func(context.Context, *pb.GetPortfolioRequest, ...grpc.CallOption) (*pb.GetPortfolioResponse, error)
	getProfitFn         func(context.Context, *pb.GetProfitRequest, ...grpc.CallOption) (*pb.GetProfitResponse, error)
	getMyTaxFn          func(context.Context, *pb.GetMyTaxRequest, ...grpc.CallOption) (*pb.GetMyTaxResponse, error)
	getTaxListFn        func(context.Context, *pb.GetTaxListRequest, ...grpc.CallOption) (*pb.GetTaxListResponse, error)
	collectTaxFn        func(context.Context, *pb.CollectTaxRequest, ...grpc.CallOption) (*pb.CollectTaxResponse, error)
	collectTaxForUserFn func(context.Context, *pb.CollectTaxForUserRequest, ...grpc.CallOption) (*pb.CollectTaxForUserResponse, error)
	setPublicModeFn     func(context.Context, *pb.SetPublicModeRequest, ...grpc.CallOption) (*pb.SetPublicModeResponse, error)
}

func (s *stubPortfolioClient) UpdateHolding(ctx context.Context, in *pb.UpdateHoldingRequest, opts ...grpc.CallOption) (*pb.UpdateHoldingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetPortfolio(ctx context.Context, in *pb.GetPortfolioRequest, opts ...grpc.CallOption) (*pb.GetPortfolioResponse, error) {
	if s.getPortfolioFn != nil {
		return s.getPortfolioFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetProfit(ctx context.Context, in *pb.GetProfitRequest, opts ...grpc.CallOption) (*pb.GetProfitResponse, error) {
	if s.getProfitFn != nil {
		return s.getProfitFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) SetPublicAmount(ctx context.Context, in *pb.SetPublicAmountRequest, opts ...grpc.CallOption) (*pb.SetPublicAmountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetMyTax(ctx context.Context, in *pb.GetMyTaxRequest, opts ...grpc.CallOption) (*pb.GetMyTaxResponse, error) {
	if s.getMyTaxFn != nil {
		return s.getMyTaxFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetTaxList(ctx context.Context, in *pb.GetTaxListRequest, opts ...grpc.CallOption) (*pb.GetTaxListResponse, error) {
	if s.getTaxListFn != nil {
		return s.getTaxListFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) CollectTax(ctx context.Context, in *pb.CollectTaxRequest, opts ...grpc.CallOption) (*pb.CollectTaxResponse, error) {
	if s.collectTaxFn != nil {
		return s.collectTaxFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) CollectTaxForUser(ctx context.Context, in *pb.CollectTaxForUserRequest, opts ...grpc.CallOption) (*pb.CollectTaxForUserResponse, error) {
	if s.collectTaxForUserFn != nil {
		return s.collectTaxForUserFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) SetPublicMode(ctx context.Context, in *pb.SetPublicModeRequest, opts ...grpc.CallOption) (*pb.SetPublicModeResponse, error) {
	if s.setPublicModeFn != nil {
		return s.setPublicModeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) CreateWatchlist(ctx context.Context, in *pb.CreateWatchlistRequest, opts ...grpc.CallOption) (*pb.WatchlistResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) ListWatchlists(ctx context.Context, in *pb.ListWatchlistsRequest, opts ...grpc.CallOption) (*pb.ListWatchlistsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) DeleteWatchlist(ctx context.Context, in *pb.DeleteWatchlistRequest, opts ...grpc.CallOption) (*pb.DeleteWatchlistResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) AddWatchlistItem(ctx context.Context, in *pb.AddWatchlistItemRequest, opts ...grpc.CallOption) (*pb.WatchlistItemResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) RemoveWatchlistItem(ctx context.Context, in *pb.RemoveWatchlistItemRequest, opts ...grpc.CallOption) (*pb.RemoveWatchlistItemResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetWatchlistItems(ctx context.Context, in *pb.GetWatchlistItemsRequest, opts ...grpc.CallOption) (*pb.GetWatchlistItemsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) CreateDividendPayout(ctx context.Context, in *pb.CreateDividendPayoutRequest, opts ...grpc.CallOption) (*pb.DividendPayoutResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetDividendHistory(ctx context.Context, in *pb.GetDividendHistoryRequest, opts ...grpc.CallOption) (*pb.GetDividendHistoryResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPortfolioClient) GetHoldersByListing(ctx context.Context, in *pb.GetHoldersByListingRequest, opts ...grpc.CallOption) (*pb.GetHoldersByListingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- GetPortfolio ----

func TestGetPortfolio_NoToken(t *testing.T) {
	w := serveHandlerFull(GetPortfolio(&stubPortfolioClient{}, "EMPLOYEE"), "GET", "/portfolio", "/portfolio", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetPortfolio_GrpcError(t *testing.T) {
	client := &stubPortfolioClient{
		getPortfolioFn: func(_ context.Context, _ *pb.GetPortfolioRequest, _ ...grpc.CallOption) (*pb.GetPortfolioResponse, error) {
			return nil, fmt.Errorf("portfolio service down")
		},
	}
	w := serveHandlerFull(GetPortfolio(client, "EMPLOYEE"), "GET", "/portfolio", "/portfolio", "", makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetPortfolio_Happy(t *testing.T) {
	client := &stubPortfolioClient{
		getPortfolioFn: func(_ context.Context, _ *pb.GetPortfolioRequest, _ ...grpc.CallOption) (*pb.GetPortfolioResponse, error) {
			return &pb.GetPortfolioResponse{Entries: []*pb.PortfolioEntry{}}, nil
		},
	}
	w := serveHandlerFull(GetPortfolio(client, "EMPLOYEE"), "GET", "/portfolio", "/portfolio", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- GetProfit ----

func TestGetProfit_NoToken(t *testing.T) {
	w := serveHandlerFull(GetProfit(&stubPortfolioClient{}, "EMPLOYEE"), "GET", "/portfolio/profit", "/portfolio/profit", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetProfit_GrpcError(t *testing.T) {
	client := &stubPortfolioClient{
		getProfitFn: func(_ context.Context, _ *pb.GetProfitRequest, _ ...grpc.CallOption) (*pb.GetProfitResponse, error) {
			return nil, fmt.Errorf("portfolio service down")
		},
	}
	w := serveHandlerFull(GetProfit(client, "EMPLOYEE"), "GET", "/portfolio/profit", "/portfolio/profit", "", makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetProfit_Happy(t *testing.T) {
	client := &stubPortfolioClient{
		getProfitFn: func(_ context.Context, _ *pb.GetProfitRequest, _ ...grpc.CallOption) (*pb.GetProfitResponse, error) {
			return &pb.GetProfitResponse{TotalProfit: 12500.50}, nil
		},
	}
	w := serveHandlerFull(GetProfit(client, "EMPLOYEE"), "GET", "/portfolio/profit", "/portfolio/profit", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
}
