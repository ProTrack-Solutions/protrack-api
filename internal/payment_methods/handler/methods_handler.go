package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/payment_methods/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/payment_methods/service"
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

func (h *Handler) CreatePaymentMethod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreatePaymentMethodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	if err := h.service.CreatePaymentMethod(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) GetPaymentMethodById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	method, err := h.service.GetPaymentMethodById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_method": method})
}

func (h *Handler) ListPaymentMethod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	method, err := h.service.ListPaymentMethod(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": method})
}

func (h *Handler) ListPaymentMethodIsActive(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	method, err := h.service.ListPaymentMethodIsActive(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": method})
}

func (h *Handler) TogglePaymentMethodActive(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.TogglePaymentMethodActiveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.TogglePaymentMethodActive(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GetPaymentMethodsStats(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	row, err := h.service.GetPaymentMethodsStats(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	c.JSON(http.StatusOK, gin.H{"payment_method_stats": row})
}
