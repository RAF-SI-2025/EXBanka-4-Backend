package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9]+$`)
)

func validateEmployeeInput(email, phone, dob string) error {
	if email != "" && !emailRegex.MatchString(email) {
		return status.Error(codes.InvalidArgument, "invalid email format")
	}
	if phone != "" && !phoneRegex.MatchString(phone) {
		return status.Error(codes.InvalidArgument, "phone must contain only digits and optional leading +")
	}
	if dob != "" {
		t, err := time.Parse("2006-01-02", dob)
		if err != nil {
			return status.Error(codes.InvalidArgument, "date of birth must be in YYYY-MM-DD format")
		}
		if !time.Now().After(t) {
			return status.Error(codes.InvalidArgument, "date of birth must be in the past")
		}
	}
	return nil
}

// containsPermission returns true if perms contains the given permission (case-insensitive).
func containsPermission(perms []string, perm string) bool {
	upper := strings.ToUpper(perm)
	for _, p := range perms {
		if strings.ToUpper(p) == upper {
			return true
		}
	}
	return false
}

type EmployeeServer struct {
	pb.UnimplementedEmployeeServiceServer
	DB *sql.DB
}

const defaultPageSize = 20
const maxPageSize = 100

func paginate(page, pageSize int32) (limit, offset int32) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page <= 0 {
		page = 1
	}
	return pageSize, (page - 1) * pageSize
}

func (s *EmployeeServer) GetAllEmployees(ctx context.Context, req *pb.GetAllEmployeesRequest) (*pb.GetAllEmployeesResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM employees`).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var permissions pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
			&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
			&e.Department, &e.Active, &permissions, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Permissions = permissions
		employees = append(employees, &e)
	}
	return &pb.GetAllEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) SearchEmployees(ctx context.Context, req *pb.SearchEmployeesRequest) (*pb.SearchEmployeesResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM employees
		WHERE ($1 = '' OR email      ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR first_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR last_name  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR position   ILIKE '%' || $4 || '%')`,
		req.Email, req.FirstName, req.LastName, req.Position,
	).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees
		WHERE ($1 = '' OR email      ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR first_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR last_name  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR position   ILIKE '%' || $4 || '%')
		LIMIT $5 OFFSET $6`,
		req.Email, req.FirstName, req.LastName, req.Position, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var permissions pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
			&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
			&e.Department, &e.Active, &permissions, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Permissions = permissions
		employees = append(employees, &e)
	}
	return &pb.SearchEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) GetEmployeeCredentials(ctx context.Context, req *pb.GetEmployeeCredentialsRequest) (*pb.GetEmployeeCredentialsResponse, error) {
	var id int64
	var passwordHash, email, firstName string
	var active bool
	var failedAttempts int32
	var lockedUntil sql.NullTime
	var permissions pq.StringArray
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, password, active, permissions, email, first_name,
		        failed_login_attempts, account_locked_until
		 FROM employees WHERE email = $1`,
		req.Email,
	).Scan(&id, &passwordHash, &active, &permissions, &email, &firstName,
		&failedAttempts, &lockedUntil)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	lockedUntilStr := ""
	if lockedUntil.Valid {
		lockedUntilStr = lockedUntil.Time.UTC().Format(time.RFC3339)
	}
	return &pb.GetEmployeeCredentialsResponse{
		Id:                  id,
		PasswordHash:        passwordHash,
		Active:              active,
		Permissions:         permissions,
		Email:               email,
		FirstName:           firstName,
		FailedLoginAttempts: failedAttempts,
		AccountLockedUntil:  lockedUntilStr,
	}, nil
}

func (s *EmployeeServer) UpdateLoginAttempts(ctx context.Context, req *pb.UpdateLoginAttemptsRequest) (*pb.UpdateLoginAttemptsResponse, error) {
	var lockedUntil interface{}
	if req.LockedUntil == "" {
		lockedUntil = nil
	} else {
		t, err := time.Parse(time.RFC3339, req.LockedUntil)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid locked_until format, expected RFC3339")
		}
		lockedUntil = t
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE employees SET failed_login_attempts = $1, account_locked_until = $2 WHERE id = $3`,
		req.Attempts, lockedUntil, req.Id,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update login attempts: %v", err)
	}
	return &pb.UpdateLoginAttemptsResponse{}, nil
}

