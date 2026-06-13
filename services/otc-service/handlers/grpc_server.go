package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// retryExec retries a DB Exec up to 3 times with linear backoff.
// Uses db.Exec (not ExecContext) so it survives a cancelled request context (e.g. during compensation).
func retryExec(db *sql.DB, query string, args ...interface{}) {
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := db.Exec(query, args...); err == nil {
			return
		}
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
}

// sagaFaultHook checks X-Saga-* gRPC metadata and injects failures or delays.
// Only active when OTC_SAGA_TEST_HOOKS=true. Returns non-nil error if the phase should fail.
// compensateFailCounts tracks how many times each compensator has failed (for -Times: N).
func sagaFaultHook(ctx context.Context, phase string, compensateFailCounts map[string]int) error {
	if os.Getenv("OTC_SAGA_TEST_HOOKS") != "true" {
		return nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	// X-Saga-Inject-Delay: Fi:Nms
	if vals := md.Get("x-saga-inject-delay"); len(vals) > 0 {
		// format: "F3:5000"
		var delayPhase string
		var delayMs int
		if n, _ := fmt.Sscanf(vals[0], "%5s:%d", &delayPhase, &delayMs); n == 2 && delayPhase == phase {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	// X-Saga-Force-Fail: Fi  (fail kind defaults to "before", meaning fail without side effects)
	if vals := md.Get("x-saga-force-fail"); len(vals) > 0 && vals[0] == phase {
		kind := "before"
		if kv := md.Get("x-saga-force-fail-kind"); len(kv) > 0 {
			kind = kv[0]
		}
		// "before" = caller checks this before executing the step → return error immediately
		// "after"  = caller checks this after executing the step → side effects already applied
		_ = kind
		return fmt.Errorf("fault injected for phase %s", phase)
	}

	// X-Saga-Compensate-Fail: Ci  +  X-Saga-Compensate-Fail-Times: N
	if vals := md.Get("x-saga-compensate-fail"); len(vals) > 0 && vals[0] == phase {
		maxFails := 1
		if tv := md.Get("x-saga-compensate-fail-times"); len(tv) > 0 {
			if n, err := strconv.Atoi(tv[0]); err == nil {
				maxFails = n
			}
		}
		if compensateFailCounts[phase] < maxFails {
			compensateFailCounts[phase]++
			return fmt.Errorf("compensate fault injected for %s (attempt %d)", phase, compensateFailCounts[phase])
		}
	}

	return nil
}

func (s *OtcServer) insertOtcInterbankTx(ctx context.Context, req *pb.OtcInterbankPrepareRequest, negID int64, vote string) {
	txType := "EXERCISE"
	if req.IsAccept {
		txType = "ACCEPT"
	}
	_, _ = s.DB.ExecContext(ctx, `
		INSERT INTO otc_interbank_tx
			(idem_routing_number, idem_key, tx_routing_number, tx_id,
			 negotiation_id, tx_type, stock_amount, status, cached_vote)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', $8)
		ON CONFLICT (idem_routing_number, idem_key) DO NOTHING`,
		req.IdemRoutingNumber, req.IdemKey, req.TxRoutingNumber, req.TxId,
		negID, txType, req.StockAmount, vote,
	)
}

// currency code → currency_id mapping (stable seed values from account-service)
var currencyIDMap = map[string]int64{
	"RSD": 1, "EUR": 2, "CHF": 3, "USD": 4,
	"GBP": 5, "JPY": 6, "CAD": 7, "AUD": 8,
}

type OtcServer struct {
	pb.UnimplementedOtcServiceServer
	DB           *sql.DB // otc_db
	EmployeeDB   *sql.DB // employee_db
	ClientDB     *sql.DB // client_db
	AccountDB    *sql.DB // account_db
	PortfolioDB  *sql.DB // portfolio_db
	SecuritiesDB *sql.DB // securities_db
	ExchangeDB   *sql.DB // exchange_db (daily_exchange_rates)
}

func getUserName(employeeDB, clientDB *sql.DB, userID int64, userType string) string {
	if userID == 0 {
		return ""
	}
	var name string
	var err error
	if userType == "EMPLOYEE" {
		err = employeeDB.QueryRow(`SELECT first_name || ' ' || last_name FROM employees WHERE id = $1`, userID).Scan(&name)
	} else {
		err = clientDB.QueryRow(`SELECT first_name || ' ' || last_name FROM clients WHERE id = $1`, userID).Scan(&name)
	}
	if err != nil {
		return ""
	}
	return name
}

// portfolioUserID returns the user_id as stored in portfolio_entry (EMPLOYEE → shared 0).
func portfolioUserID(userID int64, userType string) int64 {
	if userType == "EMPLOYEE" {
		return 0
	}
	return userID
}

// listingIDForTicker resolves ticker → listing.id in securities_db.
func listingIDForTicker(ctx context.Context, securitiesDB *sql.DB, ticker string) (int64, error) {
	var id int64
	err := securitiesDB.QueryRowContext(ctx, `SELECT id FROM listing WHERE ticker = $1`, ticker).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("listing not found for ticker %s", ticker)
	}
	return id, err
}

// findAccount returns the first account_id for owner with matching currency.
func findAccount(ctx context.Context, accountDB *sql.DB, ownerID int64, currencyID int64) (int64, error) {
	var id int64
	err := accountDB.QueryRowContext(ctx,
		`SELECT id FROM accounts WHERE owner_id = $1 AND currency_id = $2 AND status = 'ACTIVE' LIMIT 1`,
		ownerID, currencyID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no active account found for owner %d with currency_id %d", ownerID, currencyID)
	}
	return id, err
}

// currencyCodeMap is the inverse of currencyIDMap.
var currencyCodeMap = map[int64]string{
	1: "RSD", 2: "EUR", 3: "CHF", 4: "USD",
	5: "GBP", 6: "JPY", 7: "CAD", 8: "AUD",
}

// getAccountCurrencyID returns the currency_id of a given account.
func getAccountCurrencyID(ctx context.Context, accountDB *sql.DB, accountID int64) (int64, error) {
	var cid int64
	err := accountDB.QueryRowContext(ctx, `SELECT currency_id FROM accounts WHERE id = $1`, accountID).Scan(&cid)
	return cid, err
}

// convertAmount converts amount from fromCurrencyID to toCurrencyID using today's middle rates.
// Returns amount unchanged when currencies are equal.
func convertAmount(ctx context.Context, exchangeDB *sql.DB, amount float64, fromCurrencyID, toCurrencyID int64) (float64, error) {
	if fromCurrencyID == toCurrencyID {
		return amount, nil
	}
	fromCode, toCode := currencyCodeMap[fromCurrencyID], currencyCodeMap[toCurrencyID]

	getRate := func(code string) (float64, error) {
		if code == "RSD" {
			return 1.0, nil
		}
		var r float64
		err := exchangeDB.QueryRowContext(ctx,
			`SELECT middle_rate FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
			code,
		).Scan(&r)
		if err != nil {
			return 0, fmt.Errorf("no exchange rate for %s: %v", code, err)
		}
		return r, nil
	}

	fromRate, err := getRate(fromCode)
	if err != nil {
		return 0, err
	}
	toRate, err := getRate(toCode)
	if err != nil {
		return 0, err
	}
	// fromRate and toRate are both in RSD per 1 unit of currency
	return amount * fromRate / toRate, nil
}

func (s *OtcServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "otc-service ok"}, nil
}

func (s *OtcServer) recordOtcTax(userID int64, userType string, taxableAmount float64, currencyCode string, month, year int) {
	if userType == "EMPLOYEE" {
		return
	}
	fromID, ok := currencyIDMap[currencyCode]
	if !ok {
		return
	}
	amountRSD, err := convertAmount(context.Background(), s.ExchangeDB, taxableAmount, fromID, 1)
	if err != nil {
		return
	}
	taxRSD := amountRSD * 0.15
	if taxRSD == 0 {
		return
	}
	tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.PortfolioDB.ExecContext(tctx,
		`INSERT INTO tax_record (user_id, user_type, amount_rsd, month, year) VALUES ($1, $2, $3, $4, $5)`,
		userID, userType, taxRSD, month, year)
}

func (s *OtcServer) CreateNegotiation(ctx context.Context, req *pb.CreateNegotiationRequest) (*pb.NegotiationResponse, error) {
	if req.Ticker == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.PricePerStock <= 0 {
		return nil, status.Error(codes.InvalidArgument, "price_per_stock must be positive")
	}
	if req.SettlementDate == "" {
		return nil, status.Error(codes.InvalidArgument, "settlement date is required")
	}
	if settlementDate, parseErr := time.Parse("2006-01-02", req.SettlementDate); parseErr != nil || !settlementDate.After(time.Now().Truncate(24*time.Hour)) {
		return nil, status.Error(codes.InvalidArgument, "settlement date must be in the future")
	}

	isCrossBank := req.SellerRoutingNumber != 0 &&
		fmt.Sprintf("%d", req.SellerRoutingNumber) != os.Getenv("OWN_ROUTING_NUMBER")
	if isCrossBank && req.Premium <= 0 {
		return nil, status.Error(codes.InvalidArgument, "premium must be positive for cross-bank negotiations")
	}

	if isCrossBank {
		return s.createNegotiationCrossBank(ctx, req)
	}

	// Check seller has enough free shares before creating the negotiation.
	listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, req.Ticker)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unknown ticker: %s", req.Ticker)
	}
	// Free shares = portfolio amount minus saga-reserved shares (transient ExerciseContract reservations).
	var portfolioFree int64
	portfolioErr := s.PortfolioDB.QueryRowContext(ctx, `
		SELECT COALESCE(amount - reserved_amount, 0) FROM portfolio_entry
		WHERE user_id = $1 AND user_type = $2 AND listing_id = $3`,
		portfolioUserID(req.SellerId, req.SellerType), req.SellerType, listingID,
	).Scan(&portfolioFree)
	if portfolioErr != nil && portfolioErr != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to check seller portfolio: %v", portfolioErr)
	}
	var activeContractsSum int64
	_ = s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM otc_contracts
		WHERE ticker = $1 AND seller_id = $2 AND seller_type = $3 AND status = 'ACTIVE'`,
		req.Ticker, req.SellerId, req.SellerType,
	).Scan(&activeContractsSum)
	var pendingNegotiationsSum int64
	_ = s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM otc_negotiations
		WHERE ticker = $1 AND seller_id = $2 AND seller_type = $3
		  AND status IN ('PENDING_SELLER', 'PENDING_BUYER')`,
		req.Ticker, req.SellerId, req.SellerType,
	).Scan(&pendingNegotiationsSum)
	committed := activeContractsSum + pendingNegotiationsSum
	if portfolioFree < committed+int64(req.Amount) {
		return nil, status.Errorf(codes.InvalidArgument,
			"seller does not have enough free shares (available: %d, requested: %d)",
			portfolioFree-committed, req.Amount)
	}

	now := time.Now()

	var id int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type,
			 amount, price_per_stock, settlement_date, premium, currency,
			 last_modified, modified_by_id, modified_by_type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'PENDING_SELLER')
		RETURNING id`,
		req.Ticker, req.SellerId, req.SellerType, req.BuyerId, req.BuyerType,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium, req.Currency,
		now, req.BuyerId, req.BuyerType,
	).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create negotiation: %v", err)
	}

	return s.fetchNegotiationByID(ctx, id)
}

func (s *OtcServer) ListNegotiations(ctx context.Context, req *pb.ListNegotiationsRequest) (*pb.ListNegotiationsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id FROM otc_negotiations
		WHERE (seller_id = $1 AND seller_type = $2)
		   OR (buyer_id  = $1 AND buyer_type  = $2)
		   OR ($2 = 'EMPLOYEE' AND seller_id = 0 AND seller_type = 'EMPLOYEE')
		ORDER BY last_modified DESC`,
		req.CallerId, req.CallerType,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list negotiations: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan id: %v", err)
		}
		ids = append(ids, id)
	}

	negotiations := make([]*pb.NegotiationResponse, 0, len(ids))
	for _, id := range ids {
		neg, err := s.fetchNegotiationByID(ctx, id)
		if err != nil {
			return nil, err
		}
		negotiations = append(negotiations, neg)
	}
	return &pb.ListNegotiationsResponse{Negotiations: negotiations}, nil
}

func (s *OtcServer) GetNegotiation(ctx context.Context, req *pb.GetNegotiationRequest) (*pb.NegotiationResponse, error) {
	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) CounterOffer(ctx context.Context, req *pb.CounterOfferRequest) (*pb.NegotiationResponse, error) {
	if req.SettlementDate == "" {
		return nil, status.Error(codes.InvalidArgument, "settlement date is required")
	}
	if settlementDate, parseErr := time.Parse("2006-01-02", req.SettlementDate); parseErr != nil || !settlementDate.After(time.Now().Truncate(24*time.Hour)) {
		return nil, status.Error(codes.InvalidArgument, "settlement date must be in the future")
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status
		FROM otc_negotiations WHERE id = $1 FOR UPDATE`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	isSeller := req.CallerType == sellerType && (req.CallerId == sellerID || sellerID == 0)
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}
	if currentStatus == "PENDING_SELLER" && !isSeller {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for seller")
	}
	if currentStatus == "PENDING_BUYER" && !isBuyer {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for buyer")
	}
	if currentStatus != "PENDING_SELLER" && currentStatus != "PENDING_BUYER" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is in terminal state: %s", currentStatus)
	}

	newStatus := "PENDING_BUYER"
	if isBuyer {
		newStatus = "PENDING_SELLER"
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET amount = $1, price_per_stock = $2, settlement_date = $3, premium = $4,
		    last_modified = $5, modified_by_id = $6, modified_by_type = $7, status = $8
		WHERE id = $9`,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium,
		now, req.CallerId, req.CallerType, newStatus,
		req.NegotiationId,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update negotiation: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit counter offer: %v", err)
	}
	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) AcceptNegotiation(ctx context.Context, req *pb.AcceptNegotiationRequest) (*pb.NegotiationResponse, error) {
	// Lock the negotiation row to prevent concurrent accept/counter/reject.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	var ticker, currency string
	var amount int32
	var premium float64
	var settlementDate string
	var strikePrice float64
	err = tx.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status,
		       ticker, amount, premium, currency,
		       settlement_date::text, price_per_stock
		FROM otc_negotiations WHERE id = $1 FOR UPDATE`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus,
		&ticker, &amount, &premium, &currency, &settlementDate, &strikePrice)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	isSeller := req.CallerType == sellerType && (req.CallerId == sellerID || sellerID == 0)
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}
	if currentStatus == "PENDING_SELLER" && !isSeller {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for seller")
	}
	if currentStatus == "PENDING_BUYER" && !isBuyer {
		return nil, status.Error(codes.AlreadyExists, "not your turn: waiting for buyer")
	}
	if currentStatus != "PENDING_SELLER" && currentStatus != "PENDING_BUYER" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is in terminal state: %s", currentStatus)
	}

	// Cross-bank: our buyer accepts an offer from a seller on a partner bank.
	if sellerType == "INTERBANK" && isBuyer {
		return s.acceptCrossBank(ctx, req, tx, sellerID, buyerID, buyerType,
			ticker, amount, strikePrice, premium, currency, settlementDate, req.NegotiationId)
	}

	// --- Seller capacity check ---
	listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve ticker: %v", err)
	}
	var portfolioAmount int64
	portfolioErr := s.PortfolioDB.QueryRowContext(ctx, `
		SELECT COALESCE(amount - reserved_amount, 0) FROM portfolio_entry
		WHERE user_id = $1 AND user_type = $2 AND listing_id = $3`,
		portfolioUserID(sellerID, sellerType), sellerType, listingID,
	).Scan(&portfolioAmount)
	if portfolioErr != nil && portfolioErr != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to check seller portfolio: %v", portfolioErr)
	}
	// Include active contracts already committed (reserved_amount tracks transient saga reservations,
	// so we also check active contracts to prevent overselling across multiple accepted negotiations).
	var activeContractsSum int64
	_ = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM otc_contracts
		WHERE ticker = $1 AND seller_id = $2 AND seller_type = $3 AND status = 'ACTIVE'`,
		ticker, sellerID, sellerType,
	).Scan(&activeContractsSum)
	if portfolioAmount < activeContractsSum+int64(amount) {
		return nil, status.Error(codes.InvalidArgument, "Seller does not have enough free shares")
	}

	// Cross-bank: we are the seller, buyer is on a partner bank.
	// Commit ACCEPTED then run 2PC (NEW_TX) with the buyer's bank.
	if buyerType == "INTERBANK" && isSeller {
		now := time.Now()
		if _, err = tx.ExecContext(ctx,
			`UPDATE otc_negotiations
			 SET status = 'ACCEPTED', last_modified = $1, modified_by_id = $2, modified_by_type = $3
			 WHERE id = $4`,
			now, req.CallerId, req.CallerType, req.NegotiationId,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to accept negotiation: %v", err)
		}
		if err = tx.Commit(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
		}
		if twopcErr := s.executeInterbankAcceptOutgoing(ctx, req.NegotiationId); twopcErr != nil {
			_, _ = s.DB.ExecContext(context.Background(),
				`UPDATE otc_negotiations SET status='PENDING_SELLER' WHERE id = $1`, req.NegotiationId)
			return nil, status.Errorf(codes.Unavailable, "accept 2PC failed: %v", twopcErr)
		}
		return s.fetchNegotiationByID(ctx, req.NegotiationId)
	}

	// --- Buyer balance check ---
	currencyID, ok := currencyIDMap[currency]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", currency)
	}
	buyerAccountID := req.BuyerAccountId
	if buyerAccountID == 0 {
		buyerAccountID, err = findAccount(ctx, s.AccountDB, portfolioUserID(buyerID, buyerType), currencyID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find buyer account: %v", err)
		}
	}
	buyerCurrencyID, err := getAccountCurrencyID(ctx, s.AccountDB, buyerAccountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read buyer account currency: %v", err)
	}
	premiumToPay, err := convertAmount(ctx, s.ExchangeDB, premium, currencyID, buyerCurrencyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "currency conversion failed: %v", err)
	}
	var buyerBalance float64
	err = s.AccountDB.QueryRowContext(ctx,
		`SELECT available_balance FROM accounts WHERE id = $1`, buyerAccountID,
	).Scan(&buyerBalance)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check buyer balance: %v", err)
	}
	if buyerBalance < premiumToPay {
		return nil, status.Error(codes.InvalidArgument, "Insufficient funds for premium")
	}

	// --- Find seller account (prefer contract currency, fall back to any active account with conversion) ---
	sellerAccountID, err := findAccount(ctx, s.AccountDB, portfolioUserID(sellerID, sellerType), currencyID)
	sellerPremiumToCredit := premium
	if err != nil {
		var sellerCurrencyID int64
		fbErr := s.AccountDB.QueryRowContext(ctx,
			`SELECT id, currency_id FROM accounts WHERE owner_id = $1 AND status = 'ACTIVE' LIMIT 1`,
			portfolioUserID(sellerID, sellerType),
		).Scan(&sellerAccountID, &sellerCurrencyID)
		if fbErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to find seller account: %v", err)
		}
		sellerPremiumToCredit, err = convertAmount(ctx, s.ExchangeDB, premium, currencyID, sellerCurrencyID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert seller premium: %v", err)
		}
	}

	// --- Deduct buyer premium (with retry compensation on failure) ---
	if _, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
		premiumToPay, buyerAccountID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to deduct premium from buyer: %v", err)
	}

	// --- Credit seller premium ---
	if _, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		sellerPremiumToCredit, sellerAccountID,
	); err != nil {
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			premiumToPay, buyerAccountID)
		return nil, status.Errorf(codes.Internal, "failed to credit premium to seller: %v", err)
	}

	// --- Create contract and mark negotiation ACCEPTED (inside OTC tx) ---
	var contractID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO otc_contracts
			(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
			 ticker, amount, strike_price, premium, currency, settlement_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		req.NegotiationId, sellerID, sellerType, buyerID, buyerType,
		ticker, amount, strikePrice, premium, currency, settlementDate,
	).Scan(&contractID); err != nil {
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			premiumToPay, buyerAccountID)
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			premium, sellerAccountID)
		return nil, status.Errorf(codes.Internal, "failed to create contract: %v", err)
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET status = 'ACCEPTED', last_modified = $1, modified_by_id = $2, modified_by_type = $3
		WHERE id = $4`,
		now, req.CallerId, req.CallerType, req.NegotiationId,
	); err != nil {
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			premiumToPay, buyerAccountID)
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			premium, sellerAccountID)
		return nil, status.Errorf(codes.Internal, "failed to accept negotiation: %v", err)
	}

	if err = tx.Commit(); err != nil {
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			premiumToPay, buyerAccountID)
		retryExec(s.AccountDB,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			premium, sellerAccountID)
		return nil, status.Errorf(codes.Internal, "failed to commit accept: %v", err)
	}

	_ = contractID
	if sellerType != "INTERBANK" {
		s.recordOtcTax(sellerID, sellerType, premium, currency, int(now.Month()), now.Year())
	}
	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) RejectNegotiation(ctx context.Context, req *pb.RejectNegotiationRequest) (*pb.NegotiationResponse, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerID, buyerID int64
	var sellerType, buyerType, currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status
		FROM otc_negotiations WHERE id = $1 FOR UPDATE`, req.NegotiationId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &currentStatus)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	isSeller := req.CallerType == sellerType && (req.CallerId == sellerID || sellerID == 0)
	isBuyer := req.CallerId == buyerID && req.CallerType == buyerType
	if !isSeller && !isBuyer {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant in this negotiation")
	}
	if currentStatus == "ACCEPTED" || currentStatus == "REJECTED" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is already in terminal state: %s", currentStatus)
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET status = 'REJECTED', last_modified = $1, modified_by_id = $2, modified_by_type = $3
		WHERE id = $4`,
		now, req.CallerId, req.CallerType, req.NegotiationId,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reject negotiation: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit rejection: %v", err)
	}
	return s.fetchNegotiationByID(ctx, req.NegotiationId)
}

