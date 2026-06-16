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
		{"normal page 1", 1, 10, 10, 0},
		{"page 2", 2, 10, 10, 10},
		{"page 3 size 5", 3, 5, 5, 10},
		{"page 0 defaults to 1", 0, 10, 10, 0},
		{"pageSize 0 defaults to 20", 1, 0, 20, 0},
		{"pageSize over max", 1, 200, 100, 0},
		{"both zero", 0, 0, 20, 0},
		{"large page", 5, 25, 25, 100},
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
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").
		WillReturnRows(sqlmock.NewRows([]string{"active", "password"}))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestActivateEmployee_AlreadyActive(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").
		WillReturnRows(sqlmock.NewRows([]string{"active", "password"}).AddRow(true, ""))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateEmployee_PasswordAlreadySet(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").
		WillReturnRows(sqlmock.NewRows([]string{"active", "password"}).AddRow(false, "existinghash"))

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateEmployee_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").
		WillReturnRows(sqlmock.NewRows([]string{"active", "password"}).AddRow(false, ""))
	dbMock.ExpectExec("UPDATE employees SET password").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	resp, err := s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- UpdateEmployee tests ----

// expectNonAdminPermissions mocks the first SELECT in UpdateEmployee that reads
// the target employee's permissions (needed for admin-edit-admin guard).
func expectNonAdminPermissions(dbMock sqlmock.Sqlmock) {
	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).
			AddRow(pq.StringArray{"READ"}))
}

// expectAdminPermissions mocks the same query but returns an ADMIN row.
func expectAdminPermissions(dbMock sqlmock.Sqlmock) {
	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).
			AddRow(pq.StringArray{"ADMIN"}))
}

func TestUpdateEmployee_ActivateWithoutPassword(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	// Active=true triggers the password pre-check
	dbMock.ExpectQuery("SELECT password FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(""))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: true})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpdateEmployee_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	// Active=false skips the password pre-check, goes straight to UPDATE RETURNING
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "ime", "prezime", "datum_rodjenja", "pol", "email",
			"broj_telefona", "adresa", "username", "pozicija", "departman", "aktivan", "dozvole", "jmbg",
		}))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateEmployee_UniqueUsernameViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	pqErr := &pq.Error{Code: "23505", Constraint: "employees_username_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "username")
}

func TestUpdateEmployee_UniqueJmbgViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	pqErr := &pq.Error{Code: "23505", Constraint: "employees_jmbg_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "jmbg")
}

func TestUpdateEmployee_UniqueEmailViolation(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	pqErr := &pq.Error{Code: "23505", Constraint: "employees_email_key"}
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(pqErr)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "email")
}

// ---- UpdateEmployee: new #143 validations ----

func TestUpdateEmployee_AgentAndSupervisorConflict(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{
		Id:          1,
		Permissions: []string{"AGENT", "SUPERVISOR"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUpdateEmployee_AdminCannotEditAdmin(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectAdminPermissions(dbMock) // target employee is also an admin
	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 2, Active: false})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestUpdateEmployee_AdminAutoAddsSupervisor(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock) // target is not admin
	// UPDATE returns the employee with ADMIN + SUPERVISOR
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false,
				pq.StringArray{"ADMIN", "SUPERVISOR"}, "1111111111111"))
	// actuary_info INSERT (supervisor gained)
	dbMock.ExpectExec("INSERT INTO actuary_info").WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	resp, err := s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{
		Id:          1,
		Permissions: []string{"ADMIN"}, // SUPERVISOR should be auto-added
	})
	require.NoError(t, err)
	// Verify SUPERVISOR was included in the UPDATE call (checked via mock)
	assert.Contains(t, resp.Employee.Permissions, "ADMIN")
	assert.Contains(t, resp.Employee.Permissions, "SUPERVISOR")
}

func TestUpdateEmployee_ActuaryInfoCreatedOnAgentAssign(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Target has no agent/supervisor permissions yet
	expectNonAdminPermissions(dbMock)
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "j@j.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false,
				pq.StringArray{"AGENT"}, "1111111111111"))
	dbMock.ExpectExec("INSERT INTO actuary_info").WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{
		Id:          1,
		Permissions: []string{"AGENT"},
	})
	require.NoError(t, err)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUpdateEmployee_ActuaryInfoDeletedOnRoleRemoval(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Target currently has AGENT
	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).
			AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "j@j.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false,
				pq.StringArray{"READ"}, "1111111111111"))
	dbMock.ExpectExec("DELETE FROM actuary_info").WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{
		Id:          1,
		Permissions: []string{"READ"}, // AGENT removed
	})
	require.NoError(t, err)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- UpdatePassword tests ----

