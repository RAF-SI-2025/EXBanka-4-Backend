package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newPaymentServer returns a PaymentServer backed by two independent sqlmock DBs.
// Used by tests for CreatePayment, CreatePaymentRecipient, GetPaymentRecipients,
// UpdatePaymentRecipient, DeletePaymentRecipient.
func newPaymentServer(t *testing.T) (*PaymentServer, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	s := &PaymentServer{DB: db, AccountDB: accountDB}
	t.Cleanup(func() { db.Close(); accountDB.Close() })
	return s, dbMock, accountMock
}

// newTransferServer returns a PaymentServer backed by three independent sqlmock DBs
// (payment_db, account_db, exchange_db) for CreateTransfer tests.
func newTransferServer(t *testing.T) (*PaymentServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	exchangeDB, exchangeMock, err := sqlmock.New()
	require.NoError(t, err)
	s := &PaymentServer{DB: db, AccountDB: accountDB, ExchangeDB: exchangeDB}
	t.Cleanup(func() { db.Close(); accountDB.Close(); exchangeDB.Close() })
	return s, dbMock, accountMock, exchangeMock
}

// newMockServer is an alias used by GetPaymentById and GetPayments tests.
func newMockServer(t *testing.T) (*PaymentServer, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	paymentDB, paymentMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		paymentDB.Close()
		accountDB.Close()
	})
	return &PaymentServer{DB: paymentDB, AccountDB: accountDB}, paymentMock, accountMock
}

// newMockServerWithClientDB returns a PaymentServer backed by three sqlmock DBs:
// payment_db, account_db, and client_db — for tests that exercise sender info lookup.
func newMockServerWithClientDB(t *testing.T) (*PaymentServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	paymentDB, paymentMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	clientDB, clientMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { paymentDB.Close(); accountDB.Close(); clientDB.Close() })
	return &PaymentServer{DB: paymentDB, AccountDB: accountDB, ClientDB: clientDB}, paymentMock, accountMock, clientMock
}

// ---- CreatePayment ----

func TestCreatePayment_SourceNotFound(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnError(sql.ErrNoRows)

	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "123", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreatePayment_WrongOwner(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(99), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)

	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreatePayment_InsufficientFunds(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(50), nil, nil, float64(0), float64(0), int64(1)),
	)

	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreatePayment_DailyLimitExceeded(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(5000), sql.NullFloat64{Float64: 500, Valid: true}, nil, float64(450), float64(0), int64(1)),
	)

	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreatePayment_MonthlyLimitExceeded(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(5000), nil, sql.NullFloat64{Float64: 2000, Valid: true}, float64(0), float64(1950), int64(1)),
	)

	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreatePayment_HappyPath_SameCurrency(t *testing.T) {
	s, dbMock, accountMock := newPaymentServer(t)

	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "currency_id"}).AddRow(int64(2), int64(1)),
	)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO payments").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(1)),
	)

	resp, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", RecipientAccount: "ACC2", ClientId: 1, Amount: 200,
		PaymentCode: "289", Purpose: "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.Fee, "fee must be 0 for same currency")
	assert.Equal(t, float64(200), resp.FinalAmount)
	assert.Equal(t, "COMPLETED", resp.Status)
}

func TestCreatePayment_HappyPath_ExternalAccount(t *testing.T) {
	s, dbMock, accountMock := newPaymentServer(t)

	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnError(sql.ErrNoRows)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO payments").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(2)),
	)

	resp, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", RecipientAccount: "EXTERNAL123", ClientId: 1, Amount: 150,
		PaymentCode: "289", Purpose: "Eksterno",
	})
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", resp.Status)
}

func TestCreatePayment_SourceInternalError(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnError(sql.ErrConnDone)
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePayment_HappyPath_DifferentCurrency(t *testing.T) {
	s, dbMock, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "currency_id"}).AddRow(int64(2), int64(2)),
	)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO payments").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(1)),
	)
	resp, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", RecipientAccount: "ACC2", ClientId: 1, Amount: 200,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Fee, float64(0))
	assert.Less(t, resp.FinalAmount, float64(200)+0.001, "finalAmount <= amount for different currency")
}

