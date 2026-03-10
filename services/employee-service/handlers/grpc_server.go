package handlers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	pb "github.com/exbanka/backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EmployeeServer struct {
	pb.UnimplementedEmployeeServiceServer
	DB *sql.DB
}

func (s *EmployeeServer) GetAllEmployees(ctx context.Context, _ *pb.GetAllEmployeesRequest) (*pb.GetAllEmployeesResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, ime, prezime, datum_rodjenja::text, pol, email,
		       broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole
		FROM employees`)
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
			&e.Departman, &e.Aktivan, &dozvole,
		); err != nil {
			return nil, err
		}
		e.Dozvole = dozvole
		employees = append(employees, &e)
	}
	return &pb.GetAllEmployeesResponse{Employees: employees}, nil
}

func (s *EmployeeServer) SearchEmployees(ctx context.Context, req *pb.SearchEmployeesRequest) (*pb.SearchEmployeesResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, ime, prezime, datum_rodjenja::text, pol, email,
		       broj_telefona, adresa, username, pozicija, departman, aktivan, dozvole
		FROM employees
		WHERE ($1 = '' OR email    = $1)
		  AND ($2 = '' OR ime      = $2)
		  AND ($3 = '' OR prezime  = $3)
		  AND ($4 = '' OR pozicija = $4)`,
		req.Email, req.Ime, req.Prezime, req.Pozicija)
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
			&e.Departman, &e.Aktivan, &dozvole,
		); err != nil {
			return nil, err
		}
		e.Dozvole = dozvole
		employees = append(employees, &e)
	}
	return &pb.SearchEmployeesResponse{Employees: employees}, nil
}

func (s *EmployeeServer) GetEmployeeCredentials(ctx context.Context, req *pb.GetEmployeeCredentialsRequest) (*pb.GetEmployeeCredentialsResponse, error) {
	var id int64
	var passwordHash string
	var dozvole pq.StringArray
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, password, dozvole FROM employees WHERE username = $1`,
		req.Username,
	).Scan(&id, &passwordHash, &dozvole)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetEmployeeCredentialsResponse{Id: id, PasswordHash: passwordHash, Dozvole: dozvole}, nil
}

func (s *EmployeeServer) CreateEmployee(ctx context.Context, req *pb.CreateEmployeeRequest) (*pb.CreateEmployeeResponse, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO employees
			(ime, prezime, datum_rodjenja, pol, email, broj_telefona, adresa, username,
			 password, pozicija, departman, aktivan, dozvole)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, '', $9, $10, false, '{}')
		RETURNING id`,
		req.Ime, req.Prezime, req.DatumRodjenja, req.Pol, req.Email,
		req.BrojTelefona, req.Adresa, req.Username, req.Pozicija, req.Departman,
	).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
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
		},
	}, nil
}