func (s *EmployeeServer) GetEmployeeById(ctx context.Context, req *pb.GetEmployeeByIdRequest) (*pb.GetEmployeeByIdResponse, error) {
	var e pb.Employee
	var permissions pq.StringArray
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees WHERE id = $1`, req.Id,
	).Scan(
		&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
		&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
		&e.Department, &e.Active, &permissions, &e.Jmbg,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	e.Permissions = permissions
	return &pb.GetEmployeeByIdResponse{Employee: &e}, nil
}

func (s *EmployeeServer) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.UpdateEmployeeResponse, error) {
	if err := validateEmployeeInput(req.Email, req.PhoneNumber, req.DateOfBirth); err != nil {
		return nil, err
	}
	// --- Permission validation (#143) ---
	hasAgent, hasSupervisor, hasAdmin := false, false, false
	for _, p := range req.Permissions {
		switch strings.ToUpper(p) {
		case "AGENT":
			hasAgent = true
		case "SUPERVISOR":
			hasSupervisor = true
		case "ADMIN":
			hasAdmin = true
		}
	}
	if hasAgent && hasSupervisor {
		return nil, status.Error(codes.InvalidArgument, "employee cannot have both AGENT and SUPERVISOR")
	}
	// Spec: Admin → automatically also Supervisor
	if hasAdmin && !hasSupervisor {
		req.Permissions = append(req.Permissions, "SUPERVISOR")
		hasSupervisor = true
	}

	// Spec: Admin cannot edit another admin
	var targetPerms pq.StringArray
	err := s.DB.QueryRowContext(ctx, `SELECT permissions FROM employees WHERE id = $1`, req.Id).Scan(&targetPerms)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if containsPermission(targetPerms, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "admins cannot be edited by other admins")
	}

	// Track old actuary roles for actuary_info sync (#144)
	hadAgent := containsPermission(targetPerms, "AGENT")
	hadSupervisor := containsPermission(targetPerms, "SUPERVISOR")

	// --- Active check ---
	if req.Active {
		var pwd string
		err := s.DB.QueryRowContext(ctx, `SELECT password FROM employees WHERE id = $1`, req.Id).Scan(&pwd)
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		if err != nil {
			return nil, err
		}
		if pwd == "" {
			return nil, status.Error(codes.FailedPrecondition, "cannot activate employee: no password set")
		}
	}

	var e pb.Employee
	var permissions pq.StringArray
	err = s.DB.QueryRowContext(ctx, `
		UPDATE employees
		SET first_name=$2, last_name=$3, date_of_birth=$4::date, gender=$5, email=$6,
		    phone_number=$7, address=$8, username=$9, position=$10,
		    department=$11, active=$12, permissions=$13, jmbg=$14
		WHERE id=$1
		RETURNING id, first_name, last_name, date_of_birth::text, gender, email,
		          phone_number, address, username, position, department, active, permissions, jmbg`,
		req.Id, req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.Email,
		req.PhoneNumber, req.Address, req.Username, req.Position,
		req.Department, req.Active, pq.StringArray(req.Permissions), req.Jmbg,
	).Scan(
		&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
		&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
		&e.Department, &e.Active, &permissions, &e.Jmbg,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			switch pqErr.Constraint {
			case "employees_username_key":
				return nil, status.Error(codes.AlreadyExists, "username already exists")
			case "employees_jmbg_key":
				return nil, status.Error(codes.AlreadyExists, "jmbg already exists")
			default:
				return nil, status.Error(codes.AlreadyExists, "email already exists")
			}
		}
		return nil, err
	}
	e.Permissions = permissions

	// --- Sync actuary_info (#144) ---
	// If employee gained AGENT or SUPERVISOR for the first time → insert row
	willHaveAgent := hasAgent
	willHaveSupervisor := hasSupervisor
	if (!hadAgent && willHaveAgent) || (!hadSupervisor && willHaveSupervisor) {
		if !hadAgent && !hadSupervisor {
			_, _ = s.DB.ExecContext(ctx,
				`INSERT INTO actuary_info (employee_id) VALUES ($1) ON CONFLICT DO NOTHING`,
				req.Id)
		}
	}
	// If employee lost both AGENT and SUPERVISOR → delete row
	if (hadAgent || hadSupervisor) && !willHaveAgent && !willHaveSupervisor {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM actuary_info WHERE employee_id = $1`, req.Id)
	}

	if req.ActorId != 0 {
		oldPerms := strings.Join(targetPerms, ",")
		newPerms := strings.Join(req.Permissions, ",")
		if oldPerms != newPerms {
			s.insertAuditLog(ctx, req.ActorId, "EMPLOYEE", "PERMISSION_CHANGE", req.Id, oldPerms, newPerms)
		}
	}

	return &pb.UpdateEmployeeResponse{Employee: &e}, nil
}