func TestCreatePayment_BeginTxError(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnError(sql.ErrNoRows)
	accountMock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePayment_DebitError(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnError(sql.ErrNoRows)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnError(sql.ErrConnDone)
	accountMock.ExpectRollback()
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePayment_CreditError(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "currency_id"}).AddRow(int64(2), int64(1)),
	)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnError(sql.ErrConnDone)
	accountMock.ExpectRollback()
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", RecipientAccount: "ACC2", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePayment_CommitError(t *testing.T) {
	s, _, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnError(sql.ErrNoRows)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", RecipientAccount: "EXTERNAL", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePayment_PersistError(t *testing.T) {
	s, dbMock, accountMock := newPaymentServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnError(sql.ErrNoRows)
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO payments").WillReturnError(sql.ErrConnDone)
	_, err := s.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		FromAccount: "ACC1", ClientId: 1, Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- CreatePaymentRecipient ----

func TestCreatePaymentRecipient_DBError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("INSERT INTO payment_recipients").WillReturnError(sql.ErrConnDone)

	_, err := s.CreatePaymentRecipient(context.Background(), &pb.CreatePaymentRecipientRequest{
		ClientId: 1, Name: "Ana", AccountNumber: "ACC1",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreatePaymentRecipient_HappyPath(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("INSERT INTO payment_recipients").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(1)),
	)

	resp, err := s.CreatePaymentRecipient(context.Background(), &pb.CreatePaymentRecipientRequest{
		ClientId: 1, Name: "Ana", AccountNumber: "ACC1",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Recipient.Id)
	assert.Equal(t, "Ana", resp.Recipient.Name)
}

// ---- GetPaymentRecipients ----

func TestGetPaymentRecipients_DBError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("SELECT id, client_id, name, account_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetPaymentRecipients(context.Background(), &pb.GetPaymentRecipientsRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPaymentRecipients_Empty(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("SELECT id, client_id, name, account_number").WillReturnRows(
		sqlmock.NewRows([]string{"id", "client_id", "name", "account_number"}),
	)

	resp, err := s.GetPaymentRecipients(context.Background(), &pb.GetPaymentRecipientsRequest{ClientId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Recipients)
}

func TestGetPaymentRecipients_HappyPath(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("SELECT id, client_id, name, account_number").WillReturnRows(
		sqlmock.NewRows([]string{"id", "client_id", "name", "account_number"}).
			AddRow(int64(1), int64(5), "Ana", "ACC1").
			AddRow(int64(2), int64(5), "Marko", "ACC2"),
	)

	resp, err := s.GetPaymentRecipients(context.Background(), &pb.GetPaymentRecipientsRequest{ClientId: 5})
	require.NoError(t, err)
	assert.Len(t, resp.Recipients, 2)
	assert.Equal(t, "Ana", resp.Recipients[0].Name)
}

func TestGetPaymentRecipients_ScanError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("SELECT id, client_id, name, account_number").WillReturnRows(
		sqlmock.NewRows([]string{"id", "client_id", "name", "account_number"}).
			AddRow("not-an-int", 1, "Ana", "ACC1"),
	)
	_, err := s.GetPaymentRecipients(context.Background(), &pb.GetPaymentRecipientsRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- UpdatePaymentRecipient ----

func TestUpdatePaymentRecipient_NotFound(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("UPDATE payment_recipients").WillReturnError(sql.ErrNoRows)

	_, err := s.UpdatePaymentRecipient(context.Background(), &pb.UpdatePaymentRecipientRequest{
		Id: 99, ClientId: 1, Name: "Novi", AccountNumber: "ACC1",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdatePaymentRecipient_HappyPath(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("UPDATE payment_recipients").WillReturnRows(
		sqlmock.NewRows([]string{"id", "client_id", "name", "account_number"}).
			AddRow(int64(1), int64(1), "Novi naziv", "ACC1"),
	)

	resp, err := s.UpdatePaymentRecipient(context.Background(), &pb.UpdatePaymentRecipientRequest{
		Id: 1, ClientId: 1, Name: "Novi naziv", AccountNumber: "ACC1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Novi naziv", resp.Recipient.Name)
}

func TestUpdatePaymentRecipient_InternalError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("UPDATE payment_recipients").WillReturnError(sql.ErrConnDone)
	_, err := s.UpdatePaymentRecipient(context.Background(), &pb.UpdatePaymentRecipientRequest{
		Id: 1, ClientId: 1, Name: "X", AccountNumber: "ACC1",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- DeletePaymentRecipient ----

func TestDeletePaymentRecipient_NotFound(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("UPDATE payments SET recipient_id").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 99, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeletePaymentRecipient_HappyPath(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("UPDATE payments SET recipient_id").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 1, ClientId: 1})
	require.NoError(t, err)
}

func TestDeletePaymentRecipient_ExecError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("UPDATE payments SET recipient_id").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnError(sql.ErrConnDone)
	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 1, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetPaymentById ----

func TestGetPaymentById_HappyPathWithRecipient(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(1, "ORD-001", "ACC-100", "ACC-200", 300.0, 300.0, 0.0, "289", "RF01", "rent", ts, "COMPLETED", "Ana Petrović"))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 1, ClientId: 42})
	require.NoError(t, err)
	p := resp.Payment
	assert.Equal(t, int64(1), p.Id)
	assert.Equal(t, "Ana Petrović", p.RecipientName)
	assert.Equal(t, 300.0, p.InitialAmount)
	assert.Equal(t, ts.Format(time.RFC3339), p.Timestamp)
}

func TestGetPaymentById_HappyPathNoRecipient(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(2, "ORD-002", "ACC-100", "EXT-999", 150.0, 150.0, 0.0, "", "", "phone", ts, "COMPLETED", nil))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 2, ClientId: 42})
	require.NoError(t, err)
	assert.Equal(t, "", resp.Payment.RecipientName)
}

func TestGetPaymentById_NotFound(t *testing.T) {
	s, paymentMock, _ := newMockServer(t)

	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{}))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 999, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetPaymentById_PermissionDenied(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(3, "ORD-003", "ACC-500", "ACC-600", 100.0, 100.0, 0.0, "", "", "", ts, "COMPLETED", nil))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-500").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(99)))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 3, ClientId: 42})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetPaymentById_PaymentDBError(t *testing.T) {
	s, paymentMock, _ := newMockServer(t)

	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(4)).
		WillReturnError(fmt.Errorf("db unavailable"))

	_, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 4, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPaymentById_IncomingPayment_SenderInfo(t *testing.T) {
	s, paymentMock, accountMock, clientMock := newMockServerWithClientDB(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(10, "ORD-010", "EXT-001", "ACC-100", 500.0, 500.0, 0.0, "", "", "gift", ts, "COMPLETED", nil))

	// EXT-001 belongs to owner 55 (not client 42)
	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("EXT-001").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(55)))
	// ACC-100 belongs to owner 42 (our client)
	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))
	// Client lookup for sender (owner 55)
	clientMock.ExpectQuery("SELECT first_name").
		WithArgs(int64(55)).
		WillReturnRows(sqlmock.NewRows([]string{"name", "address"}).AddRow("Marko Marković", "Bulevar 1"))

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 10, ClientId: 42})
	require.NoError(t, err)
	assert.Equal(t, "Marko Marković", resp.Payment.SenderName)
	assert.Equal(t, "Bulevar 1", resp.Payment.SenderAddress)
}

