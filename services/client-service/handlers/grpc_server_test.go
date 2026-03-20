package handlers

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
)

// ---- helpers ----

func clientColumns() []string {
	return []string{
		"id", "first_name", "last_name", "jmbg", "date_of_birth",
		"gender", "email", "phone_number", "address", "username", "active",
	}
}

func addClientRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(
		int64(1), "Ana", "Anić", "0101990710001", "1990-01-01",
		"F", "ana@example.com", "0601234567", "Main St 1", "anaanic", true,
	)
}

// ---- paginate tests ----

func TestPaginate(t *testing.T) {
	tests := []struct {
		name       string
		page       int32
		pageSize   int32
		wantLimit  int32
		wantOffset int32
	}{
		{"normal page 1", 1, 10, 10, 0},
		{"page 2", 2, 10, 10, 10},
		{"page 0 defaults to 1", 0, 10, 10, 0},
		{"pageSize 0 defaults to 20", 1, 0, 20, 0},
		{"pageSize over max", 1, 200, 100, 0},
		{"both zero", 0, 0, 20, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limit, offset := paginate(tc.page, tc.pageSize)
			assert.Equal(t, tc.wantLimit, limit)
			assert.Equal(t, tc.wantOffset, offset)
		})
	}
}

// ---- GetAllClients tests ----

func TestGetAllClients_CountFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := &ClientServer{DB: db}
	_, err = s.GetAllClients(context.Background(), &pb.GetAllClientsRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetAllClients_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(2)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(addClientRow(addClientRow(sqlmock.NewRows(clientColumns()))))

	s := &ClientServer{DB: db}
	resp, err := s.GetAllClients(context.Background(), &pb.GetAllClientsRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.TotalCount)
	assert.Len(t, resp.Clients, 2)
}

// ---- GetClientById tests ----

func TestGetClientById_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, first_name").
		WillReturnRows(sqlmock.NewRows(clientColumns()))

	s := &ClientServer{DB: db}
	_, err = s.GetClientById(context.Background(), &pb.GetClientByIdRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetClientById_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, first_name").
		WillReturnRows(addClientRow(sqlmock.NewRows(clientColumns())))

	s := &ClientServer{DB: db}
	resp, err := s.GetClientById(context.Background(), &pb.GetClientByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Client.Id)
	assert.Equal(t, "Ana", resp.Client.FirstName)
	assert.Equal(t, "ana@example.com", resp.Client.Email)
	assert.True(t, resp.Client.Active)
}

// ---- CreateClient tests ----

func TestCreateClient_UniqueEmailViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_email_key"}
	dbMock.ExpectQuery("INSERT INTO clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.CreateClient(context.Background(), &pb.CreateClientRequest{Email: "taken@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "email")
}

func TestCreateClient_UniqueUsernamneViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_username_key"}
	dbMock.ExpectQuery("INSERT INTO clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.CreateClient(context.Background(), &pb.CreateClientRequest{Username: "taken"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "username")
}

func TestCreateClient_UniqueJmbgViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_jmbg_key"}
	dbMock.ExpectQuery("INSERT INTO clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.CreateClient(context.Background(), &pb.CreateClientRequest{Jmbg: "1234567890123"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "jmbg")
}

func TestCreateClient_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("INSERT INTO clients").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	s := &ClientServer{DB: db}
	resp, err := s.CreateClient(context.Background(), &pb.CreateClientRequest{
		FirstName: "Ana", LastName: "Anić", Jmbg: "0101990710001",
		DateOfBirth: "1990-01-01", Gender: "F", Email: "ana@example.com",
		PhoneNumber: "0601234567", Address: "Main St 1", Username: "anaanic",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.Client.Id)
	assert.Equal(t, "Ana", resp.Client.FirstName)
	assert.False(t, resp.Client.Active)
}

// ---- UpdateClient tests ----

func TestUpdateClient_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("UPDATE clients").
		WillReturnRows(sqlmock.NewRows(clientColumns()))

	s := &ClientServer{DB: db}
	_, err = s.UpdateClient(context.Background(), &pb.UpdateClientRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateClient_UniqueEmailViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_email_key"}
	dbMock.ExpectQuery("UPDATE clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.UpdateClient(context.Background(), &pb.UpdateClientRequest{Id: 1, Email: "taken@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "email")
}

