package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type actuaryResponse struct {
	EmployeeId   int64   `json:"employee_id"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	Email        string  `json:"email"`
	Position     string  `json:"position"`
	LimitAmount  float64 `json:"limit"`
	UsedLimit    float64 `json:"used_limit"`
	NeedApproval bool    `json:"need_approval"`
}

func toActuaryResponse(a *pb.ActuaryInfo) actuaryResponse {
	return actuaryResponse{
		EmployeeId:   a.EmployeeId,
		FirstName:    a.FirstName,
		LastName:     a.LastName,
		Email:        a.Email,
		Position:     a.Position,
		LimitAmount:  a.LimitAmount,
		UsedLimit:    a.UsedLimit,
		NeedApproval: a.NeedApproval,
	}
}

// GetActuaries godoc
// @Summary      List all agents with their limit info
// @Description  Returns a filtered list of agents (employees with AGENT permission) and their actuary info.
// @Tags         actuaries
// @Produce      json
// @Param        email       query     string  false  "Filter by email"
// @Param        first_name  query     string  false  "Filter by first name"
// @Param        last_name   query     string  false  "Filter by last name"
// @Param        position    query     string  false  "Filter by position"
// @Success      200  {array}   actuaryResponse
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/actuaries [get]
func GetActuaries(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetActuaries(ctx, &pb.GetActuariesRequest{
			Email:     c.Query("email"),
			FirstName: c.Query("first_name"),
			LastName:  c.Query("last_name"),
			Position:  c.Query("position"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]actuaryResponse, len(resp.Actuaries))
		for i, a := range resp.Actuaries {
			result[i] = toActuaryResponse(a)
		}
		c.JSON(http.StatusOK, result)
	}
}

// SetAgentLimit godoc
// @Summary      Set agent spending limit
// @Description  Updates the daily spending limit for an agent. Supervisor only.
// @Tags         actuaries
// @Accept       json
// @Produce      json
// @Param        id    path      int   true  "Employee ID"
// @Param        body  body      object true  "Limit value"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/actuaries/{id}/limit [put]
func SetAgentLimit(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			Limit float64 `json:"limit" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Limit < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be non-negative"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		actorID, _ := middleware.GetUserIDFromToken(c)
		_, err = client.SetAgentLimit(ctx, &pb.SetAgentLimitRequest{
			EmployeeId:  id,
			LimitAmount: req.Limit,
			ActorId:     actorID,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "limit updated"})
	}
}

// ResetAgentUsedLimit godoc
// @Summary      Reset agent used limit to 0
// @Description  Sets used_limit = 0 for the given agent. Supervisor only.
// @Tags         actuaries
// @Produce      json
// @Param        id  path  int  true  "Employee ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/actuaries/{id}/reset-used-limit [post]
func ResetAgentUsedLimit(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		actorID, _ := middleware.GetUserIDFromToken(c)
		_, err = client.ResetAgentUsedLimit(ctx, &pb.ResetAgentUsedLimitRequest{EmployeeId: id, ActorId: actorID})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "used_limit reset"})
	}
}

// SetNeedApproval godoc
// @Summary      Set agent need_approval flag
// @Description  Updates the need_approval flag for an agent. Supervisor only.
// @Tags         actuaries
// @Accept       json
// @Produce      json
// @Param        id    path      int     true  "Employee ID"
// @Param        body  body      object  true  "need_approval value"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/actuaries/{id}/need-approval [put]
func SetNeedApproval(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			NeedApproval bool `json:"need_approval"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = client.SetNeedApproval(ctx, &pb.SetNeedApprovalRequest{
			EmployeeId:   id,
			NeedApproval: req.NeedApproval,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "need_approval updated"})
	}
}

// GetActuaryPerformances godoc
// @Summary      Bank profit portal — actuary P&L
// @Description  Returns realized profit in RSD for every actuary (AGENT, SUPERVISOR, ADMIN). SUPERVISOR only.
// @Tags         bank-profit
// @Produce      json
// @Success      200  {array}   object
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /bank/profit/actuaries [get]
func GetActuaryPerformances(empClient pb.EmployeeServiceClient, ordClient pb_order.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// 1. All employees eligible for actuary performance view (AGENT + SUPERVISOR + ADMIN)
		perfResp, err := empClient.GetActuaryPerformers(ctx, &pb.GetActuaryPerformersRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 2. Collect user IDs
		userIDs := make([]int64, 0, len(perfResp.Performers))
		for _, p := range perfResp.Performers {
			userIDs = append(userIDs, p.UserId)
		}

		// 3. Realized profit per user from order-service
		profitResp, err := ordClient.GetActuaryProfits(ctx, &pb_order.GetActuaryProfitsRequest{UserIds: userIDs})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		profitMap := make(map[int64]float64, len(profitResp.Profits))
		for _, ap := range profitResp.Profits {
			profitMap[ap.UserId] = ap.ProfitRsd
		}

		// 4. Join and return; default profit = 0 for employees with no completed orders
		result := make([]gin.H, 0, len(perfResp.Performers))
		for _, p := range perfResp.Performers {
			result = append(result, gin.H{
				"userId":    p.UserId,
				"firstName": p.FirstName,
				"lastName":  p.LastName,
				"position":  p.Position,
				"profit":    profitMap[p.UserId],
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetSupervisors godoc
// @Summary      List active supervisors for fund manager selection
// @Tags         actuaries
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Security     BearerAuth
// @Router       /api/supervisors [get]
func GetSupervisors(client pb.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetSupervisors(ctx, &pb.GetSupervisorsRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]gin.H, len(resp.Supervisors))
		for i, s := range resp.Supervisors {
			result[i] = gin.H{
				"employee_id": s.EmployeeId,
				"first_name":  s.FirstName,
				"last_name":   s.LastName,
				"email":       s.Email,
				"position":    s.Position,
			}
		}
		c.JSON(http.StatusOK, result)
	}
}