func TestGetPaymentById_IncomingPayment_ClientDBError(t *testing.T) {
	s, paymentMock, accountMock, clientMock := newMockServerWithClientDB(t)

	ts := time.Now()
	paymentMock.ExpectQuery("SELECT p.id, p.order_number").
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(11, "ORD-011", "EXT-002", "ACC-100", 100.0, 100.0, 0.0, "", "", "", ts, "COMPLETED", nil))

	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("EXT-002").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(77)))
	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("ACC-100").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(42)))
	// ClientDB lookup fails — payment still returned, sender info empty
	clientMock.ExpectQuery("SELECT first_name").
		WithArgs(int64(77)).
		WillReturnError(sql.ErrConnDone)

	resp, err := s.GetPaymentById(context.Background(), &pb.GetPaymentByIdRequest{PaymentId: 11, ClientId: 42})
	require.NoError(t, err)
	assert.Equal(t, "", resp.Payment.SenderName)
}

// ---- GetPayments ----

func TestGetPayments_NoAccounts(t *testing.T) {
	s, _, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 99})
	require.NoError(t, err)
	assert.Empty(t, resp.Payments)
	assert.NoError(t, accountMock.ExpectationsWereMet())
}

func TestGetPayments_AccountDBError(t *testing.T) {
	s, _, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("connection refused"))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_NoFilters(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).
			AddRow("ACC-001").AddRow("ACC-002"))

	ts := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(1, "ORD-001", "ACC-001", "EXT-999", 500.0, 500.0, 0.0, "289", "RF001", "rent", ts, "COMPLETED", nil))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	p := resp.Payments[0]
	assert.Equal(t, int64(1), p.Id)
	assert.Equal(t, "ORD-001", p.OrderNumber)
	assert.Equal(t, "ACC-001", p.FromAccount)
	assert.Equal(t, "EXT-999", p.ToAccount)
	assert.Equal(t, 500.0, p.InitialAmount)
	assert.Equal(t, "COMPLETED", p.Status)
	assert.Equal(t, ts.Format(time.RFC3339), p.Timestamp)

	assert.NoError(t, accountMock.ExpectationsWereMet())
	assert.NoError(t, paymentMock.ExpectationsWereMet())
}

