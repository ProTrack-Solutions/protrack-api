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

// CreatePaymentMethod godoc
// @Summary      Cria um método de pagamento
// @Tags         payment-methods
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        method body domain.CreatePaymentMethodRequest true "Método de pagamento"
// @Success      201
// @Router       /payment-methods [post]
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

// GetPaymentMethodById godoc
// @Summary      Busca método de pagamento por ID
// @Tags         payment-methods
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do método"
// @Success      200 {object} domain.PaymentMethodResponse
// @Router       /payment-methods/{id} [get]
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

// ListPaymentMethod godoc
// @Summary      Lista métodos de pagamento da empresa
// @Tags         payment-methods
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.PaymentMethodResponse
// @Router       /payment-methods [get]
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

// ListPaymentMethodIsActive godoc
// @Summary      Lista métodos de pagamento ativos
// @Tags         payment-methods
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.PaymentMethodResponse
// @Router       /payment-methods/is-active [get]
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

// TogglePaymentMethodActive godoc
// @Summary      Ativa ou desativa um método de pagamento
// @Tags         payment-methods
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do método"
// @Param        toggle body domain.TogglePaymentMethodActiveRequest true "Status"
// @Success      200
// @Router       /payment-methods/{id} [put]
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

// GetPaymentMethodsStats godoc
// @Summary      Estatísticas dos métodos de pagamento
// @Tags         payment-methods
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetPaymentMethodsStatsResponse
// @Router       /payment-methods/stats [get]
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

	c.JSON(http.StatusOK, row)
}
