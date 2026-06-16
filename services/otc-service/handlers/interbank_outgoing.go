package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	otcInterbank "github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/interbank"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- JSON types for the /interbank envelope (OTC outgoing) ---

type otcIbTransactionID struct {
	RoutingNumber int    `json:"routingNumber"`
	ID            string `json:"id"`
}

type otcIbIdempotenceKey struct {
	RoutingNumber       int    `json:"routingNumber"`
	LocallyGeneratedKey string `json:"locallyGeneratedKey"`
}

// Nested posting structs matching Banka 4's interbank format.
type ibOutPartyID struct {
	RoutingNumber int    `json:"routingNumber"`
	ID            string `json:"id"`
}
type ibOutAccount struct {
	Type string        `json:"type"`
	Num  string        `json:"num,omitempty"`
	ID   *ibOutPartyID `json:"id,omitempty"`
}
type ibOutStock struct {
	Ticker string `json:"ticker"`
}
type ibOutMoney struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}
type ibOutAssetInner struct {
	Currency       string        `json:"currency,omitempty"`
	Ticker         string        `json:"ticker,omitempty"`
	NegotiationID  *ibOutPartyID `json:"negotiationId,omitempty"`
	Stock          *ibOutStock   `json:"stock,omitempty"`
	PricePerUnit   *ibOutMoney   `json:"pricePerUnit,omitempty"`
	SettlementDate string        `json:"settlementDate,omitempty"`
	Amount         float64       `json:"amount,omitempty"`
}
type ibOutAsset struct {
	Type  string           `json:"type"`
	Asset *ibOutAssetInner `json:"asset,omitempty"`
}
type ibOutPosting struct {
	Account ibOutAccount `json:"account"`
	Amount  float64      `json:"amount"`
	Asset   ibOutAsset   `json:"asset"`
}

type otcIbNewTxMessage struct {
	TransactionID  otcIbTransactionID `json:"transactionId"`
	Message        string             `json:"message"`
	PaymentCode    string             `json:"paymentCode"`
	PaymentPurpose string             `json:"paymentPurpose"`
	Postings       []ibOutPosting     `json:"postings"`
}

type otcIbCommitMessage struct {
	TransactionID otcIbTransactionID `json:"transactionId"`
}

type otcIbEnvelope struct {
	IdempotenceKey otcIbIdempotenceKey `json:"idempotenceKey"`
	MessageType    string              `json:"messageType"`
	Message        any                 `json:"message"`
}

type otcIbVoteResponse struct {
	Vote    string   `json:"vote"`
	Reasons []string `json:"reasons"`
}

type otcOutgoing2PCReq struct {
	sellerExtID          string
	partnerNegotiationID string
	partnerRoutingNum    int // seller's bank routing number
	stockAmount          int32
	buyerAccountNum      string
	buyerExternalID      string // buyer's local ID as string
	totalCost            float64
	currency             string
	ticker               string // stock ticker for STOCK asset posting
}

func otcGenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func otcOwnRoutingInt() int {
	var n int
	_, _ = fmt.Sscanf(os.Getenv("OWN_ROUTING_NUMBER"), "%d", &n)
	return n
}

func otcSendInterbankRequest(ctx context.Context, url, apiKey string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", apiKey)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusAccepted {
			_ = resp.Body.Close()
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("all 3 attempts failed or returned 202")
}

