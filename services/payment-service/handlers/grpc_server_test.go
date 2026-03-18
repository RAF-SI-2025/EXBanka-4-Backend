package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	// Load source account
	accountMock.ExpectQuery("SELECT id, owner_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "available_balance", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "currency_id"}).
			AddRow(int64(1), int64(1), float64(1000), nil, nil, float64(0), float64(0), int64(1)),
	)
	// Lookup destination account (same currency)
	accountMock.ExpectQuery("SELECT id, currency_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "currency_id"}).AddRow(int64(2), int64(1)),
	)
	// Begin tx
	accountMock.ExpectBegin()
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET").WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectCommit()
	// Insert payment record
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
	// Destination account not in our system
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

// ---- DeletePaymentRecipient ----

func TestDeletePaymentRecipient_NotFound(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 99, ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeletePaymentRecipient_HappyPath(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 1, ClientId: 1})
	require.NoError(t, err)
}

// ---- Additional coverage ----

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
	// Destination account with different currency
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

func TestUpdatePaymentRecipient_InternalError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectQuery("UPDATE payment_recipients").WillReturnError(sql.ErrConnDone)
	_, err := s.UpdatePaymentRecipient(context.Background(), &pb.UpdatePaymentRecipientRequest{
		Id: 1, ClientId: 1, Name: "X", AccountNumber: "ACC1",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestDeletePaymentRecipient_ExecError(t *testing.T) {
	s, dbMock, _ := newPaymentServer(t)
	dbMock.ExpectExec("DELETE FROM payment_recipients").WillReturnError(sql.ErrConnDone)
	_, err := s.DeletePaymentRecipient(context.Background(), &pb.DeletePaymentRecipientRequest{Id: 1, ClientId: 1})
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
