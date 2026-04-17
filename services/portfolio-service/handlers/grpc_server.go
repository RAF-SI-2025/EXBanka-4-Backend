package handlers

import (
	"context"
	"database/sql"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/repository"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PortfolioServer struct {
	pb.UnimplementedPortfolioServiceServer
	DB *sql.DB
}

func (s *PortfolioServer) UpdateHolding(ctx context.Context, req *pb.UpdateHoldingRequest) (*pb.UpdateHoldingResponse, error) {
	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be positive")
	}
	if req.Direction != "BUY" && req.Direction != "SELL" {
		return nil, status.Error(codes.InvalidArgument, "direction must be BUY or SELL")
	}

	err := repository.UpsertHolding(ctx, s.DB, req.UserId, req.UserType, req.ListingId, req.AccountId, req.Quantity, req.Price, req.Direction)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert holding: %v", err)
	}
	return &pb.UpdateHoldingResponse{}, nil
}

func (s *PortfolioServer) GetPortfolio(ctx context.Context, req *pb.GetPortfolioRequest) (*pb.GetPortfolioResponse, error) {
	entries, err := repository.GetHoldings(ctx, s.DB, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get holdings: %v", err)
	}

	pbEntries := make([]*pb.PortfolioEntry, 0, len(entries))
	for _, e := range entries {
		pbEntries = append(pbEntries, &pb.PortfolioEntry{
			Id:           e.ID,
			ListingId:    e.ListingID,
			Amount:       e.Amount,
			BuyPrice:     e.BuyPrice,
			LastModified: e.LastModified.Format("2006-01-02T15:04:05"),
			IsPublic:     e.IsPublic,
			PublicAmount: e.PublicAmount,
			AccountId:    e.AccountID,
		})
	}
	return &pb.GetPortfolioResponse{Entries: pbEntries}, nil
}

func (s *PortfolioServer) GetProfit(_ context.Context, _ *pb.GetProfitRequest) (*pb.GetProfitResponse, error) {
	return nil, status.Error(codes.Unimplemented, "implemented in issue #156")
}

func (s *PortfolioServer) SetPublicAmount(_ context.Context, _ *pb.SetPublicAmountRequest) (*pb.SetPublicAmountResponse, error) {
	return nil, status.Error(codes.Unimplemented, "implemented in issue #147")
}