// executeOtcOutgoing2PC sends NEW_TX + COMMIT_TX to the partner bank for cross-bank exercise.
// Returns "YES" on successful commit, error otherwise.
func executeOtcOutgoing2PC(ctx context.Context, bank otcInterbank.BankInfo, req otcOutgoing2PCReq) (string, error) {
	bankURL := bank.BankURL + "/interbank"
	ownRouting := otcOwnRoutingInt()
	txID := otcIbTransactionID{RoutingNumber: ownRouting, ID: otcGenerateUUID()}
	idemKey := otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()}

	newTxResp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: otcIbNewTxMessage{
			TransactionID:  txID,
			Message:        "Peer OTC option exercise",
			PaymentCode:    "289",
			PaymentPurpose: "OTC option exercise",
			Postings: []ibOutPosting{
				{ // buyer's settlement account debited
					Account: ibOutAccount{Type: "ACCOUNT", Num: req.buyerAccountNum},
					Amount:  -req.totalCost,
					Asset:   ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: req.currency}},
				},
				{ // OPTION contract receives money (seller's bank credits seller via escrow)
					Account: ibOutAccount{Type: "OPTION", ID: &ibOutPartyID{
						RoutingNumber: req.partnerRoutingNum,
						ID:            req.partnerNegotiationID,
					}},
					Amount: req.totalCost,
					Asset:  ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: req.currency}},
				},
				{ // OPTION contract gives shares
					Account: ibOutAccount{Type: "OPTION", ID: &ibOutPartyID{
						RoutingNumber: req.partnerRoutingNum,
						ID:            req.partnerNegotiationID,
					}},
					Amount: -float64(req.stockAmount),
					Asset:  ibOutAsset{Type: "STOCK", Asset: &ibOutAssetInner{Ticker: req.ticker}},
				},
				{ // buyer receives shares at our bank
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{
						RoutingNumber: otcOwnRoutingInt(),
						ID:            req.buyerExternalID,
					}},
					Amount: float64(req.stockAmount),
					Asset:  ibOutAsset{Type: "STOCK", Asset: &ibOutAssetInner{Ticker: req.ticker}},
				},
			},
		},
	})
	if err != nil {
		return "NO", fmt.Errorf("NEW_TX request failed: %w", err)
	}
	defer func() { _ = newTxResp.Body.Close() }()

	var voteResp otcIbVoteResponse
	if err := json.NewDecoder(newTxResp.Body).Decode(&voteResp); err != nil {
		return "NO", fmt.Errorf("decode vote response: %w", err)
	}
	if voteResp.Vote != "YES" {
		return "NO", nil
	}

	commitResp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
		MessageType:    "COMMIT_TX",
		Message:        otcIbCommitMessage{TransactionID: txID},
	})
	if err != nil {
		// Best-effort rollback
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return "NO", fmt.Errorf("COMMIT_TX request failed: %w", err)
	}
	defer func() { _ = commitResp.Body.Close() }()

	if commitResp.StatusCode != http.StatusNoContent {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return "NO", fmt.Errorf("COMMIT_TX returned status %d", commitResp.StatusCode)
	}

	return "YES", nil
}

