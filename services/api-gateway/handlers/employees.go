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
	authpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	emailpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// employeeResponse is the JSON representation of an employee.
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
	Jmbg          string   `json:"jmbg"`
}

// CreateEmployeeRequest is the request body for creating an employee.
type CreateEmployeeRequest struct {
	FirstName   string `json:"first_name"    binding:"required" example:"Marko"`
	LastName    string `json:"last_name"     binding:"required" example:"Marković"`
	DateOfBirth string `json:"date_of_birth" binding:"required" example:"1990-01-15"`
	Gender      string `json:"gender"        binding:"required" example:"M"`
	Email       string `json:"email"         binding:"required" example:"marko@exbanka.rs"`
	PhoneNumber string `json:"phone_number"  binding:"required" example:"+381641234567"`
	Address     string `json:"address"       binding:"required" example:"Bulevar Kralja Aleksandra 73"`
	Username    string `json:"username"      binding:"required" example:"mmarkovic"`
	Position    string `json:"position"      binding:"required" example:"Teller"`
	Department  string `json:"department"    binding:"required" example:"Retail"`
	Jmbg        string `json:"jmbg"          binding:"required" example:"0101990710006"`
}

// UpdateEmployeeRequest is the request body for updating an employee.
type UpdateEmployeeRequest struct {
	FirstName   string   `json:"first_name"    binding:"required" example:"Marko"`
	LastName    string   `json:"last_name"     binding:"required" example:"Marković"`
	DateOfBirth string   `json:"date_of_birth" binding:"required" example:"1990-01-15"`
	Gender      string   `json:"gender"        binding:"required" example:"M"`
	Email       string   `json:"email"         binding:"required" example:"marko@exbanka.rs"`
	PhoneNumber string   `json:"phone_number"  binding:"required" example:"+381641234567"`
	Address     string   `json:"address"       binding:"required" example:"Bulevar Kralja Aleksandra 73"`
	Username    string   `json:"username"      binding:"required" example:"mmarkovic"`
	Position    string   `json:"position"      binding:"required" example:"Teller"`
	Department  string   `json:"department"    binding:"required" example:"Retail"`
	Active      bool     `json:"active"        example:"true"`
	Permissions []string `json:"permissions"   example:"LOANS"`
	Jmbg        string   `json:"jmbg"          example:"0101990710006"`
}

// EmployeeListResponse wraps a paginated list of employees.
type EmployeeListResponse struct {
	Employees  []employeeResponse `json:"employees"`
	TotalCount int32              `json:"total_count"`
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
		Jmbg:          e.Jmbg,
	}
}

// GetEmployeeById godoc
// @Summary      Get employee by ID
// @Description  Retrieve a single employee by their numeric ID.
// @Tags         employees
// @Produce      json
// @Param        id   path      int  true  "Employee ID"
// @Success      200  {object}  employeeResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /employees/{id} [get]
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

// UpdateEmployee godoc
// @Summary      Update employee
// @Description  Update all fields of an existing employee.
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        id    path      int                   true  "Employee ID"
// @Param        body  body      UpdateEmployeeRequest true  "Updated employee data"
// @Success      200   {object}  employeeResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      422   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /employees/{id} [put]
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
			DateOfBirth string   `json:"date_of_birth"`
			Gender      string   `json:"gender"`
			Email       string   `json:"email"         binding:"required"`
			PhoneNumber string   `json:"phone_number"`
			Address     string   `json:"address"`
			Username    string   `json:"username"      binding:"required"`
			Position    string   `json:"position"`
			Department  string   `json:"department"`
			Active      bool     `json:"active"`
			Permissions []string `json:"permissions"`
			Jmbg        string   `json:"jmbg"`
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
			Jmbg:          req.Jmbg,
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

// SearchEmployees godoc
// @Summary      Search employees
// @Description  Search employees by email, first name, last name, or position with pagination.
// @Tags         employees
// @Produce      json
// @Param        email     query     string  false  "Filter by email"
// @Param        ime       query     string  false  "Filter by first name"
// @Param        prezime   query     string  false  "Filter by last name"
// @Param        pozicija  query     string  false  "Filter by position"
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 20)"
// @Success      200       {object}  EmployeeListResponse
// @Failure      500       {object}  map[string]string
// @Security     BearerAuth
// @Router       /employees/search [get]
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

// GetEmployees godoc
// @Summary      List all employees
// @Description  Retrieve a paginated list of all employees.
// @Tags         employees
// @Produce      json
// @Param        page      query     int  false  "Page number (default 1)"
// @Param        page_size query     int  false  "Page size (default 20)"
// @Success      200       {object}  EmployeeListResponse
// @Failure      500       {object}  map[string]string
// @Security     BearerAuth
// @Router       /employees [get]
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

// CreateEmployee godoc
// @Summary      Create employee
// @Description  Create a new inactive employee and send an activation email.
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        body  body      CreateEmployeeRequest  true  "New employee data"
// @Success      201   {object}  employeeResponse
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /employees [post]
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
			Jmbg        string `json:"jmbg"          binding:"required"`
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
			Jmbg:          req.Jmbg,
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

		link := fmt.Sprintf("http://localhost:5173/set-password?token=%s", tokenResp.Token)
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
