package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service/utils"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// bankIIN is the bank's 5-digit Issuer Identification Number embedded in every card number.
// Change this to the bank's real registered IIN before going to production.
const bankIIN = "26500"

type CardServer struct {
	pb.UnimplementedCardServiceServer
	DB        *sql.DB // card_db
	AccountDB *sql.DB // account_db (for cross-DB lookups)
}

// ── CreateCard ────────────────────────────────────────────────────────────────

func (s *CardServer) CreateCard(ctx context.Context, req *pb.CreateCardRequest) (*pb.CreateCardResponse, error) {
	if req.AccountNumber == "" || req.CardName == "" {
		return nil, status.Error(codes.InvalidArgument, "account_number and card_name are required")
	}

	// 1. Get account type
	accountType, err := s.getAccountType(ctx, req.AccountNumber)
	if err != nil {
		return nil, err
	}

	// 2. Check card limit
	var existingCount int
	switch accountType {
	case "CURRENT", "SAVINGS", "FOREIGN_CURRENCY":
		existingCount, err = s.countAllCards(ctx, req.AccountNumber)
		if err != nil {
			return nil, err
		}
		if err := utils.CheckCardLimit("PERSONAL", true, existingCount); err != nil {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
	case "BUSINESS":
		if req.ForSelf {
			existingCount, err = s.countOwnerCards(ctx, req.AccountNumber)
			if err != nil {
				return nil, err
			}
		}
		if err := utils.CheckCardLimit("BUSINESS", req.ForSelf, existingCount); err != nil {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
	}

	// 3. Insert authorized person if needed
	var authorizedPersonID *int64
	if !req.ForSelf {
		if req.AuthorizedPerson == nil {
			return nil, status.Error(codes.InvalidArgument, "authorized_person data is required when for_self = false")
		}
		ap := req.AuthorizedPerson
		dob, err := time.Parse("2006-01-02", ap.DateOfBirth)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid date_of_birth format: %v", err)
		}
		var apID int64
		err = s.DB.QueryRowContext(ctx, `
			INSERT INTO authorized_persons (first_name, last_name, date_of_birth, gender, email, phone_number, address, account_number)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			ap.FirstName, ap.LastName, dob.Format("2006-01-02"), ap.Gender,
			ap.Email, ap.PhoneNumber, ap.Address, req.AccountNumber,
		).Scan(&apID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to insert authorized person: %v", err)
		}
		authorizedPersonID = &apID
	}

	// 4. Generate unique card number (retry on collision)
	var cardNumber string
	for {
		cardNumber = utils.GenerateCardNumber(req.CardName, bankIIN)
		var exists bool
		err := s.DB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM cards WHERE card_number = $1)`, cardNumber,
		).Scan(&exists)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check card number uniqueness: %v", err)
		}
		if !exists {
			break
		}
	}

	// 5. Generate CVV (hashed) and expiry date
	cvv := utils.GenerateCVV()
	cvvHash, err := utils.HashCVV(cvv)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash CVV: %v", err)
	}
	expiryDate := utils.GenerateExpiryDate()

	// 6. Insert card
	var card struct {
		id            int64
		cardType      string
		createdAt     time.Time
		cardLimit     sql.NullFloat64
		status        string
	}
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO cards (card_number, card_type, card_name, expiry_date, account_number, cvv, status, authorized_person_id)
		VALUES ($1, 'DEBIT', $2, $3, $4, $5, 'ACTIVE', $6)
		RETURNING id, card_type, created_at, card_limit, status`,
		cardNumber, req.CardName, expiryDate.Format("2006-01-02"),
		req.AccountNumber, cvvHash, authorizedPersonID,
	).Scan(&card.id, &card.cardType, &card.createdAt, &card.cardLimit, &card.status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert card: %v", err)
	}

	resp := &pb.CardResponse{
		Id:            card.id,
		CardNumber:    cardNumber,
		CardType:      card.cardType,
		CardName:      req.CardName,
		ExpiryDate:    expiryDate.Format("2006-01-02"),
		AccountNumber: req.AccountNumber,
		Status:        card.status,
		CreatedAt:     card.createdAt.Format(time.RFC3339),
	}
	if card.cardLimit.Valid {
		resp.CardLimit = card.cardLimit.Float64
	}
	return &pb.CreateCardResponse{Card: resp}, nil
}

// ── GetCardsByAccount ─────────────────────────────────────────────────────────

func (s *CardServer) GetCardsByAccount(ctx context.Context, req *pb.GetCardsByAccountRequest) (*pb.GetCardsByAccountResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, card_number, card_type, card_name, expiry_date, account_number, card_limit, status, created_at
		FROM cards WHERE account_number = $1 ORDER BY created_at DESC`,
		req.AccountNumber,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query cards: %v", err)
	}
	defer rows.Close()

	var cards []*pb.CardResponse
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan card: %v", err)
		}
		c.CardNumber = maskCardNumber(c.CardNumber)
		cards = append(cards, c)
	}
	return &pb.GetCardsByAccountResponse{Cards: cards}, nil
}

