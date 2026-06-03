package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- stub ----

type stubPaymentClient struct {
	createPaymentFn            func(context.Context, *pb.CreatePaymentRequest, ...grpc.CallOption) (*pb.CreatePaymentResponse, error)
	getPaymentsFn              func(context.Context, *pb.GetPaymentsRequest, ...grpc.CallOption) (*pb.GetPaymentsResponse, error)
	getPaymentByIdFn           func(context.Context, *pb.GetPaymentByIdRequest, ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error)
	createTransferFn           func(context.Context, *pb.CreateTransferRequest, ...grpc.CallOption) (*pb.CreateTransferResponse, error)
	createPaymentRecipientFn   func(context.Context, *pb.CreatePaymentRecipientRequest, ...grpc.CallOption) (*pb.CreatePaymentRecipientResponse, error)
	getPaymentRecipientsFn     func(context.Context, *pb.GetPaymentRecipientsRequest, ...grpc.CallOption) (*pb.GetPaymentRecipientsResponse, error)
	updatePaymentRecipientFn   func(context.Context, *pb.UpdatePaymentRecipientRequest, ...grpc.CallOption) (*pb.UpdatePaymentRecipientResponse, error)
	deletePaymentRecipientFn   func(context.Context, *pb.DeletePaymentRecipientRequest, ...grpc.CallOption) (*pb.DeletePaymentRecipientResponse, error)
	reorderPaymentRecipientsFn func(context.Context, *pb.ReorderPaymentRecipientsRequest, ...grpc.CallOption) (*pb.ReorderPaymentRecipientsResponse, error)
	getTransfersFn             func(context.Context, *pb.GetTransfersRequest, ...grpc.CallOption) (*pb.GetTransfersResponse, error)
	previewPaymentFn           func(context.Context, *pb.PreviewPaymentRequest, ...grpc.CallOption) (*pb.PreviewPaymentResponse, error)
}

