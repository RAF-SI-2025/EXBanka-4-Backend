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
