package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
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

type otcIbPosting struct {
	AccountType string  `json:"accountType"`
	AccountNum  string  `json:"accountNum"`
	Amount      float64 `json:"amount"`
	AssetType   string  `json:"assetType,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

type otcIbNewTxMessage struct {
	TransactionID otcIbTransactionID `json:"transactionId"`
	Postings      []otcIbPosting     `json:"postings"`
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
	partnerNegotiationID int64
	stockAmount          int32
	buyerAccountNum      string
	totalCost            float64
	currency             string
}

func otcGenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func otcOwnRoutingInt() int {
	var n int
	fmt.Sscanf(os.Getenv("OWN_ROUTING_NUMBER"), "%d", &n)
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
			TransactionID: txID,
			Postings: []otcIbPosting{
				// Credit seller on partner bank (identified by owner_id / external ID)
				{AccountType: "PERSON", AccountNum: req.sellerExtID, Amount: +req.totalCost, AssetType: "MONAS", Currency: req.currency},
				// Consume the OPTION (partner's local negotiation ID)
				{AccountType: "OPTION", AccountNum: fmt.Sprintf("%d", req.partnerNegotiationID), Amount: -float64(req.stockAmount)},
				// Debit buyer account on our side
				{AccountType: "ACCOUNT", AccountNum: req.buyerAccountNum, Amount: -req.totalCost, AssetType: "MONAS", Currency: req.currency},
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
	var partnerNegotiationID int64
	var sellerExtID string
	err := s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(n.seller_routing_number, 0),
		       COALESCE(n.partner_negotiation_id, 0),
		       COALESCE(n.seller_external_id, '')
		FROM otc_negotiations n
		JOIN otc_contracts c ON c.negotiation_id = n.id
		WHERE c.id = $1`, req.ContractId,
	).Scan(&sellerRoutingNum, &partnerNegotiationID, &sellerExtID)
	if err != nil || sellerRoutingNum == 0 || partnerNegotiationID == 0 {
		return nil, status.Error(codes.Internal,
			"cross-bank exercise requires seller_routing_number and partner_negotiation_id (populated by outgoing negotiation flow)")
	}

	// Resolve buyer account number.
	buyerAccountID := req.BuyerAccountId
	if buyerAccountID == 0 {
		buyerAccountID, err = findAccount(s.AccountDB, portfolioUserID(buyerID, buyerType), currencyID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find buyer account: %v", err)
		}
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
		totalCost, buyerAccountID,
	)
	if err != nil {
		sagaLog(1, "FAILED", err.Error())
		return nil, status.Errorf(codes.Internal, "step 1 failed: %v", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		sagaLog(1, "FAILED", fmt.Sprintf("insufficient funds: need %.2f", totalCost))
		return nil, status.Error(codes.InvalidArgument, "Insufficient funds")
	}
	sagaLog(1, "SUCCESS", "")

	comp1 := func() {
		retryExec(s.AccountDB,
			`UPDATE accounts SET available_balance = available_balance + $1 WHERE id = $2`,
			totalCost, buyerAccountID)
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
		stockAmount:          amount,
		buyerAccountNum:      buyerAccountNum,
		totalCost:            totalCost,
		currency:             currency,
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
		totalCost, buyerAccountID)

	// Step 4: Add shares to buyer's portfolio.
	listingID, _ := listingIDForTicker(s.SecuritiesDB, ticker)
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
// /otc/interbank/negotiations endpoint and returns the partner bank's local negotiation ID.
func (s *OtcServer) forwardNegotiationToPartner(
	bank otcInterbank.BankInfo,
	req *pb.CreateNegotiationRequest,
) (int64, error) {
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
		Stock          stock   `json:"stock"`
		SettlementDate string  `json:"settlementDate"`
		PricePerUnit   money   `json:"pricePerUnit"`
		Premium        money   `json:"premium"`
		BuyerID        partyID `json:"buyerId"`
		SellerID       partyID `json:"sellerId"`
		Amount         int32   `json:"amount"`
		SellerType     string  `json:"sellerType"`
	}
	ownRouting := otcOwnRoutingInt()
	b := body{
		Stock:          stock{Ticker: req.Ticker},
		SettlementDate: req.SettlementDate,
		PricePerUnit:   money{Currency: req.Currency, Amount: req.PricePerStock},
		Premium:        money{Currency: req.Currency, Amount: req.Premium},
		BuyerID:        partyID{RoutingNumber: ownRouting, ID: fmt.Sprintf("%d", req.BuyerId)},
		SellerID:       partyID{RoutingNumber: int(req.SellerRoutingNumber), ID: req.SellerExternalId},
		Amount:         req.Amount,
		SellerType:     req.SellerType,
	}
	data, _ := json.Marshal(b)
	httpReq, err := http.NewRequest(http.MethodPost, bank.BankURL+"/otc/interbank/negotiations", bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", bank.APIKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("http call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("partner returned status %d", resp.StatusCode)
	}
	var result struct {
		RoutingNumber int32  `json:"routingNumber"`
		ID            string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}
	id, err := strconv.ParseInt(result.ID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse partner negotiation id: %w", err)
	}
	return id, nil
}