func (s *stubPaymentClient) CreatePayment(ctx context.Context, in *pb.CreatePaymentRequest, opts ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
	if s.createPaymentFn != nil {
		return s.createPaymentFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) GetPayments(ctx context.Context, in *pb.GetPaymentsRequest, opts ...grpc.CallOption) (*pb.GetPaymentsResponse, error) {
	if s.getPaymentsFn != nil {
		return s.getPaymentsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) GetPaymentById(ctx context.Context, in *pb.GetPaymentByIdRequest, opts ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error) {
	if s.getPaymentByIdFn != nil {
		return s.getPaymentByIdFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) CreateTransfer(ctx context.Context, in *pb.CreateTransferRequest, opts ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
	if s.createTransferFn != nil {
		return s.createTransferFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) CreatePaymentRecipient(ctx context.Context, in *pb.CreatePaymentRecipientRequest, opts ...grpc.CallOption) (*pb.CreatePaymentRecipientResponse, error) {
	if s.createPaymentRecipientFn != nil {
		return s.createPaymentRecipientFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) GetPaymentRecipients(ctx context.Context, in *pb.GetPaymentRecipientsRequest, opts ...grpc.CallOption) (*pb.GetPaymentRecipientsResponse, error) {
	if s.getPaymentRecipientsFn != nil {
		return s.getPaymentRecipientsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) UpdatePaymentRecipient(ctx context.Context, in *pb.UpdatePaymentRecipientRequest, opts ...grpc.CallOption) (*pb.UpdatePaymentRecipientResponse, error) {
	if s.updatePaymentRecipientFn != nil {
		return s.updatePaymentRecipientFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) DeletePaymentRecipient(ctx context.Context, in *pb.DeletePaymentRecipientRequest, opts ...grpc.CallOption) (*pb.DeletePaymentRecipientResponse, error) {
	if s.deletePaymentRecipientFn != nil {
		return s.deletePaymentRecipientFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) ReorderPaymentRecipients(ctx context.Context, in *pb.ReorderPaymentRecipientsRequest, opts ...grpc.CallOption) (*pb.ReorderPaymentRecipientsResponse, error) {
	if s.reorderPaymentRecipientsFn != nil {
		return s.reorderPaymentRecipientsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) GetTransfers(ctx context.Context, in *pb.GetTransfersRequest, opts ...grpc.CallOption) (*pb.GetTransfersResponse, error) {
	if s.getTransfersFn != nil {
		return s.getTransfersFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) PrepareInterbankPayment(ctx context.Context, in *pb.PrepareInterbankPaymentRequest, opts ...grpc.CallOption) (*pb.PrepareInterbankPaymentResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) CommitInterbankPayment(ctx context.Context, in *pb.CommitRollbackInterbankRequest, opts ...grpc.CallOption) (*pb.CommitRollbackInterbankResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) RollbackInterbankPayment(ctx context.Context, in *pb.CommitRollbackInterbankRequest, opts ...grpc.CallOption) (*pb.CommitRollbackInterbankResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubPaymentClient) PreviewPayment(ctx context.Context, in *pb.PreviewPaymentRequest, opts ...grpc.CallOption) (*pb.PreviewPaymentResponse, error) {
	if s.previewPaymentFn != nil {
		return s.previewPaymentFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}

// ---- CreatePayment ----

func TestCreatePayment_BadJSON(t *testing.T) {
	w := serveHandlerFull(CreatePayment(&stubPaymentClient{}), "POST", "/payments", "/payments", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreatePayment_NoToken(t *testing.T) {
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandler(CreatePayment(&stubPaymentClient{}), "POST", "/payments", "/payments", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestCreatePayment_NotFound(t *testing.T) {
	svc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *pb.CreatePaymentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
		return nil, status.Error(codes.NotFound, "account not found")
	}}
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandlerFull(CreatePayment(svc), "POST", "/payments", "/payments", body, makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestCreatePayment_PermissionDenied(t *testing.T) {
	svc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *pb.CreatePaymentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}}
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandlerFull(CreatePayment(svc), "POST", "/payments", "/payments", body, makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestCreatePayment_FailedPrecondition(t *testing.T) {
	svc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *pb.CreatePaymentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
		return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
	}}
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandlerFull(CreatePayment(svc), "POST", "/payments", "/payments", body, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreatePayment_Error(t *testing.T) {
	svc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *pb.CreatePaymentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandlerFull(CreatePayment(svc), "POST", "/payments", "/payments", body, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestCreatePayment_Happy(t *testing.T) {
	svc := &stubPaymentClient{createPaymentFn: func(_ context.Context, _ *pb.CreatePaymentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentResponse, error) {
		return &pb.CreatePaymentResponse{Id: 1, OrderNumber: "ORD1", Status: "COMPLETED"}, nil
	}}
	body := `{"recipientName":"A","recipientAccount":"123","amount":10,"fromAccount":"acc"}`
	w := serveHandlerFull(CreatePayment(svc), "POST", "/payments", "/payments", body, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

// ---- CreatePaymentRecipient ----

func TestCreatePaymentRecipient_BadJSON(t *testing.T) {
	w := serveHandlerFull(CreatePaymentRecipient(&stubPaymentClient{}), "POST", "/recipients", "/recipients", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreatePaymentRecipient_NoToken(t *testing.T) {
	w := serveHandler(CreatePaymentRecipient(&stubPaymentClient{}), "POST", "/recipients", "/recipients", `{"name":"A","accountNumber":"123"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestCreatePaymentRecipient_Error(t *testing.T) {
	svc := &stubPaymentClient{createPaymentRecipientFn: func(_ context.Context, _ *pb.CreatePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.CreatePaymentRecipientResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(CreatePaymentRecipient(svc), "POST", "/recipients", "/recipients", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestCreatePaymentRecipient_Happy(t *testing.T) {
	svc := &stubPaymentClient{createPaymentRecipientFn: func(_ context.Context, _ *pb.CreatePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.CreatePaymentRecipientResponse, error) {
		return &pb.CreatePaymentRecipientResponse{Recipient: &pb.PaymentRecipient{Id: 1, Name: "A", AccountNumber: "123"}}, nil
	}}
	w := serveHandlerFull(CreatePaymentRecipient(svc), "POST", "/recipients", "/recipients", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

// ---- GetPaymentRecipients ----

func TestGetPaymentRecipients_NoToken(t *testing.T) {
	w := serveHandler(GetPaymentRecipients(&stubPaymentClient{}), "GET", "/recipients", "/recipients", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetPaymentRecipients_Error(t *testing.T) {
	svc := &stubPaymentClient{getPaymentRecipientsFn: func(_ context.Context, _ *pb.GetPaymentRecipientsRequest, _ ...grpc.CallOption) (*pb.GetPaymentRecipientsResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(GetPaymentRecipients(svc), "GET", "/recipients", "/recipients", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetPaymentRecipients_Happy(t *testing.T) {
	svc := &stubPaymentClient{getPaymentRecipientsFn: func(_ context.Context, _ *pb.GetPaymentRecipientsRequest, _ ...grpc.CallOption) (*pb.GetPaymentRecipientsResponse, error) {
		return &pb.GetPaymentRecipientsResponse{Recipients: []*pb.PaymentRecipient{{Id: 1, Name: "A", AccountNumber: "123", Order: 1}}}, nil
	}}
	w := serveHandlerFull(GetPaymentRecipients(svc), "GET", "/recipients", "/recipients", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- ReorderPaymentRecipients ----

func TestReorderPaymentRecipients_NoToken(t *testing.T) {
	w := serveHandler(ReorderPaymentRecipients(&stubPaymentClient{}), "PUT", "/recipients/reorder", "/recipients/reorder", `{"orderedIds":[1,2]}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestReorderPaymentRecipients_BadJSON(t *testing.T) {
	w := serveHandlerFull(ReorderPaymentRecipients(&stubPaymentClient{}), "PUT", "/recipients/reorder", "/recipients/reorder", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestReorderPaymentRecipients_Error(t *testing.T) {
	svc := &stubPaymentClient{reorderPaymentRecipientsFn: func(_ context.Context, _ *pb.ReorderPaymentRecipientsRequest, _ ...grpc.CallOption) (*pb.ReorderPaymentRecipientsResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(ReorderPaymentRecipients(svc), "PUT", "/recipients/reorder", "/recipients/reorder", `{"orderedIds":[1,2]}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestReorderPaymentRecipients_Happy(t *testing.T) {
	svc := &stubPaymentClient{reorderPaymentRecipientsFn: func(_ context.Context, _ *pb.ReorderPaymentRecipientsRequest, _ ...grpc.CallOption) (*pb.ReorderPaymentRecipientsResponse, error) {
		return &pb.ReorderPaymentRecipientsResponse{}, nil
	}}
	w := serveHandlerFull(ReorderPaymentRecipients(svc), "PUT", "/recipients/reorder", "/recipients/reorder", `{"orderedIds":[1,2]}`, makeClientToken())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

// ---- UpdatePaymentRecipient ----

func TestUpdatePaymentRecipient_BadID(t *testing.T) {
	w := serveHandlerFull(UpdatePaymentRecipient(&stubPaymentClient{}), "PUT", "/recipients/:id", "/recipients/abc", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUpdatePaymentRecipient_BadJSON(t *testing.T) {
	w := serveHandlerFull(UpdatePaymentRecipient(&stubPaymentClient{}), "PUT", "/recipients/:id", "/recipients/1", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestUpdatePaymentRecipient_NoToken(t *testing.T) {
	w := serveHandler(UpdatePaymentRecipient(&stubPaymentClient{}), "PUT", "/recipients/:id", "/recipients/1", `{"name":"A","accountNumber":"123"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestUpdatePaymentRecipient_NotFound(t *testing.T) {
	svc := &stubPaymentClient{updatePaymentRecipientFn: func(_ context.Context, _ *pb.UpdatePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.UpdatePaymentRecipientResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(UpdatePaymentRecipient(svc), "PUT", "/recipients/:id", "/recipients/1", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestUpdatePaymentRecipient_Error(t *testing.T) {
	svc := &stubPaymentClient{updatePaymentRecipientFn: func(_ context.Context, _ *pb.UpdatePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.UpdatePaymentRecipientResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(UpdatePaymentRecipient(svc), "PUT", "/recipients/:id", "/recipients/1", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestUpdatePaymentRecipient_Happy(t *testing.T) {
	svc := &stubPaymentClient{updatePaymentRecipientFn: func(_ context.Context, _ *pb.UpdatePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.UpdatePaymentRecipientResponse, error) {
		return &pb.UpdatePaymentRecipientResponse{Recipient: &pb.PaymentRecipient{Id: 1, Name: "A", AccountNumber: "123"}}, nil
	}}
	w := serveHandlerFull(UpdatePaymentRecipient(svc), "PUT", "/recipients/:id", "/recipients/1", `{"name":"A","accountNumber":"123"}`, makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetPaymentById ----

func TestGetPaymentById_BadID(t *testing.T) {
	w := serveHandlerFull(GetPaymentById(&stubPaymentClient{}), "GET", "/payments/:paymentId", "/payments/abc", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestGetPaymentById_NoToken(t *testing.T) {
	w := serveHandler(GetPaymentById(&stubPaymentClient{}), "GET", "/payments/:paymentId", "/payments/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetPaymentById_NotFound(t *testing.T) {
	svc := &stubPaymentClient{getPaymentByIdFn: func(_ context.Context, _ *pb.GetPaymentByIdRequest, _ ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(GetPaymentById(svc), "GET", "/payments/:paymentId", "/payments/1", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestGetPaymentById_PermissionDenied(t *testing.T) {
	svc := &stubPaymentClient{getPaymentByIdFn: func(_ context.Context, _ *pb.GetPaymentByIdRequest, _ ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}}
	w := serveHandlerFull(GetPaymentById(svc), "GET", "/payments/:paymentId", "/payments/1", "", makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestGetPaymentById_Error(t *testing.T) {
	svc := &stubPaymentClient{getPaymentByIdFn: func(_ context.Context, _ *pb.GetPaymentByIdRequest, _ ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(GetPaymentById(svc), "GET", "/payments/:paymentId", "/payments/1", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetPaymentById_Happy(t *testing.T) {
	svc := &stubPaymentClient{getPaymentByIdFn: func(_ context.Context, _ *pb.GetPaymentByIdRequest, _ ...grpc.CallOption) (*pb.GetPaymentByIdResponse, error) {
		return &pb.GetPaymentByIdResponse{Payment: &pb.Payment{Id: 1, Status: "COMPLETED"}}, nil
	}}
	w := serveHandlerFull(GetPaymentById(svc), "GET", "/payments/:paymentId", "/payments/1", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- GetPayments ----

func TestGetPayments_NoToken(t *testing.T) {
	w := serveHandler(GetPayments(&stubPaymentClient{}), "GET", "/payments", "/payments", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetPayments_Error(t *testing.T) {
	svc := &stubPaymentClient{getPaymentsFn: func(_ context.Context, _ *pb.GetPaymentsRequest, _ ...grpc.CallOption) (*pb.GetPaymentsResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(GetPayments(svc), "GET", "/payments", "/payments", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetPayments_Happy(t *testing.T) {
	svc := &stubPaymentClient{getPaymentsFn: func(_ context.Context, _ *pb.GetPaymentsRequest, _ ...grpc.CallOption) (*pb.GetPaymentsResponse, error) {
		return &pb.GetPaymentsResponse{Payments: []*pb.Payment{{Id: 1, Status: "COMPLETED"}}}, nil
	}}
	w := serveHandlerFull(GetPayments(svc), "GET", "/payments", "/payments?amount_min=1&amount_max=100&limit=10&offset=0", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- CreateTransfer ----

func TestCreateTransfer_BadJSON(t *testing.T) {
	w := serveHandlerFull(CreateTransfer(&stubPaymentClient{}), "POST", "/transfers", "/transfers", `{bad}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateTransfer_NoToken(t *testing.T) {
	w := serveHandler(CreateTransfer(&stubPaymentClient{}), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestCreateTransfer_InvalidArgument(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return nil, status.Error(codes.InvalidArgument, "same account")
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateTransfer_NotFound(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return nil, status.Error(codes.NotFound, "account not found")
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestCreateTransfer_PermissionDenied(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestCreateTransfer_FailedPrecondition(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateTransfer_Error(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return nil, fmt.Errorf("internal")
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestCreateTransfer_Happy(t *testing.T) {
	svc := &stubPaymentClient{createTransferFn: func(_ context.Context, _ *pb.CreateTransferRequest, _ ...grpc.CallOption) (*pb.CreateTransferResponse, error) {
		return &pb.CreateTransferResponse{Id: 1, OrderNumber: "T1"}, nil
	}}
	w := serveHandlerFull(CreateTransfer(svc), "POST", "/transfers", "/transfers", `{"fromAccount":"a","toAccount":"b","amount":10}`, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

// ---- GetTransfers ----

func TestGetTransfers_NoToken(t *testing.T) {
	w := serveHandler(GetTransfers(&stubPaymentClient{}), "GET", "/transfers", "/transfers", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestGetTransfers_Error(t *testing.T) {
	svc := &stubPaymentClient{getTransfersFn: func(_ context.Context, _ *pb.GetTransfersRequest, _ ...grpc.CallOption) (*pb.GetTransfersResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(GetTransfers(svc), "GET", "/transfers", "/transfers", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestGetTransfers_Happy(t *testing.T) {
	svc := &stubPaymentClient{getTransfersFn: func(_ context.Context, _ *pb.GetTransfersRequest, _ ...grpc.CallOption) (*pb.GetTransfersResponse, error) {
		return &pb.GetTransfersResponse{Transfers: []*pb.Transfer{{Id: 1, OrderNumber: "T1"}}}, nil
	}}
	w := serveHandlerFull(GetTransfers(svc), "GET", "/transfers", "/transfers", "", makeClientToken())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ---- DeletePaymentRecipient ----

func TestDeletePaymentRecipient_BadID(t *testing.T) {
	w := serveHandlerFull(DeletePaymentRecipient(&stubPaymentClient{}), "DELETE", "/recipients/:id", "/recipients/abc", "", makeClientToken())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestDeletePaymentRecipient_NoToken(t *testing.T) {
	w := serveHandler(DeletePaymentRecipient(&stubPaymentClient{}), "DELETE", "/recipients/:id", "/recipients/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestDeletePaymentRecipient_NotFound(t *testing.T) {
	svc := &stubPaymentClient{deletePaymentRecipientFn: func(_ context.Context, _ *pb.DeletePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.DeletePaymentRecipientResponse, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}}
	w := serveHandlerFull(DeletePaymentRecipient(svc), "DELETE", "/recipients/:id", "/recipients/1", "", makeClientToken())
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestDeletePaymentRecipient_Error(t *testing.T) {
	svc := &stubPaymentClient{deletePaymentRecipientFn: func(_ context.Context, _ *pb.DeletePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.DeletePaymentRecipientResponse, error) {
		return nil, fmt.Errorf("db error")
	}}
	w := serveHandlerFull(DeletePaymentRecipient(svc), "DELETE", "/recipients/:id", "/recipients/1", "", makeClientToken())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}

func TestDeletePaymentRecipient_Happy(t *testing.T) {
	svc := &stubPaymentClient{deletePaymentRecipientFn: func(_ context.Context, _ *pb.DeletePaymentRecipientRequest, _ ...grpc.CallOption) (*pb.DeletePaymentRecipientResponse, error) {
		return &pb.DeletePaymentRecipientResponse{}, nil
	}}
	w := serveHandlerFull(DeletePaymentRecipient(svc), "DELETE", "/recipients/:id", "/recipients/1", "", makeClientToken())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}
