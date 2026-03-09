package grpc

import (
	pb "github.com/exbanka/backend/shared/pb/employee"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewEmployeeClient(addr string) (pb.EmployeeServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pb.NewEmployeeServiceClient(conn), conn, nil
}
