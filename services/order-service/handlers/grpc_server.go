package handlers

import (
	"context"
	"database/sql"

	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	DB               *sql.DB
	AccountDB        *sql.DB
	SecuritiesDB     *sql.DB
	ExchangeDB       *sql.DB
	SecuritiesClient pb_sec.SecuritiesServiceClient
	LoanClient       pb_loan.LoanServiceClient
	EmployeeClient   pb_emp.EmployeeServiceClient
}

func (s *OrderServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "order-service OK"}, nil
}
