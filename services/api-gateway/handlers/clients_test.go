package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	clientpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- stub client service client ----

type stubClientSvcClient struct {
	getAllFn      func(context.Context, *clientpb.GetAllClientsRequest, ...grpc.CallOption) (*clientpb.GetAllClientsResponse, error)
	getByIdFn     func(context.Context, *clientpb.GetClientByIdRequest, ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error)
	createFn      func(context.Context, *clientpb.CreateClientRequest, ...grpc.CallOption) (*clientpb.CreateClientResponse, error)
	updateFn      func(context.Context, *clientpb.UpdateClientRequest, ...grpc.CallOption) (*clientpb.UpdateClientResponse, error)
	credentialsFn func(context.Context, *clientpb.GetClientCredentialsRequest, ...grpc.CallOption) (*clientpb.GetClientCredentialsResponse, error)
	activateFn    func(context.Context, *clientpb.ActivateClientRequest, ...grpc.CallOption) (*clientpb.ActivateClientResponse, error)
}

func (s *stubClientSvcClient) GetAllClients(ctx context.Context, in *clientpb.GetAllClientsRequest, opts ...grpc.CallOption) (*clientpb.GetAllClientsResponse, error) {
	if s.getAllFn != nil {
		return s.getAllFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) GetClientById(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
	if s.getByIdFn != nil {
		return s.getByIdFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) CreateClient(ctx context.Context, in *clientpb.CreateClientRequest, opts ...grpc.CallOption) (*clientpb.CreateClientResponse, error) {
	if s.createFn != nil {
		return s.createFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) UpdateClient(ctx context.Context, in *clientpb.UpdateClientRequest, opts ...grpc.CallOption) (*clientpb.UpdateClientResponse, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) GetClientCredentials(ctx context.Context, in *clientpb.GetClientCredentialsRequest, opts ...grpc.CallOption) (*clientpb.GetClientCredentialsResponse, error) {
	if s.credentialsFn != nil {
		return s.credentialsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) ActivateClient(ctx context.Context, in *clientpb.ActivateClientRequest, opts ...grpc.CallOption) (*clientpb.ActivateClientResponse, error) {
	if s.activateFn != nil {
		return s.activateFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubClientSvcClient) UpdateLoginAttempts(ctx context.Context, in *clientpb.UpdateLoginAttemptsRequest, opts ...grpc.CallOption) (*clientpb.UpdateLoginAttemptsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- helpers ----

func sampleClient() *clientpb.Client {
	return &clientpb.Client{
		Id:          1,
		FirstName:   "Ana",
		LastName:    "Anić",
		Jmbg:        "1234567890123",
		DateOfBirth: "1995-05-10",
		Gender:      "F",
		Email:       "ana@example.com",
		PhoneNumber: "+381601234567",
		Address:     "Knez Mihailova 1",
		Username:    "aanic",
		Active:      true,
	}
}

// makeClientToken produces a valid Bearer token with user_id=1 (simulates logged-in client).
func makeClientToken() string {
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"role":    "CLIENT",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(""))
	return "Bearer " + tok
}

// serveHandlerFull is like serveHandler but also accepts an Authorization header.
func serveHandlerFull(handler gin.HandlerFunc, method, path, urlPath, body, authHeader string) *httptest.ResponseRecorder {
	router := gin.New()
	router.Handle(method, path, handler)
	w := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, urlPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, urlPath, nil)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

// ---- GetMe ----

func TestGetMe_NoToken(t *testing.T) {
	svc := &stubClientSvcClient{}
	w := serveHandlerFull(GetMe(svc), "GET", "/client/me", "/client/me", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_NotFound(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(GetMe(svc), "GET", "/client/me", "/client/me", "", makeClientToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMe_InternalError(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandlerFull(GetMe(svc), "GET", "/client/me", "/client/me", "", makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetMe_Happy(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return &clientpb.GetClientByIdResponse{Client: sampleClient()}, nil
		},
	}
	w := serveHandlerFull(GetMe(svc), "GET", "/client/me", "/client/me", "", makeClientToken())
	require.Equal(t, http.StatusOK, w.Code)
	var resp clientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}

// ---- GetClients ----

func TestGetClients_Error(t *testing.T) {
	svc := &stubClientSvcClient{
		getAllFn: func(ctx context.Context, in *clientpb.GetAllClientsRequest, opts ...grpc.CallOption) (*clientpb.GetAllClientsResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetClients(svc), "GET", "/clients", "/clients", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetClients_Happy(t *testing.T) {
	svc := &stubClientSvcClient{
		getAllFn: func(ctx context.Context, in *clientpb.GetAllClientsRequest, opts ...grpc.CallOption) (*clientpb.GetAllClientsResponse, error) {
			return &clientpb.GetAllClientsResponse{Clients: []*clientpb.Client{sampleClient()}, TotalCount: 1}, nil
		},
	}
	w := serveHandler(GetClients(svc), "GET", "/clients", "/clients", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_count"])
}

// ---- GetClientById ----

func TestGetClientById_InvalidId(t *testing.T) {
	svc := &stubClientSvcClient{}
	w := serveHandler(GetClientById(svc), "GET", "/clients/:id", "/clients/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetClientById_NotFound(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(GetClientById(svc), "GET", "/clients/:id", "/clients/99", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetClientById_InternalError(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetClientById(svc), "GET", "/clients/:id", "/clients/1", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetClientById_Happy(t *testing.T) {
	svc := &stubClientSvcClient{
		getByIdFn: func(ctx context.Context, in *clientpb.GetClientByIdRequest, opts ...grpc.CallOption) (*clientpb.GetClientByIdResponse, error) {
			return &clientpb.GetClientByIdResponse{Client: sampleClient()}, nil
		},
	}
	w := serveHandler(GetClientById(svc), "GET", "/clients/:id", "/clients/1", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp clientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}

// ---- CreateClient ----

var validCreateClientBody = `{
	"first_name":"Ana","last_name":"Anić","jmbg":"1234567890123",
	"date_of_birth":"1995-05-10","gender":"F",
	"email":"ana@example.com","phone_number":"+381601234567",
	"address":"Knez Mihailova 1","username":"aanic"
}`

func TestCreateClient_BadJSON(t *testing.T) {
	w := serveHandler(CreateClient(&stubClientSvcClient{}, &stubAuthClient{}, &stubEmailClient{}), "POST", "/clients", "/clients", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateClient_MissingField(t *testing.T) {
	// missing username
	body := `{"first_name":"Ana","last_name":"Anić","jmbg":"1234567890123","date_of_birth":"1995-05-10","gender":"F","email":"ana@example.com","phone_number":"123","address":"a"}`
	w := serveHandler(CreateClient(&stubClientSvcClient{}, &stubAuthClient{}, &stubEmailClient{}), "POST", "/clients", "/clients", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateClient_InvalidEmail(t *testing.T) {
	body := `{"first_name":"Ana","last_name":"Anić","jmbg":"1234567890123","date_of_birth":"1995-05-10","gender":"F","email":"bad","phone_number":"123","address":"a","username":"u"}`
	w := serveHandler(CreateClient(&stubClientSvcClient{}, &stubAuthClient{}, &stubEmailClient{}), "POST", "/clients", "/clients", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateClient_Conflict(t *testing.T) {
	svc := &stubClientSvcClient{
		createFn: func(ctx context.Context, in *clientpb.CreateClientRequest, opts ...grpc.CallOption) (*clientpb.CreateClientResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		},
	}
	w := serveHandler(CreateClient(svc, &stubAuthClient{}, &stubEmailClient{}), "POST", "/clients", "/clients", validCreateClientBody)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateClient_SvcError(t *testing.T) {
	svc := &stubClientSvcClient{
		createFn: func(ctx context.Context, in *clientpb.CreateClientRequest, opts ...grpc.CallOption) (*clientpb.CreateClientResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(CreateClient(svc, &stubAuthClient{}, &stubEmailClient{}), "POST", "/clients", "/clients", validCreateClientBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateClient_AuthTokenError(t *testing.T) {
	svc := &stubClientSvcClient{
		createFn: func(ctx context.Context, in *clientpb.CreateClientRequest, opts ...grpc.CallOption) (*clientpb.CreateClientResponse, error) {
			return &clientpb.CreateClientResponse{Client: sampleClient()}, nil
		},
	}
	auth := &stubAuthClient{
		createClientActivationTokenFn: func(ctx context.Context, in *authpb.CreateClientActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateClientActivationTokenResponse, error) {
			return nil, fmt.Errorf("auth service down")
		},
	}
	w := serveHandler(CreateClient(svc, auth, &stubEmailClient{}), "POST", "/clients", "/clients", validCreateClientBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateClient_Happy(t *testing.T) {
	svc := &stubClientSvcClient{
		createFn: func(ctx context.Context, in *clientpb.CreateClientRequest, opts ...grpc.CallOption) (*clientpb.CreateClientResponse, error) {
			return &clientpb.CreateClientResponse{Client: sampleClient()}, nil
		},
	}
	auth := &stubAuthClient{
		createClientActivationTokenFn: func(ctx context.Context, in *authpb.CreateClientActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateClientActivationTokenResponse, error) {
			return &authpb.CreateClientActivationTokenResponse{Token: "tok123"}, nil
		},
	}
	w := serveHandler(CreateClient(svc, auth, &stubEmailClient{}), "POST", "/clients", "/clients", validCreateClientBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp clientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}

// ---- UpdateClient ----

var validUpdateClientBody = `{
	"first_name":"Ana","last_name":"Anić",
	"email":"ana@example.com","username":"aanic"
}`

func TestUpdateClient_InvalidId(t *testing.T) {
	w := serveHandler(UpdateClient(&stubClientSvcClient{}), "PUT", "/clients/:id", "/clients/abc", validUpdateClientBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClient_BadJSON(t *testing.T) {
	w := serveHandler(UpdateClient(&stubClientSvcClient{}), "PUT", "/clients/:id", "/clients/1", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClient_MissingField(t *testing.T) {
	// missing last_name
	w := serveHandler(UpdateClient(&stubClientSvcClient{}), "PUT", "/clients/:id", "/clients/1", `{"first_name":"Ana","email":"ana@example.com","username":"u"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClient_InvalidEmail(t *testing.T) {
	w := serveHandler(UpdateClient(&stubClientSvcClient{}), "PUT", "/clients/:id", "/clients/1", `{"first_name":"Ana","last_name":"Anić","email":"bad","username":"u"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClient_NotFound(t *testing.T) {
	svc := &stubClientSvcClient{
		updateFn: func(ctx context.Context, in *clientpb.UpdateClientRequest, opts ...grpc.CallOption) (*clientpb.UpdateClientResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(UpdateClient(svc), "PUT", "/clients/:id", "/clients/1", validUpdateClientBody)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateClient_Conflict(t *testing.T) {
	svc := &stubClientSvcClient{
		updateFn: func(ctx context.Context, in *clientpb.UpdateClientRequest, opts ...grpc.CallOption) (*clientpb.UpdateClientResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "email in use")
		},
	}
	w := serveHandler(UpdateClient(svc), "PUT", "/clients/:id", "/clients/1", validUpdateClientBody)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateClient_InternalError(t *testing.T) {
	svc := &stubClientSvcClient{
		updateFn: func(ctx context.Context, in *clientpb.UpdateClientRequest, opts ...grpc.CallOption) (*clientpb.UpdateClientResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(UpdateClient(svc), "PUT", "/clients/:id", "/clients/1", validUpdateClientBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateClient_Happy(t *testing.T) {
	svc := &stubClientSvcClient{
		updateFn: func(ctx context.Context, in *clientpb.UpdateClientRequest, opts ...grpc.CallOption) (*clientpb.UpdateClientResponse, error) {
			return &clientpb.UpdateClientResponse{Client: sampleClient()}, nil
		},
	}
	w := serveHandler(UpdateClient(svc), "PUT", "/clients/:id", "/clients/1", validUpdateClientBody)
	require.Equal(t, http.StatusOK, w.Code)
	var resp clientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}
