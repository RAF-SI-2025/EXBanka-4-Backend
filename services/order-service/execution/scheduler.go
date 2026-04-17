package execution

import (
	"context"
	"database/sql"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/approval"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

// Scheduler polls for approved orders and drives their partial-fill simulation.
type Scheduler struct {
	DB               *sql.DB
	AccountDB        *sql.DB
	SecuritiesDB     *sql.DB
	ExchangeDB       *sql.DB
	EmployeeDB       *sql.DB
	SecuritiesClient pb_sec.SecuritiesServiceClient
	LoanClient       pb_loan.LoanServiceClient
	EmployeeClient   pb_emp.EmployeeServiceClient
	PortfolioClient  pb_portfolio.PortfolioServiceClient

	inProgress sync.Map // map[int64]bool — orders currently being executed
}

// Start launches the background polling goroutines.
func (s *Scheduler) Start() {
	go s.loop()
	go s.expiredOrderLoop()
}

// expiredOrderLoop periodically auto-declines PENDING orders with an expired settlement date.
func (s *Scheduler) expiredOrderLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.declineExpiredOrders()
	}
}

func (s *Scheduler) declineExpiredOrders() {
	ctx := context.Background()
	pending, err := repository.GetPendingOrders(ctx, s.DB)
	if err != nil {
		log.Printf("order-scheduler: expired check query error: %v", err)
		return
	}

	for _, o := range pending {
		var settlementDate string
		err := s.SecuritiesDB.QueryRowContext(ctx, `
			SELECT settlement_date::text FROM listing_futures_contract WHERE listing_id = $1
			UNION ALL
			SELECT settlement_date::text FROM listing_option WHERE listing_id = $1
			LIMIT 1`, o.AssetID,
		).Scan(&settlementDate)

		if err != nil {
			continue // not a futures/options listing, or no settlement date
		}

		if approval.IsSettlementExpired(settlementDate) {
			if err := approval.DeclineOrder(ctx, s.DB, o.ID, 0); err != nil {
				log.Printf("order-scheduler: auto-decline order %d error: %v", o.ID, err)
			} else {
				log.Printf("order-scheduler: auto-declined expired order %d (settlement %s)", o.ID, settlementDate)
			}
		}
	}
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		s.processOrders()
	}
}

func (s *Scheduler) processOrders() {
	ctx := context.Background()
	orders, err := repository.GetApprovedActiveOrders(ctx, s.DB)
	if err != nil {
		log.Printf("order-scheduler: query error: %v", err)
		return
	}

	for _, o := range orders {
		if _, loaded := s.inProgress.LoadOrStore(o.ID, true); loaded {
			continue // already being handled
		}
		go func(order models.Order) {
			defer s.inProgress.Delete(order.ID)
			s.executeOrder(order)
		}(o)
	}
}

