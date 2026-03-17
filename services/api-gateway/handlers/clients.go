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
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type clientResponse struct {
	Id          int64  `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Jmbg        string `json:"jmbg"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
	Username    string `json:"username"`
	Active      bool   `json:"active"`
}

type createClientRequest struct {
	FirstName   string `json:"first_name"    binding:"required"`
	LastName    string `json:"last_name"     binding:"required"`
	Jmbg        string `json:"jmbg"          binding:"required"`
	DateOfBirth string `json:"date_of_birth" binding:"required"`
	Gender      string `json:"gender"        binding:"required"`
	Email       string `json:"email"         binding:"required"`
	PhoneNumber string `json:"phone_number"  binding:"required"`
	Address     string `json:"address"       binding:"required"`
	Username    string `json:"username"      binding:"required"`
}

type updateClientRequest struct {
	FirstName   string `json:"first_name"    binding:"required"`
	LastName    string `json:"last_name"     binding:"required"`
	Jmbg        string `json:"jmbg"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	Email       string `json:"email"         binding:"required"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
	Username    string `json:"username"      binding:"required"`
	Active      bool   `json:"active"`
}

func toClientResponse(c *pb.Client) clientResponse {
	return clientResponse{
		Id:          c.Id,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		Jmbg:        c.Jmbg,
		DateOfBirth: c.DateOfBirth,
		Gender:      c.Gender,
		Email:       c.Email,
		PhoneNumber: c.PhoneNumber,
		Address:     c.Address,
		Username:    c.Username,
		Active:      c.Active,
	}
}

// GetClients godoc
// @Summary      List all clients
// @Description  Returns a paginated list of clients. Requires employee authentication.
// @Tags         clients
// @Produce      json
// @Param        page      query  int  false  "Page number"
// @Param        page_size query  int  false  "Page size"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients [get]
func GetClients(client pb.ClientServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GetAllClients(ctx, &pb.GetAllClientsRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]clientResponse, len(resp.Clients))
		for i, cl := range resp.Clients {
			result[i] = toClientResponse(cl)
		}
		c.JSON(http.StatusOK, gin.H{"clients": result, "total_count": resp.TotalCount})
	}
}

// GetClientById godoc
// @Summary      Get client by ID
// @Description  Returns a single client by ID. Requires employee authentication.
// @Tags         clients
// @Produce      json
// @Param        id   path  int  true  "Client ID"
// @Success      200  {object}  clientResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients/{id} [get]
func GetClientById(client pb.ClientServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GetClientById(ctx, &pb.GetClientByIdRequest{Id: id})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, toClientResponse(resp.Client))
	}
}

// CreateClient godoc
// @Summary      Create a client
// @Description  Creates a new client and sends an activation email. Requires employee authentication.
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        body  body      createClientRequest  true  "Client data"
// @Success      201   {object}  clientResponse
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients [post]
func CreateClient(clientSvc pb.ClientServiceClient, authClient authpb.AuthServiceClient, emailClient emailpb.EmailServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createClientRequest
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
		resp, err := clientSvc.CreateClient(ctx, &pb.CreateClientRequest{
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Jmbg:        req.Jmbg,
			DateOfBirth: req.DateOfBirth,
			Gender:      req.Gender,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
			Address:     req.Address,
			Username:    req.Username,
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
		tokenResp, err := authClient.CreateClientActivationToken(tokenCtx,
			&authpb.CreateClientActivationTokenRequest{ClientId: resp.Client.Id})
		if err != nil {
			log.Printf("failed to create activation token for client %d: %v", resp.Client.Id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "client created but activation setup failed"})
			return
		}

		link := fmt.Sprintf("http://localhost:5173/client/activate?token=%s", tokenResp.Token)
		go func() {
			_, err := emailClient.SendActivationEmail(context.Background(),
				&emailpb.SendActivationEmailRequest{
					Email:          req.Email,
					FirstName:      req.FirstName,
					ActivationLink: link,
				})
			if err != nil {
				log.Printf("failed to send activation email to client %s: %v", req.Email, err)
			}
		}()

		c.JSON(http.StatusCreated, toClientResponse(resp.Client))
	}
}

// UpdateClient godoc
// @Summary      Update a client
// @Description  Updates an existing client. Requires employee authentication.
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        id    path  int                  true  "Client ID"
// @Param        body  body  updateClientRequest  true  "Client data"
// @Success      200   {object}  clientResponse
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients/{id} [put]
func UpdateClient(client pb.ClientServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req updateClientRequest
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
		resp, err := client.UpdateClient(ctx, &pb.UpdateClientRequest{
			Id:          id,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Jmbg:        req.Jmbg,
			DateOfBirth: req.DateOfBirth,
			Gender:      req.Gender,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
			Address:     req.Address,
			Username:    req.Username,
			Active:      req.Active,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			case codes.AlreadyExists:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, toClientResponse(resp.Client))
	}
}
