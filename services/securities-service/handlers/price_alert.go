package handlers

import (
	"context"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SecuritiesServer) CreatePriceAlert(ctx context.Context, req *pb.CreatePriceAlertRequest) (*pb.CreatePriceAlertResponse, error) {
	notifType := req.NotificationType
	if notifType == "" {
		notifType = "BOTH"
	}
	var id int64
	var createdAt string
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO price_alerts (user_id, user_type, listing_id, condition, threshold, notification_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at::text`,
		req.UserId, req.UserType, req.ListingId, req.Condition, req.Threshold, notifType,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create price alert: %v", err)
	}

	var ticker string
	_ = s.DB.QueryRowContext(ctx, `SELECT ticker FROM listing WHERE id = $1`, req.ListingId).Scan(&ticker)

	return &pb.CreatePriceAlertResponse{Alert: &pb.PriceAlert{
		Id:               id,
		UserId:           req.UserId,
		UserType:         req.UserType,
		ListingId:        req.ListingId,
		Condition:        req.Condition,
		Threshold:        req.Threshold,
		NotificationType: notifType,
		IsActive:         true,
		CreatedAt:        createdAt,
		Ticker:           ticker,
	}}, nil
}

func (s *SecuritiesServer) ListPriceAlerts(ctx context.Context, req *pb.ListPriceAlertsRequest) (*pb.ListPriceAlertsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT pa.id, pa.user_id, pa.user_type, pa.listing_id, pa.condition, pa.threshold,
		       pa.notification_type, pa.is_active, COALESCE(pa.triggered_at::text,''), pa.created_at::text,
		       l.ticker
		FROM price_alerts pa
		JOIN listing l ON pa.listing_id = l.id
		WHERE pa.user_id = $1 AND pa.user_type = $2
		ORDER BY pa.created_at DESC`,
		req.UserId, req.UserType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list price alerts: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var alerts []*pb.PriceAlert
	for rows.Next() {
		var a pb.PriceAlert
		if err := rows.Scan(
			&a.Id, &a.UserId, &a.UserType, &a.ListingId, &a.Condition, &a.Threshold,
			&a.NotificationType, &a.IsActive, &a.TriggeredAt, &a.CreatedAt, &a.Ticker,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "scan price alert: %v", err)
		}
		alerts = append(alerts, &a)
	}
	return &pb.ListPriceAlertsResponse{Alerts: alerts}, nil
}

func (s *SecuritiesServer) DeletePriceAlert(ctx context.Context, req *pb.DeletePriceAlertRequest) (*pb.DeletePriceAlertResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM price_alerts WHERE id = $1 AND user_id = $2 AND user_type = $3`,
		req.Id, req.UserId, req.UserType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete price alert: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "price alert not found or not owned by user")
	}
	return &pb.DeletePriceAlertResponse{}, nil
}