// executeOrder runs the full partial-fill loop for a single order.
func (s *Scheduler) executeOrder(order models.Order) {
	ctx := context.Background()

	// 1. Fetch listing details (ask, bid, volume, margin data)
	listingResp, err := s.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: order.AssetID})
	if err != nil {
		log.Printf("order-scheduler: GetListingById(%d) error: %v", order.AssetID, err)
		return
	}
	listing := listingResp.Summary

	// 2. Margin validation (only checked once per execution start)
	if order.IsMargin {
		if !s.validateMargin(ctx, order, listing.InitialMarginCost) {
			log.Printf("order-scheduler: order %d failed margin check — declining", order.ID)
			if err := repository.UpdateOrderStatus(ctx, s.DB, order.ID, "DECLINED", nil); err != nil {
				log.Printf("order-scheduler: decline order %d error: %v", order.ID, err)
			}
			return
		}
	}

	// 3. Fetch exchange currency for commission transfers
	currencyCode, err := s.listingCurrency(ctx, order.AssetID)
	if err != nil {
		log.Printf("order-scheduler: currency lookup for asset %d error: %v", order.AssetID, err)
		return
	}

	remaining := order.RemainingPortions

	for remaining > 0 {
		// Re-fetch fresh prices each cycle
		freshResp, err := s.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: order.AssetID})
		if err != nil {
			log.Printf("order-scheduler: price refresh error for order %d: %v", order.ID, err)
			time.Sleep(10 * time.Second)
			continue
		}
		fresh := freshResp.Summary

		var limitValue, stopValue float64
		if order.LimitValue != nil {
			limitValue = *order.LimitValue
		}
		if order.StopValue != nil {
			stopValue = *order.StopValue
		}

		pricePerUnit, canExecute := CalculatePrice(
			order.OrderType, order.Direction,
			fresh.Ask, fresh.Bid,
			limitValue, stopValue,
		)

		if !canExecute {
			// Price conditions not met; wait and retry
			time.Sleep(FillInterval(fresh.Volume, remaining, order.AfterHours))
			continue
		}

		// 4. AON: only fill if we can fill everything at once
		fillQty := int32(rand.Int32N(remaining) + 1)
		if order.IsAON {
			if remaining != order.Quantity {
				// A prior partial fill happened — shouldn't occur for AON, but guard anyway
				time.Sleep(5 * time.Second)
				continue
			}
			fillQty = remaining // must fill all at once
		}

		totalPrice := float64(fillQty) * float64(order.ContractSize) * pricePerUnit
		commission := CalculateCommission(order.OrderType, totalPrice)

		// 5. Transfer commission to bank account
		if commission > 0 {
			if err := s.transferCommission(ctx, currencyCode, commission); err != nil {
				log.Printf("order-scheduler: commission transfer error for order %d: %v", order.ID, err)
				// Non-fatal: continue with the fill
			}
		}

		// 6. Record partial fill
		portion := &models.OrderPortion{
			OrderID:  order.ID,
			Quantity: fillQty,
			Price:    pricePerUnit,
		}
		if err := repository.InsertPortion(ctx, s.DB, portion); err != nil {
			log.Printf("order-scheduler: insert portion error for order %d: %v", order.ID, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if s.PortfolioClient != nil {
			_, err := s.PortfolioClient.UpdateHolding(ctx, &pb_portfolio.UpdateHoldingRequest{
				UserId:    order.UserID,
				UserType:  order.UserType,
				ListingId: order.AssetID,
				Quantity:  fillQty,
				Price:     pricePerUnit,
				Direction: order.Direction,
				AccountId: order.AccountID,
			})
			if err != nil {
				log.Printf("order-scheduler: portfolio update error for order %d: %v", order.ID, err)
				// Non-fatal: fill already recorded
			}
		}

		remaining -= fillQty
		if err := repository.UpdateRemainingPortions(ctx, s.DB, order.ID, remaining); err != nil {
			log.Printf("order-scheduler: update remaining error for order %d: %v", order.ID, err)
		}

		if remaining == 0 {
			if err := repository.SetOrderDone(ctx, s.DB, order.ID); err != nil {
				log.Printf("order-scheduler: set done error for order %d: %v", order.ID, err)
			}
			log.Printf("order-scheduler: order %d fully executed", order.ID)
			return
		}

		// 7. Wait before next partial fill
		interval := FillInterval(fresh.Volume, remaining, order.AfterHours)
		time.Sleep(interval)
	}
}

// validateMargin checks whether the margin requirement is met for the order's user.
func (s *Scheduler) validateMargin(ctx context.Context, order models.Order, initialMarginCost float64) bool {
	var accountBalance float64
	err := s.AccountDB.QueryRowContext(ctx,
		`SELECT available_balance FROM accounts WHERE id = $1`, order.AccountID,
	).Scan(&accountBalance)
	if err != nil {
		log.Printf("order-scheduler: balance lookup error for account %d: %v", order.AccountID, err)
		accountBalance = 0
	}

	var loanAmount float64

	if order.UserType == "CLIENT" {
		loansResp, err := s.LoanClient.GetClientLoans(ctx, &pb_loan.GetClientLoansRequest{ClientId: order.UserID})
		if err == nil {
			for _, l := range loansResp.Loans {
				if l.Status == "APPROVED" && l.Amount > loanAmount {
					loanAmount = l.Amount
				}
			}
		}
	} else {
		// EMPLOYEE: check for MARGIN permission
		empResp, err := s.EmployeeClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: order.UserID})
		if err == nil {
			for _, p := range empResp.Employee.Permissions {
				if p == "MARGIN" {
					// Permission granted — treat as if loan requirement is met
					return true
				}
			}
		}
	}

	return ValidateMargin(initialMarginCost, loanAmount, accountBalance)
}

// listingCurrency returns the exchange currency code (e.g. "USD") for the given listing ID.
func (s *Scheduler) listingCurrency(ctx context.Context, listingID int64) (string, error) {
	var currency string
	err := s.SecuritiesDB.QueryRowContext(ctx, `
		SELECT e.currency
		FROM listing l
		JOIN stock_exchanges e ON l.exchange_id = e.id
		WHERE l.id = $1`, listingID,
	).Scan(&currency)
	return currency, err
}

// transferCommission credits the bank account for the given currency with the commission amount.
func (s *Scheduler) transferCommission(ctx context.Context, currencyCode string, amount float64) error {
	var currencyID int64
	err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT id FROM currencies WHERE code = $1`, currencyCode,
	).Scan(&currencyID)
	if err != nil {
		return err
	}

	var bankAccountID int64
	err = s.AccountDB.QueryRowContext(ctx,
		`SELECT id FROM accounts WHERE account_type = 'BANK' AND owner_id = 0 AND currency_id = $1`,
		currencyID,
	).Scan(&bankAccountID)
	if err != nil {
		return err
	}

	_, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		amount, bankAccountID,
	)
	return err
}