// exerciseCrossBank handles ExerciseContract when seller_type='INTERBANK'.
// It runs a local SAGA step 1 (reserve buyer funds) then an outgoing 2PC to the partner bank.
func (s *OtcServer) exerciseCrossBank(
	ctx context.Context,
	req *pb.ExerciseContractRequest,
	contractTx *sql.Tx,
	sellerID, buyerID int64,
	buyerType string,
	amount int32,
	strikePrice float64,
	currency, ticker string,
	settlementDate time.Time,
) (*pb.ExerciseContractResponse, error) {
	totalCost := strikePrice * float64(amount)
	currencyID, ok := currencyIDMap[currency]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", currency)
	}

	// Lookup routing info from the negotiation linked to this contract.
	var sellerRoutingNum int32
	var partnerNegotiationID string
	var sellerExtID string
	err := s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(n.seller_routing_number, 0),
		       COALESCE(n.partner_negotiation_id, ''),
		       COALESCE(n.seller_external_id, '')
		FROM otc_negotiations n
		JOIN otc_contracts c ON c.negotiation_id = n.id
		WHERE c.id = $1`, req.ContractId,
	).Scan(&sellerRoutingNum, &partnerNegotiationID, &sellerExtID)
	if err != nil || sellerRoutingNum == 0 || partnerNegotiationID == "" {
		return nil, status.Error(codes.Internal,
			"cross-bank exercise requires seller_routing_number and partner_negotiation_id (populated by outgoing negotiation flow)")
	}

	// Resolve buyer account number.
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
	var buyerAccountNum string
	if err = s.AccountDB.QueryRowContext(ctx,
		`SELECT account_number FROM accounts WHERE id = $1`, buyerAccountID,
	).Scan(&buyerAccountNum); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load buyer account number: %v", err)
	}

	sagaLog := func(step int, stepStatus, errMsg string) {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_, _ = s.DB.ExecContext(logCtx,
			`INSERT INTO otc_saga_log (contract_id, step, status, error_msg) VALUES ($1, $2, $3, $4)`,
			req.ContractId, step, stepStatus, sql.NullString{String: errMsg, Valid: errMsg != ""},
		)
	}

	// Step 1: Reserve buyer funds.
	result, err := s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET available_balance = available_balance - $1
		 WHERE id = $2 AND available_balance >= $1 AND balance >= $1`,
		totalCostToPay, buyerAccountID,
	)
	if err != nil {
		sagaLog(1, "FAILED", err.Error())
		return nil, status.Errorf(codes.Internal, "step 1 failed: %v", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		sagaLog(1, "FAILED", fmt.Sprintf("insufficient funds: need %.2f", totalCostToPay))
		return nil, status.Error(codes.InvalidArgument, "Insufficient funds")
	}
	sagaLog(1, "SUCCESS", "")

	comp1 := func() {
		retryExec(s.AccountDB,
			`UPDATE accounts SET available_balance = available_balance + $1 WHERE id = $2`,
			totalCostToPay, buyerAccountID)
		sagaLog(1, "COMPENSATED", "")
	}

	// Step 2: Outgoing 2PC to partner bank (seller's bank handles OPTION + PERSON postings).
	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", sellerRoutingNum))
	if err != nil {
		sagaLog(2, "FAILED", err.Error())
		comp1()
		return nil, status.Errorf(codes.Internal, "cannot resolve seller bank: %v", err)
	}

	voteResult, commitErr := executeOtcOutgoing2PC(ctx, bank, otcOutgoing2PCReq{
		sellerExtID:          sellerExtID,
		partnerNegotiationID: partnerNegotiationID,
		partnerRoutingNum:    int(sellerRoutingNum),
		stockAmount:          amount,
		buyerAccountNum:      buyerAccountNum,
		buyerExternalID:      fmt.Sprintf("%d", buyerID),
		totalCost:            totalCost,
		currency:             currency,
		ticker:               ticker,
	})
	if commitErr != nil || voteResult != "YES" {
		sagaLog(2, "FAILED", fmt.Sprintf("2PC result: %v / vote=%s", commitErr, voteResult))
		comp1()
		if commitErr != nil {
			return nil, status.Errorf(codes.Unavailable, "cross-bank 2PC failed: %v", commitErr)
		}
		return nil, status.Error(codes.FailedPrecondition, "partner bank rejected exercise")
	}
	sagaLog(2, "SUCCESS", "")

	// Step 3: Debit buyer balance (available_balance already reserved in step 1).
	_, _ = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1, available_balance = available_balance + $1 WHERE id = $2`,
		totalCostToPay, buyerAccountID)

	// Step 4: Add shares to buyer's portfolio.
	listingID, _ := listingIDForTicker(ctx, s.SecuritiesDB, ticker)
	_, _ = s.PortfolioDB.ExecContext(ctx, `
		INSERT INTO portfolio_entry (user_id, user_type, listing_id, amount, buy_price, account_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, user_type, listing_id)
		DO UPDATE SET amount = portfolio_entry.amount + EXCLUDED.amount, last_modified = NOW()`,
		portfolioUserID(buyerID, buyerType), buyerType, listingID, amount, strikePrice, buyerAccountID,
	)

	// Step 5: Mark contract EXERCISED and commit.
	now := time.Now()
	_, _ = contractTx.ExecContext(ctx, `UPDATE otc_contracts SET status='EXERCISED' WHERE id = $1`, req.ContractId)
	_ = contractTx.Commit()
	sagaLog(5, "SUCCESS", "")

	return &pb.ExerciseContractResponse{
		Status:     "EXERCISED",
		ExecutedAt: now.Format(time.RFC3339),
	}, nil
}

// forwardNegotiationToPartner sends an outgoing negotiation to the partner bank's
// /negotiations endpoint and returns the partner bank's local negotiation ID.
func (s *OtcServer) forwardNegotiationToPartner(
	bank otcInterbank.BankInfo,
	req *pb.CreateNegotiationRequest,
	buyerAccountNum string,
) (string, error) {
	type partyID struct {
		RoutingNumber int    `json:"routingNumber"`
		ID            string `json:"id"`
	}
	type money struct {
		Currency string  `json:"currency"`
		Amount   float64 `json:"amount"`
	}
	type stock struct {
		Ticker string `json:"ticker"`
	}
	type body struct {
		Stock              stock   `json:"stock"`
		SettlementDate     string  `json:"settlementDate"`
		PricePerUnit       money   `json:"pricePerUnit"`
		Premium            money   `json:"premium"`
		BuyerID            partyID `json:"buyerId"`
		SellerID           partyID `json:"sellerId"`
		Amount             int32   `json:"amount"`
		SellerType         string  `json:"sellerType"`
		LastModifiedBy     partyID `json:"lastModifiedBy"`
		BuyerAccountNumber string  `json:"buyerAccountNumber"`
	}
	ownRouting := otcOwnRoutingInt()
	b := body{
		Stock:              stock{Ticker: req.Ticker},
		SettlementDate:     req.SettlementDate,
		PricePerUnit:       money{Currency: req.Currency, Amount: req.PricePerStock},
		Premium:            money{Currency: req.Currency, Amount: req.Premium},
		BuyerID:            partyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", req.BuyerId)},
		SellerID:           partyID{RoutingNumber: int(req.SellerRoutingNumber), ID: req.SellerExternalId},
		Amount:             req.Amount,
		SellerType:         req.SellerType,
		LastModifiedBy:     partyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", req.BuyerId)},
		BuyerAccountNumber: buyerAccountNum,
	}
	data, _ := json.Marshal(b)
	httpReq, err := http.NewRequest(http.MethodPost, bank.BankURL+"/negotiations", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", bank.APIKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("partner returned status %d", resp.StatusCode)
	}
	var result struct {
		RoutingNumber int32  `json:"routingNumber"`
		ID            string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	return result.ID, nil
}

// createNegotiationCrossBank handles CreateNegotiation when the seller is on a partner bank.
func (s *OtcServer) createNegotiationCrossBank(ctx context.Context, req *pb.CreateNegotiationRequest) (*pb.NegotiationResponse, error) {
	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", req.SellerRoutingNumber))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot resolve seller bank: %v", err)
	}

	// Resolve buyer's settlement account so CommitOtcInterbank can debit it.
	// Prefer an account in the negotiation's currency; fall back to any active account.
	// CommitOtcInterbank uses convertAmount (exchange service) to handle currency differences.
	currencyID := currencyIDMap[req.Currency]
	var buyerAccountNum string
	if acctID, acctErr := findAccount(ctx, s.AccountDB, req.BuyerId, currencyID); acctErr == nil {
		_ = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number FROM accounts WHERE id = $1`, acctID,
		).Scan(&buyerAccountNum)
	}
	if buyerAccountNum == "" {
		_ = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number FROM accounts WHERE owner_id = $1 AND status = 'ACTIVE' LIMIT 1`,
			req.BuyerId,
		).Scan(&buyerAccountNum)
	}

	now := time.Now()
	var id int64
	if err = s.DB.QueryRowContext(ctx, `
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type,
			 amount, price_per_stock, settlement_date, premium, currency,
			 last_modified, modified_by_id, modified_by_type, status,
			 seller_routing_number, seller_external_id,
			 buyer_account_number)
		VALUES ($1, 0, 'INTERBANK', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'PENDING_SELLER', $12, $13, $14)
		RETURNING id`,
		req.Ticker, req.BuyerId, req.BuyerType,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium, req.Currency,
		now, req.BuyerId, req.BuyerType,
		req.SellerRoutingNumber, req.SellerExternalId,
		buyerAccountNum,
	).Scan(&id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create cross-bank negotiation: %v", err)
	}

	partnerNegID, fwdErr := s.forwardNegotiationToPartner(bank, req, buyerAccountNum)
	if fwdErr != nil {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM otc_negotiations WHERE id = $1`, id)
		return nil, status.Errorf(codes.Unavailable, "failed to forward negotiation to partner bank: %v", fwdErr)
	}

	if _, err = s.DB.ExecContext(ctx,
		`UPDATE otc_negotiations SET partner_negotiation_id = $1 WHERE id = $2`,
		partnerNegID, id,
	); err != nil {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM otc_negotiations WHERE id = $1`, id)
		return nil, status.Errorf(codes.Internal, "failed to store partner_negotiation_id: %v", err)
	}

	return s.fetchNegotiationByID(ctx, id)
}

// acceptCrossBank handles AcceptNegotiation when the seller_type is 'INTERBANK'.
// Banka 4 is the seller: we commit ACCEPTED, then call their /accept endpoint.
// Banka 4 synchronously sends us a NEW_TX→COMMIT_TX (our /interbank) which creates
// the contract and debits the buyer via CommitOtcInterbank.
func (s *OtcServer) acceptCrossBank(
	ctx context.Context,
	req *pb.AcceptNegotiationRequest,
	tx *sql.Tx,
	sellerID, buyerID int64,
	buyerType, ticker string,
	amount int32, strikePrice, premium float64,
	currency, settlementDate string,
	negotiationID int64,
) (*pb.NegotiationResponse, error) {
	var sellerRoutingNum int32
	var partnerNegotiationID string
	if err := s.DB.QueryRowContext(ctx,
		`SELECT COALESCE(seller_routing_number, 0), COALESCE(partner_negotiation_id, '')
		 FROM otc_negotiations WHERE id = $1`, negotiationID,
	).Scan(&sellerRoutingNum, &partnerNegotiationID); err != nil || sellerRoutingNum == 0 || partnerNegotiationID == "" {
		return nil, status.Error(codes.Internal, "cross-bank accept: missing seller routing info")
	}
	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", sellerRoutingNum))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot resolve seller bank: %v", err)
	}

	// Commit ACCEPTED status BEFORE calling Banka 4 so the row lock is released.
	// PrepareOtcInterbank (triggered by Banka 4's synchronous 2PC) must read ACCEPTED.
	now := time.Now()
	if _, err = tx.ExecContext(ctx,
		`UPDATE otc_negotiations SET status='ACCEPTED', last_modified=$1 WHERE id=$2`, now, negotiationID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to accept negotiation: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}

	// Call Banka 4's accept endpoint. Banka 4 will synchronously send us a NEW_TX→COMMIT_TX
	// via /interbank, which creates the contract and debits buyer in CommitOtcInterbank.
	acceptURL := fmt.Sprintf("%s/negotiations/%d/%s/accept",
		bank.BankURL, sellerRoutingNum, partnerNegotiationID)
	acceptReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, acceptURL, nil)
	acceptReq.Header.Set("X-Api-Key", bank.APIKey)
	acceptResp, acceptErr := (&http.Client{Timeout: 30 * time.Second}).Do(acceptReq)
	if acceptErr != nil {
		_, _ = s.DB.ExecContext(context.Background(),
			`UPDATE otc_negotiations SET status='PENDING_BUYER' WHERE id=$1`, negotiationID)
		return nil, status.Errorf(codes.Unavailable, "partner accept call failed: %v", acceptErr)
	}
	_ = acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusNoContent && acceptResp.StatusCode != http.StatusOK {
		_, _ = s.DB.ExecContext(context.Background(),
			`UPDATE otc_negotiations SET status='PENDING_BUYER' WHERE id=$1`, negotiationID)
		return nil, status.Errorf(codes.FailedPrecondition, "partner accept returned %d", acceptResp.StatusCode)
	}
	// Contract and buyer debit created by CommitOtcInterbank during synchronous 2PC above.
	return s.fetchNegotiationByID(ctx, negotiationID)
}

// forwardSellerCounterOfferToPartner sends PUT /negotiations/{ownRn}/{negID} to the buyer's bank
// (used when we are the seller and submit a counter-offer on an inbound interbank negotiation).
func (s *OtcServer) forwardSellerCounterOfferToPartner(ctx context.Context, negID int64, amount int32, pricePerStock, premium float64, settlementDate string) error {
	type partyID struct {
		RoutingNumber int    `json:"routingNumber"`
		ID            string `json:"id"`
	}
	type money struct {
		Currency string  `json:"currency"`
		Amount   float64 `json:"amount"`
	}
	type stock struct {
		Ticker string `json:"ticker"`
	}
	type putBody struct {
		Stock              stock   `json:"stock"`
		SettlementDate     string  `json:"settlementDate"`
		PricePerUnit       money   `json:"pricePerUnit"`
		Premium            money   `json:"premium"`
		BuyerID            partyID `json:"buyerId"`
		SellerID           partyID `json:"sellerId"`
		Amount             int32   `json:"amount"`
		LastModifiedBy     partyID `json:"lastModifiedBy"`
		BuyerAccountNumber string  `json:"buyerAccountNumber"`
	}

	var ticker, currency string
	var buyerExtID, buyerAcctNum sql.NullString
	var buyerRoutingNum sql.NullInt32
	var sellerID int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT ticker, currency, buyer_routing_number, buyer_external_id, seller_id, buyer_account_number
		FROM otc_negotiations WHERE id = $1`, negID,
	).Scan(&ticker, &currency, &buyerRoutingNum, &buyerExtID, &sellerID, &buyerAcctNum)
	if err != nil {
		return fmt.Errorf("load negotiation: %w", err)
	}
	if !buyerRoutingNum.Valid {
		return fmt.Errorf("no buyer routing number stored for negotiation %d", negID)
	}

	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", buyerRoutingNum.Int32))
	if err != nil {
		return fmt.Errorf("resolve buyer bank: %w", err)
	}

	ownRouting := otcOwnRoutingInt()
	b := putBody{
		Stock:              stock{Ticker: ticker},
		SettlementDate:     settlementDate,
		PricePerUnit:       money{Currency: currency, Amount: pricePerStock},
		Premium:            money{Currency: currency, Amount: premium},
		BuyerID:            partyID{RoutingNumber: int(buyerRoutingNum.Int32), ID: buyerExtID.String},
		SellerID:           partyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", sellerID)},
		Amount:             amount,
		LastModifiedBy:     partyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", sellerID)},
		BuyerAccountNumber: buyerAcctNum.String,
	}
	data, _ := json.Marshal(b)
	url := fmt.Sprintf("%s/negotiations/%d/%d", bank.BankURL, ownRouting, negID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", bank.APIKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("partner returned status %d", resp.StatusCode)
	}
	return nil
}