func (s *OtcServer) ListContracts(ctx context.Context, req *pb.ListContractsRequest) (*pb.ListContractsResponse, error) {
	query := `
		SELECT id, negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
		       ticker, amount, strike_price, premium, currency,
		       settlement_date::text, status, created_at
		FROM otc_contracts
		WHERE ((seller_id = $1 AND seller_type = $2)
		    OR  (buyer_id  = $1 AND buyer_type  = $2)
		    OR  ($2 = 'EMPLOYEE' AND seller_id = 0 AND seller_type = 'EMPLOYEE'))`
	args := []interface{}{req.CallerId, req.CallerType}

	if req.StatusFilter != "" {
		query += ` AND status = $3`
		args = append(args, req.StatusFilter)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list contracts: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var contracts []*pb.ContractResponse
	for rows.Next() {
		var c pb.ContractResponse
		var createdAt time.Time
		if err := rows.Scan(
			&c.Id, &c.NegotiationId, &c.SellerId, &c.SellerType, &c.BuyerId, &c.BuyerType,
			&c.Ticker, &c.Amount, &c.StrikePrice, &c.Premium, &c.Currency,
			&c.SettlementDate, &c.Status, &createdAt,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan contract: %v", err)
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.SellerName = getUserName(s.EmployeeDB, s.ClientDB, c.SellerId, c.SellerType)
		c.BuyerName = getUserName(s.EmployeeDB, s.ClientDB, c.BuyerId, c.BuyerType)
		c.Profit = s.calcContractProfit(c.Ticker, c.StrikePrice, int(c.Amount), c.Premium, c.Status)
		contracts = append(contracts, &c)
	}

	return &pb.ListContractsResponse{Contracts: contracts}, nil
}

func (s *OtcServer) calcContractProfit(ticker string, strikePrice float64, amount int, premium float64, contractStatus string) float64 {
	var marketPrice float64
	err := s.SecuritiesDB.QueryRow(`SELECT price FROM listing WHERE ticker = $1`, ticker).Scan(&marketPrice)
	if err != nil || marketPrice == 0 {
		return 0
	}
	if contractStatus == "EXERCISED" {
		return (marketPrice-strikePrice)*float64(amount) - premium
	}
	return (marketPrice - strikePrice) * float64(amount)
}

func (s *OtcServer) ExerciseContract(ctx context.Context, req *pb.ExerciseContractRequest) (*pb.ExerciseContractResponse, error) {
	// Idempotency: if saga for this contract already completed, return immediately.
	var existingSagaID int64
	var existingSagaStatus string
	if idErr := s.DB.QueryRowContext(ctx,
		`SELECT id, status FROM otc_saga WHERE contract_id=$1`,
		req.ContractId,
	).Scan(&existingSagaID, &existingSagaStatus); idErr == nil && existingSagaStatus == "Completed" {
		return &pb.ExerciseContractResponse{
			Status:     "EXERCISED",
			ExecutedAt: time.Now().Format(time.RFC3339),
			SagaId:     existingSagaID,
		}, nil
	}

	// Lock the contract row to prevent concurrent exercises of the same contract.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerID, buyerID int64
	var sellerType, buyerType, contractStatus, ticker, currency string
	var amount int32
	var strikePrice, premium float64
	var settlementDate time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT seller_id, seller_type, buyer_id, buyer_type, status,
		       ticker, amount, strike_price, premium, currency, settlement_date
		FROM otc_contracts WHERE id = $1 FOR UPDATE`, req.ContractId,
	).Scan(&sellerID, &sellerType, &buyerID, &buyerType, &contractStatus,
		&ticker, &amount, &strikePrice, &premium, &currency, &settlementDate)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "contract not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load contract: %v", err)
	}

	if req.CallerId != buyerID || req.CallerType != buyerType {
		return nil, status.Error(codes.PermissionDenied, "only the buyer can exercise the contract")
	}
	if contractStatus != "ACTIVE" {
		return nil, status.Errorf(codes.InvalidArgument, "Contract has expired or is already %s", contractStatus)
	}
	if time.Now().After(settlementDate.Add(24 * time.Hour)) {
		return nil, status.Error(codes.InvalidArgument, "Contract settlement date has passed")
	}

	listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ticker not found: %v", err)
	}

	// Cross-bank exercise: seller is on partner bank — delegate to 2PC flow.
	// (otc_saga row is NOT created for cross-bank; 2PC has its own tracking)
	if sellerType == "INTERBANK" {
		resp, rerr := s.exerciseCrossBank(ctx, req, tx, sellerID, buyerID, buyerType,
			amount, strikePrice, currency, ticker, settlementDate)
		if rerr == nil && buyerType != "EMPLOYEE" {
			var mktPrice float64
			if qErr := s.SecuritiesDB.QueryRowContext(ctx,
				`SELECT price FROM listing WHERE id = $1`, listingID).Scan(&mktPrice); qErr == nil {
				profit := (mktPrice-strikePrice)*float64(amount) - premium
				if profit > 0 {
					t := time.Now()
					s.recordOtcTax(buyerID, buyerType, profit, currency, int(t.Month()), t.Year())
				}
			}
		}
		return resp, rerr
	}

	totalCost := strikePrice * float64(amount)
	currencyID, ok := currencyIDMap[currency]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", currency)
	}

	buyerAccountID := req.BuyerAccountId
	if buyerAccountID == 0 {
		buyerAccountID, err = findAccount(ctx, s.AccountDB, portfolioUserID(buyerID, buyerType), currencyID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find buyer account: %v", err)
		}
	}
	buyerCurrencyID, err := getAccountCurrencyID(ctx, s.AccountDB, buyerAccountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read buyer account currency: %v", err)
	}
	totalCostToPay, err := convertAmount(ctx, s.ExchangeDB, totalCost, currencyID, buyerCurrencyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "currency conversion failed: %v", err)
	}

	// INSERT global saga tracker row (Running, step=0).
	// sagaID stays 0 on failure (e.g. mock error in tests) — sagaStatus becomes a no-op.
	var sagaID int64
	_ = s.DB.QueryRowContext(ctx,
		`INSERT INTO otc_saga (contract_id, status, current_step) VALUES ($1, 'Running', 0) RETURNING id`,
		req.ContractId,
	).Scan(&sagaID)

	// sagaLog writes a per-step record. Uses a fresh background context so it survives
	// a cancelled request context (e.g. during compensation after client disconnect).
	sagaLog := func(step int, stepStatus, errMsg string) {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_, _ = s.DB.ExecContext(logCtx,
			`INSERT INTO otc_saga_log (contract_id, step, status, error_msg) VALUES ($1, $2, $3, $4)`,
			req.ContractId, step, stepStatus, sql.NullString{String: errMsg, Valid: errMsg != ""},
		)
	}

	// sagaStatus updates the global saga tracker. No-op if sagaID is 0 (e.g. in unit tests).
	sagaStatus := func(newStatus string, step int) {
		if sagaID == 0 {
			return
		}
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_, _ = s.DB.ExecContext(logCtx,
			`UPDATE otc_saga SET status=$1, current_step=$2, updated_at=NOW() WHERE id=$3`,
			newStatus, step, sagaID,
		)
	}

	// compensateFailCounts tracks X-Saga-Compensate-Fail-Times retries per compensator.
	compensateFailCounts := make(map[string]int)

	sellerPortfolioID := portfolioUserID(sellerID, sellerType)

	// ── Step 1: Reserve buyer funds ──────────────────────────────────────────────
	if hookErr := sagaFaultHook(ctx, "F1", compensateFailCounts); hookErr != nil {
		sagaLog(1, "FAILED", hookErr.Error())
		sagaStatus("Compensating", 1)
		sagaStatus("Compensated", 1)
		return nil, status.Errorf(codes.Internal, "F1 fault injected: %v", hookErr)
	}
	result, err := s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET available_balance = available_balance - $1
		 WHERE id = $2 AND available_balance >= $1 AND balance >= $1`,
		totalCostToPay, buyerAccountID,
	)
	if err != nil {
		sagaLog(1, "FAILED", err.Error())
		sagaStatus("Compensating", 1)
		sagaStatus("Compensated", 1)
		return nil, status.Errorf(codes.Internal, "step 1 failed: %v", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		sagaLog(1, "FAILED", fmt.Sprintf("insufficient funds: need %.2f", totalCostToPay))
		sagaStatus("Compensating", 1)
		sagaStatus("Compensated", 1)
		return nil, status.Error(codes.InvalidArgument, "Insufficient funds")
	}
	sagaLog(1, "SUCCESS", "")
	sagaStatus("Running", 1)

	comp1 := func() {
		for {
			if hookErr := sagaFaultHook(ctx, "C1", compensateFailCounts); hookErr != nil {
				sagaLog(1, "COMP_FAILED", hookErr.Error())
				time.Sleep(200 * time.Millisecond)
				continue
			}
			retryExec(s.AccountDB,
				`UPDATE accounts SET available_balance = available_balance + $1 WHERE id = $2`,
				totalCostToPay, buyerAccountID)
			sagaLog(1, "COMPENSATED", "")
			break
		}
	}

	// ── Step 2: Reserve seller securities ────────────────────────────────────────
	if hookErr := sagaFaultHook(ctx, "F2", compensateFailCounts); hookErr != nil {
		sagaLog(2, "FAILED", hookErr.Error())
		sagaStatus("Compensating", 2)
		comp1()
		sagaStatus("Compensated", 2)
		return nil, status.Errorf(codes.Internal, "F2 fault injected: %v", hookErr)
	}
	result2, err := s.PortfolioDB.ExecContext(ctx, `
		UPDATE portfolio_entry
		SET reserved_amount = reserved_amount + $1
		WHERE user_id = $2 AND user_type = $3 AND listing_id = $4
		  AND (amount - reserved_amount) >= $1`,
		amount, sellerPortfolioID, sellerType, listingID,
	)
	if err != nil {
		sagaLog(2, "FAILED", err.Error())
		sagaStatus("Compensating", 2)
		comp1()
		sagaStatus("Compensated", 2)
		return nil, status.Errorf(codes.Internal, "step 2 failed: %v", err)
	}
	if rows, _ := result2.RowsAffected(); rows == 0 {
		sagaLog(2, "FAILED", "seller insufficient free holdings")
		sagaStatus("Compensating", 2)
		comp1()
		sagaStatus("Compensated", 2)
		return nil, status.Error(codes.InvalidArgument, "Seller does not have enough free shares")
	}
	sagaLog(2, "SUCCESS", "")
	sagaStatus("Running", 2)

	comp2 := func() {
		for {
			if hookErr := sagaFaultHook(ctx, "C2", compensateFailCounts); hookErr != nil {
				sagaLog(2, "COMP_FAILED", hookErr.Error())
				time.Sleep(200 * time.Millisecond)
				continue
			}
			retryExec(s.PortfolioDB,
				`UPDATE portfolio_entry SET reserved_amount = GREATEST(0, reserved_amount - $1)
				 WHERE user_id = $2 AND user_type = $3 AND listing_id = $4`,
				amount, sellerPortfolioID, sellerType, listingID)
			sagaLog(2, "COMPENSATED", "")
			break
		}
	}

	// ── Step 3: Transfer funds ────────────────────────────────────────────────────
	sellerAccountID, err := findAccount(ctx, s.AccountDB, portfolioUserID(sellerID, sellerType), currencyID)
	sellerTotalCostToReceive := totalCost
	if err != nil {
		// No account in contract currency — fall back to any active account with conversion.
		var sellerCurrencyID int64
		fbErr := s.AccountDB.QueryRowContext(ctx,
			`SELECT id, currency_id FROM accounts WHERE owner_id = $1 AND status = 'ACTIVE' LIMIT 1`,
			portfolioUserID(sellerID, sellerType),
		).Scan(&sellerAccountID, &sellerCurrencyID)
		if fbErr != nil {
			sagaLog(3, "FAILED", err.Error())
			sagaStatus("Compensating", 3)
			comp2()
			comp1()
			sagaStatus("Compensated", 3)
			return nil, status.Errorf(codes.Internal, "step 3 failed finding seller account: %v", err)
		}
		sellerTotalCostToReceive, err = convertAmount(ctx, s.ExchangeDB, totalCost, currencyID, sellerCurrencyID)
		if err != nil {
			sagaLog(3, "FAILED", err.Error())
			sagaStatus("Compensating", 3)
			comp2()
			comp1()
			sagaStatus("Compensated", 3)
			return nil, status.Errorf(codes.Internal, "step 3 failed converting seller amount: %v", err)
		}
	}
	if hookErr := sagaFaultHook(ctx, "F3", compensateFailCounts); hookErr != nil {
		sagaLog(3, "FAILED", hookErr.Error())
		sagaStatus("Compensating", 3)
		comp2()
		comp1()
		sagaStatus("Compensated", 3)
		return nil, status.Errorf(codes.Internal, "F3 fault injected: %v", hookErr)
	}
	if _, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE id = $2`,
		totalCostToPay, buyerAccountID,
	); err != nil {
		sagaLog(3, "FAILED", err.Error())
		sagaStatus("Compensating", 3)
		comp2()
		comp1()
		sagaStatus("Compensated", 3)
		return nil, status.Errorf(codes.Internal, "step 3 failed debit buyer: %v", err)
	}
	if _, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		sellerTotalCostToReceive, sellerAccountID,
	); err != nil {
		retryExec(s.AccountDB, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, totalCostToPay, buyerAccountID)
		sagaLog(3, "FAILED", err.Error())
		sagaStatus("Compensating", 3)
		comp2()
		comp1()
		sagaStatus("Compensated", 3)
		return nil, status.Errorf(codes.Internal, "step 3 failed credit seller: %v", err)
	}
	sagaLog(3, "SUCCESS", "")
	sagaStatus("Running", 3)

	comp3 := func() {
		for {
			if hookErr := sagaFaultHook(ctx, "C3", compensateFailCounts); hookErr != nil {
				sagaLog(3, "COMP_FAILED", hookErr.Error())
				time.Sleep(200 * time.Millisecond)
				continue
			}
			retryExec(s.AccountDB, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, totalCostToPay, buyerAccountID)
			retryExec(s.AccountDB, `UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`, sellerTotalCostToReceive, sellerAccountID)
			sagaLog(3, "COMPENSATED", "")
			break
		}
	}

	// ── Step 4: Transfer ownership ────────────────────────────────────────────────
	if hookErr := sagaFaultHook(ctx, "F4", compensateFailCounts); hookErr != nil {
		sagaLog(4, "FAILED", hookErr.Error())
		sagaStatus("Compensating", 4)
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 4)
		return nil, status.Errorf(codes.Internal, "F4 fault injected: %v", hookErr)
	}
	if _, err = s.PortfolioDB.ExecContext(ctx, `
		UPDATE portfolio_entry
		SET amount          = amount - $1,
		    reserved_amount = GREATEST(0, reserved_amount - $1),
		    public_amount   = GREATEST(0, LEAST(public_amount, amount - $1)),
		    last_modified   = NOW()
		WHERE user_id = $2 AND user_type = $3 AND listing_id = $4`,
		amount, sellerPortfolioID, sellerType, listingID,
	); err != nil {
		sagaLog(4, "FAILED", err.Error())
		sagaStatus("Compensating", 4)
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 4)
		return nil, status.Errorf(codes.Internal, "step 4 failed deduct seller portfolio: %v", err)
	}
	_, _ = s.PortfolioDB.ExecContext(ctx, `
		DELETE FROM portfolio_entry WHERE user_id=$1 AND user_type=$2 AND listing_id=$3 AND amount <= 0`,
		sellerPortfolioID, sellerType, listingID,
	)

	buyerPortfolioID := portfolioUserID(buyerID, buyerType)
	if _, err = s.PortfolioDB.ExecContext(ctx, `
		INSERT INTO portfolio_entry (user_id, user_type, listing_id, amount, buy_price, account_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, user_type, listing_id) DO UPDATE
		SET amount = portfolio_entry.amount + EXCLUDED.amount, last_modified = NOW()`,
		buyerPortfolioID, buyerType, listingID, amount, strikePrice, buyerAccountID,
	); err != nil {
		sagaLog(4, "FAILED", "buyer upsert failed: "+err.Error())
		retryExec(s.PortfolioDB,
			`UPDATE portfolio_entry SET amount = amount + $1, reserved_amount = reserved_amount + $1, last_modified = NOW()
			 WHERE user_id = $2 AND user_type = $3 AND listing_id = $4`,
			amount, sellerPortfolioID, sellerType, listingID,
		)
		sagaStatus("Compensating", 4)
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 4)
		return nil, status.Errorf(codes.Internal, "step 4 failed upsert buyer portfolio: %v", err)
	}
	sagaLog(4, "SUCCESS", "")
	sagaStatus("Running", 4)

	comp4 := func() {
		for {
			if hookErr := sagaFaultHook(ctx, "C4", compensateFailCounts); hookErr != nil {
				sagaLog(4, "COMP_FAILED", hookErr.Error())
				time.Sleep(200 * time.Millisecond)
				continue
			}
			// Restore seller portfolio. F4 may have deleted the row (amount hit 0),
			// so UPSERT handles both the re-insert and the increment cases.
			retryExec(s.PortfolioDB, `
				INSERT INTO portfolio_entry
					(user_id, user_type, listing_id, amount, buy_price, account_id, reserved_amount)
				VALUES ($2, $3, $4, $1, 0, $5, $1)
				ON CONFLICT (user_id, user_type, listing_id) DO UPDATE
				SET amount          = portfolio_entry.amount + EXCLUDED.amount,
				    reserved_amount = portfolio_entry.reserved_amount + EXCLUDED.reserved_amount,
				    last_modified   = NOW()`,
				amount, sellerPortfolioID, sellerType, listingID, sellerAccountID)
			// Remove buyer shares acquired in F4, clean up row if empty.
			retryExec(s.PortfolioDB, `
				UPDATE portfolio_entry SET amount = amount - $1, last_modified = NOW()
				WHERE user_id=$2 AND user_type=$3 AND listing_id=$4`,
				amount, buyerPortfolioID, buyerType, listingID)
			_, _ = s.PortfolioDB.Exec(`
				DELETE FROM portfolio_entry
				WHERE user_id=$1 AND user_type=$2 AND listing_id=$3 AND amount <= 0`,
				buyerPortfolioID, buyerType, listingID)
			sagaLog(4, "COMPENSATED", "")
			break
		}
	}

	// ── Step 5: Verify and mark contract EXERCISED ────────────────────────────────
	if hookErr := sagaFaultHook(ctx, "F5", compensateFailCounts); hookErr != nil {
		sagaLog(5, "FAILED", hookErr.Error())
		sagaStatus("Compensating", 5)
		comp4()
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 5)
		return nil, status.Errorf(codes.Internal, "F5 fault injected: %v", hookErr)
	}

	var buyerHolding int64
	checkErr := s.PortfolioDB.QueryRowContext(ctx, `
		SELECT COALESCE(amount, 0) FROM portfolio_entry
		WHERE user_id = $1 AND user_type = $2 AND listing_id = $3`,
		buyerPortfolioID, buyerType, listingID,
	).Scan(&buyerHolding)
	if checkErr != nil || buyerHolding < int64(amount) {
		sagaLog(5, "FAILED", "double check failed: buyer portfolio inconsistent")
		sagaStatus("Compensating", 5)
		comp4()
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 5)
		return nil, status.Error(codes.Internal, "step 5 double check failed, saga rolled back")
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx,
		`UPDATE otc_contracts SET status = 'EXERCISED' WHERE id = $1`, req.ContractId,
	); err != nil {
		sagaLog(5, "FAILED", err.Error())
		sagaStatus("Compensating", 5)
		comp4()
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 5)
		return nil, status.Errorf(codes.Internal, "step 5 failed: %v", err)
	}
	if err = tx.Commit(); err != nil {
		sagaLog(5, "FAILED", "commit failed: "+err.Error())
		sagaStatus("Compensating", 5)
		comp4()
		comp3()
		comp2()
		comp1()
		sagaStatus("Compensated", 5)
		return nil, status.Errorf(codes.Internal, "step 5 commit failed: %v", err)
	}
	sagaLog(5, "SUCCESS", "")
	sagaStatus("Completed", 5)

	if buyerType != "EMPLOYEE" {
		var mktPrice float64
		if qErr := s.SecuritiesDB.QueryRowContext(ctx,
			`SELECT price FROM listing WHERE id = $1`, listingID).Scan(&mktPrice); qErr == nil {
			profit := (mktPrice-strikePrice)*float64(amount) - premium
			if profit > 0 {
				s.recordOtcTax(buyerID, buyerType, profit, currency, int(now.Month()), now.Year())
			}
		}
	}

	return &pb.ExerciseContractResponse{
		Status:     "EXERCISED",
		ExecutedAt: now.Format(time.RFC3339),
		SagaId:     sagaID,
	}, nil
}

func (s *OtcServer) GetMarket(ctx context.Context, req *pb.GetMarketRequest) (*pb.GetMarketResponse, error) {
	var query string
	var args []interface{}

	if req.CallerType == "CLIENT" {
		query = `
			SELECT user_id, user_type, listing_id, public_amount, last_modified
			FROM portfolio_entry
			WHERE user_type = 'CLIENT' AND is_public = true AND public_amount > 0
			  AND user_id != $1`
		args = []interface{}{req.CallerId}
	} else {
		// SUPERVISOR sees bank public stocks (user_id=0, user_type='EMPLOYEE')
		query = `
			SELECT user_id, user_type, listing_id, public_amount, last_modified
			FROM portfolio_entry
			WHERE user_type = 'EMPLOYEE' AND user_id = 0 AND is_public = true AND public_amount > 0`
		args = []interface{}{}
	}

	rows, err := s.PortfolioDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query market: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var items []*pb.MarketItem
	for rows.Next() {
		var ownerID int64
		var ownerType string
		var listingID int64
		var publicAmount int32
		var lastModified time.Time

		if err := rows.Scan(&ownerID, &ownerType, &listingID, &publicAmount, &lastModified); err != nil {
			return nil, status.Errorf(codes.Internal, "scan market row: %v", err)
		}

		var ticker, name, currency string
		var price float64
		secErr := s.SecuritiesDB.QueryRowContext(ctx,
			`SELECT l.ticker, l.name, l.price, se.currency
			 FROM listing l
			 JOIN stock_exchanges se ON l.exchange_id = se.id
			 WHERE l.id = $1`, listingID,
		).Scan(&ticker, &name, &price, &currency)
		if secErr != nil {
			continue
		}

		// Compute free (uncommitted) amount: subtract pending negotiations and active contracts.
		var pendingSum int64
		_ = s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM otc_negotiations
			WHERE ticker = $1 AND seller_id = $2 AND seller_type = $3
			  AND status IN ('PENDING_SELLER', 'PENDING_BUYER')`,
			ticker, ownerID, ownerType,
		).Scan(&pendingSum)
		var contractSum int64
		_ = s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM otc_contracts
			WHERE ticker = $1 AND seller_id = $2 AND seller_type = $3 AND status = 'ACTIVE'`,
			ticker, ownerID, ownerType,
		).Scan(&contractSum)
		freeAmount := int64(publicAmount) - pendingSum - contractSum
		if freeAmount <= 0 {
			continue
		}

		ownerName := "EXBanka"
		if ownerType == "CLIENT" && ownerID != 0 {
			ownerName = getUserName(s.EmployeeDB, s.ClientDB, ownerID, ownerType)
		}

		items = append(items, &pb.MarketItem{
			Ticker:        ticker,
			Name:          name,
			Amount:        int32(freeAmount),
			PricePerStock: price,
			Currency:      currency,
			LastUpdated:   lastModified.Format(time.RFC3339),
			OwnerName:     ownerName,
			OwnerBank:     "EXBanka",
			OwnerId:       ownerID,
			OwnerType:     ownerType,
		})
	}

	return &pb.GetMarketResponse{Items: items}, nil
}

// ── Cross-bank (interbank) negotiation handlers ──────────────────────────────

// fetchInterbankNegotiationByID reads a negotiation row and returns the cross-bank response shape.
func (s *OtcServer) fetchInterbankNegotiationByID(ctx context.Context, id int64) (*pb.InterbankNegotiationResponse, error) {
	var r pb.InterbankNegotiationResponse
	var ticker, currency, settlementDate, currentStatus string
	var amount int32
	var pricePerStock, premium float64
	var buyerRouting, sellerRouting, creatorRouting sql.NullInt32
	var buyerExtID, sellerExtID, creatorExtID sql.NullString

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, ticker, amount, price_per_stock, settlement_date::text, premium, currency, status,
		       buyer_routing_number, buyer_external_id,
		       seller_routing_number, seller_external_id,
		       creator_routing_number, creator_external_id
		FROM otc_negotiations WHERE id = $1`, id,
	).Scan(
		&r.LocalId, &ticker, &amount, &pricePerStock, &settlementDate, &premium, &currency, &currentStatus,
		&buyerRouting, &buyerExtID,
		&sellerRouting, &sellerExtID,
		&creatorRouting, &creatorExtID,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch negotiation: %v", err)
	}

	r.Ticker = ticker
	r.Amount = amount
	r.PricePerUnit = pricePerStock
	r.PriceCurrency = currency
	r.SettlementDate = settlementDate
	r.Premium = premium
	r.IsOngoing = currentStatus == "PENDING_SELLER" || currentStatus == "PENDING_BUYER"
	if buyerRouting.Valid {
		r.BuyerRoutingNumber = buyerRouting.Int32
	}
	if buyerExtID.Valid {
		r.BuyerExternalId = buyerExtID.String
	}
	if sellerRouting.Valid {
		r.SellerRoutingNumber = sellerRouting.Int32
	}
	if sellerExtID.Valid {
		r.SellerExternalId = sellerExtID.String
	}
	if creatorRouting.Valid {
		r.CreatorRoutingNumber = creatorRouting.Int32
	}
	if creatorExtID.Valid {
		r.CreatorExternalId = creatorExtID.String
	}
	return &r, nil
}