func TestUpdatePassword_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "active", "permissions"}))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "unknown@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeCredentials_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, password").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password", "active", "permissions", "email", "first_name", "failed_login_attempts", "account_locked_until"}).
			AddRow(int64(1), "hashedpw", true, pq.StringArray{"ADMIN"}, "user@example.com", "User", int32(0), nil))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "user@example.com"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "hashedpw", resp.PasswordHash)
	assert.True(t, resp.Active)
	assert.Equal(t, []string{"ADMIN"}, resp.Permissions)
}

// ---- GetEmployeeById tests ----

func employeeColumns() []string {
	return []string{
		"id", "first_name", "last_name", "date_of_birth", "gender", "email",
		"phone_number", "address", "username", "position", "department", "active", "permissions", "jmbg",
	}
}

func TestGetEmployeeById_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name").
		WillReturnRows(sqlmock.NewRows(employeeColumns()))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeById_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"0601234567", "Main St 1", "johndoe", "Engineer", "IT", true, pq.StringArray{"ADMIN"}, "1234567890123"))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Employee.Id)
	assert.Equal(t, "John", resp.Employee.FirstName)
	assert.Equal(t, "john@example.com", resp.Employee.Email)
	assert.True(t, resp.Employee.Active)
}

// ---- GetEmployeeByEmail tests ----

func TestGetEmployeeByEmail_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name, email").
		WillReturnRows(sqlmock.NewRows([]string{"id", "first_name", "email"}))

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeByEmail(context.Background(), &pb.GetEmployeeByEmailRequest{Email: "nobody@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEmployeeByEmail_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name, email").
		WillReturnRows(sqlmock.NewRows([]string{"id", "first_name", "email"}).
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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("INSERT INTO employees").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	req := &pb.CreateEmployeeRequest{
		FirstName: "Jane", LastName: "Doe", DateOfBirth: "1995-06-15", Gender: "F",
		Email: "jane@example.com", PhoneNumber: "0601111111", Address: "Addr 2",
		Username: "janedoe", Position: "Analyst", Department: "Finance", Jmbg: "9876543210987",
	}

	s := &EmployeeServer{DB: db}
	resp, err := s.CreateEmployee(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.Employee.Id)
	assert.Equal(t, "Jane", resp.Employee.FirstName)
	assert.Equal(t, "jane@example.com", resp.Employee.Email)
	assert.False(t, resp.Employee.Active)
}

// ---- GetAllEmployees tests ----

func TestGetAllEmployees_CountFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := &EmployeeServer{DB: db}
	_, err = s.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestGetAllEmployees_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	assert.Equal(t, "John", resp.Employees[0].FirstName)
	assert.Equal(t, "Jane", resp.Employees[1].FirstName)
}

// ---- SearchEmployees tests ----

func TestSearchEmployees_HappyPath_NoFilters(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	assert.Equal(t, "Bob", resp.Employees[0].FirstName)
}

// ---- Additional DB error / scan error tests ----

func TestGetAllEmployees_QueryFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT COUNT").
		WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestSearchEmployees_QueryFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeCredentials(context.Background(), &pb.GetEmployeeCredentialsRequest{Email: "user@example.com"})
	require.Error(t, err)
}

func TestGetEmployeeById_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeById(context.Background(), &pb.GetEmployeeByIdRequest{Id: 1})
	require.Error(t, err)
}

func TestUpdateEmployee_ActivateNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	dbMock.ExpectQuery("SELECT password FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"password"})) // ErrNoRows

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 99, Active: true})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateEmployee_ActivateDBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	dbMock.ExpectQuery("SELECT password FROM employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: true})
	require.Error(t, err)
}

func TestUpdateEmployee_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock)
	dbMock.ExpectQuery("UPDATE employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false})
	require.Error(t, err)
}

func TestActivateEmployee_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1})
	require.Error(t, err)
}

func TestActivateEmployee_ExecError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT active").
		WillReturnRows(sqlmock.NewRows([]string{"active", "password"}).AddRow(false, ""))
	dbMock.ExpectExec("UPDATE employees SET password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ActivateEmployee(context.Background(), &pb.ActivateEmployeeRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
}

