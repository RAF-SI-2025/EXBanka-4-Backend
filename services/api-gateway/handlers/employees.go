package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	authpb "github.com/exbanka/backend/shared/pb/auth"
	emailpb "github.com/exbanka/backend/shared/pb/email"
	pb "github.com/exbanka/backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type employeeResponse struct {
	Id            int64    `json:"id"`
	Ime           string   `json:"ime"`
	Prezime       string   `json:"prezime"`
	DatumRodjenja string   `json:"datum_rodjenja"`
	Pol           string   `json:"pol"`
	Email         string   `json:"email"`
	BrojTelefona  string   `json:"broj_telefona"`
	Adresa        string   `json:"adresa"`
	Username      string   `json:"username"`
	Pozicija      string   `json:"pozicija"`
	Departman     string   `json:"departman"`
	Aktivan       bool     `json:"aktivan"`
	Dozvole       []string `json:"dozvole"`
}

func toEmployeeResponse(e *pb.Employee) employeeResponse {
	dozvole := e.Dozvole
	if dozvole == nil {
		dozvole = []string{}
	}
	return employeeResponse{
		Id:            e.Id,
		Ime:           e.Ime,
		Prezime:       e.Prezime,
		DatumRodjenja: e.DatumRodjenja,
		Pol:           e.Pol,
		Email:         e.Email,
		BrojTelefona:  e.BrojTelefona,
		Adresa:        e.Adresa,
		Username:      e.Username,
		Pozicija:      e.Pozicija,
		Departman:     e.Departman,
		Aktivan:       e.Aktivan,
		Dozvole:       dozvole,
	}
}

func GetEmployeeById(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GetEmployeeById(ctx, &pb.GetEmployeeByIdRequest{Id: id})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, toEmployeeResponse(resp.Employee))
	}
}

func UpdateEmployee(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			FirstName   string   `json:"first_name"    binding:"required"`
			LastName    string   `json:"last_name"     binding:"required"`
			DateOfBirth string   `json:"date_of_birth" binding:"required"`
			Gender      string   `json:"gender"        binding:"required"`
			Email       string   `json:"email"         binding:"required"`
			PhoneNumber string   `json:"phone_number"  binding:"required"`
			Address     string   `json:"address"       binding:"required"`
			Username    string   `json:"username"      binding:"required"`
			Position    string   `json:"position"      binding:"required"`
			Department  string   `json:"department"    binding:"required"`
			Active      bool     `json:"active"`
			Permissions []string `json:"permissions"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Permissions == nil {
			req.Permissions = []string{}
		}
		if _, err := mail.ParseAddress(req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.UpdateEmployee(ctx, &pb.UpdateEmployeeRequest{
			Id:            id,
			Ime:           req.FirstName,
			Prezime:       req.LastName,
			DatumRodjenja: req.DateOfBirth,
			Pol:           req.Gender,
			Email:         req.Email,
			BrojTelefona:  req.PhoneNumber,
			Adresa:        req.Address,
			Username:      req.Username,
			Pozicija:      req.Position,
			Departman:     req.Department,
			Aktivan:       req.Active,
			Dozvole:       req.Permissions,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
			case codes.AlreadyExists:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, toEmployeeResponse(resp.Employee))
	}
}

func SearchEmployees(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.SearchEmployees(ctx, &pb.SearchEmployeesRequest{
			Email:    c.Query("email"),
			Ime:      c.Query("ime"),
			Prezime:  c.Query("prezime"),
			Pozicija: c.Query("pozicija"),
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]employeeResponse, len(resp.Employees))
		for i, e := range resp.Employees {
			result[i] = toEmployeeResponse(e)
		}
		c.JSON(http.StatusOK, gin.H{"employees": result, "total_count": resp.TotalCount})
	}
}

func GetEmployees(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GetAllEmployees(ctx, &pb.GetAllEmployeesRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]employeeResponse, len(resp.Employees))
		for i, e := range resp.Employees {
			result[i] = toEmployeeResponse(e)
		}
		c.JSON(http.StatusOK, gin.H{"employees": result, "total_count": resp.TotalCount})
	}
}

func CreateEmployee(empClient pb.EmployeeServiceClient, authClient authpb.AuthServiceClient, emailClient emailpb.EmailServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			FirstName   string `json:"first_name"    binding:"required"`
			LastName    string `json:"last_name"     binding:"required"`
			DateOfBirth string `json:"date_of_birth" binding:"required"`
			Gender      string `json:"gender"        binding:"required"`
			Email       string `json:"email"         binding:"required"`
			PhoneNumber string `json:"phone_number"  binding:"required"`
			Address     string `json:"address"       binding:"required"`
			Username    string `json:"username"      binding:"required"`
			Position    string `json:"position"      binding:"required"`
			Department  string `json:"department"    binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := mail.ParseAddress(req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := empClient.CreateEmployee(ctx, &pb.CreateEmployeeRequest{
			Ime:           req.FirstName,
			Prezime:       req.LastName,
			DatumRodjenja: req.DateOfBirth,
			Pol:           req.Gender,
			Email:         req.Email,
			BrojTelefona:  req.PhoneNumber,
			Adresa:        req.Address,
			Username:      req.Username,
			Pozicija:      req.Position,
			Departman:     req.Department,
		})
		if err != nil {
			if status.Code(err) == codes.AlreadyExists {
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tokenCtx, tokenCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer tokenCancel()
		tokenResp, err := authClient.CreateActivationToken(tokenCtx,
			&authpb.CreateActivationTokenRequest{EmployeeId: resp.Employee.Id})
		if err != nil {
			log.Printf("failed to create activation token for employee %d: %v", resp.Employee.Id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "employee created but activation setup failed"})
			return
		}

		link := fmt.Sprintf("http://localhost:8081/auth/activate?token=%s", tokenResp.Token)
		go func() {
			_, err := emailClient.SendActivationEmail(context.Background(),
				&emailpb.SendActivationEmailRequest{
					Email:          req.Email,
					FirstName:      req.FirstName,
					ActivationLink: link,
				})
			if err != nil {
				log.Printf("failed to send activation email to %s: %v", req.Email, err)
			}
		}()

		c.JSON(http.StatusCreated, toEmployeeResponse(resp.Employee))
	}
}
