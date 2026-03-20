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

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
)

// ---- paginate tests ----

func TestPaginate(t *testing.T) {
	tests := []struct {
		name       string
		page       int32
		pageSize   int32
		wantLimit  int32
		wantOffset int32
	}{
		{"normal page 1",            1,  10,  10,  0},
		{"page 2",                   2,  10,  10,  10},
		{"page 3 size 5",            3,  5,   5,   10},
		{"page 0 defaults to 1",     0,  10,  10,  0},
		{"pageSize 0 defaults to 20", 1, 0,   20,  0},
		{"pageSize over max",        1,  200, 100, 0},
		{"both zero",                0,  0,   20,  0},
		{"large page",               5,  25,  25,  100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limit, offset := paginate(tc.page, tc.pageSize)
			assert.Equal(t, tc.wantLimit, limit)
			assert.Equal(t, tc.wantOffset, offset)
		})
	}
}

// ---- ActivateEmployee tests ----

func TestActivateEmployee_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").
		WillReturnRows(sqlmock.NewRows([]string{"aktivan", "password"}))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestActivateEmployee_AlreadyActive(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").
		WillReturnRows(sqlmock.NewRows([]string{"aktivan", "password"}).AddRow(true, ""))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateEmployee_PasswordAlreadySet(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").
		WillReturnRows(sqlmock.NewRows([]string{"aktivan", "password"}).AddRow(false, "existinghash"))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateEmployee_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").
		WillReturnRows(sqlmock.NewRows([]string{"aktivan", "password"}).AddRow(false, ""))
	dbMock.ExpectExec("UPDATE employees SET password").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	resp, err := s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- UpdateEmployee tests ----

func TestUpdateEmployee_ActivateWithoutPassword(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Aktivan=true triggers the password pre-check
	dbMock.ExpectQuery("SELECT password FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(""))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: true})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpdateEmployee_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Aktivan=false skips the password pre-check, goes straight to UPDATE RETURNING
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "ime", "prezime", "datum_rodjenja", "pol", "email",
			"broj_telefona", "adresa", "username", "pozicija", "departman", "aktivan", "dozvole", "jmbg",
		}))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateEmployee_UniqueUsernameViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_username_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "username")
}

func TestUpdateEmployee_UniqueJmbgViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_jmbg_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "jmbg")
}

func TestUpdateEmployee_UniqueEmailViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_email_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "email")
}

// ---- UpdatePassword tests ----

func TestUpdatePassword_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE employees SET password").
		WillReturnResult(sqlmock.NewResult(0, 0))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdatePassword(context.Background(), &pb.UpdatePasswordRequest{EmployeeId: 999, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdatePassword_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE employees SET password").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	resp, err := s.UpdatePassword(context.Background(), &pb.UpdatePasswordRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- GetEmployeeCredentials tests ----

func TestGetEmployeeCredentials_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "aktivan", "dozvole"}))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "unknown@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeCredentials_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "aktivan", "dozvole"}).
			AddRow(int64(1), "hashedpw", true, pq.StringArray{"ADMIN"}))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "user@example.com"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "hashedpw", resp.PasswordHash)
	assert.True(t, resp.Aktivan)
	assert.Equal(t, []string{"ADMIN"}, resp.Dozvole)
}

// ---- GetEmployeeById tests ----

func employeeColumns() []string {
	return []string{
		"id", "ime", "prezime", "datum_rodjenja", "pol", "email",
		"broj_telefona", "adresa", "username", "pozicija", "departman", "aktivan", "dozvole", "jmbg",
	}
}

func TestGetEmployeeById_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime").
		WillReturnRows(sqlmock.NewRows(employeeColumns()))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeById_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"0601234567", "Main St 1", "johndoe", "Engineer", "IT", true, pq.StringArray{"ADMIN"}, "1234567890123"))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Employee.Id)
	assert.Equal(t, "John", resp.Employee.Ime)
	assert.Equal(t, "john@example.com", resp.Employee.Email)
	assert.True(t, resp.Employee.Aktivan)
}

// ---- GetEmployeeByEmail tests ----

func TestGetEmployeeByEmail_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime, email").
		WillReturnRows(sqlmock.NewRows([]string{"id", "ime", "email"}))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeByEmail(context.Background(), &pb.GetEmployeeByEmailRequest{Email: "nobody@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeByEmail_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime, email").
		WillReturnRows(sqlmock.NewRows([]string{"id", "ime", "email"}).
			AddRow(int64(7), "Alice", "alice@example.com"))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetEmployeeByEmail(context.Background(), &pb.GetEmployeeByEmailRequest{Email: "alice@example.com"})
	require.NoError(t, err)
	assert.Equal(t, int64(7), resp.Id)
	assert.Equal(t, "Alice", resp.FirstName)
	assert.Equal(t, "alice@example.com", resp.Email)
}

// ---- CreateEmployee tests ----

func TestCreateEmployee_UniqueUsernameViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_username_key"}
	dbMock.ExpectQuery("INSERT INTO employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{Username: "taken"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "username")
}