// lookupInterbankNegotiation resolves a cross-bank negotiation by path params.
// Protocol path: {sellerRn}/{sellerLocalId} — sellerRn is our routing, sellerLocalId is our DB id.
// Falls back to creator-key lookup for backward compatibility.
func (s *OtcServer) lookupInterbankNegotiation(ctx context.Context, routingNumber int32, externalID string) (int64, string, error) {
	var localID int64
	var currentStatus string

	// Primary: {sellerRn}/{ourLocalId} — look up by local primary-key id.
	if localIDInt, parseErr := strconv.ParseInt(externalID, 10, 64); parseErr == nil {
		err := s.DB.QueryRowContext(ctx,
			`SELECT id, status FROM otc_negotiations WHERE id = $1`, localIDInt,
		).Scan(&localID, &currentStatus)
		if err == nil {
			return localID, currentStatus, nil
		}
	}

	// Fallback: {creatorRn}/{creatorExtId} — backward compat for callers that pass buyer routing/id.
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, status FROM otc_negotiations
		WHERE creator_routing_number = $1 AND creator_external_id = $2`,
		routingNumber, externalID,
	).Scan(&localID, &currentStatus)
	if err == sql.ErrNoRows {
		return 0, "", status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return 0, "", status.Errorf(codes.Internal, "failed to look up negotiation: %v", err)
	}
	return localID, currentStatus, nil
}

func (s *OtcServer) CreateInterbankNegotiation(ctx context.Context, req *pb.CreateInterbankNegotiationRequest) (*pb.InterbankNegotiationResponse, error) {
	if req.Ticker == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.SettlementDate == "" {
		return nil, status.Error(codes.InvalidArgument, "settlement_date is required")
	}

	sellerType := req.SellerType
	if sellerType == "" {
		sellerType = "CLIENT"
	}

	// seller_external_id is our local user's numeric ID as a string
	sellerID, parseErr := strconv.ParseInt(req.SellerExternalId, 10, 64)
	if parseErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "seller_external_id must be a numeric local user ID")
	}

	// Read buyer account number from gRPC metadata (forwarded by api-gateway from Banka 4's request body).
	var buyerAccountNum string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("buyer-account-number"); len(v) > 0 {
			buyerAccountNum = v[0]
		}
	}

	// Idempotency: return existing row if this creator key already exists
	if existingID, _, err := s.lookupInterbankNegotiation(ctx, req.CreatorRoutingNumber, req.CreatorExternalId); err == nil {
		return s.fetchInterbankNegotiationByID(ctx, existingID)
	}

	now := time.Now()
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type,
			 amount, price_per_stock, settlement_date, premium, currency,
			 last_modified, modified_by_id, modified_by_type, status,
			 buyer_routing_number, buyer_external_id,
			 seller_routing_number, seller_external_id,
			 creator_routing_number, creator_external_id,
			 buyer_account_number)
		VALUES ($1, $2, $3, 0, 'INTERBANK', $4, $5, $6, $7, $8, $9, 0, 'INTERBANK', 'PENDING_SELLER',
		        $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`,
		req.Ticker, sellerID, sellerType,
		req.Amount, req.PricePerUnit, req.SettlementDate, req.Premium, req.PriceCurrency,
		now,
		req.BuyerRoutingNumber, req.BuyerExternalId,
		req.SellerRoutingNumber, req.SellerExternalId,
		req.CreatorRoutingNumber, req.CreatorExternalId,
		buyerAccountNum,
	).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create interbank negotiation: %v", err)
	}
	return s.fetchInterbankNegotiationByID(ctx, id)
}

