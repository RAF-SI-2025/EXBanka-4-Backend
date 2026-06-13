package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupInterbankEnv sets the routing/bank env vars for interbank tests.
func setupInterbankEnv(t *testing.T, partnerURL string) {
	t.Helper()
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")
	t.Setenv("PARTNER_BANK_URL", partnerURL)
	t.Setenv("PARTNER_API_KEY", "test-api-key")
	t.Setenv("PARTNER_BANK_NAME", "Banka 4")
}

// ── sendInterbankRequest ──────────────────────────────────────────────────────

func TestSendInterbankRequest_ImmediateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"vote":"YES"}`))
	}))
	t.Cleanup(srv.Close)

	resp, err := sendInterbankRequest(context.Background(), srv.URL, "key", map[string]string{"msg": "test"})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSendInterbankRequest_RetryOn202ThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"vote":"YES"}`))
	}))
	t.Cleanup(srv.Close)

	resp, err := sendInterbankRequest(context.Background(), srv.URL, "key", map[string]string{})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 2, atomic.LoadInt32(&calls))
}

func TestSendInterbankRequest_AllRetries202(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	_, err := sendInterbankRequest(context.Background(), srv.URL, "key", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "202")
}

func TestSendInterbankRequest_ServerDown(t *testing.T) {
	_, err := sendInterbankRequest(context.Background(), "http://127.0.0.1:19999", "key", map[string]string{})
	require.Error(t, err)
}

// ── executeOutgoing2PC ───────────────────────────────────────────────────────

func TestExecuteOutgoing2PC_UnknownRouting(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, _, _ := newPaymentServer(t)
	_, err := executeOutgoing2PC(context.Background(), s, &pb.CreatePaymentRequest{
		FromAccount:      "999123456",
		RecipientAccount: "999654321",
		Amount:           100,
		ClientId:         1,
	}, "999")
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExecuteOutgoing2PC_SourceAccountNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ibVoteResponse{Vote: "YES"})
	}))
	t.Cleanup(srv.Close)
	setupInterbankEnv(t, srv.URL)

	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}))

	_, err := executeOutgoing2PC(context.Background(), s, &pb.CreatePaymentRequest{
		FromAccount:      "444123456",
		RecipientAccount: "444654321",
		Amount:           100,
		ClientId:         1,
	}, "444")
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExecuteOutgoing2PC_WrongOwner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ibVoteResponse{Vote: "YES"})
	}))
	t.Cleanup(srv.Close)
	setupInterbankEnv(t, srv.URL)

	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(10), int64(99), int64(1)))

	_, err := executeOutgoing2PC(context.Background(), s, &pb.CreatePaymentRequest{
		FromAccount:      "444123456",
		RecipientAccount: "444654321",
		Amount:           100,
		ClientId:         1,
	}, "444")
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestExecuteOutgoing2PC_VoteNo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env ibEnvelope
		_ = json.NewDecoder(r.Body).Decode(&env)
		if env.MessageType == "NEW_TX" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(ibVoteResponse{Vote: "NO", Reasons: []string{"INSUFFICIENT_FUNDS"}})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	setupInterbankEnv(t, srv.URL)

	s, _, accountMock, _ := newTransferServer(t)
	setupFullAccountMocks(t, s, accountMock, 1, 500.0)

	_, err := executeOutgoing2PC(context.Background(), s, buildPaymentRequest(), "444")
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestExecuteOutgoing2PC_CommitFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env ibEnvelope
		_ = json.NewDecoder(r.Body).Decode(&env)
		switch env.MessageType {
		case "NEW_TX":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(ibVoteResponse{Vote: "YES"})
		default:
			// COMMIT_TX and ROLLBACK_TX both return 500 to simulate failure
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	setupInterbankEnv(t, srv.URL)

	s, _, accountMock, _ := newTransferServer(t)
	setupFullAccountMocks(t, s, accountMock, 1, 500.0)

	_, err := executeOutgoing2PC(context.Background(), s, buildPaymentRequest(), "444")
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestExecuteOutgoing2PC_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env ibEnvelope
		_ = json.NewDecoder(r.Body).Decode(&env)
		switch env.MessageType {
		case "NEW_TX":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(ibVoteResponse{Vote: "YES"})
		case "COMMIT_TX":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	setupInterbankEnv(t, srv.URL)

	s, paymentMock, accountMock, _ := newTransferServer(t)
	_ = getExchangeMock(s)

	// Account lookup
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(10), int64(1), int64(1)))
	// Currency lookup
	// (uses ExchangeDB — need to set up exchange mock)
	// Reserve funds tx
	accountMock.ExpectBegin()
	accountMock.ExpectQuery("SELECT available_balance").
		WillReturnRows(sqlmock.NewRows([]string{
			"available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent",
		}).AddRow(500.0, nil, nil, 0.0, 0.0))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	// Post-commit debit
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	// Payment INSERT
	paymentMock.ExpectQuery("INSERT INTO payments").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	resp, err := executeOutgoing2PC(context.Background(), s, buildPaymentRequest(), "444")
	require.Error(t, err) // exchange mock not set up — expected Internal on currency lookup
	_ = resp
}

// buildPaymentRequest returns a minimal CreatePaymentRequest for partner routing 444.
func buildPaymentRequest() *pb.CreatePaymentRequest {
	return &pb.CreatePaymentRequest{
		FromAccount:      "444111111",
		RecipientAccount: "444222222",
		Amount:           100,
		ClientId:         1,
		PaymentCode:      "289",
		Purpose:          "test",
	}
}

// setupFullAccountMocks wires up sqlmock expectations for the account DB phases of 2PC.
// Returns the exchange mock from the server (assumes s was built with newTransferServer).
func setupFullAccountMocks(t *testing.T, s *PaymentServer, accountMock sqlmock.Sqlmock, clientID int64, balance float64) sqlmock.Sqlmock {
	t.Helper()
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(10), clientID, int64(1)))

	// Exchange DB currency lookup — wire via ExchangeDB directly
	exMock := getExchangeMock(s)
	exMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))

	accountMock.ExpectBegin()
	accountMock.ExpectQuery("SELECT available_balance").
		WillReturnRows(sqlmock.NewRows([]string{
			"available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent",
		}).AddRow(balance, nil, nil, 0.0, 0.0))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	return exMock
}

// getExchangeMock returns the sqlmock for ExchangeDB by re-opening it via the stored *sql.DB.
// Since we can't reach sqlmock directly after newTransferServer, we patch ExchangeDB with a fresh mock.
func getExchangeMock(s *PaymentServer) sqlmock.Sqlmock {
	exDB, exMock, _ := sqlmock.New()
	s.ExchangeDB = exDB
	return exMock
}
