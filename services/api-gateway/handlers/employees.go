package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	pb "github.com/exbanka/backend/shared/pb/employee"
)

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
		c.JSON(http.StatusOK, resp.Employees)
	}
}

func GetEmployees(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := client.GetAllEmployees(context.Background(), &pb.GetAllEmployeesRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp.Employees)
	}
}
