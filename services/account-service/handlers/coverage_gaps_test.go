package handlers

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── GetAccount with company_id set ──────────────────────────────────────────
// The existing TestGetAccount_HappyPath passes nil for company_id so the
// companyID.Valid branch is never entered. This test covers that branch.

func TestGetAccount_WithCompanyData(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "account_name", "account_number", "owner_id",
			"balance", "available_balance", "reserved_funds",
			"currency_id", "status", "account_type", "account_subtype",
			"daily_limit", "monthly_limit", "daily_spent", "monthly_spent",
			"company_id",
		}).AddRow(
			int64(1), "Business Account", "888000100000000101", int64(5),
			float64(5000), float64(4500), float64(500),
			int64(1), "ACTIVE", "business", "",
			float64(0), float64(0), float64(0), float64(0),
			int64(7), // companyID.Valid = true
		),
	)
	exchangeMock.ExpectQuery("SELECT code").WillReturnRows(
		sqlmock.NewRows([]string{"code"}).AddRow("RSD"),
	)
	clientMock.ExpectQuery("SELECT first_name").WillReturnRows(
		sqlmock.NewRows([]string{"first_name", "last_name"}).AddRow("Marko", "Markovic"),
	)
	// Company data query — returns a valid company
	dbMock.ExpectQuery("SELECT name, registration_number").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"name", "registration_number", "pib", "activity_code", "address",
		}).AddRow("Firma DOO", "12345678", "987654321", "62.01", "Beograd"))

	resp, err := s.GetAccount(context.Background(), &pb.GetAccountRequest{AccountId: 1, OwnerId: 5})
	require.NoError(t, err)
	require.NotNil(t, resp.Account.CompanyData)
	assert.Equal(t, "Firma DOO", resp.Account.CompanyData.Name)
}

// ── CreateAccount company error paths with lowercase "business" ─────────────
// The existing error tests use AccountType: "BUSINESS" (uppercase) which
// doesn't match the "business" condition in the handler, so those branches
// were never entered. These tests fix that.

func TestCreateAccount_CompanyInsertError_Lowercase(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO companies").WillReturnError(sql.ErrConnDone)

	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "business", AccountName: "Biznis",
		CompanyData: &pb.CompanyData{Name: "Firma", RegistrationNumber: "123"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_CompanyLookupError_Lowercase(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnError(sql.ErrConnDone)

	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "business", AccountName: "Biznis",
		CompanyData: &pb.CompanyData{Name: "Firma", RegistrationNumber: "123"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