func (s *OtcServer) InterbankCounterOffer(ctx context.Context, req *pb.InterbankCounterOfferRequest) (*pb.InterbankNegotiationResponse, error) {
	localID, currentStatus, err := s.lookupInterbankNegotiation(ctx, req.RoutingNumber, req.ExternalId)
	if err != nil {
		return nil, err
	}
	if currentStatus != "PENDING_BUYER" && currentStatus != "PENDING_SELLER" {
		return nil, status.Errorf(codes.FailedPrecondition, "negotiation is in terminal state: %s", currentStatus)
	}
	// The partner bank is always the buyer for incoming negotiations.
	// Counter-offer is only allowed when it is the buyer's turn.
	if currentStatus != "PENDING_BUYER" {
		return nil, status.Error(codes.FailedPrecondition, "not your turn")
	}

	now := time.Now()
	if _, err = s.DB.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET amount = $1, price_per_stock = $2, settlement_date = $3, premium = $4,
		    last_modified = $5, modified_by_id = 0, modified_by_type = 'INTERBANK', status = 'PENDING_SELLER'
		WHERE id = $6`,
		req.Amount, req.PricePerUnit, req.SettlementDate, req.Premium,
		now, localID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update negotiation: %v", err)
	}
	return s.fetchInterbankNegotiationByID(ctx, localID)
}

func (s *OtcServer) InterbankGetNegotiation(ctx context.Context, req *pb.InterbankNegotiationIdRequest) (*pb.InterbankNegotiationResponse, error) {
	localID, _, err := s.lookupInterbankNegotiation(ctx, req.RoutingNumber, req.ExternalId)
	if err != nil {
		return nil, err
	}
	return s.fetchInterbankNegotiationByID(ctx, localID)
}

func (s *OtcServer) InterbankDeleteNegotiation(ctx context.Context, req *pb.InterbankNegotiationIdRequest) (*pb.OtcEmptyResponse, error) {
	localID, currentStatus, err := s.lookupInterbankNegotiation(ctx, req.RoutingNumber, req.ExternalId)
	if err != nil {
		return nil, err
	}
	if currentStatus == "ACCEPTED" {
		return nil, status.Error(codes.FailedPrecondition, "cannot cancel an already accepted negotiation")
	}
	if currentStatus == "REJECTED" {
		return &pb.OtcEmptyResponse{}, nil // idempotent
	}

	if _, err = s.DB.ExecContext(ctx, `
		UPDATE otc_negotiations SET status = 'REJECTED', last_modified = NOW()
		WHERE id = $1`, localID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel negotiation: %v", err)
	}
	return &pb.OtcEmptyResponse{}, nil
}

func (s *OtcServer) InterbankAcceptNegotiation(ctx context.Context, req *pb.InterbankNegotiationIdRequest) (*pb.OtcEmptyResponse, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var localID int64
	var currentStatus, ticker, currency, settlementDate string
	var amount int32
	var premium, strikePrice float64
	var sellerID int64
	var sellerType string
	// Protocol path: {sellerRn}/{sellerLocalId} — look up by local id first.
	localIDInt, parseErr := strconv.ParseInt(req.ExternalId, 10, 64)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid negotiation id")
	}
	err = tx.QueryRowContext(ctx, `
		SELECT id, status, ticker, currency, settlement_date::text, amount, premium, price_per_stock,
		       seller_id, seller_type
		FROM otc_negotiations
		WHERE id = $1 FOR UPDATE`,
		localIDInt,
	).Scan(&localID, &currentStatus, &ticker, &currency, &settlementDate, &amount, &premium, &strikePrice,
		&sellerID, &sellerType)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	if currentStatus == "ACCEPTED" {
		if err = tx.Commit(); err == nil {
			return &pb.OtcEmptyResponse{}, nil // idempotent
		}
	}
	// The partner bank is the buyer; they can accept only when it is their turn.
	if currentStatus != "PENDING_BUYER" {
		return nil, status.Error(codes.FailedPrecondition, "not your turn to accept")
	}

	now := time.Now()
	// Contract will be created by CommitOtcInterbank after 2PC with the buyer's bank.
	if _, err = tx.ExecContext(ctx, `
		UPDATE otc_negotiations
		SET status = 'ACCEPTED', last_modified = $1, modified_by_id = 0, modified_by_type = 'INTERBANK'
		WHERE id = $2`, now, localID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to accept negotiation: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}

	if twopcErr := s.executeInterbankAcceptOutgoing(ctx, localID); twopcErr != nil {
		_, _ = s.DB.ExecContext(context.Background(),
			`UPDATE otc_negotiations SET status='PENDING_BUYER' WHERE id = $1`, localID)
		return nil, status.Errorf(codes.Unavailable, "accept 2PC failed: %v", twopcErr)
	}
	return &pb.OtcEmptyResponse{}, nil
}

func (s *OtcServer) fetchNegotiationByID(ctx context.Context, id int64) (*pb.NegotiationResponse, error) {
	var n pb.NegotiationResponse
	var lastModified time.Time
	var settlementDate string
	var modifiedByID sql.NullInt64
	var modifiedByType sql.NullString

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, ticker, seller_id, seller_type, buyer_id, buyer_type,
		       amount, price_per_stock, settlement_date::text, premium, currency,
		       last_modified, modified_by_id, modified_by_type, status
		FROM otc_negotiations WHERE id = $1`, id,
	).Scan(
		&n.Id, &n.Ticker, &n.SellerId, &n.SellerType, &n.BuyerId, &n.BuyerType,
		&n.Amount, &n.PricePerStock, &settlementDate, &n.Premium, &n.Currency,
		&lastModified, &modifiedByID, &modifiedByType, &n.Status,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "negotiation not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch negotiation: %v", err)
	}

	n.SettlementDate = settlementDate
	n.LastModified = lastModified.Format(time.RFC3339)
	n.SellerName = getUserName(s.EmployeeDB, s.ClientDB, n.SellerId, n.SellerType)
	n.BuyerName = getUserName(s.EmployeeDB, s.ClientDB, n.BuyerId, n.BuyerType)

	if modifiedByID.Valid {
		n.ModifiedById = modifiedByID.Int64
	}
	if modifiedByType.Valid {
		n.ModifiedByType = modifiedByType.String
		n.ModifiedByName = getUserName(s.EmployeeDB, s.ClientDB, n.ModifiedById, n.ModifiedByType)
	}

	return &n, nil
}

