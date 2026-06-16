package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	accountpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	cardpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- stub account service client ----

type stubAccountClient struct {
	createFn          func(context.Context, *accountpb.CreateAccountRequest, ...grpc.CallOption) (*accountpb.CreateAccountResponse, error)
	getMyAccountsFn   func(context.Context, *accountpb.GetMyAccountsRequest, ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error)
	getAccountFn      func(context.Context, *accountpb.GetAccountRequest, ...grpc.CallOption) (*accountpb.GetAccountResponse, error)
	renameAccountFn   func(context.Context, *accountpb.RenameAccountRequest, ...grpc.CallOption) (*accountpb.RenameAccountResponse, error)
	getAllAccountsFn  func(context.Context, *accountpb.GetAllAccountsRequest, ...grpc.CallOption) (*accountpb.GetAllAccountsResponse, error)
	updateLimitsFn    func(context.Context, *accountpb.UpdateAccountLimitsRequest, ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error)
	deleteAccountFn   func(context.Context, *accountpb.DeleteAccountRequest, ...grpc.CallOption) (*accountpb.DeleteAccountResponse, error)
	getBankAccountsFn func(context.Context, *accountpb.GetBankAccountsRequest, ...grpc.CallOption) (*accountpb.GetBankAccountsResponse, error)
}

