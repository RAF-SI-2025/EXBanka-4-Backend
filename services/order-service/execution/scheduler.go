package execution

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/approval"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	orderqueue "github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/queue"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb_fund "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	amqp "github.com/rabbitmq/amqp091-go"
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
	ExchangeClient   pb_exchange.ExchangeServiceClient
	FundClient       pb_fund.FundServiceClient
	AmqpChannel      *amqp.Channel
	ClientClient     pb_client.ClientServiceClient

	inProgress sync.Map // map[int64]bool — orders currently being executed
}

func (s *Scheduler) sendNotif(ctx context.Context, orderID, userID int64, userType, ticker, direction, status string, quantity, filledQty int32, pricePerUnit float64) {
	if s.AmqpChannel == nil {
		return
	}
	var email, firstName string
	if userType == "CLIENT" && s.ClientClient != nil {
		if resp, err := s.ClientClient.GetClientById(ctx, &pb_client.GetClientByIdRequest{Id: userID}); err == nil && resp.Client != nil {
			email, firstName = resp.Client.Email, resp.Client.FirstName
		}
	} else if s.EmployeeClient != nil {
		if resp, err := s.EmployeeClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: userID}); err == nil && resp.Employee != nil {
			email, firstName = resp.Employee.Email, resp.Employee.FirstName
		}
	}
	if email == "" {
		return
	}
	_ = orderqueue.PublishOrderStatus(s.AmqpChannel, orderqueue.OrderStatusMessage{
		Email:        email,
		FirstName:    firstName,
		OrderID:      orderID,
		Ticker:       ticker,
		Direction:    direction,
		Status:       status,
		Quantity:     quantity,
		FilledQty:    filledQty,
		PricePerUnit: pricePerUnit,
	})
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

		if !canExecute || pricePerUnit == 0 {
			// Price conditions not met or no price data yet; wait and retry
			time.Sleep(FillInterval(fresh.Volume, remaining, order.AfterHours))
			continue
		}

		// 4. AON: only fill if we can fill everything at once
		aonQty, aonOk := DetermineAONFillQty(order.IsAON, remaining, order.Quantity)
		if !aonOk {
			time.Sleep(5 * time.Second)
			continue
		}
		var fillQty int32
		if order.IsAON {
			fillQty = aonQty
		} else {
			fillQty = int32(rand.Int32N(remaining) + 1)
		}

		totalPrice := float64(fillQty) * float64(order.ContractSize) * pricePerUnit
		commission := CalculateCommission(order.OrderType, totalPrice)

		// 5. Settle account balance and transfer commission to bank.
		// Returns the total in account currency (converted from security currency when needed).
		settledTotal, err := s.settleAccountAndCommission(ctx, order, totalPrice, commission, currencyCode)
		if err != nil {
			log.Printf("order-scheduler: settlement failed for order %d: %v — declining", order.ID, err)
			_ = repository.UpdateOrderStatus(ctx, s.DB, order.ID, "DECLINED", nil)
			return
		}
		if commission > 0 {
			if err := repository.AddCommissionPaid(ctx, s.DB, order.ID, commission); err != nil {
				log.Printf("order-scheduler: AddCommissionPaid for order %d: %v", order.ID, err)
			}
		}
		// Price per unit in account currency — used to keep liquid_assets and average_cost in the
		// same currency as the fund account (RSD), not the security's trading currency.
		settledPricePerUnit := settledTotal / float64(fillQty) / float64(order.ContractSize)

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

		if order.FundID != 0 && s.FundClient != nil {
			_, err := s.FundClient.UpdateFundHolding(ctx, &pb_fund.UpdateFundHoldingRequest{
				FundId:    order.FundID,
				ListingId: order.AssetID,
				Quantity:  int64(fillQty),
				Price:     settledPricePerUnit,
				Direction: order.Direction,
			})
			if err != nil {
				log.Printf("order-scheduler: fund holding update error for order %d: %v", order.ID, err)
				// Non-fatal: fill already recorded
			}
		} else if s.PortfolioClient != nil {
			_, err := s.PortfolioClient.UpdateHolding(ctx, &pb_portfolio.UpdateHoldingRequest{
				UserId:    order.UserID,
				UserType:  order.UserType,
				ListingId: order.AssetID,
				Quantity:  fillQty,
				Price:     pricePerUnit,
				Direction: order.Direction,
				AccountId: order.AccountID,
				AssetType: listing.Type,
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

		if remaining > 0 {
			filled := order.Quantity - remaining
			go s.sendNotif(context.Background(), order.ID, order.UserID, order.UserType, listing.Ticker, order.Direction, "PARTIAL_FILL", order.Quantity, filled, pricePerUnit)
		}

		if remaining == 0 {
			if err := repository.SetOrderDone(ctx, s.DB, order.ID); err != nil {
				log.Printf("order-scheduler: set done error for order %d: %v", order.ID, err)
			}
			go s.sendNotif(context.Background(), order.ID, order.UserID, order.UserType, listing.Ticker, order.Direction, "DONE", order.Quantity, order.Quantity, pricePerUnit)
			log.Printf("order-scheduler: order %d fully executed", order.ID)
			if order.FundID != 0 && order.Direction == "SELL" && s.FundClient != nil {
				if _, err := s.FundClient.CheckPendingWithdrawals(ctx, &pb_fund.CheckPendingWithdrawalsRequest{
					FundId: order.FundID,
				}); err != nil {
					log.Printf("order-scheduler: CheckPendingWithdrawals for fund %d error: %v", order.FundID, err)
				}
			}
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

// settleAccountAndCommission debits/credits the order account for the trade and
// credits the commission to the bank account.
//   - BUY:  deduct (totalPrice + commission) from the buyer's account
//   - SELL: credit (totalPrice - commission) to the seller's account
//   - commission: always credited to the bank account
//
// When the security's currency differs from the account's currency, amounts are
// converted using today's exchange rates. CLIENT orders include a 0.5% exchange
// commission; EMPLOYEE (agent) orders do not.
// settleAccountAndCommission returns the settled total in account currency (after conversion).
func (s *Scheduler) settleAccountAndCommission(ctx context.Context, order models.Order, totalPrice, commission float64, currencyCode string) (float64, error) {
	const exchangeCommRate = 0.005

	// 0. Ensure today's exchange rates are seeded (lazy population via exchange-service).
	if s.ExchangeClient != nil {
		if _, err := s.ExchangeClient.GetExchangeRates(ctx, &pb_exchange.GetExchangeRatesRequest{}); err != nil {
			log.Printf("order-scheduler: ensureTodayRates warning: %v", err)
		}
	}

	// 1. Look up account currency.
	var accountCurrencyID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT currency_id FROM accounts WHERE id = $1`, order.AccountID,
	).Scan(&accountCurrencyID); err != nil {
		return 0, fmt.Errorf("account currency_id: %w", err)
	}
	var accountCurrencyCode string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, accountCurrencyID,
	).Scan(&accountCurrencyCode); err != nil {
		return 0, fmt.Errorf("currency code: %w", err)
	}

	bankCurrencyCode := currencyCode

	// 2. Convert amounts to account currency when there is a mismatch.
	if accountCurrencyCode != currencyCode {
		bankCurrencyCode = accountCurrencyCode

		getRate := func(code, rateType string) (float64, error) {
			if code == "RSD" {
				return 1.0, nil
			}
			var r float64
			err := s.ExchangeDB.QueryRowContext(ctx,
				`SELECT `+rateType+` FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
				code,
			).Scan(&r)
			return r, err
		}

		var convRate float64
		var convErr error
		commSteps := 1

		if order.Direction == "BUY" {
			// User pays accountCurrency for securityCurrency-priced goods → bank sells security currency.
			switch {
			case accountCurrencyCode == "RSD":
				convRate, convErr = getRate(currencyCode, "selling_rate")
			case currencyCode == "RSD":
				var r float64
				r, convErr = getRate(accountCurrencyCode, "buying_rate")
				if convErr == nil {
					convRate = 1.0 / r
				}
			default:
				var sell, buy float64
				sell, convErr = getRate(currencyCode, "selling_rate")
				if convErr == nil {
					buy, convErr = getRate(accountCurrencyCode, "buying_rate")
				}
				if convErr == nil {
					convRate = sell / buy
					commSteps = 2
				}
			}
		} else { // SELL
			// User receives accountCurrency for securityCurrency-priced goods → bank buys security currency.
			switch {
			case accountCurrencyCode == "RSD":
				convRate, convErr = getRate(currencyCode, "buying_rate")
			case currencyCode == "RSD":
				var r float64
				r, convErr = getRate(accountCurrencyCode, "selling_rate")
				if convErr == nil {
					convRate = 1.0 / r
				}
			default:
				var buy, sell float64
				buy, convErr = getRate(currencyCode, "buying_rate")
				if convErr == nil {
					sell, convErr = getRate(accountCurrencyCode, "selling_rate")
				}
				if convErr == nil {
					convRate = buy / sell
					commSteps = 2
				}
			}
		}
		if convErr != nil {
			return 0, fmt.Errorf("exchange rate for %s: %w", currencyCode, convErr)
		}

		convertedTotal := totalPrice * convRate
		convertedTradeComm := commission * convRate

		var exchangeComm float64
		if order.UserType == "CLIENT" {
			exchangeComm = convertedTotal * exchangeCommRate * float64(commSteps)
		}

		totalPrice = convertedTotal
		commission = convertedTradeComm + exchangeComm
	}

	// 3. Settle order account (now in account currency).
	var accountDelta float64
	if order.Direction == "BUY" {
		accountDelta = -(totalPrice + commission)
		debitAmount := -accountDelta // positive
		result, err := s.AccountDB.ExecContext(ctx,
			`UPDATE accounts
			 SET balance = balance + $1, available_balance = available_balance + $1
			 WHERE id = $2 AND available_balance >= $3`,
			accountDelta, order.AccountID, debitAmount,
		)
		if err != nil {
			return 0, err
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return 0, fmt.Errorf("insufficient funds for order %d", order.ID)
		}
	} else {
		accountDelta = totalPrice - commission
		if _, err := s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			accountDelta, order.AccountID,
		); err != nil {
			return 0, err
		}
	}

	// 4. Credit commission to bank account (in account currency).
	if commission <= 0 {
		return totalPrice, nil
	}
	var currencyID int64
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT id FROM currencies WHERE code = $1`, bankCurrencyCode,
	).Scan(&currencyID); err != nil {
		return 0, err
	}
	var bankAccountID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT id FROM accounts WHERE account_type = 'BANK' AND owner_id = 0 AND currency_id = $1`,
		currencyID,
	).Scan(&bankAccountID); err != nil {
		return 0, err
	}
	_, err := s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		commission, bankAccountID,
	)
	return totalPrice, err
}
