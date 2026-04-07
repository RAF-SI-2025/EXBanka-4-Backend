package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- stub securities service client ----

type stubSecuritiesClient struct {
	getExchangesFn    func(context.Context, *pb.GetStockExchangesRequest, ...grpc.CallOption) (*pb.GetStockExchangesResponse, error)
	getByMICFn        func(context.Context, *pb.GetStockExchangeByMICRequest, ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error)
	createExchangeFn  func(context.Context, *pb.CreateStockExchangeRequest, ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error)
	updateExchangeFn  func(context.Context, *pb.UpdateStockExchangeRequest, ...grpc.CallOption) (*pb.UpdateStockExchangeResponse, error)
	deleteExchangeFn  func(context.Context, *pb.DeleteStockExchangeRequest, ...grpc.CallOption) (*pb.DeleteStockExchangeResponse, error)
	getHoursFn        func(context.Context, *pb.GetWorkingHoursRequest, ...grpc.CallOption) (*pb.GetWorkingHoursResponse, error)
	setHoursFn        func(context.Context, *pb.SetWorkingHoursRequest, ...grpc.CallOption) (*pb.SetWorkingHoursResponse, error)
	getHolidaysFn     func(context.Context, *pb.GetHolidaysRequest, ...grpc.CallOption) (*pb.GetHolidaysResponse, error)
	addHolidayFn      func(context.Context, *pb.AddHolidayRequest, ...grpc.CallOption) (*pb.AddHolidayResponse, error)
	deleteHolidayFn   func(context.Context, *pb.DeleteHolidayRequest, ...grpc.CallOption) (*pb.DeleteHolidayResponse, error)
	isOpenFn          func(context.Context, *pb.IsExchangeOpenRequest, ...grpc.CallOption) (*pb.IsExchangeOpenResponse, error)
	getTestModeFn     func(context.Context, *pb.GetTestModeRequest, ...grpc.CallOption) (*pb.GetTestModeResponse, error)
	setTestModeFn     func(context.Context, *pb.SetTestModeRequest, ...grpc.CallOption) (*pb.SetTestModeResponse, error)
}

func (s *stubSecuritiesClient) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetStockExchanges(ctx context.Context, in *pb.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb.GetStockExchangesResponse, error) {
	if s.getExchangesFn != nil {
		return s.getExchangesFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetStockExchangeByMIC(ctx context.Context, in *pb.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb.GetStockExchangeByMICResponse, error) {
	if s.getByMICFn != nil {
		return s.getByMICFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) CreateStockExchange(ctx context.Context, in *pb.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb.CreateStockExchangeResponse, error) {
	if s.createExchangeFn != nil {
		return s.createExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) UpdateStockExchange(ctx context.Context, in *pb.UpdateStockExchangeRequest, opts ...grpc.CallOption) (*pb.UpdateStockExchangeResponse, error) {
	if s.updateExchangeFn != nil {
		return s.updateExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) DeleteStockExchange(ctx context.Context, in *pb.DeleteStockExchangeRequest, opts ...grpc.CallOption) (*pb.DeleteStockExchangeResponse, error) {
	if s.deleteExchangeFn != nil {
		return s.deleteExchangeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetWorkingHours(ctx context.Context, in *pb.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb.GetWorkingHoursResponse, error) {
	if s.getHoursFn != nil {
		return s.getHoursFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) SetWorkingHours(ctx context.Context, in *pb.SetWorkingHoursRequest, opts ...grpc.CallOption) (*pb.SetWorkingHoursResponse, error) {
	if s.setHoursFn != nil {
		return s.setHoursFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetHolidays(ctx context.Context, in *pb.GetHolidaysRequest, opts ...grpc.CallOption) (*pb.GetHolidaysResponse, error) {
	if s.getHolidaysFn != nil {
		return s.getHolidaysFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) AddHoliday(ctx context.Context, in *pb.AddHolidayRequest, opts ...grpc.CallOption) (*pb.AddHolidayResponse, error) {
	if s.addHolidayFn != nil {
		return s.addHolidayFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) DeleteHoliday(ctx context.Context, in *pb.DeleteHolidayRequest, opts ...grpc.CallOption) (*pb.DeleteHolidayResponse, error) {
	if s.deleteHolidayFn != nil {
		return s.deleteHolidayFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) IsExchangeOpen(ctx context.Context, in *pb.IsExchangeOpenRequest, opts ...grpc.CallOption) (*pb.IsExchangeOpenResponse, error) {
	if s.isOpenFn != nil {
		return s.isOpenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) GetTestMode(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
	if s.getTestModeFn != nil {
		return s.getTestModeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubSecuritiesClient) SetTestMode(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
	if s.setTestModeFn != nil {
		return s.setTestModeFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}

// ---- GetTestMode ----

func TestGetTestMode_Happy(t *testing.T) {
	client := &stubSecuritiesClient{
		getTestModeFn: func(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
			return &pb.GetTestModeResponse{Enabled: true}, nil
		},
	}
	w := serveHandler(GetTestMode(client), "GET", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestGetTestMode_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		getTestModeFn: func(ctx context.Context, in *pb.GetTestModeRequest, opts ...grpc.CallOption) (*pb.GetTestModeResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(GetTestMode(client), "GET", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- SetTestMode ----

func TestSetTestMode_Enable(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return &pb.SetTestModeResponse{Enabled: in.Enabled}, nil
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestSetTestMode_Disable(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return &pb.SetTestModeResponse{Enabled: in.Enabled}, nil
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":false}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":false`)
}

func TestSetTestMode_InvalidBody(t *testing.T) {
	client := &stubSecuritiesClient{}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetTestMode_Error(t *testing.T) {
	client := &stubSecuritiesClient{
		setTestModeFn: func(ctx context.Context, in *pb.SetTestModeRequest, opts ...grpc.CallOption) (*pb.SetTestModeResponse, error) {
			return nil, status.Error(codes.Internal, "db error")
		},
	}
	w := serveHandler(SetTestMode(client), "POST", "/stock-exchanges/test-mode", "/stock-exchanges/test-mode", `{"enabled":true}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