func (s *stubAccountClient) CreateAccount(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
	if s.createFn != nil {
		return s.createFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) GetMyAccounts(ctx context.Context, in *accountpb.GetMyAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
	if s.getMyAccountsFn != nil {
		return s.getMyAccountsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) GetAccount(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
	if s.getAccountFn != nil {
		return s.getAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) RenameAccount(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
	if s.renameAccountFn != nil {
		return s.renameAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) GetAllAccounts(ctx context.Context, in *accountpb.GetAllAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetAllAccountsResponse, error) {
	if s.getAllAccountsFn != nil {
		return s.getAllAccountsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) UpdateAccountLimits(ctx context.Context, in *accountpb.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
	if s.updateLimitsFn != nil {
		return s.updateLimitsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) DeleteAccount(ctx context.Context, in *accountpb.DeleteAccountRequest, opts ...grpc.CallOption) (*accountpb.DeleteAccountResponse, error) {
	if s.deleteAccountFn != nil {
		return s.deleteAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAccountClient) GetBankAccounts(ctx context.Context, in *accountpb.GetBankAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetBankAccountsResponse, error) {
	if s.getBankAccountsFn != nil {
		return s.getBankAccountsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}

// ---- stub card service client ----

type stubCardClient struct {
	createCardFn          func(context.Context, *cardpb.CreateCardRequest, ...grpc.CallOption) (*cardpb.CreateCardResponse, error)
	getCardsByAccountFn   func(context.Context, *cardpb.GetCardsByAccountRequest, ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error)
	getCardByNumberFn     func(context.Context, *cardpb.GetCardByNumberRequest, ...grpc.CallOption) (*cardpb.GetCardByNumberResponse, error)
	getCardByIdFn         func(context.Context, *cardpb.GetCardByIdRequest, ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error)
	blockCardFn           func(context.Context, *cardpb.BlockCardRequest, ...grpc.CallOption) (*cardpb.BlockCardResponse, error)
	unblockCardFn         func(context.Context, *cardpb.UnblockCardRequest, ...grpc.CallOption) (*cardpb.UnblockCardResponse, error)
	deactivateCardFn      func(context.Context, *cardpb.DeactivateCardRequest, ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error)
	updateCardLimitFn     func(context.Context, *cardpb.UpdateCardLimitRequest, ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error)
	initiateCardRequestFn func(context.Context, *cardpb.InitiateCardRequestRequest, ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error)
	confirmCardRequestFn  func(context.Context, *cardpb.ConfirmCardRequestRequest, ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error)
}

func (s *stubCardClient) CreateCard(ctx context.Context, in *cardpb.CreateCardRequest, opts ...grpc.CallOption) (*cardpb.CreateCardResponse, error) {
	if s.createCardFn != nil {
		return s.createCardFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) GetCardsByAccount(ctx context.Context, in *cardpb.GetCardsByAccountRequest, opts ...grpc.CallOption) (*cardpb.GetCardsByAccountResponse, error) {
	if s.getCardsByAccountFn != nil {
		return s.getCardsByAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) GetCardByNumber(ctx context.Context, in *cardpb.GetCardByNumberRequest, opts ...grpc.CallOption) (*cardpb.GetCardByNumberResponse, error) {
	if s.getCardByNumberFn != nil {
		return s.getCardByNumberFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) GetCardById(ctx context.Context, in *cardpb.GetCardByIdRequest, opts ...grpc.CallOption) (*cardpb.GetCardByIdResponse, error) {
	if s.getCardByIdFn != nil {
		return s.getCardByIdFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) BlockCard(ctx context.Context, in *cardpb.BlockCardRequest, opts ...grpc.CallOption) (*cardpb.BlockCardResponse, error) {
	if s.blockCardFn != nil {
		return s.blockCardFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) UnblockCard(ctx context.Context, in *cardpb.UnblockCardRequest, opts ...grpc.CallOption) (*cardpb.UnblockCardResponse, error) {
	if s.unblockCardFn != nil {
		return s.unblockCardFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) DeactivateCard(ctx context.Context, in *cardpb.DeactivateCardRequest, opts ...grpc.CallOption) (*cardpb.DeactivateCardResponse, error) {
	if s.deactivateCardFn != nil {
		return s.deactivateCardFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) UpdateCardLimit(ctx context.Context, in *cardpb.UpdateCardLimitRequest, opts ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
	if s.updateCardLimitFn != nil {
		return s.updateCardLimitFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) InitiateCardRequest(ctx context.Context, in *cardpb.InitiateCardRequestRequest, opts ...grpc.CallOption) (*cardpb.InitiateCardRequestResponse, error) {
	if s.initiateCardRequestFn != nil {
		return s.initiateCardRequestFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubCardClient) ConfirmCardRequest(ctx context.Context, in *cardpb.ConfirmCardRequestRequest, opts ...grpc.CallOption) (*cardpb.ConfirmCardRequestResponse, error) {
	if s.confirmCardRequestFn != nil {
		return s.confirmCardRequestFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}

// ---- GetMyAccounts ----

func TestGetMyAccounts_NoToken(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(GetMyAccounts(svc), "GET", "/client/accounts", "/client/accounts", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMyAccounts_Error(t *testing.T) {
	svc := &stubAccountClient{
		getMyAccountsFn: func(ctx context.Context, in *accountpb.GetMyAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandlerFull(GetMyAccounts(svc), "GET", "/client/accounts", "/client/accounts", "", makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetMyAccounts_Happy(t *testing.T) {
	svc := &stubAccountClient{
		getMyAccountsFn: func(ctx context.Context, in *accountpb.GetMyAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetMyAccountsResponse, error) {
			return &accountpb.GetMyAccountsResponse{Accounts: []*accountpb.AccountSummary{
				{Id: 1, AccountName: "Test", AccountNumber: "265001", AvailableBalance: 5000, CurrencyCode: "RSD"},
			}}, nil
		},
	}
	w := serveHandlerFull(GetMyAccounts(svc), "GET", "/client/accounts", "/client/accounts", "", makeClientToken())
	require.Equal(t, http.StatusOK, w.Code)
	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
}

// ---- GetAccount ----

func TestGetAccount_InvalidId(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/abc", "", makeClientToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAccount_NoToken(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/1", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/99", "", makeClientToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAccount_PermissionDenied(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "not yours")
		},
	}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/1", "", makeClientToken())
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetAccount_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/1", "", makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAccount_Happy(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return &accountpb.GetAccountResponse{Account: &accountpb.AccountDetails{
				AccountName: "Test", AccountNumber: "265001", CurrencyCode: "RSD",
			}}, nil
		},
	}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/1", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- GetAccountAdmin ----

func TestGetAccountAdmin_InvalidId(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandler(GetAccountAdmin(svc), "GET", "/admin/accounts/:accountId", "/admin/accounts/bad", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAccountAdmin_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(GetAccountAdmin(svc), "GET", "/admin/accounts/:accountId", "/admin/accounts/99", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAccountAdmin_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetAccountAdmin(svc), "GET", "/admin/accounts/:accountId", "/admin/accounts/1", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAccountAdmin_Happy(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return &accountpb.GetAccountResponse{Account: &accountpb.AccountDetails{
				AccountName: "Test", AccountNumber: "265001",
			}}, nil
		},
	}
	w := serveHandler(GetAccountAdmin(svc), "GET", "/admin/accounts/:accountId", "/admin/accounts/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- RenameAccount ----

func TestRenameAccount_InvalidId(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/bad/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenameAccount_NoToken(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRenameAccount_MissingField(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{}`, makeClientToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenameAccount_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		renameAccountFn: func(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenameAccount_PermissionDenied(t *testing.T) {
	svc := &stubAccountClient{
		renameAccountFn: func(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "not yours")
		},
	}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRenameAccount_InvalidArgument(t *testing.T) {
	svc := &stubAccountClient{
		renameAccountFn: func(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "bad name")
		},
	}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenameAccount_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		renameAccountFn: func(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRenameAccount_Happy(t *testing.T) {
	svc := &stubAccountClient{
		renameAccountFn: func(ctx context.Context, in *accountpb.RenameAccountRequest, opts ...grpc.CallOption) (*accountpb.RenameAccountResponse, error) {
			return &accountpb.RenameAccountResponse{}, nil
		},
	}
	w := serveHandlerFull(RenameAccount(svc), "PUT", "/client/accounts/:accountId/name", "/client/accounts/1/name", `{"newAccountName":"New"}`, makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- GetAllAccounts ----

func TestGetAllAccounts_Error(t *testing.T) {
	svc := &stubAccountClient{
		getAllAccountsFn: func(ctx context.Context, in *accountpb.GetAllAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetAllAccountsResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetAllAccounts(svc), "GET", "/admin/accounts", "/admin/accounts", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAllAccounts_Happy(t *testing.T) {
	svc := &stubAccountClient{
		getAllAccountsFn: func(ctx context.Context, in *accountpb.GetAllAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetAllAccountsResponse, error) {
			return &accountpb.GetAllAccountsResponse{Accounts: []*accountpb.AccountListItem{
				{Id: 1, AccountNumber: "265001", AccountName: "Test", CurrencyCode: "RSD"},
			}}, nil
		},
	}
	w := serveHandler(GetAllAccounts(svc), "GET", "/admin/accounts", "/admin/accounts", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- UpdateAccountLimits (gateway handler) ----

func TestUpdateAccountLimitsHandler_InvalidId(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandler(UpdateAccountLimits(svc, &stubEmailClient{}, &stubAuthClient{}, &stubClientSvcClient{}), "PUT", "/admin/accounts/:accountId/limits", "/admin/accounts/bad/limits", `{"dailyLimit":100,"monthlyLimit":500}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateAccountLimitsHandler_MissingField(t *testing.T) {
	svc := &stubAccountClient{}
	w := serveHandler(UpdateAccountLimits(svc, &stubEmailClient{}, &stubAuthClient{}, &stubClientSvcClient{}), "PUT", "/admin/accounts/:accountId/limits", "/admin/accounts/1/limits", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateAccountLimitsHandler_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		updateLimitsFn: func(ctx context.Context, in *accountpb.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(UpdateAccountLimits(svc, &stubEmailClient{}, &stubAuthClient{}, &stubClientSvcClient{}), "PUT", "/admin/accounts/:accountId/limits", "/admin/accounts/1/limits", `{"dailyLimit":100,"monthlyLimit":500}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAccountLimitsHandler_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		updateLimitsFn: func(ctx context.Context, in *accountpb.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(UpdateAccountLimits(svc, &stubEmailClient{}, &stubAuthClient{}, &stubClientSvcClient{}), "PUT", "/admin/accounts/:accountId/limits", "/admin/accounts/1/limits", `{"dailyLimit":100,"monthlyLimit":500}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateAccountLimitsHandler_Happy(t *testing.T) {
	svc := &stubAccountClient{
		updateLimitsFn: func(ctx context.Context, in *accountpb.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
			return &accountpb.UpdateAccountLimitsResponse{}, nil
		},
	}
	w := serveHandler(UpdateAccountLimits(svc, &stubEmailClient{}, &stubAuthClient{}, &stubClientSvcClient{}), "PUT", "/admin/accounts/:accountId/limits", "/admin/accounts/1/limits", `{"dailyLimit":100,"monthlyLimit":500}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- GetBankAccounts (gateway handler) ----

func TestGetBankAccountsHandler_Error(t *testing.T) {
	svc := &stubAccountClient{
		getBankAccountsFn: func(ctx context.Context, in *accountpb.GetBankAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetBankAccountsResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetBankAccounts(svc), "GET", "/admin/bank-accounts", "/admin/bank-accounts", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetBankAccountsHandler_Happy(t *testing.T) {
	svc := &stubAccountClient{
		getBankAccountsFn: func(ctx context.Context, in *accountpb.GetBankAccountsRequest, opts ...grpc.CallOption) (*accountpb.GetBankAccountsResponse, error) {
			return &accountpb.GetBankAccountsResponse{Accounts: []*accountpb.BankAccountItem{
				{AccountNumber: "265000", AccountName: "EUR Bank", CurrencyCode: "EUR"},
			}}, nil
		},
	}
	w := serveHandler(GetBankAccounts(svc), "GET", "/admin/bank-accounts", "/admin/bank-accounts", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
}

// ---- DeleteAccount (gateway handler) ----

func TestDeleteAccountHandler_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		deleteAccountFn: func(ctx context.Context, in *accountpb.DeleteAccountRequest, opts ...grpc.CallOption) (*accountpb.DeleteAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(DeleteAccount(svc), "DELETE", "/admin/accounts/:accountId", "/admin/accounts/99", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAccountHandler_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		deleteAccountFn: func(ctx context.Context, in *accountpb.DeleteAccountRequest, opts ...grpc.CallOption) (*accountpb.DeleteAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(DeleteAccount(svc), "DELETE", "/admin/accounts/:accountId", "/admin/accounts/1", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteAccountHandler_Happy(t *testing.T) {
	svc := &stubAccountClient{
		deleteAccountFn: func(ctx context.Context, in *accountpb.DeleteAccountRequest, opts ...grpc.CallOption) (*accountpb.DeleteAccountResponse, error) {
			return &accountpb.DeleteAccountResponse{}, nil
		},
	}
	w := serveHandler(DeleteAccount(svc), "DELETE", "/admin/accounts/:accountId", "/admin/accounts/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- CreateAccount (gateway handler) ----

var validCreateAccountBody = `{
	"clientId":1,"accountType":"CURRENT","currencyCode":"RSD"
}`

func TestCreateAccountHandler_BadJSON(t *testing.T) {
	svc := &stubAccountClient{}
	card := &stubCardClient{}
	w := serveHandler(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", `{bad}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccountHandler_MissingField(t *testing.T) {
	svc := &stubAccountClient{}
	card := &stubCardClient{}
	w := serveHandler(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", `{"clientId":1}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccountHandler_NoToken(t *testing.T) {
	svc := &stubAccountClient{}
	card := &stubCardClient{}
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", validCreateAccountBody, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateAccountHandler_NotFound(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "client not found")
		},
	}
	card := &stubCardClient{}
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", validCreateAccountBody, makeClientToken())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateAccountHandler_InvalidArgument(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "bad currency")
		},
	}
	card := &stubCardClient{}
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", validCreateAccountBody, makeClientToken())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccountHandler_InternalError(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	card := &stubCardClient{}
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", validCreateAccountBody, makeClientToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateAccountHandler_Happy(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{
				Id: 1, AccountNumber: "265001", AccountName: "Test", CurrencyCode: "RSD", Status: "ACTIVE",
			}}, nil
		},
	}
	card := &stubCardClient{}
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", validCreateAccountBody, makeClientToken())
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateAccountHandler_WithLimits(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{
				Id: 1, AccountNumber: "265001",
			}}, nil
		},
		updateLimitsFn: func(ctx context.Context, in *accountpb.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*accountpb.UpdateAccountLimitsResponse, error) {
			return &accountpb.UpdateAccountLimitsResponse{}, nil
		},
	}
	card := &stubCardClient{}
	body := `{"clientId":1,"accountType":"CURRENT","currencyCode":"RSD","dailyLimit":1000,"monthlyLimit":5000}`
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateAccountHandler_WithCard(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{
				Id: 1, AccountNumber: "265001",
			}}, nil
		},
	}
	card := &stubCardClient{
		createCardFn: func(ctx context.Context, in *cardpb.CreateCardRequest, opts ...grpc.CallOption) (*cardpb.CreateCardResponse, error) {
			return &cardpb.CreateCardResponse{Card: &cardpb.CardResponse{CardNumber: "4111111111111111"}}, nil
		},
		updateCardLimitFn: func(ctx context.Context, in *cardpb.UpdateCardLimitRequest, opts ...grpc.CallOption) (*cardpb.UpdateCardLimitResponse, error) {
			return &cardpb.UpdateCardLimitResponse{}, nil
		},
	}
	body := `{"clientId":1,"accountType":"CURRENT","currencyCode":"RSD","createCard":true,"cardLimit":500}`
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateAccountHandler_WithCompanyData(t *testing.T) {
	svc := &stubAccountClient{
		createFn: func(ctx context.Context, in *accountpb.CreateAccountRequest, opts ...grpc.CallOption) (*accountpb.CreateAccountResponse, error) {
			return &accountpb.CreateAccountResponse{Account: &accountpb.AccountResponse{
				Id: 1, AccountNumber: "265001",
			}}, nil
		},
	}
	card := &stubCardClient{}
	body := `{"clientId":1,"accountType":"BUSINESS","currencyCode":"RSD","companyData":{"name":"Firma d.o.o.","registrationNumber":"12345678","pib":"987654321","activityCode":"6419","address":"Beograd"}}`
	w := serveHandlerFull(CreateAccount(svc, card), "POST", "/admin/accounts", "/admin/accounts", body, makeClientToken())
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---- GetAccount / GetAccountAdmin with CompanyData ----

func TestGetAccount_WithCompanyData(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return &accountpb.GetAccountResponse{Account: &accountpb.AccountDetails{
				AccountName: "Business", CompanyData: &accountpb.CompanyData{Name: "Firma", Pib: "123"},
			}}, nil
		},
	}
	w := serveHandlerFull(GetAccount(svc), "GET", "/client/accounts/:accountId", "/client/accounts/1", "", makeClientToken())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "company")
}

func TestGetAccountAdmin_WithCompanyData(t *testing.T) {
	svc := &stubAccountClient{
		getAccountFn: func(ctx context.Context, in *accountpb.GetAccountRequest, opts ...grpc.CallOption) (*accountpb.GetAccountResponse, error) {
			return &accountpb.GetAccountResponse{Account: &accountpb.AccountDetails{
				AccountName: "Business", CompanyData: &accountpb.CompanyData{Name: "Firma"},
			}}, nil
		},
	}
	w := serveHandler(GetAccountAdmin(svc), "GET", "/admin/accounts/:accountId", "/admin/accounts/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "company")
}
