package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

// CreateCustomer godoc
// @Summary      Cria um cliente
// @Tags         customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        customer body domain.CreateCustomersRequest true "Cliente"
// @Success      201 {object} map[string]string
// @Router       /customers [post]
func (h *Handler) CreateCustomer(c *gin.Context) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
		return
	}

	userIdStr := userIdAny.(string)

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreateCustomersRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("erro no JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId
	req.CreatedBy = userID

	id, err := h.service.CreateCustomer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer_id": id})
}

// DeleteCustomer godoc
// @Summary      Remove um cliente
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      204
// @Router       /customers/{id} [delete]
func (h *Handler) DeleteCustomer(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.DeleteCustomerRequest

	req.ID = id

	req.DeletedBy = companyId

	if err := h.service.DeleteCustomer(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.Status(http.StatusNoContent)
}

// GetCustomerByCPF godoc
// @Summary      Busca cliente por CPF
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        cpf path string true "CPF"
// @Success      200 {object} domain.CustomerResponse
// @Router       /customers/cpf/{cpf} [get]
func (h *Handler) GetCustomerByCPF(c *gin.Context) {
	cpf := c.Param("cpf")

	customer, err := h.service.GetCustomerByCPF(c.Request.Context(), cpf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

// GetCustomerById godoc
// @Summary      Busca cliente por ID
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      200 {object} domain.CustomerResponse
// @Router       /customers/{id} [get]
func (h *Handler) GetCustomerById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.GetCustomerById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

// ListCustomers godoc
// @Summary      Lista clientes da empresa com paginação
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        Page header int false "Página"
// @Param        PerPage header int false "Itens por página"
// @Success      200 {object} domain.CustomerPaginatedResponse
// @Router       /customers/list [get]
func (h *Handler) ListCustomers(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globalDomain.PaginationParams
	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customers, err := h.service.ListCustomersPaginated(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, customers)
}

// UpdateBalanceDueCustomer godoc
// @Summary      Atualiza saldo devedor do cliente
// @Tags         customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Param        balance body domain.UpdateBalanceDueCustomerRequest true "Saldo"
// @Success      200
// @Router       /customers/balanceDue/{id} [put]
func (h *Handler) UpdateBalanceDueCustomer(c *gin.Context) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
	}

	userIdStr := userIdAny.(string)

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	}

	var req domain.UpdateBalanceDueCustomerRequest

	req.UpdatedBy = userID

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateBalanceDueCustomer(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UpdateCustomer godoc
// @Summary      Atualiza um cliente
// @Tags         customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Param        customer body domain.UpdateCustomerRequest true "Cliente"
// @Success      200
// @Router       /customers/{id} [put]
func (h *Handler) UpdateCustomer(c *gin.Context) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
	}

	userIdStr := userIdAny.(string)

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	}

	var req domain.UpdateCustomerRequest

	req.UpdatedBy = userID

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateCustomer(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// CountCustomers godoc
// @Summary      Conta clientes da empresa
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]int64
// @Router       /customers/count [get]
func (h *Handler) CountCustomers(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	count, err := h.service.CountCustomers(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetCustomersPerformanceSummary godoc
// @Summary      Resumo de performance dos clientes
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]float64
// @Router       /customers/percentage [get]
func (h *Handler) GetCustomersPerformanceSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	percentage, err := h.service.GetCustomersPerformanceSummary(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"percentage": percentage})
}