// createNegotiationCrossBank handles CreateNegotiation when the seller is on a partner bank.
func (s *OtcServer) createNegotiationCrossBank(ctx context.Context, req *pb.CreateNegotiationRequest) (*pb.NegotiationResponse, error) {
	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", req.SellerRoutingNumber))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot resolve seller bank: %v", err)
	}

	now := time.Now()
	var id int64
	if err = s.DB.QueryRowContext(ctx, `
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type,
			 amount, price_per_stock, settlement_date, premium, currency,
			 last_modified, modified_by_id, modified_by_type, status,
			 seller_routing_number, seller_external_id)
		VALUES ($1, 0, 'INTERBANK', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'PENDING_SELLER', $12, $13)
		RETURNING id`,
		req.Ticker, req.BuyerId, req.BuyerType,
		req.Amount, req.PricePerStock, req.SettlementDate, req.Premium, req.Currency,
		now, req.BuyerId, req.BuyerType,
		req.SellerRoutingNumber, req.SellerExternalId,
	).Scan(&id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create cross-bank negotiation: %v", err)
	}

	partnerNegID, fwdErr := s.forwardNegotiationToPartner(bank, req)
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

// executeOtcAccept2PC sends a premium-payment 2PC to the partner bank during cross-bank accept.
// The OPTION posting uses a positive amount (reserve/accept), unlike exercise which uses negative.
func executeOtcAccept2PC(ctx context.Context, bank otcInterbank.BankInfo, req otcOutgoing2PCReq) (string, error) {
	bankURL := bank.BankURL + "/interbank"
	ownRouting := otcOwnRoutingInt()
	txID := otcIbTransactionID{RoutingNumber: ownRouting, ID: otcGenerateUUID()}
	idemKey := otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()}

	resp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: otcIbNewTxMessage{
			TransactionID: txID,
			Postings: []otcIbPosting{
				{AccountType: "PERSON", AccountNum: req.sellerExtID, Amount: +req.totalCost, AssetType: "MONAS", Currency: req.currency},
				{AccountType: "OPTION", AccountNum: fmt.Sprintf("%d", req.partnerNegotiationID), Amount: +float64(req.stockAmount)},
				{AccountType: "ACCOUNT", AccountNum: req.buyerAccountNum, Amount: -req.totalCost, AssetType: "MONAS", Currency: req.currency},
			},
		},
	})
	if err != nil {
		return "NO", fmt.Errorf("NEW_TX: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var vote otcIbVoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&vote); err != nil {
		return "NO", err
	}
	if vote.Vote != "YES" {
		return "NO", nil
	}

	commitResp, err := otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
		IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
		MessageType:    "COMMIT_TX",
		Message:        otcIbCommitMessage{TransactionID: txID},
	})
	if err != nil {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return "NO", fmt.Errorf("COMMIT_TX: %w", err)
	}
	defer func() { _ = commitResp.Body.Close() }()

	if commitResp.StatusCode != http.StatusNoContent {
		_, _ = otcSendInterbankRequest(ctx, bankURL, bank.APIKey, otcIbEnvelope{
			IdempotenceKey: otcIbIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: otcGenerateUUID()},
			MessageType:    "ROLLBACK_TX",
			Message:        otcIbCommitMessage{TransactionID: txID},
		})
		return "NO", fmt.Errorf("COMMIT_TX status %d", commitResp.StatusCode)
	}
	return "YES", nil
}

