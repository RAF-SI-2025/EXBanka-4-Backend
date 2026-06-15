package execution

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

type RecurringScheduler struct {
	DB           *sql.DB
	AccountDB    *sql.DB
	SecuritiesDB *sql.DB
	EmployeeDB   *sql.DB

	SecuritiesClient pb_sec.SecuritiesServiceClient
}

func (rs *RecurringScheduler) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			rs.processRecurring()
		}
	}()
}

func (rs *RecurringScheduler) processRecurring() {
	ctx := context.Background()
	rows, err := rs.DB.QueryContext(ctx, `
		SELECT id, user_id, user_type, asset_id, direction, mode, value, account_id, cadence
		FROM recurring_orders
		WHERE active = true AND next_run <= NOW()`)
	if err != nil {
		log.Printf("recurring-scheduler: query error: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	type recurringRow struct {
		id, userID, assetID, accountID     int64
		userType, direction, mode, cadence string
		value                              float64
	}

	var items []recurringRow
	for rows.Next() {
		var r recurringRow
		if err := rows.Scan(&r.id, &r.userID, &r.userType, &r.assetID, &r.direction, &r.mode, &r.value, &r.accountID, &r.cadence); err != nil {
			log.Printf("recurring-scheduler: scan error: %v", err)
			continue
		}
		items = append(items, r)
	}
	_ = rows.Close()

	for _, r := range items {
		rs.executeRecurring(ctx, r.id, r.userID, r.assetID, r.accountID, r.userType, r.direction, r.mode, r.cadence, r.value)
	}
}

func (rs *RecurringScheduler) executeRecurring(ctx context.Context, recurringID, userID, assetID, accountID int64, userType, direction, mode, cadence string, value float64) {
	// Fetch listing for price and contract size
	var pricePerUnit float64
	var contractSize int32 = 1
	if rs.SecuritiesClient != nil {
		resp, err := rs.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: assetID})
		if err == nil && resp.Summary != nil {
			pricePerUnit = resp.Summary.Price
			if futures := resp.GetFutures(); futures != nil {
				contractSize = int32(futures.ContractSize)
			}
		}
	}

	// Determine quantity from mode
	var quantity int32
	if mode == "BY_AMOUNT" && pricePerUnit > 0 {
		qty := value / (pricePerUnit * float64(contractSize))
		quantity = int32(qty)
		if quantity == 0 {
			quantity = 1
		}
	} else {
		quantity = int32(value)
	}

	// Determine approval status
	initialStatus := "APPROVED"
	if userType == "EMPLOYEE" {
		_, _, needApproval, err := repository.GetActuaryInfo(ctx, rs.EmployeeDB, userID)
		if err == nil && needApproval {
			initialStatus = "PENDING"
		}
	}

	o := &models.Order{
		UserID:            userID,
		UserType:          userType,
		AssetID:           assetID,
		OrderType:         "MARKET",
		Quantity:          quantity,
		ContractSize:      contractSize,
		PricePerUnit:      pricePerUnit,
		Direction:         direction,
		Status:            initialStatus,
		RemainingPortions: quantity,
		AccountID:         accountID,
	}

	_, err := repository.InsertOrder(ctx, rs.DB, o)
	if err != nil {
		log.Printf("recurring-scheduler: insert order for recurring %d: %v", recurringID, err)
	} else if userType == "EMPLOYEE" && initialStatus == "APPROVED" && direction == "BUY" {
		approxAmount := pricePerUnit * float64(quantity) * float64(contractSize)
		approxAmount += CalculateCommission(o.OrderType, approxAmount)
		if err := repository.DeductActuaryUsedLimit(ctx, rs.EmployeeDB, userID, approxAmount); err != nil {
			log.Printf("recurring-scheduler: deduct actuary limit for user %d: %v", userID, err)
		}
	}

	// Advance next_run regardless of success/failure
	nextRun := advanceNextRun(cadence, time.Now())
	_, _ = rs.DB.ExecContext(ctx,
		`UPDATE recurring_orders SET next_run = $2 WHERE id = $1`,
		recurringID, nextRun,
	)
}

func advanceNextRun(cadence string, from time.Time) time.Time {
	switch cadence {
	case "WEEKLY":
		return from.AddDate(0, 0, 7)
	case "MONTHLY":
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}
