package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
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

func SearchEmployees(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := client.SearchEmployees(context.Background(), &pb.SearchEmployeesRequest{
			Email:    c.Query("email"),
			Ime:      c.Query("ime"),
			Prezime:  c.Query("prezime"),
			Pozicija: c.Query("pozicija"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]employeeResponse, len(resp.Employees))
		for i, e := range resp.Employees {
			result[i] = toEmployeeResponse(e)
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetEmployees(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := client.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]employeeResponse, len(resp.Employees))
		for i, e := range resp.Employees {
			result[i] = toEmployeeResponse(e)
		}
		c.JSON(http.StatusOK, result)
	}
}

func CreateEmployee(client pb.EmployeeServiceClient) gin.HandlerFunc {
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

		resp, err := client.CreateEmployee(context.Background(), &pb.CreateEmployeeRequest{
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
				c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, toEmployeeResponse(resp.Employee))
	}
}