func (s *EmployeeServer) ActivateEmployee(ctx context.Context, req *pb.ActivateEmployeeRequest) (*pb.ActivateEmployeeResponse, error) {
	var active bool
	var pwd string
	err := s.DB.QueryRowContext(ctx, `SELECT active, password FROM employees WHERE id = $1`, req.EmployeeId).Scan(&active, &pwd)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if active || pwd != "" {
		return nil, status.Error(codes.FailedPrecondition, "employee already activated")
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE employees SET password = $2, active = true WHERE id = $1`, req.EmployeeId, req.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &pb.ActivateEmployeeResponse{}, nil
}

func (s *EmployeeServer) GetEmployeeByEmail(ctx context.Context, req *pb.GetEmployeeByEmailRequest) (*pb.GetEmployeeByEmailResponse, error) {
	var id int64
	var firstName, email string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, first_name, email FROM employees WHERE email = $1`,
		req.Email,
	).Scan(&id, &firstName, &email)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user with this email doesn't exist")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetEmployeeByEmailResponse{Id: id, FirstName: firstName, Email: email}, nil
}

func (s *EmployeeServer) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE employees SET password = $2 WHERE id = $1`,
		req.EmployeeId, req.PasswordHash,
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	return &pb.UpdatePasswordResponse{}, nil
}

func (s *EmployeeServer) CreateEmployee(ctx context.Context, req *pb.CreateEmployeeRequest) (*pb.CreateEmployeeResponse, error) {
	if err := validateEmployeeInput(req.Email, req.PhoneNumber, req.DateOfBirth); err != nil {
		return nil, err
	}
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO employees
			(first_name, last_name, date_of_birth, gender, email, phone_number, address, username,
			 password, position, department, active, permissions, jmbg)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, '', $9, $10, false, '{}', $11)
		RETURNING id`,
		req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.Email,
		req.PhoneNumber, req.Address, req.Username, req.Position, req.Department, req.Jmbg,
	).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			switch pqErr.Constraint {
			case "employees_username_key":
				return nil, status.Error(codes.AlreadyExists, "username already exists")
			case "employees_jmbg_key":
				return nil, status.Error(codes.AlreadyExists, "jmbg already exists")
			default:
				return nil, status.Error(codes.AlreadyExists, "email already exists")
			}
		}
		return nil, err
	}
	return &pb.CreateEmployeeResponse{
		Employee: &pb.Employee{
			Id:          id,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			Gender:      req.Gender,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
			Address:     req.Address,
			Username:    req.Username,
			Position:    req.Position,
			Department:  req.Department,
			Active:      false,
			Permissions: []string{},
			Jmbg:        req.Jmbg,
		},
	}, nil
}

// --- Actuary management handlers (issues #144–#145) ---

