package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makePreviewReq(fromAccount, toAccount string, amount float64, clientID int64) *pb.PreviewPaymentRequest {
	return &pb.PreviewPaymentRequest{
		FromAccount:      fromAccount,
		RecipientAccount: toAccount,
		Amount:           amount,
		ClientId:         clientID,
	}
}

func TestPreviewPayment_SourceAccountNotFound(t *testing.T) {
	s, _, accountMock, _ := newPaymentServerWithExchange(t)
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}))

	_, err := s.PreviewPayment(context.Background(), makePreviewReq("888123", "888456", 100, 1))
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestPreviewPayment_WrongOwner(t *testing.T) {
	s, _, accountMock, _ := newPaymentServerWithExchange(t)
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}).AddRow(int64(99), int64(1)))

	_, err := s.PreviewPayment(context.Background(), makePreviewReq("888123", "888456", 100, 1))
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestPreviewPayment_CrossBank_PartnerDown(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")
	t.Setenv("PARTNER_BANK_URL", "http://127.0.0.1:19997")
	t.Setenv("PARTNER_API_KEY", "key")

	s, _, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}).AddRow(int64(1), int64(1)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))

	resp, err := s.PreviewPayment(context.Background(), makePreviewReq("888111", "444222", 100, 1))
	require.NoError(t, err)
	assert.True(t, resp.IsCrossBank)
	assert.Equal(t, float64(100), resp.FinalAmount)
}

func TestPreviewPayment_CrossBank_VoteYes(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// NEW_TX → vote YES with rate and fee
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vote":"YES","exchange_rate":1.17,"fee":1.5,"final_amount":98.5}`))
		} else {
			// ROLLBACK_TX → ok
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PARTNER_BANK_URL", srv.URL)
	t.Setenv("PARTNER_API_KEY", "key")

	s, _, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}).AddRow(int64(1), int64(1)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))

	resp, err := s.PreviewPayment(context.Background(), makePreviewReq("888111", "444222", 100, 1))
	require.NoError(t, err)
	assert.True(t, resp.IsCrossBank)
	assert.Equal(t, 1.17, resp.ExchangeRate)
	assert.Equal(t, 1.5, resp.Fee)
	assert.Equal(t, 98.5, resp.FinalAmount)
}

func TestPreviewPayment_OwnBank_SameCurrency(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, _, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	// Source account lookup
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}).AddRow(int64(1), int64(1)))
	// Source currency
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	// Recipient currency (same)
	accountMock.ExpectQuery("SELECT currency_id FROM accounts WHERE account_number").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))

	resp, err := s.PreviewPayment(context.Background(), makePreviewReq("888111", "888222", 100, 1))
	require.NoError(t, err)
	assert.False(t, resp.IsCrossBank)
	assert.Equal(t, float64(1), resp.ExchangeRate)
	assert.Equal(t, float64(0), resp.Fee)
	assert.Equal(t, float64(100), resp.FinalAmount)
}

func TestPreviewPayment_OwnBank_DifferentCurrency_FromRSD(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, _, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	accountMock.ExpectQuery("SELECT owner_id, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "currency_id"}).AddRow(int64(1), int64(1)))
	// Source: RSD
	exchangeMock.ExpectQuery("SELECT code FROM currencies WHERE id = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	// Recipient: EUR (different currency ID)
	accountMock.ExpectQuery("SELECT currency_id FROM accounts WHERE account_number").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	// Dest currency code
	exchangeMock.ExpectQuery("SELECT code FROM currencies WHERE id = \\$1").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	// selling rate for EUR
	exchangeMock.ExpectQuery("SELECT selling_rate FROM daily_exchange_rates").
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(117.0))

	resp, err := s.PreviewPayment(context.Background(), makePreviewReq("888111", "888222", 100, 1))
	require.NoError(t, err)
	assert.False(t, resp.IsCrossBank)
	assert.Greater(t, resp.FinalAmount, float64(0))
}
