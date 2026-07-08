package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/service"
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

// CreateBillCategories godoc
// @Summary      Cria uma categoria de conta
// @Tags         bill-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        category body domain.CreateBillCategoriesRequest true "Categoria"
// @Success      201
// @Router       /bill-categories [post]
func (h *Handler) CreateBillCategories(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreateBillCategoriesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	if err := h.service.CreateBillCategories(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// DeleteBillCategories godoc
// @Summary      Remove uma categoria de conta
// @Tags         bill-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Success      204
// @Router       /bill-categories/{id} [delete]
func (h *Handler) DeleteBillCategories(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteBillCategories(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetBillCategoriesById godoc
// @Summary      Busca categoria de conta por ID
// @Tags         bill-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Success      200 {object} domain.BillCategoryResponse
// @Router       /bill-categories/{id} [get]
func (h *Handler) GetBillCategoriesById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	billCategory, err := h.service.GetBillCategoriesById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bill_categories": billCategory})
}

// ListBillCategories godoc
// @Summary      Lista categorias de conta da empresa
// @Tags         bill-categories
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.BillCategoryResponse
// @Router       /bill-categories [get]
func (h *Handler) ListBillCategories(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	billCategories, err := h.service.ListBillCategories(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bill_categories": billCategories})
}

// ListBillCategoriesActive godoc
// @Summary      Lista categorias de conta ativas
// @Tags         bill-categories
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.BillCategoryResponse
// @Router       /bill-categories/active [get]
func (h *Handler) ListBillCategoriesActive(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	billCategories, err := h.service.ListBillCategoriesActive(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bill_categories": billCategories})
}

// ToggleBillCategoriesActive godoc
// @Summary      Ativa ou desativa uma categoria de conta
// @Tags         bill-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Param        toggle body domain.ToggleBillCategoriesActiveRequest true "Status"
// @Success      200
// @Router       /bill-categories/toggle/{id} [put]
func (h *Handler) ToggleBillCategoriesActive(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.ToggleBillCategoriesActiveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = id

	if err := h.service.ToggleBillCategoriesActive(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