// ── GetCardByNumber ───────────────────────────────────────────────────────────

func (s *CardServer) GetCardByNumber(ctx context.Context, req *pb.GetCardByNumberRequest) (*pb.GetCardByNumberResponse, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, card_number, card_type, card_name, expiry_date, account_number, card_limit, status, created_at
		FROM cards WHERE card_number = $1`,
		req.CardNumber,
	)
	c, err := scanCard(row)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "card %s not found", req.CardNumber)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query card: %v", err)
	}
	return &pb.GetCardByNumberResponse{Card: c}, nil
}

// ── GetCardById ───────────────────────────────────────────────────────────────

func (s *CardServer) GetCardById(ctx context.Context, req *pb.GetCardByIdRequest) (*pb.GetCardByIdResponse, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, card_number, card_type, card_name, expiry_date, account_number, card_limit, status, created_at
		FROM cards WHERE id = $1`,
		req.Id,
	)
	c, err := scanCard(row)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "card %d not found", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query card: %v", err)
	}
	return &pb.GetCardByIdResponse{Card: c}, nil
}

// ── BlockCard ─────────────────────────────────────────────────────────────────

func (s *CardServer) BlockCard(ctx context.Context, req *pb.BlockCardRequest) (*pb.BlockCardResponse, error) {
	cardStatus, accountNumber, err := s.fetchCardStatusAndAccount(ctx, req.CardNumber)
	if err != nil {
		return nil, err
	}

	if req.CallerRole == "CLIENT" {
		ownerID, err := s.getAccountOwnerID(ctx, accountNumber)
		if err != nil {
			return nil, err
		}
		if ownerID != req.CallerClientId {
			return nil, status.Error(codes.PermissionDenied, "you do not own this card")
		}
	}

	if cardStatus != "ACTIVE" {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot block card with status %s", cardStatus)
	}

	_, err = s.DB.ExecContext(ctx, `UPDATE cards SET status = 'BLOCKED' WHERE card_number = $1`, req.CardNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to block card: %v", err)
	}
	return &pb.BlockCardResponse{}, nil
}

// ── UnblockCard ───────────────────────────────────────────────────────────────

func (s *CardServer) UnblockCard(ctx context.Context, req *pb.UnblockCardRequest) (*pb.UnblockCardResponse, error) {
	cardStatus, _, err := s.fetchCardStatusAndAccount(ctx, req.CardNumber)
	if err != nil {
		return nil, err
	}

	if cardStatus == "DEACTIVATED" {
		return nil, status.Error(codes.FailedPrecondition, "cannot unblock a deactivated card")
	}
	if cardStatus != "BLOCKED" {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot unblock card with status %s", cardStatus)
	}

	_, err = s.DB.ExecContext(ctx, `UPDATE cards SET status = 'ACTIVE' WHERE card_number = $1`, req.CardNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unblock card: %v", err)
	}
	return &pb.UnblockCardResponse{}, nil
}

// ── DeactivateCard ────────────────────────────────────────────────────────────