// notifyPartnerRejection sends DELETE /negotiations/{ownRn}/{negID} to the buyer's bank
// (used when we are the seller and reject an inbound interbank negotiation).
func (s *OtcServer) notifyPartnerRejection(ctx context.Context, negID int64) error {
	var buyerRoutingNum sql.NullInt32
	err := s.DB.QueryRowContext(ctx,
		`SELECT buyer_routing_number FROM otc_negotiations WHERE id = $1`, negID,
	).Scan(&buyerRoutingNum)
	if err != nil || !buyerRoutingNum.Valid {
		return fmt.Errorf("load buyer routing number: %w", err)
	}

	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", buyerRoutingNum.Int32))
	if err != nil {
		return fmt.Errorf("resolve buyer bank: %w", err)
	}

	ownRouting := otcOwnRoutingInt()
	url := fmt.Sprintf("%s/negotiations/%d/%d", bank.BankURL, ownRouting, negID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("X-Api-Key", bank.APIKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("partner returned status %d", resp.StatusCode)
	}
	return nil
}

// executeInterbankAcceptOutgoing is called by InterbankAcceptNegotiation (seller-side) after
// the negotiation row is committed to ACCEPTED. It sends a 4-posting accept 2PC to the buyer's
// bank, reserves seller shares, and inserts the seller-side contract.
func (s *OtcServer) executeInterbankAcceptOutgoing(ctx context.Context, localNegID int64) error {
	var buyerAcctNum, buyerExtID, currency, ticker, sellerType sql.NullString
	var buyerRoutingNum sql.NullInt32
	var sellerID int64
	var premium, strikePrice float64
	var amount int32
	var settlementDate string
	err := s.DB.QueryRowContext(ctx, `
		SELECT buyer_account_number, buyer_routing_number, buyer_external_id,
		       seller_id, seller_type, premium, currency, amount, ticker,
		       settlement_date::text, price_per_stock
		FROM otc_negotiations WHERE id = $1`, localNegID,
	).Scan(&buyerAcctNum, &buyerRoutingNum, &buyerExtID,
		&sellerID, &sellerType, &premium, &currency, &amount, &ticker,
		&settlementDate, &strikePrice)
	if err != nil {
		return fmt.Errorf("load negotiation: %w", err)
	}
	if !buyerRoutingNum.Valid {
		return fmt.Errorf("missing buyer_routing_number on negotiation %d", localNegID)
	}
	if !buyerAcctNum.Valid || buyerAcctNum.String == "" {
		return fmt.Errorf("missing buyer_account_number on negotiation %d — partner bank must include buyerAccountNumber in the offer", localNegID)
	}

	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", buyerRoutingNum.Int32))
	if err != nil {
		return fmt.Errorf("resolve buyer bank: %w", err)
	}

	bankURL := bank.BankURL + "/interbank"
	ownRouting := otcOwnRoutingInt()
	txID := otcIbTransactionID{RoutingNumber: ownRouting, ID: otcGenerateUUID()}
	idemKey := otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()}

	envelope := otcIbEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: otcIbNewTxMessage{
			TransactionID:  txID,
			Message:        "Peer OTC option premium and contract acceptance",
			PaymentCode:    "289",
			PaymentPurpose: "OTC option premium",
			Postings: []ibOutPosting{
				{ // posting 1: buyer's account debited (premium)
					Account: ibOutAccount{Type: "ACCOUNT", Num: buyerAcctNum.String},
					Amount:  -premium,
					Asset:   ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: currency.String}},
				},
				{ // posting 2: seller (us) receives premium
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", sellerID)}},
					Amount:  premium,
					Asset:   ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: currency.String}},
				},
				{ // posting 4: buyer (bank4) receives option right — triggers bank4's commitOptionPosting to create their mirror contract
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{RoutingNumber: int(buyerRoutingNum.Int32), ID: buyerExtID.String}},
					Amount:  1,
					Asset: ibOutAsset{Type: "OPTION", Asset: &ibOutAssetInner{
						NegotiationID:  &ibOutPartyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", localNegID)},
						Stock:          &ibOutStock{Ticker: ticker.String},
						PricePerUnit:   &ibOutMoney{Currency: currency.String, Amount: strikePrice},
						SettlementDate: settlementDate,
						Amount:         float64(amount),
					}},
				},
			},
		},
	}

	resp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, envelope)
	if err != nil {
		return fmt.Errorf("NEW_TX: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, _ := io.ReadAll(resp.Body)
	var vote otcIbVoteResponse
	if jsonErr := json.Unmarshal(rawBody, &vote); jsonErr != nil || vote.Vote != "YES" {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return fmt.Errorf("buyer bank voted NO (status=%d body=%s)", resp.StatusCode, string(rawBody))
	}

	// YES: reserve seller shares and create seller-side contract.
	listingID, err := listingIDForTicker(ctx, s.SecuritiesDB, ticker.String)
	if err != nil {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return fmt.Errorf("ticker not found: %w", err)
	}
	result, err := s.PortfolioDB.ExecContext(ctx,
		`UPDATE portfolio_entry SET reserved_amount = reserved_amount + $1
		 WHERE user_id = $2 AND user_type = $3 AND listing_id = $4
		   AND (amount - reserved_amount) >= $1`,
		amount, portfolioUserID(sellerID, sellerType.String), sellerType.String, listingID,
	)
	if err != nil {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return fmt.Errorf("reserve shares: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return fmt.Errorf("seller has insufficient free shares")
	}

	_, _ = s.DB.ExecContext(ctx, `
		INSERT INTO otc_contracts
			(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
			 ticker, amount, strike_price, premium, currency, settlement_date)
		VALUES ($1, $2, $3, 0, 'INTERBANK', $4, $5, $6, $7, $8, $9)
		ON CONFLICT (negotiation_id) DO NOTHING`,
		localNegID, sellerID, sellerType.String,
		ticker.String, amount, strikePrice, premium, currency.String, settlementDate,
	)

	// COMMIT.
	_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
		MessageType:    "COMMIT_TX",
		Message:        otcIbCommitMessage{TransactionID: txID},
	})

	// Credit seller's account locally — the 2PC posting notifies Banka 4 but does not
	// touch our account DB. We credit after COMMIT so the buyer's debit is already settled.
	if currID, ok := currencyIDMap[currency.String]; ok && premium > 0 {
		if sellerAcctID, findErr := findAccount(ctx, s.AccountDB, portfolioUserID(sellerID, sellerType.String), currID); findErr == nil {
			_, _ = s.AccountDB.ExecContext(ctx,
				`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
				premium, sellerAcctID)
		}
	}

	return nil
}

// executeInterbankAcceptBuyerSide coordinates the 2PC when we are the buyer and bank4 (seller)
// has accepted. Bank4 called our /accept endpoint (AcceptAsLocal); we form the NEW_TX, send it
// to bank4, debit our buyer's account on YES, create our buyer-side contract, and send COMMIT.
// Bank4 on COMMIT creates their authoritative seller contract via commitOptionPosting.
func (s *OtcServer) executeInterbankAcceptBuyerSide(ctx context.Context, localNegID int64) error {
	var ticker, currency, buyerType string
	var sellerExtID, buyerAcctNum sql.NullString
	var settlementDate string
	var buyerID int64
	var sellerRoutingNum sql.NullInt32
	var partnerNegID sql.NullString
	var amount int32
	var strikePrice, premium float64

	err := s.DB.QueryRowContext(ctx, `
		SELECT ticker, currency, settlement_date::text,
		       buyer_id, buyer_type, buyer_account_number,
		       seller_routing_number, seller_external_id,
		       partner_negotiation_id, amount, price_per_stock, premium
		FROM otc_negotiations WHERE id = $1`, localNegID,
	).Scan(&ticker, &currency, &settlementDate,
		&buyerID, &buyerType, &buyerAcctNum,
		&sellerRoutingNum, &sellerExtID,
		&partnerNegID, &amount, &strikePrice, &premium)
	if err != nil {
		return fmt.Errorf("load negotiation: %w", err)
	}
	if !sellerRoutingNum.Valid || !partnerNegID.Valid || partnerNegID.String == "" {
		return fmt.Errorf("missing seller_routing_number or partner_negotiation_id on outbound negotiation %d", localNegID)
	}
	if !buyerAcctNum.Valid || buyerAcctNum.String == "" {
		return fmt.Errorf("missing buyer_account_number on outbound negotiation %d", localNegID)
	}

	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", sellerRoutingNum.Int32))
	if err != nil {
		return fmt.Errorf("resolve seller bank: %w", err)
	}

	bankURL := bank.BankURL + "/interbank"
	ownRouting := otcOwnRoutingInt()
	txID := otcIbTransactionID{RoutingNumber: ownRouting, ID: otcGenerateUUID()}
	idemKey := otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()}

	// OPTION asset: negotiationId uses bank4's routing + bank4's UUID (they are authoritative seller).
	optionAI := ibOutAssetInner{
		NegotiationID:  &ibOutPartyID{RoutingNumber: int(sellerRoutingNum.Int32), ID: partnerNegID.String},
		Stock:          &ibOutStock{Ticker: ticker},
		PricePerUnit:   &ibOutMoney{Currency: currency, Amount: strikePrice},
		SettlementDate: settlementDate,
		Amount:         float64(amount),
	}

	envelope := otcIbEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: otcIbNewTxMessage{
			TransactionID:  txID,
			Message:        "Peer OTC option premium and contract acceptance",
			PaymentCode:    "289",
			PaymentPurpose: "OTC option premium",
			Postings: []ibOutPosting{
				{ // posting 1: buyer ACCOUNT MONAS credit — buyer pays premium (non-local to bank4, skipped by them)
					Account: ibOutAccount{Type: "ACCOUNT", Num: buyerAcctNum.String},
					Amount:  -premium,
					Asset:   ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: currency}},
				},
				{ // posting 2: seller PERSON MONAS debit — seller receives premium
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{RoutingNumber: int(sellerRoutingNum.Int32), ID: sellerExtID.String}},
					Amount:  premium,
					Asset:   ibOutAsset{Type: "MONAS", Asset: &ibOutAssetInner{Currency: currency}},
				},
				{ // posting 3: seller PERSON OPTION credit — seller gives option (reserves shares; triggers authoritative contract on COMMIT)
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{RoutingNumber: int(sellerRoutingNum.Int32), ID: sellerExtID.String}},
					Amount:  -1,
					Asset:   ibOutAsset{Type: "OPTION", Asset: &optionAI},
				},
				{ // posting 4: buyer PERSON OPTION debit — buyer receives option (non-local to bank4, skipped by them; we handle locally)
					Account: ibOutAccount{Type: "PERSON", ID: &ibOutPartyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", buyerID)}},
					Amount:  1,
					Asset:   ibOutAsset{Type: "OPTION", Asset: &optionAI},
				},
			},
		},
	}

	resp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, envelope)
	if err != nil {
		return fmt.Errorf("NEW_TX: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, _ := io.ReadAll(resp.Body)
	var vote otcIbVoteResponse
	if jsonErr := json.Unmarshal(rawBody, &vote); jsonErr != nil || vote.Vote != "YES" {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return fmt.Errorf("seller bank voted NO (status=%d body=%s)", resp.StatusCode, string(rawBody))
	}

	// YES: debit buyer's account (converting currency if needed) and create our buyer-side contract.
	premiumToPay := premium
	if ncurrencyID, ok := currencyIDMap[currency]; ok {
		var buyerAcctCurrencyID int64
		if qErr := s.AccountDB.QueryRowContext(ctx,
			`SELECT currency_id FROM accounts WHERE account_number = $1`, buyerAcctNum.String,
		).Scan(&buyerAcctCurrencyID); qErr == nil {
			if converted, convErr := convertAmount(ctx, s.ExchangeDB, premium, ncurrencyID, buyerAcctCurrencyID); convErr == nil {
				premiumToPay = converted
			}
		}
	}
	_, _ = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1
		 WHERE account_number = $2 AND available_balance >= $1`,
		premiumToPay, buyerAcctNum.String)

	_, _ = s.DB.ExecContext(ctx, `
		INSERT INTO otc_contracts
			(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
			 ticker, amount, strike_price, premium, currency, settlement_date)
		VALUES ($1, 0, 'INTERBANK', $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (negotiation_id) DO NOTHING`,
		localNegID, buyerID, buyerType,
		ticker, amount, strikePrice, premium, currency, settlementDate)

	// COMMIT — bank4 will commit their postings (credit seller's MONAS, create authoritative seller contract).
	_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
		MessageType:    "COMMIT_TX",
		Message:        otcIbCommitMessage{TransactionID: txID},
	})

	return nil
}