// acceptCrossBank handles AcceptNegotiation when the seller_type is 'INTERBANK'.
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
	var partnerNegotiationID int64
	var sellerExtID string
	if err := s.DB.QueryRowContext(ctx,
		`SELECT COALESCE(seller_routing_number, 0), COALESCE(partner_negotiation_id, 0),
		        COALESCE(seller_external_id, '')
		 FROM otc_negotiations WHERE id = $1`, negotiationID,
	).Scan(&sellerRoutingNum, &partnerNegotiationID, &sellerExtID); err != nil || sellerRoutingNum == 0 {
		return nil, status.Error(codes.Internal, "cross-bank accept: missing seller routing info")
	}

	bank, err := otcInterbank.ResolveBankByRoutingNumber(fmt.Sprintf("%d", sellerRoutingNum))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot resolve seller bank: %v", err)
	}

	// Call partner bank's accept endpoint.
	// Partner identifies the negotiation by creator_routing_number + creator_external_id,
	// which was set to OWN_ROUTING_NUMBER + buyerID when we forwarded the negotiation.
	acceptURL := fmt.Sprintf("%s/otc/interbank/negotiations/%s/%d/accept",
		bank.BankURL, os.Getenv("OWN_ROUTING_NUMBER"), buyerID)
	acceptReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, acceptURL, nil)
	acceptReq.Header.Set("X-Api-Key", bank.APIKey)
	acceptResp, acceptErr := (&http.Client{Timeout: 10 * time.Second}).Do(acceptReq)
	if acceptErr != nil {
		return nil, status.Errorf(codes.Unavailable, "partner accept call failed: %v", acceptErr)
	}
	_ = acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusNoContent && acceptResp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.FailedPrecondition, "partner accept returned %d", acceptResp.StatusCode)
	}

	// If premium > 0: debit buyer locally, then 2PC to partner for PERSON credit + OPTION reserve.
	if premium > 0 {
		currencyID, ok := currencyIDMap[currency]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", currency)
		}
		buyerAccountID := req.BuyerAccountId
		if buyerAccountID == 0 {
			buyerAccountID, err = findAccount(s.AccountDB, portfolioUserID(buyerID, buyerType), currencyID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find buyer account: %v", err)
			}
		}
		var buyerBalance float64
		if err = s.AccountDB.QueryRowContext(ctx,
			`SELECT available_balance FROM accounts WHERE id = $1`, buyerAccountID,
		).Scan(&buyerBalance); err != nil || buyerBalance < premium {
			return nil, status.Error(codes.InvalidArgument, "Insufficient funds for premium")
		}
		var buyerAccountNum string
		if err = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number FROM accounts WHERE id = $1`, buyerAccountID,
		).Scan(&buyerAccountNum); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load buyer account: %v", err)
		}

		if _, err = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			premium, buyerAccountID,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to debit buyer premium: %v", err)
		}

		voteResult, commitErr := executeOtcAccept2PC(ctx, bank, otcOutgoing2PCReq{
			sellerExtID:          sellerExtID,
			partnerNegotiationID: partnerNegotiationID,
			stockAmount:          amount,
			buyerAccountNum:      buyerAccountNum,
			totalCost:            premium,
			currency:             currency,
		})
		if commitErr != nil || voteResult != "YES" {
			retryExec(s.AccountDB,
				`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
				premium, buyerAccountID)
			return nil, status.Errorf(codes.FailedPrecondition,
				"partner rejected accept 2PC (vote=%s err=%v)", voteResult, commitErr)
		}
	}

	now := time.Now()
	settlDate, _ := time.Parse("2006-01-02", settlementDate)
	var contractID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO otc_contracts
			(negotiation_id, seller_id, seller_type, buyer_id, buyer_type,
			 ticker, amount, strike_price, premium, currency, settlement_date)
		VALUES ($1, $2, 'INTERBANK', $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		negotiationID, sellerID, buyerID, buyerType,
		ticker, amount, strikePrice, premium, currency, settlDate,
	).Scan(&contractID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create contract: %v", err)
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE otc_negotiations SET status='ACCEPTED', last_modified=$1 WHERE id=$2`,
		now, negotiationID,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update negotiation: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}
	return s.fetchNegotiationByID(ctx, negotiationID)
}
