package handlers

import (
	"context"
	"fmt"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EmployeeServer) LogAuditEvent(ctx context.Context, req *pb.AuditLogRequest) (*pb.AuditLogResponse, error) {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO audit_logs (actor_id, actor_type, actor_name, action, target_id, target_type, target_name, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ActorId, req.ActorType, req.ActorName, req.Action,
		req.TargetId, req.TargetType, req.TargetName, req.OldValue, req.NewValue,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "log audit event: %v", err)
	}
	return &pb.AuditLogResponse{}, nil
}

func (s *EmployeeServer) ListAuditLogs(ctx context.Context, req *pb.ListAuditLogsRequest) (*pb.ListAuditLogsResponse, error) {
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
		SELECT id, actor_id, actor_type, COALESCE(actor_name,''), action,
		       COALESCE(target_id,0), COALESCE(target_type,''), COALESCE(target_name,''),
		       COALESCE(old_value,''), COALESCE(new_value,''), timestamp::text
		FROM audit_logs
		WHERE ($1 = '' OR action = $1)
		  AND ($2 = 0 OR actor_id = $2)
		  AND ($3 = '' OR timestamp::date >= $3::date)
		  AND ($4 = '' OR timestamp::date <= $4::date)
		ORDER BY timestamp DESC
		LIMIT $5 OFFSET $6`,
		req.Action, req.ActorId, req.FromDate, req.ToDate, pageSize, offset,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audit logs: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []*pb.AuditLogEntry
	for rows.Next() {
		var e pb.AuditLogEntry
		if err := rows.Scan(
			&e.Id, &e.ActorId, &e.ActorType, &e.ActorName, &e.Action,
			&e.TargetId, &e.TargetType, &e.TargetName,
			&e.OldValue, &e.NewValue, &e.Timestamp,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "scan audit log: %v", err)
		}
		entries = append(entries, &e)
	}
	return &pb.ListAuditLogsResponse{Entries: entries}, nil
}

// insertAuditLog is a helper used by the other employee handlers.
func (s *EmployeeServer) insertAuditLog(ctx context.Context, actorID int64, actorType, action string, targetID int64, oldValue, newValue string) {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO audit_logs (actor_id, actor_type, action, target_id, target_type, old_value, new_value)
		VALUES ($1, $2, $3, $4, 'EMPLOYEE', $5, $6)`,
		actorID, actorType, action, targetID, oldValue, newValue,
	)
	if err != nil {
		_ = fmt.Errorf("audit log insert: %v", err)
	}
}
