package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc"
)

// ─── CreateNegotiation: cross-bank fields ──────────────────────────────────

var crossBankNegotiationBody = `{
	"sellerId": 1,
	"sellerType": "CLIENT",
	"sellerRoutingNumber": 999,
	"sellerExternalId": "ext-seller-42",
	"ticker": "AAPL",
	"amount": 100,
	"pricePerStock": 150.0,
	"settlementDate": "2027-06-01",
	"premium": 5.0,
	"currency": "RSD"
}`

// TestCreateNegotiation_CrossBankFields_ForwardedToGRPC verifies that
// sellerRoutingNumber and sellerExternalId from the JSON body reach the gRPC call.
func TestCreateNegotiation_CrossBankFields_ForwardedToGRPC(t *testing.T) {
	var captured *pb.CreateNegotiationRequest
	svc := &stubOtcClient{
		createNegotiationFn: func(_ context.Context, req *pb.CreateNegotiationRequest, _ ...grpc.CallOption) (*pb.NegotiationResponse, error) {
			captured = req
			return sampleNegotiation(), nil
		},
	}
	w := serveHandlerFull(CreateNegotiation(svc),
		"POST", "/otc/negotiations", "/otc/negotiations",
		crossBankNegotiationBody, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d body=%s", w.Code, w.Body.String())
	}
	if captured == nil {
		t.Fatal("gRPC CreateNegotiation was not called")
	}
	if captured.SellerRoutingNumber != 999 {
		t.Errorf("expected SellerRoutingNumber=999, got %d", captured.SellerRoutingNumber)
	}
	if captured.SellerExternalId != "ext-seller-42" {
		t.Errorf("expected SellerExternalId='ext-seller-42', got '%s'", captured.SellerExternalId)
	}
}

// TestCreateNegotiation_CrossBankFields_ZeroWhenOmitted ensures that
// omitting the cross-bank fields still works (intra-bank case, defaults to 0/"").
func TestCreateNegotiation_CrossBankFields_ZeroWhenOmitted(t *testing.T) {
	var captured *pb.CreateNegotiationRequest
	svc := &stubOtcClient{
		createNegotiationFn: func(_ context.Context, req *pb.CreateNegotiationRequest, _ ...grpc.CallOption) (*pb.NegotiationResponse, error) {
			captured = req
			return sampleNegotiation(), nil
		},
	}
	w := serveHandlerFull(CreateNegotiation(svc),
		"POST", "/otc/negotiations", "/otc/negotiations",
		validCreateNegotiationBody, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	if captured.SellerRoutingNumber != 0 {
		t.Errorf("expected SellerRoutingNumber=0 (omitted), got %d", captured.SellerRoutingNumber)
	}
	if captured.SellerExternalId != "" {
		t.Errorf("expected SellerExternalId='' (omitted), got '%s'", captured.SellerExternalId)
	}
}

// TestCreateNegotiation_CrossBankFields_ResponseShape checks the JSON response
// still has the expected shape when called with cross-bank fields.
func TestCreateNegotiation_CrossBankFields_ResponseShape(t *testing.T) {
	svc := &stubOtcClient{
		createNegotiationFn: func(_ context.Context, _ *pb.CreateNegotiationRequest, _ ...grpc.CallOption) (*pb.NegotiationResponse, error) {
			neg := sampleNegotiation()
			neg.SellerType = "INTERBANK"
			return neg, nil
		},
	}
	w := serveHandlerFull(CreateNegotiation(svc),
		"POST", "/otc/negotiations", "/otc/negotiations",
		crossBankNegotiationBody, makeClientToken())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	for _, field := range []string{"id", "ticker", "status", "sellerType"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing expected field %q", field)
		}
	}
	if resp["sellerType"] != "INTERBANK" {
		t.Errorf("expected sellerType='INTERBANK', got %v", resp["sellerType"])
	}
}
