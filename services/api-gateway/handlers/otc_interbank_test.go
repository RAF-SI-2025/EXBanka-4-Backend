package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockOtcClient is a minimal stub implementing pb.OtcServiceClient.
type mockOtcClient struct {
	createInterbankFn      func(ctx context.Context, req *pb.CreateInterbankNegotiationRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error)
	counterOfferFn         func(ctx context.Context, req *pb.InterbankCounterOfferRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error)
	getNegotiationFn       func(ctx context.Context, req *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error)
	deleteNegotiationFn    func(ctx context.Context, req *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error)
	acceptNegotiationFn    func(ctx context.Context, req *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error)
}

func (m *mockOtcClient) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) CreateNegotiation(ctx context.Context, in *pb.CreateNegotiationRequest, opts ...grpc.CallOption) (*pb.NegotiationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) ListNegotiations(ctx context.Context, in *pb.ListNegotiationsRequest, opts ...grpc.CallOption) (*pb.ListNegotiationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) GetNegotiation(ctx context.Context, in *pb.GetNegotiationRequest, opts ...grpc.CallOption) (*pb.NegotiationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) CounterOffer(ctx context.Context, in *pb.CounterOfferRequest, opts ...grpc.CallOption) (*pb.NegotiationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) AcceptNegotiation(ctx context.Context, in *pb.AcceptNegotiationRequest, opts ...grpc.CallOption) (*pb.NegotiationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) RejectNegotiation(ctx context.Context, in *pb.RejectNegotiationRequest, opts ...grpc.CallOption) (*pb.NegotiationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) ListContracts(ctx context.Context, in *pb.ListContractsRequest, opts ...grpc.CallOption) (*pb.ListContractsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) ExerciseContract(ctx context.Context, in *pb.ExerciseContractRequest, opts ...grpc.CallOption) (*pb.ExerciseContractResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) GetMarket(ctx context.Context, in *pb.GetMarketRequest, opts ...grpc.CallOption) (*pb.GetMarketResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) CreateInterbankNegotiation(ctx context.Context, in *pb.CreateInterbankNegotiationRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
	if m.createInterbankFn != nil {
		return m.createInterbankFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) InterbankCounterOffer(ctx context.Context, in *pb.InterbankCounterOfferRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
	if m.counterOfferFn != nil {
		return m.counterOfferFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) InterbankGetNegotiation(ctx context.Context, in *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
	if m.getNegotiationFn != nil {
		return m.getNegotiationFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) InterbankDeleteNegotiation(ctx context.Context, in *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
	if m.deleteNegotiationFn != nil {
		return m.deleteNegotiationFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) InterbankAcceptNegotiation(ctx context.Context, in *pb.InterbankNegotiationIdRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
	if m.acceptNegotiationFn != nil {
		return m.acceptNegotiationFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) PrepareOtcInterbank(ctx context.Context, in *pb.OtcInterbankPrepareRequest, opts ...grpc.CallOption) (*pb.OtcInterbankVoteResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) CommitOtcInterbank(ctx context.Context, in *pb.OtcInterbankTxRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}
func (m *mockOtcClient) RollbackOtcInterbank(ctx context.Context, in *pb.OtcInterbankTxRequest, opts ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not mocked")
}

// doOtcRequest fires a single POST request against the given handler on a fresh gin router.
func doOtcRequest(t *testing.T, handler gin.HandlerFunc, method, _ string, body any, apiKey string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/test", handler)

	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, "/test", reqBody)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── IncomingCreateNegotiation ─────────────────────────────────────────────────

func TestIncomingCreateNegotiation_NoApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{}
	w := doOtcRequest(t, IncomingCreateNegotiation(mock), http.MethodPost, "/test", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingCreateNegotiation_WrongApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{}
	w := doOtcRequest(t, IncomingCreateNegotiation(mock), http.MethodPost, "/test", nil, "wrong-key")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingCreateNegotiation_InvalidBody(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	router := gin.New()
	mock := &mockOtcClient{}
	router.POST("/test", IncomingCreateNegotiation(mock))

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncomingCreateNegotiation_GrpcError(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{
		createInterbankFn: func(_ context.Context, _ *pb.CreateInterbankNegotiationRequest, _ ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}

	body := validNegotiationBody()
	w := doOtcRequest(t, IncomingCreateNegotiation(mock), http.MethodPost, "/test", body, "secret")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIncomingCreateNegotiation_Success(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{
		createInterbankFn: func(_ context.Context, _ *pb.CreateInterbankNegotiationRequest, _ ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
			return &pb.InterbankNegotiationResponse{LocalId: 42}, nil
		},
	}

	body := validNegotiationBody()
	w := doOtcRequest(t, IncomingCreateNegotiation(mock), http.MethodPost, "/test", body, "secret")
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "42", resp["id"])
	assert.Equal(t, float64(888), resp["routingNumber"])
}

// ── GetExternalStocks ─────────────────────────────────────────────────────────

func TestGetExternalStocks_PartnerDown(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")
	t.Setenv("PARTNER_BANK_URL", "http://127.0.0.1:19998")
	t.Setenv("PARTNER_API_KEY", "key")
	t.Setenv("PARTNER_BANK_NAME", "Banka 4")

	router := gin.New()
	router.GET("/public-stock", GetExternalStocks())
	req := httptest.NewRequest(http.MethodGet, "/public-stock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	items, _ := resp["items"].([]any)
	assert.Empty(t, items)
}

func TestGetExternalStocks_PartnerReturnsStocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"ticker":"AAPL"},{"ticker":"GOOG"}]`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")
	t.Setenv("PARTNER_BANK_URL", srv.URL)
	t.Setenv("PARTNER_API_KEY", "key")
	t.Setenv("PARTNER_BANK_NAME", "Banka 4")

	router := gin.New()
	router.GET("/public-stock", GetExternalStocks())
	req := httptest.NewRequest(http.MethodGet, "/public-stock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	items, _ := resp["items"].([]any)
	assert.Len(t, items, 2)
}

// ── IncomingCounterOffer ──────────────────────────────────────────────────────

func TestIncomingCounterOffer_NoApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{}
	w := doPathRequest(t, "PUT", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingCounterOffer(mock), nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingCounterOffer_Success(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{
		counterOfferFn: func(_ context.Context, _ *pb.InterbankCounterOfferRequest, _ ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
			return &pb.InterbankNegotiationResponse{LocalId: 42, Ticker: "AAPL", Amount: 10}, nil
		},
	}
	body := map[string]any{
		"pricePerUnit":   map[string]any{"currency": "RSD", "amount": 110.0},
		"premium":        map[string]any{"currency": "RSD", "amount": 5.0},
		"amount":         10,
		"settlementDate": "2026-12-31",
	}
	w := doPathRequest(t, "PUT", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingCounterOffer(mock), body, "secret")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── IncomingGetNegotiation ────────────────────────────────────────────────────

func TestIncomingGetNegotiation_NoApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingGetNegotiation(mock), nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingGetNegotiation_NotFound(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{
		getNegotiationFn: func(_ context.Context, _ *pb.InterbankNegotiationIdRequest, _ ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingGetNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncomingGetNegotiation_Success(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	t.Setenv("OWN_ROUTING_NUMBER", "888")

	mock := &mockOtcClient{
		getNegotiationFn: func(_ context.Context, req *pb.InterbankNegotiationIdRequest, _ ...grpc.CallOption) (*pb.InterbankNegotiationResponse, error) {
			assert.Equal(t, int32(444), req.RoutingNumber)
			assert.Equal(t, "42", req.ExternalId)
			return &pb.InterbankNegotiationResponse{Ticker: "AAPL", Amount: 10, PriceCurrency: "RSD"}, nil
		},
	}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingGetNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── IncomingDeleteNegotiation ─────────────────────────────────────────────────

func TestIncomingDeleteNegotiation_NoApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{}
	w := doPathRequest(t, "DELETE", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingDeleteNegotiation(mock), nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingDeleteNegotiation_Success(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{
		deleteNegotiationFn: func(_ context.Context, _ *pb.InterbankNegotiationIdRequest, _ ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
			return &pb.OtcEmptyResponse{}, nil
		},
	}
	w := doPathRequest(t, "DELETE", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/444/42", IncomingDeleteNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── IncomingAcceptNegotiation ─────────────────────────────────────────────────

func TestIncomingAcceptNegotiation_NoApiKey(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id/accept",
		"/otc/interbank/negotiations/444/42/accept", IncomingAcceptNegotiation(mock), nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncomingAcceptNegotiation_GrpcError(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{
		acceptNegotiationFn: func(_ context.Context, _ *pb.InterbankNegotiationIdRequest, _ ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "already accepted")
		},
	}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id/accept",
		"/otc/interbank/negotiations/444/42/accept", IncomingAcceptNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestIncomingAcceptNegotiation_Success(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{
		acceptNegotiationFn: func(_ context.Context, _ *pb.InterbankNegotiationIdRequest, _ ...grpc.CallOption) (*pb.OtcEmptyResponse, error) {
			return &pb.OtcEmptyResponse{}, nil
		},
	}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id/accept",
		"/otc/interbank/negotiations/444/42/accept", IncomingAcceptNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── parseInterbankNegPath (via a handler) ─────────────────────────────────────

func TestParseInterbankNegPath_InvalidRoutingNumber(t *testing.T) {
	t.Setenv("OWN_INTERBANK_API_KEY", "secret")
	mock := &mockOtcClient{}
	w := doPathRequest(t, "GET", "/otc/interbank/negotiations/:routingNumber/:id",
		"/otc/interbank/negotiations/notanumber/42", IncomingGetNegotiation(mock), nil, "secret")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// doPathRequest fires a request against a handler registered at a route with gin path params.
func doPathRequest(t *testing.T, method, route, path string, handler gin.HandlerFunc, body any, apiKey string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Handle(method, route, handler)

	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func validNegotiationBody() map[string]any {
	return map[string]any{
		"stock":          map[string]any{"ticker": "AAPL"},
		"settlementDate": "2026-12-31",
		"pricePerUnit":   map[string]any{"currency": "RSD", "amount": 100.0},
		"premium":        map[string]any{"currency": "RSD", "amount": 5.0},
		"buyerId":        map[string]any{"routingNumber": 444, "id": "42"},
		"sellerId":       map[string]any{"routingNumber": 888, "id": "7"},
		"amount":         10,
		"sellerType":     "CLIENT",
	}
}