func TestGetEmployeeByEmail_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT id, first_name, email").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetEmployeeByEmail(context.Background(), &pb.GetEmployeeByEmailRequest{Email: "user@example.com"})
	require.Error(t, err)
}

func TestUpdatePassword_ExecError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectExec("UPDATE employees SET password").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.UpdatePassword(context.Background(), &pb.UpdatePasswordRequest{EmployeeId: 1, PasswordHash: "hash"})
	require.Error(t, err)
}

func TestCreateEmployee_GenericError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("INSERT INTO employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{FirstName: "X"})
	require.Error(t, err)
}

func TestUpdateEmployee_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectNonAdminPermissions(dbMock) // target not admin
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false, pq.StringArray{"ADMIN", "SUPERVISOR"}, "1111111111111"))
	// ADMIN req → SUPERVISOR auto-added → INSERT actuary_info (target had no actuary role before)
	dbMock.ExpectExec("INSERT INTO actuary_info").WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	resp, err := s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{Id: 1, Active: false, Permissions: []string{"ADMIN"}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Employee.Id)
	assert.Equal(t, "John", resp.Employee.FirstName)
}

// ---- GetActuaries tests ----

func actuaryColumns() []string {
	return []string{"id", "first_name", "last_name", "email", "position",
		"limit_amount", "used_limit", "need_approval"}
}

func TestGetActuaries_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT e.id").
		WillReturnRows(sqlmock.NewRows(actuaryColumns()).
			AddRow(int64(1), "Marko", "Markovic", "m@banka.rs", "Agent", 100000.0, 15000.0, false))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetActuaries(context.Background(), &pb.GetActuariesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Actuaries, 1)
	assert.Equal(t, int64(1), resp.Actuaries[0].EmployeeId)
	assert.Equal(t, "Marko", resp.Actuaries[0].FirstName)
	assert.Equal(t, 100000.0, resp.Actuaries[0].LimitAmount)
	assert.Equal(t, 15000.0, resp.Actuaries[0].UsedLimit)
	assert.False(t, resp.Actuaries[0].NeedApproval)
}

func TestGetActuaries_EmptyResult(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT e.id").
		WillReturnRows(sqlmock.NewRows(actuaryColumns()))

	s := &EmployeeServer{DB: db}
	resp, err := s.GetActuaries(context.Background(), &pb.GetActuariesRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp.Actuaries)
	assert.Len(t, resp.Actuaries, 0)
}

func TestGetActuaries_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT e.id").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.GetActuaries(context.Background(), &pb.GetActuariesRequest{})
	require.Error(t, err)
}

// ---- SetAgentLimit tests ----

func TestSetAgentLimit_NegativeLimit(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: -1})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetAgentLimit_EmployeeNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"})) // ErrNoRows

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 99, LimitAmount: 5000})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestSetAgentLimit_NotAgent(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"READ"}))

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: 5000})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetAgentLimit_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET limit_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: 50000})
	require.NoError(t, err)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- ResetAgentUsedLimit tests ----

func TestResetAgentUsedLimit_NotAgent(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"READ"}))

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestResetAgentUsedLimit_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET used_limit").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 1})
	require.NoError(t, err)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- SetNeedApproval tests ----

func TestSetNeedApproval_NotAgent(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"READ"}))

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 1, NeedApproval: true})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetNeedApproval_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET need_approval").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 1, NeedApproval: true})
	require.NoError(t, err)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ---- GetActuaries scan error ----

func TestGetActuaries_ScanError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT e.id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := &EmployeeServer{DB: db}
	_, err = s.GetActuaries(context.Background(), &pb.GetActuariesRequest{})
	require.Error(t, err)
}

// ---- SetAgentLimit additional error paths ----

func TestSetAgentLimit_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: 1000})
	require.Error(t, err)
}

func TestSetAgentLimit_UpdateError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET limit_amount").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: 1000})
	require.Error(t, err)
}

func TestSetAgentLimit_UpdateNoRows(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET limit_amount").WillReturnResult(sqlmock.NewResult(0, 0))

	s := &EmployeeServer{DB: db}
	_, err = s.SetAgentLimit(context.Background(), &pb.SetAgentLimitRequest{EmployeeId: 1, LimitAmount: 1000})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- ResetAgentUsedLimit additional error paths ----

func TestResetAgentUsedLimit_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}))

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestResetAgentUsedLimit_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 1})
	require.Error(t, err)
}

func TestResetAgentUsedLimit_UpdateError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET used_limit").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 1})
	require.Error(t, err)
}

