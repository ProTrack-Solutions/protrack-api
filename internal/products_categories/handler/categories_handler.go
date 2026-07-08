package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/products_categories/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products_categories/service"
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

// CreateProductCategory godoc
// @Summary      Cria uma categoria de produto
// @Tags         products-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        category body domain.CreateProductCategoryRequest true "Categoria"
// @Success      201 {object} map[string]string
// @Router       /products-categories [post]
func (h *Handler) CreateProductCategory(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.CreateProductCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.CreateProductCategory(c.Request.Context(), userId, companyId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"category": "success"})
}

// DeleteProductCategory godoc
// @Summary      Remove uma categoria de produto
// @Tags         products-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Success      204
// @Router       /products-categories/{id} [delete]
func (h *Handler) DeleteProductCategory(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteProductCategory(c.Request.Context(), domain.DeleteProductCategoryRequest{
		ID:        id,
		DeletedBy: uuid.Nil,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProductCategoryById godoc
// @Summary      Busca categoria de produto por ID
// @Tags         products-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Success      200 {object} domain.ProductCategoryResponse
// @Router       /products-categories/{id} [get]
func (h *Handler) GetProductCategoryById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.service.GetProductCategoryById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}

// ListProductCategoryByCompanyId godoc
// @Summary      Lista categorias de produto da empresa
// @Tags         products-categories
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.ProductCategoryResponse
// @Router       /products-categories/list/company [get]
func (h *Handler) ListProductCategoryByCompanyId(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	categories, err := h.service.ListProductCategoryByCompanyId(c.Request.Context(), companyId)
	if err != nil {
		log.Err(err).Msg("Esse é o erro")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// SetProductCategoryStatus godoc
// @Summary      Altera o status de uma categoria de produto
// @Tags         products-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        status body domain.SetProductCategoryStatusRequest true "Status"
// @Success      200 {object} map[string]int64
// @Router       /products-categories/status/ [put]
func (h *Handler) SetProductCategoryStatus(c *gin.Context) {
	var req domain.SetProductCategoryStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.service.SetProductCategoryStatus(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// UpdateProductCategory godoc
// @Summary      Atualiza uma categoria de produto
// @Tags         products-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Param        category body domain.UpdateProductCategoryRequest true "Categoria"
// @Success      200 {object} domain.ProductCategoryResponse
// @Router       /products-categories/{id} [put]
func (h *Handler) UpdateProductCategory(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateProductCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.service.UpdateProductCategory(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}
