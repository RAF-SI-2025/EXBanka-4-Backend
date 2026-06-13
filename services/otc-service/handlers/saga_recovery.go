package handlers

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// RecoverInFlightSagas is called at service startup. It finds any SAGA that was left in
// Running or Compensating state (e.g. because the service was killed mid-flight) and
// compensates all completed forward steps.
func (s *OtcServer) RecoverInFlightSagas() {
	type sagaRow struct {
		id, contractID int64
		currentStep    int
		sagaStatus     string
	}

	rows, err := s.DB.Query(
		`SELECT id, contract_id, current_step, status FROM otc_saga WHERE status IN ('Running', 'Compensating')`)
	if err != nil {
		log.Printf("RecoverInFlightSagas: query error: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	var sagas []sagaRow
	for rows.Next() {
		var r sagaRow
		if err := rows.Scan(&r.id, &r.contractID, &r.currentStep, &r.sagaStatus); err == nil {
			sagas = append(sagas, r)
		}
	}
	_ = rows.Close()

	for _, saga := range sagas {
		log.Printf("RecoverInFlightSagas: recovering saga %d (contract %d, step %d, status %s)",
			saga.id, saga.contractID, saga.currentStep, saga.sagaStatus)
		go s.recoverSaga(saga.id, saga.contractID)
	}
}

// recoverSaga compensates all completed forward steps for the given contract.
func (s *OtcServer) recoverSaga(sagaID, contractID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Mark as Compensating so the recovery is idempotent.
	_, _ = s.DB.ExecContext(ctx,
		`UPDATE otc_saga SET status='Compensating', updated_at=NOW() WHERE id=$1 AND status IN ('Running','Compensating')`,
		sagaID)

	// Determine which forward steps completed by reading the log.
	successRows, err := s.DB.QueryContext(ctx,
		`SELECT DISTINCT step FROM otc_saga_log WHERE contract_id=$1 AND status='SUCCESS' ORDER BY step DESC`,
		contractID)
	if err != nil {
		log.Printf("recoverSaga %d: failed to read log: %v", sagaID, err)
		return
	}
	var completedSteps []int
	for successRows.Next() {
		var step int
		if err := successRows.Scan(&step); err == nil {
			completedSteps = append(completedSteps, step)
		}
	}
	_ = successRows.Close()

	if len(completedSteps) == 0 {
		// Nothing completed, just mark as Compensated.
		_, _ = s.DB.ExecContext(ctx,
			`UPDATE otc_saga SET status='Compensated', updated_at=NOW() WHERE id=$1`, sagaID)
		return
	}

	// Load contract details needed to build compensation operations.
	var sellerID, buyerID int64
	var sellerType, buyerType, ticker, currency string
	var amount int32
	var strikePrice float64
	err = s.DB.QueryRowContext(ctx,
		`SELECT seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, currency
		 FROM otc_contracts WHERE id=$1`, contractID,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &ticker, &amount, &strikePrice, &currency)
	if err != nil {
		log.Printf("recoverSaga %d: failed to load contract: %v", sagaID, err)
		return
	}

	currencyID, ok := currencyIDMap[currency]
	if !ok {
		log.Printf("recoverSaga %d: unknown currency %s", sagaID, currency)
		return
	}

	buyerAccountID, err := findAccount(ctx, s.AccountDB, portfolioUserID(buyerID, buyerType), currencyID)
	if err != nil {
		log.Printf("recoverSaga %d: cannot find buyer account: %v", sagaID, err)
		return
	}

	buyerCurrencyID, err := getAccountCurrencyID(ctx, s.AccountDB, buyerAccountID)
	if err != nil {
		log.Printf("recoverSaga %d: cannot get buyer currency: %v", sagaID, err)
		return
	}

	totalCostToPay, err := convertAmount(ctx, s.ExchangeDB, strikePrice*float64(amount), currencyID, buyerCurrencyID)
	if err != nil {
		log.Printf("recoverSaga %d: currency conversion failed: %v", sagaID, err)
		return
	}

	sellerAccountID, err := findAccount(ctx, s.AccountDB, portfolioUserID(sellerID, sellerType), currencyID)
	if err != nil {
		log.Printf("recoverSaga %d: cannot find seller account: %v", sagaID, err)
		return
	}

	listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
	if err != nil {
		log.Printf("recoverSaga %d: cannot find listing for ticker %s: %v", sagaID, ticker, err)
		return
	}

	sellerPortfolioID := portfolioUserID(sellerID, sellerType)
	buyerPortfolioID := portfolioUserID(buyerID, buyerType)

	sagaLogRecovery := func(step int, stepStatus, errMsg string) {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_, _ = s.DB.ExecContext(logCtx,
			`INSERT INTO otc_saga_log (contract_id, step, status, error_msg) VALUES ($1, $2, $3, $4)`,
			contractID, step, stepStatus, sql.NullString{String: errMsg, Valid: errMsg != ""},
		)
	}

	// Run compensators in reverse order for all completed forward steps.
	// completedSteps is already in DESC order.
	highestStep := completedSteps[0]

	if highestStep >= 4 {
		retryExec(s.PortfolioDB, `
			UPDATE portfolio_entry SET amount=amount+$1, reserved_amount=reserved_amount+$1, public_amount=public_amount+$1, last_modified=NOW()
			WHERE user_id=$2 AND user_type=$3 AND listing_id=$4`,
			amount, sellerPortfolioID, sellerType, listingID)
		retryExec(s.PortfolioDB, `
			UPDATE portfolio_entry SET amount=amount-$1, last_modified=NOW()
			WHERE user_id=$2 AND user_type=$3 AND listing_id=$4`,
			amount, buyerPortfolioID, buyerType, listingID)
		sagaLogRecovery(4, "COMPENSATED", "recovery")
	}

	if highestStep >= 3 {
		retryExec(s.AccountDB, `UPDATE accounts SET balance=balance+$1 WHERE id=$2`, totalCostToPay, buyerAccountID)
		retryExec(s.AccountDB, `UPDATE accounts SET balance=balance-$1, available_balance=available_balance-$1 WHERE id=$2`,
			strikePrice*float64(amount), sellerAccountID)
		sagaLogRecovery(3, "COMPENSATED", "recovery")
	}

	if highestStep >= 2 {
		retryExec(s.PortfolioDB,
			`UPDATE portfolio_entry SET reserved_amount=GREATEST(0, reserved_amount-$1)
			 WHERE user_id=$2 AND user_type=$3 AND listing_id=$4`,
			amount, sellerPortfolioID, sellerType, listingID)
		sagaLogRecovery(2, "COMPENSATED", "recovery")
	}

	if highestStep >= 1 {
		retryExec(s.AccountDB,
			`UPDATE accounts SET available_balance=available_balance+$1 WHERE id=$2`,
			totalCostToPay, buyerAccountID)
		sagaLogRecovery(1, "COMPENSATED", "recovery")
	}

	_, _ = s.DB.ExecContext(ctx,
		`UPDATE otc_saga SET status='Compensated', updated_at=NOW() WHERE id=$1`, sagaID)
	log.Printf("recoverSaga %d: recovery complete (compensated up to step %d)", sagaID, highestStep)
}
