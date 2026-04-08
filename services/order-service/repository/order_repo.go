package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
)

func GetApprovedActiveOrders(ctx context.Context, db *sql.DB) ([]models.Order, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, user_type, asset_id, order_type, quantity, contract_size,
		       price_per_unit, limit_value, stop_value, direction, status, approved_by,
		       is_done, last_modification, remaining_portions, after_hours, is_aon, is_margin, account_id
		FROM orders
		WHERE status = 'APPROVED' AND is_done = false`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.UserType, &o.AssetID, &o.OrderType, &o.Quantity, &o.ContractSize,
			&o.PricePerUnit, &o.LimitValue, &o.StopValue, &o.Direction, &o.Status, &o.ApprovedBy,
			&o.IsDone, &o.LastModification, &o.RemainingPortions, &o.AfterHours, &o.IsAON, &o.IsMargin, &o.AccountID,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func GetOrderByID(ctx context.Context, db *sql.DB, id int64) (models.Order, error) {
	var o models.Order
	err := db.QueryRowContext(ctx, `
		SELECT id, user_id, user_type, asset_id, order_type, quantity, contract_size,
		       price_per_unit, limit_value, stop_value, direction, status, approved_by,
		       is_done, last_modification, remaining_portions, after_hours, is_aon, is_margin, account_id
		FROM orders WHERE id = $1`, id).Scan(
		&o.ID, &o.UserID, &o.UserType, &o.AssetID, &o.OrderType, &o.Quantity, &o.ContractSize,
		&o.PricePerUnit, &o.LimitValue, &o.StopValue, &o.Direction, &o.Status, &o.ApprovedBy,
		&o.IsDone, &o.LastModification, &o.RemainingPortions, &o.AfterHours, &o.IsAON, &o.IsMargin, &o.AccountID,
	)
	return o, err
}

func InsertOrder(ctx context.Context, db *sql.DB, o *models.Order) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO orders
		  (user_id, user_type, asset_id, order_type, quantity, contract_size,
		   price_per_unit, limit_value, stop_value, direction, status,
		   remaining_portions, after_hours, is_aon, is_margin, account_id,
		   last_modification)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id`,
		o.UserID, o.UserType, o.AssetID, o.OrderType, o.Quantity, o.ContractSize,
		o.PricePerUnit, o.LimitValue, o.StopValue, o.Direction, o.Status,
		o.RemainingPortions, o.AfterHours, o.IsAON, o.IsMargin, o.AccountID,
		time.Now(),
	).Scan(&id)
	return id, err
}

func UpdateOrderStatus(ctx context.Context, db *sql.DB, id int64, status string, approvedBy *int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE orders SET status = $1, approved_by = $2, last_modification = $3 WHERE id = $4`,
		status, approvedBy, time.Now(), id)
	return err
}

func SetOrderDone(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE orders SET is_done = true, last_modification = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

func UpdateRemainingPortions(ctx context.Context, db *sql.DB, id int64, remaining int32) error {
	_, err := db.ExecContext(ctx,
		`UPDATE orders SET remaining_portions = $1, last_modification = $2 WHERE id = $3`,
		remaining, time.Now(), id)
	return err
}

func InsertPortion(ctx context.Context, db *sql.DB, p *models.OrderPortion) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO order_portions (order_id, quantity, price, filled_at) VALUES ($1, $2, $3, $4)`,
		p.OrderID, p.Quantity, p.Price, time.Now())
	return err
}

func GetPendingOrders(ctx context.Context, db *sql.DB) ([]models.Order, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, user_type, asset_id, order_type, quantity, contract_size,
		       price_per_unit, limit_value, stop_value, direction, status, approved_by,
		       is_done, last_modification, remaining_portions, after_hours, is_aon, is_margin, account_id
		FROM orders
		WHERE status = 'PENDING'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.UserType, &o.AssetID, &o.OrderType, &o.Quantity, &o.ContractSize,
			&o.PricePerUnit, &o.LimitValue, &o.StopValue, &o.Direction, &o.Status, &o.ApprovedBy,
			&o.IsDone, &o.LastModification, &o.RemainingPortions, &o.AfterHours, &o.IsAON, &o.IsMargin, &o.AccountID,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetActuaryInfo returns limit management data for an employee.
// Returns sql.ErrNoRows if the employee is not an actuary.
func GetActuaryInfo(ctx context.Context, employeeDB *sql.DB, employeeID int64) (limitAmount, usedLimit float64, needApproval bool, err error) {
	err = employeeDB.QueryRowContext(ctx,
		`SELECT limit_amount, used_limit, need_approval FROM actuary_info WHERE employee_id = $1`,
		employeeID,
	).Scan(&limitAmount, &usedLimit, &needApproval)
	return
}

// IsActuary reports whether the employee has a row in actuary_info.
func IsActuary(ctx context.Context, employeeDB *sql.DB, employeeID int64) bool {
	var exists bool
	employeeDB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM actuary_info WHERE employee_id = $1)`, employeeID,
	).Scan(&exists)
	return exists
}

// DeductActuaryUsedLimit increments used_limit by amount for the given employee.
func DeductActuaryUsedLimit(ctx context.Context, employeeDB *sql.DB, employeeID int64, amount float64) error {
	_, err := employeeDB.ExecContext(ctx,
		`UPDATE actuary_info SET used_limit = used_limit + $1 WHERE employee_id = $2`,
		amount, employeeID)
	return err
}

// ListOrders returns orders optionally filtered by status and/or agent (user_id).
// status="" or "ALL" returns all statuses. agentID=0 returns all agents.
func ListOrders(ctx context.Context, db *sql.DB, statusFilter string, agentID int64) ([]models.Order, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, user_type, asset_id, order_type, quantity, contract_size,
		       price_per_unit, limit_value, stop_value, direction, status, approved_by,
		       is_done, last_modification, remaining_portions, after_hours, is_aon, is_margin, account_id
		FROM orders
		WHERE ($1 = '' OR $1 = 'ALL' OR status::text = $1)
		  AND ($2 = 0 OR user_id = $2)
		ORDER BY last_modification DESC`,
		statusFilter, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.UserType, &o.AssetID, &o.OrderType, &o.Quantity, &o.ContractSize,
			&o.PricePerUnit, &o.LimitValue, &o.StopValue, &o.Direction, &o.Status, &o.ApprovedBy,
			&o.IsDone, &o.LastModification, &o.RemainingPortions, &o.AfterHours, &o.IsAON, &o.IsMargin, &o.AccountID,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// CancelOrder marks an order as fully done with no remaining portions.
func CancelOrder(ctx context.Context, db *sql.DB, orderID int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE orders SET is_done = true, remaining_portions = 0, last_modification = $1 WHERE id = $2`,
		time.Now(), orderID)
	return err
}
