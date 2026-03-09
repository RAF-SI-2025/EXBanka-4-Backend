package handlers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	pb "github.com/exbanka/backend/shared/pb/employee"
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
