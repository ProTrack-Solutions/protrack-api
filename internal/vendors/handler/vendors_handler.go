package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/vendors/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/vendors/service"
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

// CreateVendors godoc
// @Summary      Cria um fornecedor
// @Tags         vendors
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        vendor body domain.CreateVendorsRequest true "Fornecedor"
// @Success      201
// @Router       /vendors [post]
func (h *Handler) CreateVendors(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreateVendorsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}

	req.CompanyID = companyId

	if err := h.service.CreateVendors(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// GetVendorsById godoc
// @Summary      Busca fornecedor por ID
// @Tags         vendors
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do fornecedor"
// @Success      200 {object} domain.VendorResponse
// @Router       /vendors/{id} [get]
func (h *Handler) GetVendorsById(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.GetVendorsByIdRequest

	req.CompanyID = companyId
	req.ID = id

	vendor, err := h.service.GetVendorsById(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendor": vendor})
}

// ListVendors godoc
// @Summary      Lista fornecedores da empresa
// @Tags         vendors
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.VendorResponse
// @Router       /vendors/list [get]
func (h *Handler) ListVendors(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	vendors, err := h.service.ListVendors(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendors": vendors})
}

// ListVendorsIsActive godoc
// @Summary      Lista fornecedores ativos
// @Tags         vendors
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.VendorResponse
// @Router       /vendors/list/is-active [get]
func (h *Handler) ListVendorsIsActive(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	vendors, err := h.service.ListVendorsIsActive(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendors_active": vendors})
}

// ToggleVendorsActive godoc
// @Summary      Ativa ou desativa um fornecedor
// @Tags         vendors
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do fornecedor"
// @Param        toggle body domain.ToggleVendorsActiveParams true "Status"
// @Success      200
// @Router       /vendors/toggle/{id} [put]
func (h *Handler) ToggleVendorsActive(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.ToggleVendorsActiveParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ToggleVendorsActive(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UpdateVendors godoc
// @Summary      Atualiza um fornecedor
// @Tags         vendors
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do fornecedor"
// @Param        vendor body domain.UpdateVendorsRequest true "Fornecedor"
// @Success      200
// @Router       /vendors/{id} [put]
func (h *Handler) UpdateVendors(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.UpdateVendorsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var reqById domain.GetVendorsByIdRequest

	reqById.ID = id
	reqById.CompanyID = companyId

	if err := h.service.UpdateVendors(c.Request.Context(), reqById, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
