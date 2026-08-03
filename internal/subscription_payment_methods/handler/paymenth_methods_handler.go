package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

func (h *Handler) CreateSubscriptionPaymentMethodHandler(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	companyIdStr, ok := companyIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "company_id is not a string"})
		return
	}

	companyId, err := uuid.Parse(companyIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid company_id format"})
		return
	}

	userIdAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIdStr, ok := userIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id is not a string"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
	}

	var req domain.CreateSubscriptionPaymentMethodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateSubscriptionPaymentMethod(c.Request.Context(), companyId, userId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Payment method created successfully"})
}

func (h *Handler) GetSubscriptionPaymentMethodByCompanyId(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	companyIdStr, ok := companyIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "company_id is not a string"})
		return
	}

	companyId, err := uuid.Parse(companyIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid company_id format"})
		return
	}

	paymentMethods, err := h.service.GetSubscriptionPaymentMethodByCompanyId(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, paymentMethods)
}

func (h *Handler) GetSubscriptionPaymentMethodById(c *gin.Context) {
	paymentMethodIdStr := c.Param("id")
	paymentMethodId, err := uuid.Parse(paymentMethodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method ID format"})
		return
	}

	paymentMethod, err := h.service.GetSubscriptionPaymentMethodById(c.Request.Context(), paymentMethodId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

func (h *Handler) UpdateSubscriptionPaymentMethodHandler(c *gin.Context) {
	paymentMethodIdStr := c.Param("id")
	paymentMethodId, err := uuid.Parse(paymentMethodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method ID format"})
		return
	}

	userIdAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIdStr, ok := userIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id is not a string"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id format"})
		return
	}

	var req domain.UpdateSubscriptionPaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateSubscriptionPaymentMethod(c.Request.Context(), paymentMethodId, userId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment method updated successfully"})
}

func (h *Handler) SetDefaultSubscriptionPaymentMethod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	companyIdStr, ok := companyIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "company_id is not a string"})
		return
	}

	companyId, err := uuid.Parse(companyIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid company_id format"})
		return
	}

	userIdAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIdStr, ok := userIdAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id is not a string"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
	}

	paymentMethodIdStr := c.Param("id")
	paymentMethodId, err := uuid.Parse(paymentMethodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method ID format"})
		return
	}

	if err := h.service.SetDefaultSubscriptionPaymentMethod(c.Request.Context(), companyId, userId, paymentMethodId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) DeleteSubscriptionPaymentMethod(c *gin.Context) {
	paymentMethodIdStr := c.Param("id")
	paymentMethodId, err := uuid.Parse(paymentMethodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method ID format"})
		return
	}

	if err := h.service.DeleteSubscriptionPaymentMethod(c.Request.Context(), paymentMethodId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
