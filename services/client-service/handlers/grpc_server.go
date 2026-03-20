package handlers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClientServer struct {
	pb.UnimplementedClientServiceServer
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

func scanClient(row interface {
	Scan(...any) error
}) (*pb.Client, error) {
	var c pb.Client
	return &c, row.Scan(
		&c.Id, &c.FirstName, &c.LastName, &c.Jmbg, &c.DateOfBirth,
		&c.Gender, &c.Email, &c.PhoneNumber, &c.Address, &c.Username, &c.Active,
	)
}

func clientUniqueError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		switch pqErr.Constraint {
		case "clients_email_key":
			return status.Error(codes.AlreadyExists, "email already exists")
		case "clients_username_key":
			return status.Error(codes.AlreadyExists, "username already exists")
		case "clients_jmbg_key":
			return status.Error(codes.AlreadyExists, "jmbg already exists")
		}
	}
	return nil
}

func (s *ClientServer) GetAllClients(ctx context.Context, req *pb.GetAllClientsRequest) (*pb.GetAllClientsResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients`).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, jmbg, date_of_birth::text,
		       gender, email, phone_number, address, username, active
		FROM clients
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*pb.Client
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return &pb.GetAllClientsResponse{Clients: clients, TotalCount: total}, nil
}

func (s *ClientServer) GetClientById(ctx context.Context, req *pb.GetClientByIdRequest) (*pb.GetClientByIdResponse, error) {
	c, err := scanClient(s.DB.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, jmbg, date_of_birth::text,
		       gender, email, phone_number, address, username, active
		FROM clients WHERE id = $1`, req.Id))
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "client not found")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetClientByIdResponse{Client: c}, nil
}

func (s *ClientServer) CreateClient(ctx context.Context, req *pb.CreateClientRequest) (*pb.CreateClientResponse, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO clients
			(first_name, last_name, jmbg, date_of_birth, gender, email, phone_number, address, username, active)
		VALUES ($1, $2, $3, $4::date, $5, $6, $7, $8, $9, false)
		RETURNING id`,
		req.FirstName, req.LastName, req.Jmbg, req.DateOfBirth, req.Gender,
		req.Email, req.PhoneNumber, req.Address, req.Username,
	).Scan(&id)
	if err != nil {
		if uniqueErr := clientUniqueError(err); uniqueErr != nil {
			return nil, uniqueErr
		}
		return nil, err
	}
	return &pb.CreateClientResponse{
		Client: &pb.Client{
			Id:          id,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Jmbg:        req.Jmbg,
			DateOfBirth: req.DateOfBirth,
			Gender:      req.Gender,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
			Address:     req.Address,
			Username:    req.Username,
			Active:      false,
		},
	}, nil
}

func (s *ClientServer) UpdateClient(ctx context.Context, req *pb.UpdateClientRequest) (*pb.UpdateClientResponse, error) {
	c, err := scanClient(s.DB.QueryRowContext(ctx, `
		UPDATE clients
		SET first_name=$2, last_name=$3, jmbg=$4, date_of_birth=$5::date,
		    gender=$6, email=$7, phone_number=$8, address=$9, username=$10, active=$11
		WHERE id=$1
		RETURNING id, first_name, last_name, jmbg, date_of_birth::text,
		          gender, email, phone_number, address, username, active`,
		req.Id, req.FirstName, req.LastName, req.Jmbg, req.DateOfBirth,
		req.Gender, req.Email, req.PhoneNumber, req.Address, req.Username, req.Active,
	))
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "client not found")
	}
	if err != nil {
		if uniqueErr := clientUniqueError(err); uniqueErr != nil {
			return nil, uniqueErr
		}
		return nil, err
	}
	return &pb.UpdateClientResponse{Client: c}, nil
}

func (s *ClientServer) ActivateClient(ctx context.Context, req *pb.ActivateClientRequest) (*pb.ActivateClientResponse, error) {
	result, err := s.DB.ExecContext(ctx,
		`UPDATE clients SET password = $1, active = true WHERE id = $2`,
		req.PasswordHash, req.ClientId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to activate client: %v", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "client not found")
	}
	return &pb.ActivateClientResponse{}, nil
}

func (s *ClientServer) GetClientCredentials(ctx context.Context, req *pb.GetClientCredentialsRequest) (*pb.GetClientCredentialsResponse, error) {
	var id int64
	var passwordHash string
	var active bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, password, active FROM clients WHERE email = $1`,
		req.Email,
	).Scan(&id, &passwordHash, &active)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "client not found")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetClientCredentialsResponse{Id: id, PasswordHash: passwordHash, Active: active}, nil
}
