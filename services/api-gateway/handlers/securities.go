package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type exchangeJSON struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Acronym  string `json:"acronym"`
	MICCode  string `json:"micCode"`
	Polity   string `json:"polity"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

func toExchangeJSON(e *pb.StockExchange) exchangeJSON {
	return exchangeJSON{
		ID:       e.Id,
		Name:     e.Name,
		Acronym:  e.Acronym,
		MICCode:  e.MicCode,
		Polity:   e.Polity,
		Currency: e.Currency,
		Timezone: e.Timezone,
	}
}

// GetStockExchanges godoc
// @Summary      List all stock exchanges
// @Tags         securities
// @Produce      json
// @Success      200  {array}   exchangeJSON
// @Router       /stock-exchanges [get]
func GetStockExchanges(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetStockExchanges(ctx, &pb.GetStockExchangesRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stock exchanges"})
			return
		}

		result := make([]exchangeJSON, 0, len(resp.Exchanges))
		for _, e := range resp.Exchanges {
			result = append(result, toExchangeJSON(e))
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetStockExchangeByMIC godoc
// @Summary      Get stock exchange by MIC code
// @Tags         securities
// @Produce      json
// @Param        mic  path  string  true  "MIC code"
// @Success      200  {object}  exchangeJSON
// @Router       /stock-exchanges/{mic} [get]
func GetStockExchangeByMIC(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetStockExchangeByMIC(ctx, &pb.GetStockExchangeByMICRequest{MicCode: mic})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch exchange"})
			return
		}
		c.JSON(http.StatusOK, toExchangeJSON(resp.Exchange))
	}
}

// CreateStockExchange godoc
// @Summary      Create a new stock exchange
// @Tags         securities
// @Accept       json
// @Produce      json
// @Success      201  {object}  exchangeJSON
// @Router       /stock-exchanges [post]
func CreateStockExchange(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name     string `json:"name"     binding:"required"`
			Acronym  string `json:"acronym"  binding:"required"`
			MICCode  string `json:"micCode"  binding:"required"`
			Polity   string `json:"polity"   binding:"required"`
			Currency string `json:"currency" binding:"required"`
			Timezone string `json:"timezone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.CreateStockExchange(ctx, &pb.CreateStockExchangeRequest{
			Name:     body.Name,
			Acronym:  body.Acronym,
			MicCode:  body.MICCode,
			Polity:   body.Polity,
			Currency: body.Currency,
			Timezone: body.Timezone,
		})
		if err != nil {
			if status.Code(err) == codes.AlreadyExists {
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exchange"})
			return
		}
		c.JSON(http.StatusCreated, toExchangeJSON(resp.Exchange))
	}
}

// UpdateStockExchange godoc
// @Summary      Update a stock exchange by MIC code
// @Tags         securities
// @Accept       json
// @Produce      json
// @Param        mic  path  string  true  "MIC code"
// @Success      200  {object}  exchangeJSON
// @Router       /stock-exchanges/{mic} [put]
func UpdateStockExchange(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		var body struct {
			Name     string `json:"name"     binding:"required"`
			Acronym  string `json:"acronym"  binding:"required"`
			Polity   string `json:"polity"   binding:"required"`
			Currency string `json:"currency" binding:"required"`
			Timezone string `json:"timezone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.UpdateStockExchange(ctx, &pb.UpdateStockExchangeRequest{
			MicCode:  mic,
			Name:     body.Name,
			Acronym:  body.Acronym,
			Polity:   body.Polity,
			Currency: body.Currency,
			Timezone: body.Timezone,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update exchange"})
			return
		}
		c.JSON(http.StatusOK, toExchangeJSON(resp.Exchange))
	}
}

// DeleteStockExchange godoc
// @Summary      Delete a stock exchange by MIC code
// @Tags         securities
// @Param        mic  path  string  true  "MIC code"
// @Success      204
// @Router       /stock-exchanges/{mic} [delete]
func DeleteStockExchange(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err := client.DeleteStockExchange(ctx, &pb.DeleteStockExchangeRequest{MicCode: mic})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete exchange"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ── Working Hours ─────────────────────────────────────────────────────────────

type workingHoursJSON struct {
	ID        int64  `json:"id"`
	Polity    string `json:"polity"`
	Segment   string `json:"segment"`
	OpenTime  string `json:"openTime"`
	CloseTime string `json:"closeTime"`
}

func toWorkingHoursJSON(h *pb.ExchangeWorkingHours) workingHoursJSON {
	return workingHoursJSON{
		ID: h.Id, Polity: h.Polity, Segment: h.Segment,
		OpenTime: h.OpenTime, CloseTime: h.CloseTime,
	}
}

// GetWorkingHours godoc
// @Summary      Get working hours for an exchange
// @Tags         securities
// @Param        mic  path  string  true  "MIC code"
// @Success      200  {array}  workingHoursJSON
// @Router       /stock-exchanges/{mic}/hours [get]
func GetWorkingHours(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetWorkingHours(ctx, &pb.GetWorkingHoursRequest{MicCode: mic})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch working hours"})
			return
		}
		result := make([]workingHoursJSON, 0, len(resp.Hours))
		for _, h := range resp.Hours {
			result = append(result, toWorkingHoursJSON(h))
		}
		c.JSON(http.StatusOK, result)
	}
}

// SetWorkingHours godoc
// @Summary      Set (upsert) working hours for a polity
// @Tags         securities
// @Accept       json
// @Produce      json
// @Success      200  {object}  workingHoursJSON
// @Router       /stock-exchanges/hours [post]
func SetWorkingHours(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Polity    string `json:"polity"     binding:"required"`
			Segment   string `json:"segment"    binding:"required"`
			OpenTime  string `json:"openTime"   binding:"required"`
			CloseTime string `json:"closeTime"  binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.SetWorkingHours(ctx, &pb.SetWorkingHoursRequest{
			Polity:    body.Polity,
			Segment:   body.Segment,
			OpenTime:  body.OpenTime,
			CloseTime: body.CloseTime,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set working hours"})
			return
		}
		c.JSON(http.StatusOK, toWorkingHoursJSON(resp.Hours))
	}
}

// ── Holidays ──────────────────────────────────────────────────────────────────

type holidayJSON struct {
	ID          int64  `json:"id"`
	Polity      string `json:"polity"`
	HolidayDate string `json:"holidayDate"`
	Description string `json:"description"`
}

func toHolidayJSON(h *pb.ExchangeHoliday) holidayJSON {
	return holidayJSON{
		ID: h.Id, Polity: h.Polity, HolidayDate: h.HolidayDate, Description: h.Description,
	}
}

// GetHolidays godoc
// @Summary      Get holidays for an exchange's polity
// @Tags         securities
// @Param        mic  path  string  true  "MIC code"
// @Success      200  {array}  holidayJSON
// @Router       /stock-exchanges/{mic}/holidays [get]
func GetHolidays(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// Resolve polity via GetStockExchangeByMIC first, then GetHolidays by polity
		exchResp, err := client.GetStockExchangeByMIC(ctx, &pb.GetStockExchangeByMICRequest{MicCode: mic})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch exchange"})
			return
		}

		resp, err := client.GetHolidays(ctx, &pb.GetHolidaysRequest{Polity: exchResp.Exchange.Polity})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch holidays"})
			return
		}
		result := make([]holidayJSON, 0, len(resp.Holidays))
		for _, h := range resp.Holidays {
			result = append(result, toHolidayJSON(h))
		}
		c.JSON(http.StatusOK, result)
	}
}

