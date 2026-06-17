package handlers

import (
	"context"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *PortfolioServer) CreateDividendPayout(ctx context.Context, req *pb.CreateDividendPayoutRequest) (*pb.DividendPayoutResponse, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO dividend_payouts
		  (user_id, user_type, stock_listing_id, quantity, gross_amount, currency, tax_rsd, net_amount, account_id, payment_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		req.UserId, req.UserType, req.StockListingId, req.Quantity, req.GrossAmount,
		req.Currency, req.TaxRsd, req.NetAmount, req.AccountId, req.PaymentDate,
	).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create dividend payout: %v", err)
	}
	return &pb.DividendPayoutResponse{Id: id}, nil
}

func (s *PortfolioServer) GetDividendHistory(ctx context.Context, req *pb.GetDividendHistoryRequest) (*pb.GetDividendHistoryResponse, error) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := s.DB.QueryContext(ctx, `
		SELECT dp.id, dp.user_id, dp.user_type, dp.stock_listing_id, dp.quantity,
		       dp.gross_amount, dp.currency, dp.tax_rsd, dp.net_amount,
		       COALESCE(dp.account_id, 0), dp.payment_date::text
		FROM dividend_payouts dp
		WHERE dp.user_id = $1 AND dp.user_type = $2
		  AND ($3 = 0 OR dp.stock_listing_id = $3)
		  AND ($4 = '' OR dp.payment_date >= $4::date)
		  AND ($5 = '' OR dp.payment_date <= $5::date)
		ORDER BY dp.payment_date DESC, dp.created_at DESC
		LIMIT $6 OFFSET $7`,
		req.UserId, req.UserType, req.ListingId, req.FromDate, req.ToDate, pageSize, offset,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get dividend history: %v", err)
	}
	defer func() { _ = rows.Close() }()

	payouts := []*pb.DividendPayout{}
	for rows.Next() {
		var dp pb.DividendPayout
		if err := rows.Scan(
			&dp.Id, &dp.UserId, &dp.UserType, &dp.StockListingId, &dp.Quantity,
			&dp.GrossAmount, &dp.Currency, &dp.TaxRsd, &dp.NetAmount, &dp.AccountId, &dp.PaymentDate,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "scan dividend payout: %v", err)
		}
		payouts = append(payouts, &dp)
	}

	// Enrich with ticker from securities DB in one query, same pattern as GetWatchlistItems.
	if s.SecuritiesDB != nil && len(payouts) > 0 {
		listingIDs := make([]int64, len(payouts))
		for i, p := range payouts {
			listingIDs[i] = p.StockListingId
		}
		secRows, secErr := s.SecuritiesDB.QueryContext(ctx, `
			SELECT id, ticker FROM listing WHERE id = ANY($1)`,
			pq.Array(listingIDs),
		)
		if secErr == nil {
			tickerMap := make(map[int64]string)
			for secRows.Next() {
				var id int64
				var ticker string
				if err := secRows.Scan(&id, &ticker); err == nil {
					tickerMap[id] = ticker
				}
			}
			_ = secRows.Close()
			for _, p := range payouts {
				if ticker, ok := tickerMap[p.StockListingId]; ok {
					p.Ticker = ticker
				}
			}
		}
	}

	return &pb.GetDividendHistoryResponse{Payouts: payouts}, nil
}

func (s *PortfolioServer) GetHoldersByListing(ctx context.Context, req *pb.GetHoldersByListingRequest) (*pb.GetHoldersByListingResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT user_id, user_type, amount, account_id
		FROM portfolio_entry
		WHERE listing_id = $1 AND amount > 0`,
		req.ListingId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get holders by listing: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var holders []*pb.PortfolioHolder
	for rows.Next() {
		var h pb.PortfolioHolder
		var amount int32
		if err := rows.Scan(&h.UserId, &h.UserType, &amount, &h.AccountId); err != nil {
			return nil, status.Errorf(codes.Internal, "scan holder: %v", err)
		}
		h.Amount = float64(amount)
		holders = append(holders, &h)
	}
	return &pb.GetHoldersByListingResponse{Holders: holders}, nil
}
