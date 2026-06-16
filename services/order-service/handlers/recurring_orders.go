package handlers

import (
	"context"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func (s *OrderServer) CreateRecurringOrder(ctx context.Context, req *pb.CreateRecurringOrderRequest) (*pb.RecurringOrderResponse, error) {
	nextRun := calcNextRun(req.Cadence, time.Now())
	var id int64
	var createdAt string
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO recurring_orders (user_id, user_type, asset_id, direction, mode, value, account_id, cadence, next_run)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at::text`,
		req.UserId, req.UserType, req.AssetId, req.Direction, req.Mode, req.Value, req.AccountId, req.Cadence, nextRun,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "create recurring order: %v", err)
	}
	return &pb.RecurringOrderResponse{Order: &pb.RecurringOrder{
		Id:        id,
		UserId:    req.UserId,
		UserType:  req.UserType,
		AssetId:   req.AssetId,
		Direction: req.Direction,
		Mode:      req.Mode,
		Value:     req.Value,
		AccountId: req.AccountId,
		Cadence:   req.Cadence,
		NextRun:   nextRun.Format(time.RFC3339),
		Active:    true,
		CreatedAt: createdAt,
	}}, nil
}

func (s *OrderServer) ListRecurringOrders(ctx context.Context, req *pb.ListRecurringOrdersRequest) (*pb.ListRecurringOrdersResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, user_id, user_type, asset_id, direction, mode, value, account_id, cadence, next_run::text, active, created_at::text
		FROM recurring_orders
		WHERE user_id = $1 AND user_type = $2
		ORDER BY created_at DESC`,
		req.UserId, req.UserType,
	)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "list recurring orders: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var orders []*pb.RecurringOrder
	for rows.Next() {
		var o pb.RecurringOrder
		if err := rows.Scan(
			&o.Id, &o.UserId, &o.UserType, &o.AssetId, &o.Direction, &o.Mode, &o.Value,
			&o.AccountId, &o.Cadence, &o.NextRun, &o.Active, &o.CreatedAt,
		); err != nil {
			return nil, grpcstatus.Errorf(codes.Internal, "scan recurring order: %v", err)
		}
		orders = append(orders, &o)
	}
	return &pb.ListRecurringOrdersResponse{Orders: orders}, nil
}

func (s *OrderServer) PauseRecurringOrder(ctx context.Context, req *pb.RecurringOrderIdRequest) (*pb.RecurringOrderResponse, error) {
	return s.setRecurringActive(ctx, req, false)
}

func (s *OrderServer) ResumeRecurringOrder(ctx context.Context, req *pb.RecurringOrderIdRequest) (*pb.RecurringOrderResponse, error) {
	return s.setRecurringActive(ctx, req, true)
}

func (s *OrderServer) CancelRecurringOrder(ctx context.Context, req *pb.RecurringOrderIdRequest) (*pb.RecurringOrderResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM recurring_orders WHERE id = $1 AND user_id = $2 AND user_type = $3`,
		req.Id, req.UserId, req.UserType,
	)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "cancel recurring order: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, grpcstatus.Errorf(codes.NotFound, "recurring order not found")
	}
	return &pb.RecurringOrderResponse{}, nil
}

func (s *OrderServer) setRecurringActive(ctx context.Context, req *pb.RecurringOrderIdRequest, active bool) (*pb.RecurringOrderResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE recurring_orders SET active = $4 WHERE id = $1 AND user_id = $2 AND user_type = $3`,
		req.Id, req.UserId, req.UserType, active,
	)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "update recurring order: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, grpcstatus.Errorf(codes.NotFound, "recurring order not found")
	}
	return &pb.RecurringOrderResponse{}, nil
}

func calcNextRun(cadence string, from time.Time) time.Time {
	switch cadence {
	case "WEEKLY":
		return from.AddDate(0, 0, 7)
	case "MONTHLY":
		return from.AddDate(0, 1, 0)
	default: // DAILY
		return from.AddDate(0, 0, 1)
	}
}
