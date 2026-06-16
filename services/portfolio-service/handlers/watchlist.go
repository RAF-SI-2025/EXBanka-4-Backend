package handlers

import (
	"context"
	"fmt"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *PortfolioServer) CreateWatchlist(ctx context.Context, req *pb.CreateWatchlistRequest) (*pb.WatchlistResponse, error) {
	var id int64
	var name, userType, createdAt string
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO watchlists (user_id, user_type, name)
		VALUES ($1, $2, $3)
		RETURNING id, name, user_type, created_at::text`,
		req.UserId, req.UserType, req.Name,
	).Scan(&id, &name, &userType, &createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create watchlist: %v", err)
	}
	return &pb.WatchlistResponse{Watchlist: &pb.Watchlist{
		Id: id, UserId: req.UserId, UserType: userType, Name: name, CreatedAt: createdAt,
	}}, nil
}

func (s *PortfolioServer) ListWatchlists(ctx context.Context, req *pb.ListWatchlistsRequest) (*pb.ListWatchlistsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, user_id, user_type, name, created_at::text
		FROM watchlists
		WHERE user_id = $1 AND user_type = $2
		ORDER BY created_at DESC`,
		req.UserId, req.UserType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list watchlists: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var watchlists []*pb.Watchlist
	for rows.Next() {
		var w pb.Watchlist
		if err := rows.Scan(&w.Id, &w.UserId, &w.UserType, &w.Name, &w.CreatedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "scan watchlist: %v", err)
		}
		watchlists = append(watchlists, &w)
	}
	return &pb.ListWatchlistsResponse{Watchlists: watchlists}, nil
}

func (s *PortfolioServer) DeleteWatchlist(ctx context.Context, req *pb.DeleteWatchlistRequest) (*pb.DeleteWatchlistResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM watchlists WHERE id = $1 AND user_id = $2 AND user_type = $3`,
		req.WatchlistId, req.UserId, req.UserType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete watchlist: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "watchlist not found or not owned by user")
	}
	return &pb.DeleteWatchlistResponse{}, nil
}

func (s *PortfolioServer) AddWatchlistItem(ctx context.Context, req *pb.AddWatchlistItemRequest) (*pb.WatchlistItemResponse, error) {
	// Verify ownership
	var ownerID int64
	if err := s.DB.QueryRowContext(ctx,
		`SELECT user_id FROM watchlists WHERE id = $1 AND user_type = $2`, req.WatchlistId, req.UserType,
	).Scan(&ownerID); err != nil || ownerID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "watchlist not found or not owned by user")
	}

	var itemID int64
	var addedAt string
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO watchlist_items (watchlist_id, listing_id)
		VALUES ($1, $2)
		ON CONFLICT (watchlist_id, listing_id) DO UPDATE SET added_at = NOW()
		RETURNING id, added_at::text`,
		req.WatchlistId, req.ListingId,
	).Scan(&itemID, &addedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add watchlist item: %v", err)
	}

	item := &pb.WatchlistItem{Id: itemID, WatchlistId: req.WatchlistId, ListingId: req.ListingId, AddedAt: addedAt}
	s.enrichWatchlistItem(ctx, item)
	return &pb.WatchlistItemResponse{Item: item}, nil
}

func (s *PortfolioServer) RemoveWatchlistItem(ctx context.Context, req *pb.RemoveWatchlistItemRequest) (*pb.RemoveWatchlistItemResponse, error) {
	res, err := s.DB.ExecContext(ctx, `
		DELETE FROM watchlist_items wi
		USING watchlists w
		WHERE wi.watchlist_id = w.id
		  AND wi.watchlist_id = $1
		  AND wi.listing_id = $2
		  AND w.user_id = $3
		  AND w.user_type = $4`,
		req.WatchlistId, req.ListingId, req.UserId, req.UserType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "remove watchlist item: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "watchlist item not found or not owned by user")
	}
	return &pb.RemoveWatchlistItemResponse{}, nil
}

func (s *PortfolioServer) GetWatchlistItems(ctx context.Context, req *pb.GetWatchlistItemsRequest) (*pb.GetWatchlistItemsResponse, error) {
	// Verify ownership
	var ownerID int64
	if err := s.DB.QueryRowContext(ctx,
		`SELECT user_id FROM watchlists WHERE id = $1 AND user_type = $2`, req.WatchlistId, req.UserType,
	).Scan(&ownerID); err != nil || ownerID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "watchlist not found or not owned by user")
	}

	// Step 1: get items from portfolio DB
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, listing_id, added_at::text
		FROM watchlist_items
		WHERE watchlist_id = $1
		ORDER BY added_at DESC`,
		req.WatchlistId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get watchlist items: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var items []*pb.WatchlistItem
	for rows.Next() {
		var item pb.WatchlistItem
		item.WatchlistId = req.WatchlistId
		if err := rows.Scan(&item.Id, &item.ListingId, &item.AddedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "scan watchlist item: %v", err)
		}
		items = append(items, &item)
	}
	_ = rows.Close()

	// Step 2: enrich from securities DB in one query if available
	if s.SecuritiesDB != nil && len(items) > 0 {
		listingIDs := make([]int64, len(items))
		for i, item := range items {
			listingIDs[i] = item.ListingId
		}
		secRows, secErr := s.SecuritiesDB.QueryContext(ctx, `
			SELECT id, ticker, name, type, price, "change", volume
			FROM listing WHERE id = ANY($1)`,
			pq.Array(listingIDs),
		)
		if secErr == nil {
			type listingInfo struct {
				ticker, name, assetType string
				price, change, volume   float64
			}
			infoMap := make(map[int64]listingInfo)
			for secRows.Next() {
				var id int64
				var li listingInfo
				if err := secRows.Scan(&id, &li.ticker, &li.name, &li.assetType, &li.price, &li.change, &li.volume); err == nil {
					infoMap[id] = li
				}
			}
			_ = secRows.Close()
			for _, item := range items {
				if li, ok := infoMap[item.ListingId]; ok {
					item.Ticker = li.ticker
					item.Name = li.name
					item.AssetType = li.assetType
					item.Price = li.price
					item.Change = li.change
					item.Volume = li.volume
					if item.Price != 0 {
						item.ChangePercent = li.change / item.Price * 100
					}
				}
			}
		}
	}

	return &pb.GetWatchlistItemsResponse{Items: items}, nil
}

// enrichWatchlistItem populates ticker/price/etc from the securities DB for a newly added item.
func (s *PortfolioServer) enrichWatchlistItem(ctx context.Context, item *pb.WatchlistItem) {
	if s.SecuritiesDB == nil {
		return
	}
	var absChange float64
	err := s.SecuritiesDB.QueryRowContext(ctx,
		`SELECT ticker, name, type, price, "change", volume FROM listing WHERE id = $1`, item.ListingId,
	).Scan(&item.Ticker, &item.Name, &item.AssetType, &item.Price, &absChange, &item.Volume)
	if err != nil {
		_ = fmt.Errorf("enrich watchlist item: %v", err)
		return
	}
	item.Change = absChange
	if item.Price != 0 {
		item.ChangePercent = absChange / item.Price * 100
	}
}
