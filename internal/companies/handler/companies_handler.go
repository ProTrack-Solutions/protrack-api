package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
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

// CreateCompany godoc
// @Summary      Cria uma empresa
// @Tags         companies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        company body domain.CreateCompanyParams true "Empresa"
// @Success      201 {object} domain.CompanyResponse
// @Router       /companies [post]
func (h *Handler) CreateCompany(c *gin.Context) {
	idStr := c.GetString("sub")
	if idStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.CreateCompanyParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CreatedBy = id

	company, err := h.service.CreateCompany(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"company": company})
}

// DeleteCompany godoc
// @Summary      Remove uma empresa
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da empresa"
// @Success      204
// @Router       /companies/{id} [delete]
func (h *Handler) DeleteCompany(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteCompany(c.Request.Context(), domain.DeleteCompanyParams{
		ID:        id,
		DeletedBy: uuid.Nil,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetCompanyByDocument godoc
// @Summary      Busca empresa por documento
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        document path string true "Documento (CNPJ/CPF)"
// @Success      200 {object} domain.CompanyResponse
// @Router       /companies/document/{document} [get]
func (h *Handler) GetCompanyByDocument(c *gin.Context) {
	document := c.Param("document")

	company, err := h.service.GetCompanyByDocument(c.Request.Context(), document)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"company": company})
}

// GetCompanyByID godoc
// @Summary      Busca empresa por ID
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da empresa"
// @Success      200 {object} domain.CompanyResponse
// @Router       /companies/{id} [get]
func (h *Handler) GetCompanyByID(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	company, err := h.service.GetCompanyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"company": company})
}

// ListCompanies godoc
// @Summary      Lista todas as empresas
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.CompanyResponse
// @Router       /companies [get]
func (h *Handler) ListCompanies(c *gin.Context) {
	companies, err := h.service.ListCompanies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"companies": companies})
}

// SetCompanyStatus godoc
// @Summary      Altera o status de uma empresa
// @Tags         companies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        status body domain.SetCompanyStatusParams true "Status"
// @Success      200 {object} map[string]int64
// @Router       /companies/set/ [post]
func (h *Handler) SetCompanyStatus(c *gin.Context) {
	var req domain.SetCompanyStatusParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.service.SetCompanyStatus(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// UpdateCompany godoc
// @Summary      Atualiza uma empresa
// @Tags         companies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da empresa"
// @Param        company body domain.UpdateCompanyRequest true "Empresa"
// @Success      200 {object} domain.CompanyResponse
// @Router       /companies/{id} [put]
func (h *Handler) UpdateCompany(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateCompanyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	company, err := h.service.UpdateCompany(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"company": company})
}