func (s *CardServer) DeactivateCard(ctx context.Context, req *pb.DeactivateCardRequest) (*pb.DeactivateCardResponse, error) {
	cardStatus, _, err := s.fetchCardStatusAndAccount(ctx, req.CardNumber)
	if err != nil {
		return nil, err
	}

	if cardStatus == "DEACTIVATED" {
		return nil, status.Error(codes.FailedPrecondition, "card is already deactivated")
	}

	_, err = s.DB.ExecContext(ctx, `UPDATE cards SET status = 'DEACTIVATED' WHERE card_number = $1`, req.CardNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to deactivate card: %v", err)
	}
	return &pb.DeactivateCardResponse{}, nil
}

// ── UpdateCardLimit ───────────────────────────────────────────────────────────

func (s *CardServer) UpdateCardLimit(ctx context.Context, req *pb.UpdateCardLimitRequest) (*pb.UpdateCardLimitResponse, error) {
	cardStatus, _, err := s.fetchCardStatusAndAccount(ctx, req.CardNumber)
	if err != nil {
		return nil, err
	}

	if cardStatus == "DEACTIVATED" {
		return nil, status.Error(codes.FailedPrecondition, "cannot update limit on a deactivated card")
	}

	_, err = s.DB.ExecContext(ctx, `UPDATE cards SET card_limit = $1 WHERE card_number = $2`, req.NewLimit, req.CardNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update card limit: %v", err)
	}
	return &pb.UpdateCardLimitResponse{}, nil
}

// ── InitiateCardRequest ───────────────────────────────────────────────────────

