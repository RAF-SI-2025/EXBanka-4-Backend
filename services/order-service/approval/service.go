package approval

import (
	"context"
	"database/sql"
	"errors"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
)

var ErrNotPending = errors.New("order is not in PENDING status")

// ApproveOrder approves a PENDING order on behalf of a supervisor.
// Deducts the order's approximate total from the agent's usedLimit if the order was placed by an agent.
// Returns ErrNotPending if the order has already been approved or declined.
func ApproveOrder(ctx context.Context, orderDB, employeeDB *sql.DB, orderID, supervisorID int64) error {
	order, err := repository.GetOrderByID(ctx, orderDB, orderID)
	if err != nil {
		return err
	}
	if order.Status != "PENDING" {
		return ErrNotPending
	}

	if err := repository.UpdateOrderStatus(ctx, orderDB, orderID, "APPROVED", &supervisorID); err != nil {
		return err
	}

	// Deduct from agent's used limit if placed by an actuary
	if order.UserType == "EMPLOYEE" && employeeDB != nil {
		isActuary := repository.IsActuary(ctx, employeeDB, order.UserID)
		if isActuary {
			orderTotal := float64(order.ContractSize) * order.PricePerUnit * float64(order.Quantity)
			if err := repository.DeductActuaryUsedLimit(ctx, employeeDB, order.UserID, orderTotal); err != nil {
				// Non-fatal: log but don't fail the approval
				_ = err
			}
		}
	}

	return nil
}

// DeclineOrder declines a PENDING order.
// Pass supervisorID = 0 for auto-decline (e.g. expired settlement date).
// Returns ErrNotPending if the order has already been processed.
func DeclineOrder(ctx context.Context, orderDB *sql.DB, orderID, supervisorID int64) error {
	order, err := repository.GetOrderByID(ctx, orderDB, orderID)
	if err != nil {
		return err
	}
	if order.Status != "PENDING" {
		return ErrNotPending
	}

	var approvedBy *int64
	if supervisorID != 0 {
		approvedBy = &supervisorID
	}

	return repository.UpdateOrderStatus(ctx, orderDB, orderID, "DECLINED", approvedBy)
}