func TestGetPayments_MultiplePayments(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-100"))

	ts1 := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "order_number", "from_account", "to_account",
		"initial_amount", "final_amount", "fee",
		"payment_code", "reference_number", "purpose",
		"timestamp", "status", "name",
	}).
		AddRow(10, "ORD-010", "ACC-100", "EXT-111", 200.0, 200.0, 0.0, "", "", "electricity", ts1, "COMPLETED", nil).
		AddRow(9, "ORD-009", "ACC-100", "EXT-222", 100.0, 99.5, 0.5, "", "", "phone", ts2, "COMPLETED", nil)

	paymentMock.ExpectQuery("FROM payments p").WillReturnRows(rows)

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 2})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 2)
	assert.Equal(t, int64(10), resp.Payments[0].Id)
	assert.Equal(t, int64(9), resp.Payments[1].Id)
}

func TestGetPayments_FilterByStatus(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-300"))

	ts := time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(5, "ORD-005", "ACC-300", "EXT-400", 150.0, 150.0, 0.0, "", "", "", ts, "PROCESSING", nil))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId: 3,
		Status:   "PROCESSING",
	})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	assert.Equal(t, "PROCESSING", resp.Payments[0].Status)
}

func TestGetPayments_FilterByDateRange(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-400"))

	ts := time.Date(2025, 2, 10, 12, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(7, "ORD-007", "ACC-400", "EXT-500", 300.0, 300.0, 0.0, "", "", "", ts, "COMPLETED", nil))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId: 4,
		DateFrom: "2025-01-01T00:00:00Z",
		DateTo:   "2025-03-01T00:00:00Z",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 1)
}

func TestGetPayments_FilterByAmountRange(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-500"))

	ts := time.Now()
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(8, "ORD-008", "ACC-500", "EXT-600", 250.0, 250.0, 0.0, "", "", "", ts, "COMPLETED", nil))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId:  5,
		AmountMin: 100.0,
		AmountMax: 500.0,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Payments, 1)
	assert.Equal(t, 250.0, resp.Payments[0].InitialAmount)
}

func TestGetPayments_EmptyResult(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(6)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-600"))

	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 6})
	require.NoError(t, err)
	assert.Empty(t, resp.Payments)
}

func TestGetPayments_PaymentDBError(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-700"))

	paymentMock.ExpectQuery("FROM payments p").
		WillReturnError(fmt.Errorf("query timeout"))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 7})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_ScanError(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-800"))

	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	_, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 8})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetPayments_AllFilters(t *testing.T) {
	s, paymentMock, accountMock := newMockServer(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-900"))

	ts := time.Date(2025, 7, 20, 0, 0, 0, 0, time.UTC)
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(20, "ORD-020", "ACC-900", "EXT-999", 400.0, 395.0, 5.0, "221", "RF020", "salary", ts, "COMPLETED", nil))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{
		ClientId:  9,
		DateFrom:  "2025-07-01T00:00:00Z",
		DateTo:    "2025-08-01T00:00:00Z",
		AmountMin: 100.0,
		AmountMax: 1000.0,
		Status:    "COMPLETED",
	})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	p := resp.Payments[0]
	assert.Equal(t, int64(20), p.Id)
	assert.Equal(t, 400.0, p.InitialAmount)
	assert.Equal(t, 5.0, p.Fee)
	assert.Equal(t, "salary", p.Purpose)
}

func TestGetPayments_IncomingPayment_SenderInfo(t *testing.T) {
	s, paymentMock, accountMock, clientMock := newMockServerWithClientDB(t)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-100"))

	ts := time.Now()
	paymentMock.ExpectQuery("FROM payments p").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "fee",
			"payment_code", "reference_number", "purpose",
			"timestamp", "status", "name",
		}).AddRow(20, "ORD-020", "EXT-999", "ACC-100", 300.0, 300.0, 0.0, "", "", "salary", ts, "COMPLETED", nil))

	// Incoming: EXT-999 → owner 55
	accountMock.ExpectQuery("SELECT owner_id FROM accounts").
		WithArgs("EXT-999").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(int64(55)))
	// Client lookup for owner 55
	clientMock.ExpectQuery("SELECT first_name").
		WithArgs(int64(55)).
		WillReturnRows(sqlmock.NewRows([]string{"name", "address"}).AddRow("Ana Anić", "Ulica 5"))

	resp, err := s.GetPayments(context.Background(), &pb.GetPaymentsRequest{ClientId: 42})
	require.NoError(t, err)
	require.Len(t, resp.Payments, 1)
	assert.Equal(t, "Ana Anić", resp.Payments[0].SenderName)
	assert.Equal(t, "Ulica 5", resp.Payments[0].SenderAddress)
}

