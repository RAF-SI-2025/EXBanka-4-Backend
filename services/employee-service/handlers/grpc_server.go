package handlers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		SELECT id, ime, prezime, datum_rodjenja::text, pol, email,
		       broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole, jmbg
		FROM employees
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var dozvole pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.Ime, &e.Prezime, &e.DatumRodjenja, &e.Pol, &e.Email,
			&e.BrojTelefona, &e.Adresa, &e.Username, &e.Pozicija,
			&e.Departman, &e.Aktivan, &dozvole, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Dozvole = dozvole
		employees = append(employees, &e)
	}
	return &pb.GetAllEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) SearchEmployees(ctx context.Context, req *pb.SearchEmployeesRequest) (*pb.SearchEmployeesResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM employees
		WHERE ($1 = '' OR email    ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR ime      ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR prezime  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR pozicija ILIKE '%' || $4 || '%')`,
		req.Email, req.Ime, req.Prezime, req.Pozicija,
	).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, ime, prezime, datum_rodjenja::text, pol, email,
		       broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole, jmbg
		FROM employees
		WHERE ($1 = '' OR email    ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR ime      ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR prezime  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR pozicija ILIKE '%' || $4 || '%')
		LIMIT $5 OFFSET $6`,
		req.Email, req.Ime, req.Prezime, req.Pozicija, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var dozvole pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.Ime, &e.Prezime, &e.DatumRodjenja, &e.Pol, &e.Email,
			&e.BrojTelefona, &e.Adresa, &e.Username, &e.Pozicija,
			&e.Departman, &e.Aktivan, &dozvole, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Dozvole = dozvole
		employees = append(employees, &e)
	}
	return &pb.SearchEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) GetEmployeeCredentials(ctx context.Context, req *pb.GetEmployeeCredentialsRequest) (*pb.GetEmployeeCredentialsResponse, error) {
	var id int64
	var passwordHash string
	var aktivan bool
	var dozvole pq.StringArray
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, password, aktivan, dozvole FROM employees WHERE email = $1`,
		req.Email,
	).Scan(&id, &passwordHash, &aktivan, &dozvole)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetEmployeeCredentialsResponse{Id: id, PasswordHash: passwordHash, Aktivan: aktivan, Dozvole: dozvole}, nil
}

func (s *EmployeeServer) GetEmployeeById(ctx context.Context, req *pb.GetEmployeeByIdRequest) (*pb.GetEmployeeByIdResponse, error) {
	var e pb.Employee
	var dozvole pq.StringArray
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, ime, prezime, datum_rodjenja::text, pol, email,
		       broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole, jmbg
		FROM employees WHERE id = $1`, req.Id,
	).Scan(
		&e.Id, &e.Ime, &e.Prezime, &e.DatumRodjenja, &e.Pol, &e.Email,
		&e.BrojTelefona, &e.Adresa, &e.Username, &e.Pozicija,
		&e.Departman, &e.Aktivan, &dozvole, &e.Jmbg,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	e.Dozvole = dozvole
	return &pb.GetEmployeeByIdResponse{Employee: &e}, nil
}

func (s *EmployeeServer) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.UpdateEmployeeResponse, error) {
	// Activating an employee requires a password to be set first.
	if req.Aktivan {
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
	var dozvole pq.StringArray
	err := s.DB.QueryRowContext(ctx, `
		UPDATE employees
		SET ime=$2, prezime=$3, datum_rodjenja=$4::date, pol=$5, email=$6,
		    broj_telefona=$7, adresa=$8, username=$9, pozicija=$10,
		    departman=$11, aktivan=$12, dozvole=$13, jmbg=$14
		WHERE id=$1
		RETURNING id, ime, prezime, datum_rodjenja::text, pol, email,
		          broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole, jmbg`,
		req.Id, req.Ime, req.Prezime, req.DatumRodjenja, req.Pol, req.Email,
		req.BrojTelefona, req.Adresa, req.Username, req.Pozicija,
		req.Departman, req.Aktivan, pq.StringArray(req.Dozvole), req.Jmbg,
	).Scan(
		&e.Id, &e.Ime, &e.Prezime, &e.DatumRodjenja, &e.Pol, &e.Email,
		&e.BrojTelefona, &e.Adresa, &e.Username, &e.Pozicija,
		&e.Departman, &e.Aktivan, &dozvole, &e.Jmbg,
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
	e.Dozvole = dozvole
	return &pb.UpdateEmployeeResponse{Employee: &e}, nil
}

func (s *EmployeeServer) ActivateEmployee(ctx context.Context, req *pb.ActivateEmployeeRequest) (*pb.ActivateEmployeeResponse, error) {
	var aktivan bool
	var pwd string
	err := s.DB.QueryRowContext(ctx, `SELECT aktivan, password FROM employees WHERE id = $1`, req.EmployeeId).Scan(&aktivan, &pwd)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if aktivan || pwd != "" {
		return nil, status.Error(codes.FailedPrecondition, "employee already activated")
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE employees SET password = $2, aktivan = true WHERE id = $1`, req.EmployeeId, req.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &pb.ActivateEmployeeResponse{}, nil
}

func (s *EmployeeServer) GetEmployeeByEmail(ctx context.Context, req *pb.GetEmployeeByEmailRequest) (*pb.GetEmployeeByEmailResponse, error) {
	var id int64
	var firstName, email string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, ime, email FROM employees WHERE email = $1`,
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
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO employees
			(ime, prezime, datum_rodjenja, pol, email, broj_telefona, adresa, username,
			 password, pozicija, departman, aktivan, dozvole, jmbg)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, '', $9, $10, false, '{}', $11)
		RETURNING id`,
		req.Ime, req.Prezime, req.DatumRodjenja, req.Pol, req.Email,
		req.BrojTelefona, req.Adresa, req.Username, req.Pozicija, req.Departman, req.Jmbg,
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
			Id:            id,
			Ime:           req.Ime,
			Prezime:       req.Prezime,
			DatumRodjenja: req.DatumRodjenja,
			Pol:           req.Pol,
			Email:         req.Email,
			BrojTelefona:  req.BrojTelefona,
			Adresa:        req.Adresa,
			Username:      req.Username,
			Pozicija:      req.Pozicija,
			Departman:     req.Departman,
			Aktivan:       false,
			Dozvole:       []string{},
			Jmbg:          req.Jmbg,
		},
	}, nil
}