func TestCreateEmployee_UniqueJmbgViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_jmbg_key"}
	dbMock.ExpectQuery("INSERT INTO employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{Jmbg: "1234567890123"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "jmbg")
}

func TestCreateEmployee_UniqueEmailViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pqErr := &pq.Error{Code: "23505", Constraint: "employees_email_key"}
	dbMock.ExpectQuery("INSERT INTO employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{Email: "taken@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "email")
}

func TestCreateEmployee_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("INSERT INTO employees").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	req := &pb.CreateEmployeeRequest{
		Ime: "Jane", Prezime: "Doe", DatumRodjenja: "1995-06-15", Pol: "F",
		Email: "jane@example.com", BrojTelefona: "0601111111", Adresa: "Addr 2",
		Username: "janedoe", Pozicija: "Analyst", Departman: "Finance", Jmbg: "9876543210987",
	}

	s := &EmployeeServer{DB: db}
	resp, err := s.CreateEmployee(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.Employee.Id)
	assert.Equal(t, "Jane", resp.Employee.Ime)
	assert.Equal(t, "jane@example.com", resp.Employee.Email)
	assert.False(t, resp.Employee.Aktivan)
}

// ---- GetAllEmployees tests ----

func TestGetAllEmployees_CountFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := &EmployeeServer{DB: db}
	_, err = s.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetAllEmployees_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(2)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", true, pq.StringArray{"ADMIN"}, "1111111111111").
			AddRow(int64(2), "Jane", "Doe", "1992-03-20", "F", "jane@example.com",
				"060222", "Addr2", "janedoe", "QA", "IT", true, pq.StringArray{}, "2222222222222"))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.TotalCount)
	assert.Len(t, resp.Employees, 2)
	assert.Equal(t, "John", resp.Employees[0].Ime)
	assert.Equal(t, "Jane", resp.Employees[1].Ime)
}

// ---- SearchEmployees tests ----

func TestSearchEmployees_HappyPath_NoFilters(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", true, pq.StringArray{"ADMIN"}, "1111111111111"))

	s := &EmployeeServer{DB: db}
	resp, err := s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Len(t, resp.Employees, 1)
}

func TestSearchEmployees_HappyPath_WithEmailFilter(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(3), "Bob", "Smith", "1985-07-04", "M", "bob@example.com",
				"060333", "Addr3", "bobsmith", "PM", "HR", true, pq.StringArray{}, "3333333333333"))

	s := &EmployeeServer{DB: db}
	resp, err := s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{
		Email: "bob@example.com", Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Equal(t, "Bob", resp.Employees[0].Ime)
}

// ---- Additional DB error / scan error tests ----

func TestGetAllEmployees_QueryFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetAllEmployees_ScanFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	// wrong column type for id to trigger scan error
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow("not-an-int", "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", true, pq.StringArray{}, "1111111111111"))

	s := &EmployeeServer{DB: db}
	_, err = s.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestSearchEmployees_CountFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestSearchEmployees_QueryFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestSearchEmployees_ScanFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int32(1)))
	dbMock.ExpectQuery("LIMIT").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow("not-an-int", "X", "Y", "2000-01-01", "M", "x@example.com",
				"000", "Addr", "xy", "Dev", "IT", false, pq.StringArray{}, "0000000000000"))

	s := &EmployeeServer{DB: db}
	_, err = s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetEmployeeCredentials_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "user@example.com"})
	require.Error(t, err)
}

func TestGetEmployeeById_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 1})
	require.Error(t, err)
}

func TestUpdateEmployee_ActivateNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT password FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"password"})) // ErrNoRows

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 99, Aktivan: true})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateEmployee_ActivateDBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT password FROM employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: true})
	require.Error(t, err)
}

func TestUpdateEmployee_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("UPDATE employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false})
	require.Error(t, err)
}

func TestActivateEmployee_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1})
	require.Error(t, err)
}

func TestActivateEmployee_ExecError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT aktivan").
		WillReturnRows(sqlmock.NewRows([]string{"aktivan", "password"}).AddRow(false, ""))
	dbMock.ExpectExec("UPDATE employees SET password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
}

func TestGetEmployeeByEmail_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("SELECT id, ime, email").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeByEmail(context.Background(), &pb.GetEmployeeByEmailRequest{Email: "user@example.com"})
	require.Error(t, err)
}

func TestUpdatePassword_ExecError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("UPDATE employees SET password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdatePassword(context.Background(), &pb.UpdatePasswordRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
}

func TestCreateEmployee_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("INSERT INTO employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{Ime: "X"})
	require.Error(t, err)
}

func TestUpdateEmployee_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false, pq.StringArray{"ADMIN"}, "1111111111111"))

	s := &EmployeeServer{DB: db}
	resp, err := s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Aktivan: false, Dozvole: []string{"ADMIN"}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Employee.Id)
	assert.Equal(t, "John", resp.Employee.Ime)
}