func TestResetAgentUsedLimit_UpdateNoRows(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET used_limit").WillReturnResult(sqlmock.NewResult(0, 0))

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAgentUsedLimit(context.Background(), &pb.ResetAgentUsedLimitRequest{EmployeeId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- SetNeedApproval additional error paths ----

func TestSetNeedApproval_NotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}))

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 99, NeedApproval: true})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestSetNeedApproval_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 1, NeedApproval: true})
	require.Error(t, err)
}

func TestSetNeedApproval_UpdateError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET need_approval").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 1, NeedApproval: false})
	require.Error(t, err)
}

func TestSetNeedApproval_UpdateNoRows(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectExec("UPDATE actuary_info SET need_approval").WillReturnResult(sqlmock.NewResult(0, 0))

	s := &EmployeeServer{DB: db}
	_, err = s.SetNeedApproval(context.Background(), &pb.SetNeedApprovalRequest{EmployeeId: 1, NeedApproval: false})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- ResetAllActuaryUsedLimits ----

func TestResetAllActuaryUsedLimits_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectExec("UPDATE actuary_info SET used_limit").WillReturnError(sql.ErrConnDone)

	s := &EmployeeServer{DB: db}
	_, err = s.ResetAllActuaryUsedLimits(context.Background(), &pb.ResetAllActuaryUsedLimitsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestResetAllActuaryUsedLimits_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectExec("UPDATE actuary_info SET used_limit").WillReturnResult(sqlmock.NewResult(0, 5))

	s := &EmployeeServer{DB: db}
	resp, err := s.ResetAllActuaryUsedLimits(context.Background(), &pb.ResetAllActuaryUsedLimitsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ---- UpdateEmployee: outer actuary condition true but inner false ----

func TestUpdateEmployee_ActuaryInfoSkipsInsertWhenHadAgent(t *testing.T) {
	// Target employee has AGENT (hadAgent=true). Request gives ADMIN → SUPERVISOR auto-added.
	// willHaveAgent=false, willHaveSupervisor=true → outer condition true (!hadSupervisor && willHaveSupervisor).
	// Inner: !hadAgent && !hadSupervisor = false → no INSERT. No DELETE either (still has actuary role in request).
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	dbMock.ExpectQuery("SELECT permissions FROM employees WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow(pq.StringArray{"AGENT"}))
	dbMock.ExpectQuery("UPDATE employees").
		WillReturnRows(sqlmock.NewRows(employeeColumns()).
			AddRow(int64(1), "John", "Doe", "1990-01-01", "M", "john@example.com",
				"060111", "Addr", "johndoe", "Dev", "IT", false, pq.StringArray{"ADMIN", "SUPERVISOR"}, "1111111111111"))

	s := &EmployeeServer{DB: db}
	resp, err := s.UpdateEmployee(context.Background(), &pb.UpdateEmployeeRequest{
		Id: 1, Active: false, Permissions: []string{"ADMIN"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Employee.Id)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

// ── GetActuaryPerformers ──────────────────────────────────────────────────────

func TestGetActuaryPerformers_ReturnsAgentAndSupervisor(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	s := &EmployeeServer{DB: db}

	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "position"}).
		AddRow(int64(1), "Ana", "Jovanovic", "AGENT").
		AddRow(int64(2), "Marko", "Petrovic", "SUPERVISOR")
	dbMock.ExpectQuery(`SELECT id, first_name, last_name`).
		WillReturnRows(rows)

	resp, err := s.GetActuaryPerformers(context.Background(), &pb.GetActuaryPerformersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Performers, 2)
	assert.Equal(t, int64(1), resp.Performers[0].UserId)
	assert.Equal(t, "Ana", resp.Performers[0].FirstName)
	assert.Equal(t, "SUPERVISOR", resp.Performers[1].Position)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGetActuaryPerformers_Empty(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	s := &EmployeeServer{DB: db}

	dbMock.ExpectQuery(`SELECT id, first_name, last_name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "first_name", "last_name", "position"}))

	resp, err := s.GetActuaryPerformers(context.Background(), &pb.GetActuaryPerformersRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Performers)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGetActuaryPerformers_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	s := &EmployeeServer{DB: db}

	dbMock.ExpectQuery(`SELECT id, first_name, last_name`).
		WillReturnError(sql.ErrConnDone)

	_, err = s.GetActuaryPerformers(context.Background(), &pb.GetActuaryPerformersRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