func (s *OtcServer) PrepareOtcInterbank(ctx context.Context, req *pb.OtcInterbankPrepareRequest) (*pb.OtcInterbankVoteResponse, error) {
	// Idempotency: return cached vote if we've already seen this key.
	var cachedVote string
	err := s.DB.QueryRowContext(ctx,
		`SELECT cached_vote FROM otc_interbank_tx
		 WHERE idem_routing_number = $1 AND idem_key = $2`,
		req.IdemRoutingNumber, req.IdemKey,
	).Scan(&cachedVote)
	if err == nil {
		return &pb.OtcInterbankVoteResponse{Vote: cachedVote}, nil
	}
	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "idempotency check failed: %v", err)
	}

	// BUYER-SIDE accept: Banka 4 is seller; req.NegotiationId is their local ID, not ours.
	// Must be handled before the general negotiation load (which uses our local IDs).
	if req.IsAccept {
		partnerRouting := os.Getenv("PARTNER_ROUTING_NUMBER")
		if req.IdemRoutingNumber == partnerRouting {
			partnerRoutingInt, _ := strconv.ParseInt(partnerRouting, 10, 64)
			var localNegID int64
			if lookupErr := s.DB.QueryRowContext(ctx,
				`SELECT id FROM otc_negotiations WHERE partner_negotiation_id = $1 AND seller_routing_number = $2`,
				req.NegotiationId, partnerRoutingInt,
			).Scan(&localNegID); lookupErr == sql.ErrNoRows {
				s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
				return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_NEGOTIATION_NOT_FOUND"}, nil
			} else if lookupErr != nil {
				return nil, status.Errorf(codes.Internal, "lookup failed: %v", lookupErr)
			}

			var buyerAcctNum, buyerNegStatus string
			var prem float64
			if err = s.DB.QueryRowContext(ctx,
				`SELECT buyer_account_number, premium, status FROM otc_negotiations WHERE id = $1`,
				localNegID,
			).Scan(&buyerAcctNum, &prem, &buyerNegStatus); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
			}
			if buyerNegStatus != "ACCEPTED" {
				s.insertOtcInterbankTx(ctx, req, localNegID, "NO")
				return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_USED_OR_EXPIRED"}, nil
			}
			var availBal float64
			_ = s.AccountDB.QueryRowContext(ctx,
				`SELECT available_balance FROM accounts WHERE account_number = $1`, buyerAcctNum,
			).Scan(&availBal)
			if availBal < prem {
				s.insertOtcInterbankTx(ctx, req, localNegID, "NO")
				return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "INSUFFICIENT_FUNDS"}, nil
			}
			s.insertOtcInterbankTx(ctx, req, localNegID, "YES")
			return &pb.OtcInterbankVoteResponse{Vote: "YES"}, nil
		}
	}

	// Load negotiation (by our local ID; applies to seller-side accept and all exercise requests).
	var negStatus, ticker, sellerType string
	var sellerID int64
	var negAmount int32
	err = s.DB.QueryRowContext(ctx,
		`SELECT status, ticker, seller_id, seller_type, amount
		 FROM otc_negotiations WHERE id = $1`, req.NegotiationId,
	).Scan(&negStatus, &ticker, &sellerID, &sellerType, &negAmount)
	if err == sql.ErrNoRows {
		s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
		return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_NEGOTIATION_NOT_FOUND"}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
	}

	if req.IsAccept {
		// SELLER-SIDE accept: partner bank is buyer, req.NegotiationId is our local ID.
		if negStatus != "ACCEPTED" {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_USED_OR_EXPIRED"}, nil
		}
		var contractExists bool
		_ = s.DB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM otc_contracts WHERE negotiation_id = $1)`,
			req.NegotiationId,
		).Scan(&contractExists)
		if contractExists {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_USED_OR_EXPIRED"}, nil
		}
		if req.StockAmount != negAmount {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_AMOUNT_INCORRECT"}, nil
		}
		listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ticker not found: %v", err)
		}
		var freeShares int64
		_ = s.PortfolioDB.QueryRowContext(ctx,
			`SELECT COALESCE(amount - reserved_amount, 0) FROM portfolio_entry
			 WHERE user_id = $1 AND user_type = $2 AND listing_id = $3`,
			portfolioUserID(sellerID, sellerType), sellerType, listingID,
		).Scan(&freeShares)
		if freeShares < int64(req.StockAmount) {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_AMOUNT_INCORRECT"}, nil
		}
	} else {
		var contractStatus string
		var contractAmount int32
		var settlementDate time.Time
		err = s.DB.QueryRowContext(ctx,
			`SELECT c.status, c.amount, n.settlement_date
			 FROM otc_contracts c JOIN otc_negotiations n ON n.id = c.negotiation_id
			 WHERE c.negotiation_id = $1`,
			req.NegotiationId,
		).Scan(&contractStatus, &contractAmount, &settlementDate)
		if err == sql.ErrNoRows {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_NEGOTIATION_NOT_FOUND"}, nil
		}
		if contractStatus != "ACTIVE" {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_USED_OR_EXPIRED"}, nil
		}
		if time.Now().After(settlementDate.Add(24 * time.Hour)) {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_USED_OR_EXPIRED"}, nil
		}
		if contractAmount != req.StockAmount {
			s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "NO")
			return &pb.OtcInterbankVoteResponse{Vote: "NO", Reason: "OPTION_AMOUNT_INCORRECT"}, nil
		}
	}

	s.insertOtcInterbankTx(ctx, req, req.NegotiationId, "YES")
	return &pb.OtcInterbankVoteResponse{Vote: "YES"}, nil
}

func (s *OtcServer) CommitOtcInterbank(ctx context.Context, req *pb.OtcInterbankTxRequest) (*pb.OtcEmptyResponse, error) {
	var id int64
	var txStatus, txType string
	var negotiationID int64
	var stockAmount int32
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, status, tx_type, negotiation_id, stock_amount
		 FROM otc_interbank_tx WHERE tx_routing_number = $1 AND tx_id = $2`,
		req.TxRoutingNumber, req.TxId,
	).Scan(&id, &txStatus, &txType, &negotiationID, &stockAmount)
	if err == sql.ErrNoRows {
		return &pb.OtcEmptyResponse{}, nil // not an OTC transaction — ignore
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load otc_interbank_tx: %v", err)
	}
	if txStatus == "COMMITTED" {
		return &pb.OtcEmptyResponse{}, nil // idempotent
	}
	if txStatus == "ROLLED_BACK" {
		return nil, status.Error(codes.FailedPrecondition, "transaction already rolled back")
	}

	if txType == "ACCEPT" {
		var ticker, currency, settlementDate, sellerType string
		var sellerID int64
		var amount int32
		var strikePrice, premium float64
		err = s.DB.QueryRowContext(ctx, `
			SELECT ticker, currency, settlement_date::text, seller_id, seller_type,
			       amount, price_per_stock, premium
			FROM otc_negotiations WHERE id = $1`, negotiationID,
		).Scan(&ticker, &currency, &settlementDate, &sellerID, &sellerType,
			&amount, &strikePrice, &premium)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load negotiation: %v", err)
		}

		if sellerType == "INTERBANK" {
			// BUYER-SIDE accept: we are the buyer, Banka 4 is seller.
			var buyerAcctNum string
			var buyerID int64
			var buyerType string
			if err = s.DB.QueryRowContext(ctx,
				`SELECT buyer_account_number, buyer_id, buyer_type FROM otc_negotiations WHERE id = $1`,
				negotiationID,
			).Scan(&buyerAcctNum, &buyerID, &buyerType); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to load buyer info: %v", err)
			}
			// Debit buyer's account for premium, converting via exchange service if currencies differ.
			premiumToPay := premium
			if ncurrencyID, ok := currencyIDMap[currency]; ok {
				var buyerAcctCurrencyID int64
				if qErr := s.AccountDB.QueryRowContext(ctx,
					`SELECT currency_id FROM accounts WHERE account_number = $1`, buyerAcctNum,
				).Scan(&buyerAcctCurrencyID); qErr == nil {
					if converted, convErr := convertAmount(ctx, s.ExchangeDB, premium, ncurrencyID, buyerAcctCurrencyID); convErr == nil {
						premiumToPay = converted
					}
				}
			}
			_, _ = s.AccountDB.ExecContext(ctx,
				`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1
				 WHERE account_number = $2 AND available_balance >= $1`,
				premiumToPay, buyerAcctNum)
			// Create buyer-side contract (seller is INTERBANK).
			_, _ = s.DB.ExecContext(ctx, `
				INSERT INTO otc_contracts
					(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
					 ticker, amount, strike_price, premium, currency, settlement_date)
				VALUES ($1, 0, 'INTERBANK', $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (negotiation_id) DO NOTHING`,
				negotiationID, buyerID, buyerType,
				ticker, amount, strikePrice, premium, currency, settlementDate)
			_, _ = s.DB.ExecContext(ctx, `UPDATE otc_interbank_tx SET status='COMMITTED' WHERE id = $1`, id)
			return &pb.OtcEmptyResponse{}, nil
		}

		// SELLER-SIDE accept: we are the seller, partner bank is buyer.
		listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ticker not found: %v", err)
		}

		if _, err = s.DB.ExecContext(ctx, `
			INSERT INTO otc_contracts
				(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
				 ticker, amount, strike_price, premium, currency, settlement_date)
			VALUES ($1, $2, $3, 0, 'INTERBANK', $4, $5, $6, $7, $8, $9)
			ON CONFLICT (negotiation_id) DO NOTHING`,
			negotiationID, sellerID, sellerType,
			ticker, amount, strikePrice, premium, currency, settlementDate,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create contract: %v", err)
		}

		result, err := s.PortfolioDB.ExecContext(ctx, `
			UPDATE portfolio_entry
			SET reserved_amount = reserved_amount + $1
			WHERE user_id = $2 AND user_type = $3 AND listing_id = $4
			  AND (amount - reserved_amount) >= $1`,
			stockAmount, portfolioUserID(sellerID, sellerType), sellerType, listingID,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to reserve shares: %v", err)
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			retryExec(s.DB, `DELETE FROM otc_contracts WHERE negotiation_id = $1 AND buyer_type = 'INTERBANK'`, negotiationID)
			return nil, status.Error(codes.FailedPrecondition, "seller no longer has enough free shares")
		}
	} else { // EXERCISE
		var sellerID int64
		var sellerType, ticker, currency string
		var amount int32
		var strikePrice float64
		err = s.DB.QueryRowContext(ctx,
			`SELECT seller_id, seller_type, ticker, amount, currency, strike_price FROM otc_contracts WHERE negotiation_id = $1`,
			negotiationID,
		).Scan(&sellerID, &sellerType, &ticker, &amount, &currency, &strikePrice)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load contract: %v", err)
		}

		listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ticker not found: %v", err)
		}

		if _, err = s.PortfolioDB.ExecContext(ctx, `
			UPDATE portfolio_entry
			SET amount          = amount - $1,
			    reserved_amount = GREATEST(0, reserved_amount - $1),
			    public_amount   = GREATEST(0, LEAST(public_amount, amount - $1)),
			    last_modified   = NOW()
			WHERE user_id = $2 AND user_type = $3 AND listing_id = $4`,
			stockAmount, portfolioUserID(sellerID, sellerType), sellerType, listingID,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update seller portfolio: %v", err)
		}
		_, _ = s.PortfolioDB.ExecContext(ctx,
			`DELETE FROM portfolio_entry WHERE user_id=$1 AND user_type=$2 AND listing_id=$3 AND amount <= 0`,
			portfolioUserID(sellerID, sellerType), sellerType, listingID,
		)

		// Credit seller's account with strike payment (buyer's bank sent us the exercise 2PC).
		strikePayment := float64(stockAmount) * strikePrice
		if currID, ok := currencyIDMap[currency]; ok && strikePayment > 0 {
			if sellerAcctID, findErr := findAccount(ctx, s.AccountDB, portfolioUserID(sellerID, sellerType), currID); findErr == nil {
				_, _ = s.AccountDB.ExecContext(ctx,
					`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
					strikePayment, sellerAcctID)
			}
		}

		if _, err = s.DB.ExecContext(ctx,
			`UPDATE otc_contracts SET status='EXERCISED' WHERE negotiation_id = $1`, negotiationID,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mark contract exercised: %v", err)
		}
	}

	_, _ = s.DB.ExecContext(ctx, `UPDATE otc_interbank_tx SET status='COMMITTED' WHERE id = $1`, id)
	return &pb.OtcEmptyResponse{}, nil
}

func (s *OtcServer) RollbackOtcInterbank(ctx context.Context, req *pb.OtcInterbankTxRequest) (*pb.OtcEmptyResponse, error) {
	_, _ = s.DB.ExecContext(ctx,
		`UPDATE otc_interbank_tx SET status='ROLLED_BACK'
		 WHERE tx_routing_number = $1 AND tx_id = $2 AND status = 'PENDING'`,
		req.TxRoutingNumber, req.TxId,
	)
	return &pb.OtcEmptyResponse{}, nil
}

// ExpireContracts atomically transitions ACTIVE contracts past their settlement window to EXPIRED,
// then records a loss-credit tax entry for each buyer whose premium is now forfeit.
func (s *OtcServer) ExpireContracts() {
	rows, err := s.DB.Query(`
		UPDATE otc_contracts SET status='EXPIRED'
		WHERE status='ACTIVE' AND settlement_date + INTERVAL '1 day' < NOW()
		RETURNING id, buyer_id, buyer_type, premium, currency`)
	if err != nil {
		log.Printf("contract expiration error: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()
	now := time.Now()
	for rows.Next() {
		var id, buyerID int64
		var buyerType, currency string
		var prem float64
		if err := rows.Scan(&id, &buyerID, &buyerType, &prem, &currency); err != nil {
			continue
		}
		s.recordOtcTax(buyerID, buyerType, -prem, currency, int(now.Month()), now.Year())
	}
}
