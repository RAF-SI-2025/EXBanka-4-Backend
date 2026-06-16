package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"github.com/gin-gonic/gin"
)

// ListAuditLogs handles GET /audit-logs (ADMIN, SUPERVISOR only)
func ListAuditLogs(employeeClient pb_emp.EmployeeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
		actorID, _ := strconv.ParseInt(c.Query("actor_id"), 10, 64)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := employeeClient.ListAuditLogs(ctx, &pb_emp.ListAuditLogsRequest{
			Action:   c.Query("action"),
			ActorId:  actorID,
			FromDate: c.Query("from_date"),
			ToDate:   c.Query("to_date"),
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"entries": resp.Entries})
	}
}