// AddHoliday godoc
// @Summary      Add a holiday for a polity
// @Tags         securities
// @Accept       json
// @Produce      json
// @Success      201  {object}  holidayJSON
// @Router       /stock-exchanges/holidays [post]
func AddHoliday(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Polity      string `json:"polity"       binding:"required"`
			HolidayDate string `json:"holidayDate"  binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.AddHoliday(ctx, &pb.AddHolidayRequest{
			Polity:      body.Polity,
			HolidayDate: body.HolidayDate,
			Description: body.Description,
		})
		if err != nil {
			if status.Code(err) == codes.AlreadyExists {
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add holiday"})
			return
		}
		c.JSON(http.StatusCreated, toHolidayJSON(resp.Holiday))
	}
}

// DeleteHoliday godoc
// @Summary      Delete a holiday for a polity
// @Tags         securities
// @Param        polity  path  string  true  "Polity (country)"
// @Param        date    path  string  true  "Holiday date (YYYY-MM-DD)"
// @Success      204
// @Router       /stock-exchanges/holidays/{polity}/{date} [delete]
func DeleteHoliday(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		polity := c.Param("polity")
		date := c.Param("date")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err := client.DeleteHoliday(ctx, &pb.DeleteHolidayRequest{Polity: polity, HolidayDate: date})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete holiday"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ── Exchange Status ───────────────────────────────────────────────────────────

// GetTestMode godoc
// @Summary      Get test mode status
// @Tags         securities
// @Produce      json
// @Success      200  {object}  map[string]bool
// @Router       /stock-exchanges/test-mode [get]
func GetTestMode(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetTestMode(ctx, &pb.GetTestModeRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test mode"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": resp.Enabled})
	}
}

// SetTestMode godoc
// @Summary      Enable or disable test mode
// @Tags         securities
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]bool
// @Router       /stock-exchanges/test-mode [post]
func SetTestMode(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.SetTestMode(ctx, &pb.SetTestModeRequest{Enabled: body.Enabled})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set test mode"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": resp.Enabled})
	}
}

// IsExchangeOpen godoc
// @Summary      Check if an exchange is currently open
// @Tags         securities
// @Param        mic  path  string  true  "MIC code"
// @Success      200  {object}  map[string]interface{}
// @Router       /stock-exchanges/{mic}/is-open [get]
func IsExchangeOpen(client pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		mic := c.Param("mic")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.IsExchangeOpen(ctx, &pb.IsExchangeOpenRequest{MicCode: mic})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check exchange status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"micCode":          resp.MicCode,
			"isOpen":           resp.IsOpen,
			"segment":          resp.Segment,
			"currentTimeLocal": resp.CurrentTimeLocal,
		})
	}
}
