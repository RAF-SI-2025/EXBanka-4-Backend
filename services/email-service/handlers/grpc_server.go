package handlers

import (
	"context"
	"net/mail"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/exbanka/backend/shared/pb/email"
	"github.com/exbanka/backend/services/email-service/queue"
)

type EmailServer struct {
	pb.UnimplementedEmailServiceServer
	Producer *queue.Producer
}

func (s *EmailServer) SendActivationEmail(_ context.Context, req *pb.SendActivationEmailRequest) (*pb.SendActivationEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.Publish(queue.ActivationMessage{
		Email:          req.Email,
		FirstName:      req.FirstName,
		ActivationLink: req.ActivationLink,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendActivationEmailResponse{}, nil
}