func (s *EmployeeServer) GetActuaries(ctx context.Context, req *pb.GetActuariesRequest) (*pb.GetActuariesResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT e.id, e.first_name, e.last_name, e.email, e.position,
		       a.limit_amount, a.used_limit, a.need_approval
		FROM employees e
		JOIN actuary_info a ON a.employee_id = e.id
		WHERE 'AGENT' = ANY(e.permissions)
		  AND ($1 = '' OR e.email      ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR e.first_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR e.last_name  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR e.position   ILIKE '%' || $4 || '%')
		ORDER BY e.last_name, e.first_name`,
		req.Email, req.FirstName, req.LastName, req.Position,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var actuaries []*pb.ActuaryInfo
	for rows.Next() {
		var a pb.ActuaryInfo
		if err := rows.Scan(&a.EmployeeId, &a.FirstName, &a.LastName, &a.Email, &a.Position,
			&a.LimitAmount, &a.UsedLimit, &a.NeedApproval); err != nil {
			return nil, err
		}
		actuaries = append(actuaries, &a)
	}
	if actuaries == nil {
		actuaries = []*pb.ActuaryInfo{}
	}
	return &pb.GetActuariesResponse{Actuaries: actuaries}, nil
}

func (s *EmployeeServer) GetSupervisors(ctx context.Context, _ *pb.GetSupervisorsRequest) (*pb.GetSupervisorsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, email, position
		FROM employees
		WHERE 'SUPERVISOR' = ANY(permissions) AND active = true
		ORDER BY last_name, first_name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var supervisors []*pb.SupervisorInfo
	for rows.Next() {
		var s pb.SupervisorInfo
		if err := rows.Scan(&s.EmployeeId, &s.FirstName, &s.LastName, &s.Email, &s.Position); err != nil {
			return nil, err
		}
		supervisors = append(supervisors, &s)
	}
	if supervisors == nil {
		supervisors = []*pb.SupervisorInfo{}
	}
	return &pb.GetSupervisorsResponse{Supervisors: supervisors}, nil
}

func (s *EmployeeServer) SetAgentLimit(ctx context.Context, req *pb.SetAgentLimitRequest) (*pb.SetAgentLimitResponse, error) {
	if req.LimitAmount < 0 {
		return nil, status.Error(codes.InvalidArgument, "limit must be non-negative")
	}
	// Verify the employee is an agent
	var perms pq.StringArray
	err := s.DB.QueryRowContext(ctx, `SELECT permissions FROM employees WHERE id = $1`, req.EmployeeId).Scan(&perms)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if !containsPermission(perms, "AGENT") {
		return nil, status.Error(codes.InvalidArgument, "employee is not an agent")
	}
	var oldLimit float64
	_ = s.DB.QueryRowContext(ctx, `SELECT limit_amount FROM actuary_info WHERE employee_id = $1`, req.EmployeeId).Scan(&oldLimit)
	res, err := s.DB.ExecContext(ctx,
		`UPDATE actuary_info SET limit_amount = $2 WHERE employee_id = $1`,
		req.EmployeeId, req.LimitAmount)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "actuary info not found")
	}
	if req.ActorId != 0 {
		s.insertAuditLog(ctx, req.ActorId, "EMPLOYEE", "LIMIT_CHANGE", req.EmployeeId,
			fmt.Sprintf("%.2f", oldLimit), fmt.Sprintf("%.2f", req.LimitAmount))
	}
	return &pb.SetAgentLimitResponse{}, nil
}

func (s *EmployeeServer) ResetAgentUsedLimit(ctx context.Context, req *pb.ResetAgentUsedLimitRequest) (*pb.ResetAgentUsedLimitResponse, error) {
	// Verify the employee is an agent
	var perms pq.StringArray
	err := s.DB.QueryRowContext(ctx, `SELECT permissions FROM employees WHERE id = $1`, req.EmployeeId).Scan(&perms)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if !containsPermission(perms, "AGENT") {
		return nil, status.Error(codes.InvalidArgument, "employee is not an agent")
	}
	var oldUsed float64
	_ = s.DB.QueryRowContext(ctx, `SELECT used_limit FROM actuary_info WHERE employee_id = $1`, req.EmployeeId).Scan(&oldUsed)
	res, err := s.DB.ExecContext(ctx,
		`UPDATE actuary_info SET used_limit = 0 WHERE employee_id = $1`,
		req.EmployeeId)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "actuary info not found")
	}
	if req.ActorId != 0 {
		s.insertAuditLog(ctx, req.ActorId, "EMPLOYEE", "RESET_USED_LIMIT", req.EmployeeId,
			fmt.Sprintf("%.2f", oldUsed), "0.00")
	}
	return &pb.ResetAgentUsedLimitResponse{}, nil
}

func (s *EmployeeServer) SetNeedApproval(ctx context.Context, req *pb.SetNeedApprovalRequest) (*pb.SetNeedApprovalResponse, error) {
	// Verify the employee is an agent
	var perms pq.StringArray
	err := s.DB.QueryRowContext(ctx, `SELECT permissions FROM employees WHERE id = $1`, req.EmployeeId).Scan(&perms)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if !containsPermission(perms, "AGENT") {
		return nil, status.Error(codes.InvalidArgument, "employee is not an agent")
	}
	res, err := s.DB.ExecContext(ctx,
		`UPDATE actuary_info SET need_approval = $2 WHERE employee_id = $1`,
		req.EmployeeId, req.NeedApproval)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "actuary info not found")
	}
	return &pb.SetNeedApprovalResponse{}, nil
}

func (s *EmployeeServer) ResetAllActuaryUsedLimits(ctx context.Context, _ *pb.ResetAllActuaryUsedLimitsRequest) (*pb.ResetAllActuaryUsedLimitsResponse, error) {
	_, err := s.DB.ExecContext(ctx, `UPDATE actuary_info SET used_limit = 0`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reset all actuary used limits: %v", err)
	}
	return &pb.ResetAllActuaryUsedLimitsResponse{}, nil
}

// GetActuaryPerformers returns all employees with AGENT, SUPERVISOR, or ADMIN permission.
// Used by the bank profit portal to show per-actuary realized P&L (#226).
func (s *EmployeeServer) GetActuaryPerformers(ctx context.Context, _ *pb.GetActuaryPerformersRequest) (*pb.GetActuaryPerformersResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name,
		       CASE
		           WHEN 'ADMIN'      = ANY(permissions) THEN 'ADMIN'
		           WHEN 'SUPERVISOR' = ANY(permissions) THEN 'SUPERVISOR'
		           ELSE 'AGENT'
		       END AS position
		FROM employees
		WHERE ('AGENT' = ANY(permissions) OR 'SUPERVISOR' = ANY(permissions) OR 'ADMIN' = ANY(permissions))
		  AND active = TRUE
		ORDER BY last_name, first_name`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get actuary performers: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var performers []*pb.ActuaryPerformer
	for rows.Next() {
		var p pb.ActuaryPerformer
		if err := rows.Scan(&p.UserId, &p.FirstName, &p.LastName, &p.Position); err != nil {
			return nil, status.Errorf(codes.Internal, "scan actuary performer: %v", err)
		}
		performers = append(performers, &p)
	}
	if performers == nil {
		performers = []*pb.ActuaryPerformer{}
	}
	return &pb.GetActuaryPerformersResponse{Performers: performers}, nil
}
