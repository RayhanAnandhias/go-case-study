package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"go-case-study/internal/order/domain"
	"go-case-study/pkg/auth"
)

type OrderHandler struct {
	service   domain.OrderService
	validate  *validator.Validate
	jwtSecret string
}

func NewOrderHandler(service domain.OrderService, jwtSecret string) *OrderHandler {
	return &OrderHandler{
		service:   service,
		validate:  validator.New(),
		jwtSecret: jwtSecret,
	}
}

func (h *OrderHandler) GenerateToken(c *gin.Context) {
	userID := uuid.New().String()
	token, err := auth.GenerateMockJWT(userID, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"token":   token,
	})
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req domain.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Override UserID with the one extracted from JWT middleware
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user_id missing in context"})
		return
	}
	req.UserID = userIDVal.(uuid.UUID)

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	order, err := h.service.CreateOrder(c.Request.Context(), req)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "payment failed") {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "Payment failed", "details": errStr})
			return
		}
		if strings.Contains(errStr, "rate limit exceeded") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}
