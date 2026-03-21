package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetMyLoans godoc
// @Summary      Get all loans for authenticated client
// @Tags         loans
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Router       /loans [get]
func GetMyLoans(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetClientLoans(ctx, &pb.GetClientLoansRequest{ClientId: clientID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch loans"})
			return
		}

		type loanJSON struct {
			ID              int64   `json:"id"`
			LoanNumber      int64   `json:"loanNumber"`
			AccountNumber   string  `json:"accountNumber"`
			LoanType        string  `json:"loanType"`
			Amount          float64 `json:"amount"`
			Currency        string  `json:"currency"`
			Status          string  `json:"status"`
			RepaymentPeriod int32   `json:"repaymentPeriod"`
		}
		result := make([]loanJSON, 0, len(resp.Loans))
		for _, l := range resp.Loans {
			result = append(result, loanJSON{
				ID: l.Id, LoanNumber: l.LoanNumber, AccountNumber: l.AccountNumber,
				LoanType: l.LoanType, Amount: l.Amount, Currency: l.Currency,
				Status: l.Status, RepaymentPeriod: l.RepaymentPeriod,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetLoanDetails godoc
// @Summary      Get details + installments for a loan
// @Tags         loans
// @Produce      json
// @Param        id  path  int  true  "Loan ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /loans/{id} [get]
func GetLoanDetails(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		loanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid loan id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetLoanDetails(ctx, &pb.GetLoanDetailsRequest{LoanId: loanID})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch loan details"})
			}
			return
		}

		type installmentJSON struct {
			ID               int64   `json:"id"`
			InstallmentAmount float64 `json:"installmentAmount"`
			InterestRate     float64 `json:"interestRate"`
			Currency         string  `json:"currency"`
			ExpectedDueDate  string  `json:"expectedDueDate"`
			ActualDueDate    string  `json:"actualDueDate,omitempty"`
			Status           string  `json:"status"`
		}
		l := resp.Loan
		installments := make([]installmentJSON, 0, len(resp.Installments))
		for _, i := range resp.Installments {
			installments = append(installments, installmentJSON{
				ID: i.Id, InstallmentAmount: i.InstallmentAmount,
				InterestRate: i.InterestRate, Currency: i.Currency,
				ExpectedDueDate: i.ExpectedDueDate, ActualDueDate: i.ActualDueDate,
				Status: i.Status,
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"id": l.Id, "loanNumber": l.LoanNumber, "accountNumber": l.AccountNumber,
			"loanType": l.LoanType, "interestRateType": l.InterestRateType,
			"amount": l.Amount, "currency": l.Currency, "repaymentPeriod": l.RepaymentPeriod,
			"nominalRate": l.NominalRate, "effectiveRate": l.EffectiveRate,
			"agreedDate": l.AgreedDate, "maturityDate": l.MaturityDate,
			"nextInstallmentAmount": l.NextInstallmentAmount,
			"nextInstallmentDate": l.NextInstallmentDate,
			"remainingDebt": l.RemainingDebt, "status": l.Status,
			"installments": installments,
		})
	}
}

// GetLoanInstallments godoc
// @Summary      Get installments for a loan
// @Tags         loans
// @Produce      json
// @Param        id  path  int  true  "Loan ID"
// @Success      200  {array}  map[string]interface{}
// @Router       /loans/{id}/installments [get]
func GetLoanInstallments(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		loanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid loan id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetLoanInstallments(ctx, &pb.GetLoanInstallmentsRequest{LoanId: loanID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch installments"})
			return
		}

		type installmentJSON struct {
			ID               int64   `json:"id"`
			InstallmentAmount float64 `json:"installmentAmount"`
			InterestRate     float64 `json:"interestRate"`
			Currency         string  `json:"currency"`
			ExpectedDueDate  string  `json:"expectedDueDate"`
			ActualDueDate    string  `json:"actualDueDate,omitempty"`
			Status           string  `json:"status"`
		}
		result := make([]installmentJSON, 0, len(resp.Installments))
		for _, i := range resp.Installments {
			result = append(result, installmentJSON{
				ID: i.Id, InstallmentAmount: i.InstallmentAmount,
				InterestRate: i.InterestRate, Currency: i.Currency,
				ExpectedDueDate: i.ExpectedDueDate, ActualDueDate: i.ActualDueDate,
				Status: i.Status,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// ApplyForLoan godoc
// @Summary      Submit a loan application
// @Tags         loans
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Loan application"
// @Success      201  {object}  map[string]interface{}
// @Router       /loans/apply [post]
func ApplyForLoan(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		var req struct {
			LoanType         string  `json:"loanType"         binding:"required"`
			InterestRateType string  `json:"interestRateType" binding:"required"`
			Amount           float64 `json:"amount"           binding:"required"`
			Currency         string  `json:"currency"         binding:"required"`
			Purpose          string  `json:"purpose"`
			MonthlySalary    float64 `json:"monthlySalary"`
			EmploymentStatus string  `json:"employmentStatus"`
			EmploymentPeriod int32   `json:"employmentPeriod"`
			RepaymentPeriod  int32   `json:"repaymentPeriod"  binding:"required"`
			ContactPhone     string  `json:"contactPhone"`
			AccountNumber    string  `json:"accountNumber"    binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.SubmitLoanApplication(ctx, &pb.SubmitLoanApplicationRequest{
			ClientId:         clientID,
			LoanType:         req.LoanType,
			InterestRateType: req.InterestRateType,
			Amount:           req.Amount,
			Currency:         req.Currency,
			Purpose:          req.Purpose,
			MonthlySalary:    req.MonthlySalary,
			EmploymentStatus: req.EmploymentStatus,
			EmploymentPeriod: req.EmploymentPeriod,
			RepaymentPeriod:  req.RepaymentPeriod,
			ContactPhone:     req.ContactPhone,
			AccountNumber:    req.AccountNumber,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"loanId":             resp.LoanId,
			"loanNumber":         resp.LoanNumber,
			"monthlyInstallment": resp.MonthlyInstallment,
		})
	}
}

// GetAllLoanApplications godoc
// @Summary      Get all pending loan applications (employee)
// @Tags         loans
// @Produce      json
// @Param        loanType      query  string  false  "Filter by loan type"
// @Param        accountNumber query  string  false  "Filter by account number"
// @Success      200  {array}  map[string]interface{}
// @Router       /admin/loans/applications [get]
func GetAllLoanApplications(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetAllLoanApplications(ctx, &pb.GetAllLoanApplicationsRequest{
			LoanType:      c.Query("loanType"),
			AccountNumber: c.Query("accountNumber"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch loan applications"})
			return
		}

		c.JSON(http.StatusOK, loanDetailsToJSON(resp.Applications))
	}
}

// ApproveLoan godoc
// @Summary      Approve a pending loan application (employee)
// @Tags         loans
// @Produce      json
// @Param        id  path  int  true  "Loan ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /admin/loans/{id}/approve [put]
func ApproveLoan(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		employeeID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		loanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid loan id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = client.ApproveLoan(ctx, &pb.ApproveLoanRequest{LoanId: loanID, EmployeeId: employeeID})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve loan"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "loan approved"})
	}
}

// RejectLoan godoc
// @Summary      Reject a pending loan application (employee)
// @Tags         loans
// @Produce      json
// @Param        id  path  int  true  "Loan ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /admin/loans/{id}/reject [put]
func RejectLoan(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		employeeID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		loanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid loan id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = client.RejectLoan(ctx, &pb.RejectLoanRequest{LoanId: loanID, EmployeeId: employeeID})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject loan"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "loan rejected"})
	}
}

// GetAllLoans godoc
// @Summary      Get all loans (employee)
// @Tags         loans
// @Produce      json
// @Param        loanType      query  string  false  "Filter by loan type"
// @Param        accountNumber query  string  false  "Filter by account number"
// @Param        status        query  string  false  "Filter by status"
// @Success      200  {array}  map[string]interface{}
// @Router       /admin/loans [get]
func GetAllLoans(client pb.LoanServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetAllLoans(ctx, &pb.GetAllLoansRequest{
			LoanType:      c.Query("loanType"),
			AccountNumber: c.Query("accountNumber"),
			Status:        c.Query("status"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch loans"})
			return
		}

		c.JSON(http.StatusOK, loanDetailsToJSON(resp.Loans))
	}
}

// loanDetailsToJSON converts a slice of LoanDetail proto messages to a JSON-serialisable slice.
func loanDetailsToJSON(loans []*pb.LoanDetail) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(loans))
	for _, l := range loans {
		result = append(result, map[string]interface{}{
			"id": l.Id, "loanNumber": l.LoanNumber, "accountNumber": l.AccountNumber,
			"loanType": l.LoanType, "interestRateType": l.InterestRateType,
			"amount": l.Amount, "currency": l.Currency, "repaymentPeriod": l.RepaymentPeriod,
			"nominalRate": l.NominalRate, "effectiveRate": l.EffectiveRate,
			"agreedDate": l.AgreedDate, "maturityDate": l.MaturityDate,
			"nextInstallmentAmount": l.NextInstallmentAmount,
			"nextInstallmentDate":   l.NextInstallmentDate,
			"remainingDebt": l.RemainingDebt, "status": l.Status,
			"purpose":          l.Purpose,
			"monthlySalary":    l.MonthlySalary,
			"employmentStatus": l.EmploymentStatus,
			"employmentPeriod": l.EmploymentPeriod,
			"contactPhone":     l.ContactPhone,
		})
	}
	return result
}
