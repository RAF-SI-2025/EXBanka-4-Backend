package grpc

import (
	emailpb "github.com/exbanka/backend/shared/pb/email"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewEmailClient(addr string) (emailpb.EmailServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return emailpb.NewEmailServiceClient(conn), conn, nil
}