func (s *CardServer) InitiateCardRequest(ctx context.Context, req *pb.InitiateCardRequestRequest) (*pb.InitiateCardRequestResponse, error) {
	if req.AccountNumber == "" || req.CardName == "" {
		return nil, status.Error(codes.InvalidArgument, "account_number and card_name are required")
	}

	accountType, err := s.getAccountType(ctx, req.AccountNumber)
	if err != nil {
		return nil, err
	}

	// Pre-check limits (same logic as CreateCard)
	var existingCount int
	switch accountType {
	case "CURRENT", "SAVINGS", "FOREIGN_CURRENCY":
		existingCount, err = s.countAllCards(ctx, req.AccountNumber)
		if err != nil {
			return nil, err
		}
		if err := utils.CheckCardLimit("PERSONAL", true, existingCount); err != nil {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
	case "BUSINESS":
		if req.ForSelf {
			existingCount, err = s.countOwnerCards(ctx, req.AccountNumber)
			if err != nil {
				return nil, err
			}
		}
		if err := utils.CheckCardLimit("BUSINESS", req.ForSelf, existingCount); err != nil {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
	}

	// Generate 6-digit confirmation code
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate code: %v", err)
	}

	// Serialize authorized person data if present
	var apDataJSON []byte
	if !req.ForSelf && req.AuthorizedPerson != nil {
		apDataJSON, err = json.Marshal(req.AuthorizedPerson)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to serialize authorized person: %v", err)
		}
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO card_requests
			(request_token, account_number, card_name, caller_client_id, for_self, authorized_person_data, confirmation_code, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		token, req.AccountNumber, req.CardName, req.CallerClientId,
		req.ForSelf, apDataJSON, code, expiresAt,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store card request: %v", err)
	}

	return &pb.InitiateCardRequestResponse{
		RequestToken:    token,
		ConfirmationCode: code,
	}, nil
}

// ── ConfirmCardRequest ────────────────────────────────────────────────────────

func (s *CardServer) ConfirmCardRequest(ctx context.Context, req *pb.ConfirmCardRequestRequest) (*pb.ConfirmCardRequestResponse, error) {
	if req.RequestToken == "" || req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "request_token and code are required")
	}

	var (
		accountNumber   string
		cardName        string
		callerClientID  int64
		forSelf         bool
		apDataJSON      []byte
		storedCode      string
		expiresAt       time.Time
		used            bool
	)
	err := s.DB.QueryRowContext(ctx, `
		SELECT account_number, card_name, caller_client_id, for_self,
		       authorized_person_data, confirmation_code, expires_at, used
		FROM card_requests WHERE request_token = $1`, req.RequestToken,
	).Scan(&accountNumber, &cardName, &callerClientID, &forSelf,
		&apDataJSON, &storedCode, &expiresAt, &used)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "card request not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch card request: %v", err)
	}
	if used {
		return nil, status.Error(codes.FailedPrecondition, "confirmation code already used")
	}
	if time.Now().After(expiresAt) {
		return nil, status.Error(codes.FailedPrecondition, "confirmation code expired")
	}
	if req.Code != storedCode {
		return nil, status.Error(codes.PermissionDenied, "invalid confirmation code")
	}

	// Mark as used
	if _, err := s.DB.ExecContext(ctx, `UPDATE card_requests SET used = true WHERE request_token = $1`, req.RequestToken); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark request as used: %v", err)
	}

	// Reconstruct AuthorizedPersonData if needed
	var apData *pb.AuthorizedPersonData
	if !forSelf && len(apDataJSON) > 0 {
		apData = &pb.AuthorizedPersonData{}
		if err := json.Unmarshal(apDataJSON, apData); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to deserialize authorized person: %v", err)
		}
	}

	// Delegate to CreateCard
	createResp, err := s.CreateCard(ctx, &pb.CreateCardRequest{
		AccountNumber:   accountNumber,
		CardName:        cardName,
		CallerClientId:  callerClientID,
		ForSelf:         forSelf,
		AuthorizedPerson: apData,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ConfirmCardRequestResponse{Card: createResp.Card}, nil
}

// generateConfirmationCode returns a random 6-digit string (with leading zeros).
func generateConfirmationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// fetchCardStatusAndAccount fetches a card's status and account_number, returning NotFound if missing.
func (s *CardServer) fetchCardStatusAndAccount(ctx context.Context, cardNumber string) (string, string, error) {
	var cardStatus, accountNumber string
	err := s.DB.QueryRowContext(ctx,
		`SELECT status, account_number FROM cards WHERE card_number = $1`, cardNumber,
	).Scan(&cardStatus, &accountNumber)
	if err == sql.ErrNoRows {
		return "", "", status.Errorf(codes.NotFound, "card %s not found", cardNumber)
	}
	if err != nil {
		return "", "", status.Errorf(codes.Internal, "failed to fetch card: %v", err)
	}
	return cardStatus, accountNumber, nil
}

// getAccountOwnerID returns the owner_id of an account from account_db.
func (s *CardServer) getAccountOwnerID(ctx context.Context, accountNumber string) (int64, error) {
	var ownerID int64
	err := s.AccountDB.QueryRowContext(ctx,
		`SELECT owner_id FROM accounts WHERE account_number = $1`, accountNumber,
	).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return 0, status.Errorf(codes.NotFound, "account %s not found", accountNumber)
	}
	if err != nil {
		return 0, status.Errorf(codes.Internal, "failed to query account owner: %v", err)
	}
	return ownerID, nil
}

// maskCardNumber returns the card number with middle 8 digits replaced by asterisks.
// e.g. "5798123456785571" → "5798********5571"
func maskCardNumber(n string) string {
	if len(n) < 8 {
		return n
	}
	return fmt.Sprintf("%s********%s", n[:4], n[len(n)-4:])
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanCard(s scanner) (*pb.CardResponse, error) {
	var (
		c         pb.CardResponse
		expiryDate time.Time
		createdAt  time.Time
		cardLimit  sql.NullFloat64
	)
	err := s.Scan(
		&c.Id, &c.CardNumber, &c.CardType, &c.CardName,
		&expiryDate, &c.AccountNumber, &cardLimit, &c.Status, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	c.ExpiryDate = expiryDate.Format("2006-01-02")
	c.CreatedAt = createdAt.Format(time.RFC3339)
	if cardLimit.Valid {
		c.CardLimit = cardLimit.Float64
	}
	return &c, nil
}