func TestUpdateClient_UniqueUsernameViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_username_key"}
	dbMock.ExpectQuery("UPDATE clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.UpdateClient(context.Background(), &pb.UpdateClientRequest{Id: 1, Username: "taken"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "username")
}

func TestUpdateClient_UniqueJmbgViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "clients_jmbg_key"}
	dbMock.ExpectQuery("UPDATE clients").WillReturnError(pqErr)

	s := &ClientServer{DB: db}
	_, err = s.UpdateClient(context.Background(), &pb.UpdateClientRequest{Id: 1, Jmbg: "1234567890123"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "jmbg")
}

func TestUpdateClient_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("UPDATE clients").
		WillReturnRows(addClientRow(sqlmock.NewRows(clientColumns())))

	s := &ClientServer{DB: db}
	resp, err := s.UpdateClient(context.Background(), &pb.UpdateClientRequest{
		Id: 1, FirstName: "Ana", LastName: "Anić", Jmbg: "0101990710001",
		DateOfBirth: "1990-01-01", Gender: "F", Email: "ana@example.com",
		PhoneNumber: "0601234567", Address: "Main St 1", Username: "anaanic", Active: true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Client.Id)
	assert.Equal(t, "Ana", resp.Client.FirstName)
	assert.True(t, resp.Client.Active)
}

// ---- GetClientCredentials tests ----

func TestGetClientCredentials_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "active"}))

	s := &ClientServer{DB: db}
	_, err = s.GetClientCredentials(context.Background(), &pb.GetClientCredentialsRequest{Email: "nobody@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetClientCredentials_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "active"}).
			AddRow(int64(1), "hashedpw", true))

	s := &ClientServer{DB: db}
	resp, err := s.GetClientCredentials(context.Background(), &pb.GetClientCredentialsRequest{Email: "ana@example.com"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "hashedpw", resp.PasswordHash)
	assert.True(t, resp.Active)
}

// ---- ActivateClient tests ----

func TestActivateClient_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE clients").
		WillReturnResult(sqlmock.NewResult(0, 0))

	s := &ClientServer{DB: db}
	_, err = s.ActivateClient(context.Background(), &pb.ActivateClientRequest{
		ClientId: 99, PasswordHash: "hash",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestActivateClient_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE clients").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := &ClientServer{DB: db}
	_, err = s.ActivateClient(context.Background(), &pb.ActivateClientRequest{
		ClientId: 1, PasswordHash: "hash",
	})
	require.Error(t, err)
}

func TestActivateClient_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE clients").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &ClientServer{DB: db}
	_, err = s.ActivateClient(context.Background(), &pb.ActivateClientRequest{
		ClientId: 1, PasswordHash: "$2a$10$somehash",
	})
	require.NoError(t, err)
}

// ---- Additional DB error / scan error tests ----

func TestGetAllClients_QueryFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").WillReturnError(sql.ErrConnDone)

	s := &ClientServer{DB: db}
	_, err = s.GetAllClients(context.Background(), &pb.GetAllClientsRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetAllClients_ScanFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(clientColumns()).
			AddRow("not-an-int", "Ana", "Anić", "0101990710001", "1990-01-01",
				"F", "ana@example.com", "0601234567", "Main St 1", "anaanic", true))

	s := &ClientServer{DB: db}
	_, err = s.GetAllClients(context.Background(), &pb.GetAllClientsRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetClientById_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, first_name").WillReturnError(sql.ErrConnDone)

	s := &ClientServer{DB: db}
	_, err = s.GetClientById(context.Background(), &pb.GetClientByIdRequest{Id: 1})
	require.Error(t, err)
}

func TestCreateClient_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("INSERT INTO clients").WillReturnError(sql.ErrConnDone)

	s := &ClientServer{DB: db}
	_, err = s.CreateClient(context.Background(), &pb.CreateClientRequest{FirstName: "X"})
	require.Error(t, err)
}

func TestUpdateClient_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("UPDATE clients").WillReturnError(sql.ErrConnDone)

	s := &ClientServer{DB: db}
	_, err = s.UpdateClient(context.Background(), &pb.UpdateClientRequest{Id: 1})
	require.Error(t, err)
}

func TestGetClientCredentials_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").WillReturnError(sql.ErrConnDone)

	s := &ClientServer{DB: db}
	_, err = s.GetClientCredentials(context.Background(), &pb.GetClientCredentialsRequest{Email: "user@example.com"})
	require.Error(t, err)
}