// ---- CreateTransfer ----

func TestCreateTransfer_SameAccount(t *testing.T) {
	s, _, _, _ := newTransferServer(t)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC1", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateTransfer_SourceNotFound(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnError(sql.ErrNoRows)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateTransfer_SourceNotOwned(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(99), float64(500), int64(1)))
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreateTransfer_DestNotFound(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(500), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnError(sql.ErrNoRows)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateTransfer_DestNotOwned(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(500), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(99), int64(1)))
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreateTransfer_InsufficientFunds(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(50), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(1)))
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 100,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreateTransfer_SameCurrency_Happy(t *testing.T) {
	s, dbMock, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(500), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(1)))
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO transfers").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	resp, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 200,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(10), resp.Id)
	assert.Equal(t, 200.0, resp.InitialAmount)
	assert.Equal(t, 200.0, resp.FinalAmount)
	assert.Equal(t, 1.0, resp.ExchangeRate)
	assert.Equal(t, 0.0, resp.Fee)
}

func TestCreateTransfer_DifferentCurrency_Happy(t *testing.T) {
	s, dbMock, accountMock, exchangeMock := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(2)))
	// resolve from currency code
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	// resolve to currency code
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	// fetch rate
	exchangeMock.ExpectQuery("SELECT rate FROM exchange_rates").
		WillReturnRows(sqlmock.NewRows([]string{"rate"}).AddRow(0.008547))
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO transfers").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(11)))

	resp, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 1000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(11), resp.Id)
	assert.Equal(t, 1000.0, resp.InitialAmount)
	assert.InDelta(t, 8.547, resp.FinalAmount, 0.001)
	assert.Equal(t, 0.008547, resp.ExchangeRate)
}

func TestCreateTransfer_RateNotFound(t *testing.T) {
	s, _, accountMock, exchangeMock := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(2)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("JPY"))
	exchangeMock.ExpectQuery("SELECT rate FROM exchange_rates").
		WillReturnError(sql.ErrNoRows)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateTransfer_CommitError(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(500), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(1)))
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 200,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateTransfer_PersistError(t *testing.T) {
	s, dbMock, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT id, owner_id, available_balance, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "currency_id"}).
			AddRow(int64(1), int64(1), float64(500), int64(1)))
	accountMock.ExpectQuery("SELECT id, owner_id, currency_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "currency_id"}).
			AddRow(int64(2), int64(1), int64(1)))
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	dbMock.ExpectQuery("INSERT INTO transfers").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		ClientId: 1, FromAccount: "ACC1", ToAccount: "ACC2", Amount: 200,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetTransfers ----

func TestGetTransfers_NoAccounts(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}))

	resp, err := s.GetTransfers(context.Background(), &pb.GetTransfersRequest{ClientId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Transfers)
}

func TestGetTransfers_AccountsQueryError(t *testing.T) {
	s, _, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetTransfers(context.Background(), &pb.GetTransfersRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetTransfers_HappyPath(t *testing.T) {
	s, dbMock, accountMock, _ := newTransferServer(t)
	ts := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).
			AddRow("ACC-100").AddRow("ACC-200"))

	dbMock.ExpectQuery("SELECT id, order_number, from_account, to_account").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_number", "from_account", "to_account",
			"initial_amount", "final_amount", "exchange_rate", "fee", "timestamp",
		}).AddRow(int64(1), "TRF-001", "ACC-100", "ACC-200", 500.0, 500.0, 1.0, 0.0, ts))

	resp, err := s.GetTransfers(context.Background(), &pb.GetTransfersRequest{ClientId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Transfers, 1)
	assert.Equal(t, "TRF-001", resp.Transfers[0].OrderNumber)
	assert.Equal(t, "ACC-100", resp.Transfers[0].FromAccount)
	assert.Equal(t, ts.Format(time.RFC3339), resp.Transfers[0].Timestamp)
}

func TestGetTransfers_TransfersQueryError(t *testing.T) {
	s, dbMock, accountMock, _ := newTransferServer(t)
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-100"))
	dbMock.ExpectQuery("SELECT id, order_number, from_account, to_account").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetTransfers(context.Background(), &pb.GetTransfersRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- DeletePaymentRecipient: UPDATE error path ----

func TestDeletePaymentRecipient_UpdateRefsError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("UPDATE payments SET recipient_id").WillReturnError(sql.ErrConnDone)

	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 1, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
