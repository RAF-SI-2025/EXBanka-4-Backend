package handlers

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
)

func newServer(t *testing.T) (*PortfolioServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return &PortfolioServer{DB: db}, mock
}

// ── UpdateHolding ─────────────────────────────────────────────────────────────

func TestUpdateHolding_Buy_NewEntry(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`INSERT INTO portfolio_entry`).
		WithArgs(int64(1), "CLIENT", int64(10), int32(5), float64(100.0), int64(42)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		UserType:  "CLIENT",
		ListingId: 10,
		Quantity:  5,
		Price:     100.0,
		Direction: "BUY",
		AccountId: 42,
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Buy_ExistingEntry_WeightedAvg(t *testing.T) {
	srv, mock := newServer(t)

	// ON CONFLICT DO UPDATE — same INSERT statement handles both cases
	mock.ExpectExec(`INSERT INTO portfolio_entry`).
		WithArgs(int64(2), "EMPLOYEE", int64(20), int32(3), float64(200.0), int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    2,
		UserType:  "EMPLOYEE",
		ListingId: 20,
		Quantity:  3,
		Price:     200.0,
		Direction: "BUY",
		AccountId: 99,
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Sell_Partial(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`UPDATE portfolio_entry`).
		WithArgs(int32(2), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM portfolio_entry`).
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  2,
		Price:     150.0,
		Direction: "SELL",
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_Sell_Full(t *testing.T) {
	srv, mock := newServer(t)

	mock.ExpectExec(`UPDATE portfolio_entry`).
		WithArgs(int32(5), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM portfolio_entry`).
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1)) // row deleted (amount hit 0)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  5,
		Price:     150.0,
		Direction: "SELL",
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateHolding_InvalidDirection(t *testing.T) {
	srv, _ := newServer(t)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  1,
		Direction: "HOLD",
	})
	require.Error(t, err)
}

func TestUpdateHolding_ZeroQuantity(t *testing.T) {
	srv, _ := newServer(t)

	_, err := srv.UpdateHolding(context.Background(), &pb.UpdateHoldingRequest{
		UserId:    1,
		ListingId: 10,
		Quantity:  0,
		Direction: "BUY",
	})
	require.Error(t, err)
}
