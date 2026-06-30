package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/service"
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

// CreateProduct godoc
// @Summary      Cria um produto
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product body domain.CreateProductRequest true "Produto"
// @Success      201 {object} domain.ProductResponse
// @Router       /product [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdStr := c.GetString("sub")
	if userIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.CreateProductRequest

	req.CompanyID = companyId

	req.CreatedBy = userId

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.service.CreateProduct(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"product": product})
}

// DeleteProduct godoc
// @Summary      Remove um produto
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do produto"
// @Success      204
// @Router       /product/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteProduct(c.Request.Context(), domain.DeleteProductRequest{
		ID:        id,
		DeletedBy: uuid.Nil,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProductByBarcode godoc
// @Summary      Busca produto por código de barras
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        barcode path string true "Código de barras"
// @Success      200 {object} domain.ProductResponse
// @Router       /product/barcode/{barcode} [get]
func (h *Handler) GetProductByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")

	product, err := h.service.GetProductByBarcode(c.Request.Context(), barcode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// GetProductById godoc
// @Summary      Busca produto por ID
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do produto"
// @Success      200 {object} domain.ProductResponse
// @Router       /product/{id} [get]
func (h *Handler) GetProductById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.service.GetProductById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// ListProductsByCategoryId godoc
// @Summary      Lista produtos por categoria
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        categoryId path string true "ID da categoria"
// @Success      200 {array} domain.ProductResponse
// @Router       /product/category/{categoryId} [get]
func (h *Handler) ListProductsByCategoryId(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	categoryIdStr := c.Param("categoryId")

	categoryId, err := uuid.Parse(categoryIdStr)

	products, err := h.service.ListProductsByCategoryId(c.Request.Context(), categoryId, companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

// ListProductsByCompany godoc
// @Summary      Lista produtos da empresa com paginação
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        Page header int false "Página"
// @Param        PerPage header int false "Itens por página"
// @Success      200 {object} domain.ProductPaginatedResponse
// @Router       /product/company [get]
func (h *Handler) ListProductsByCompany(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globalDomain.PaginationParams
	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	products, err := h.service.ListProductsByCompanyPaginated(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// UpdateProduct godoc
// @Summary      Atualiza um produto
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do produto"
// @Param        product body domain.UpdateProductRequest true "Produto"
// @Success      200 {object} domain.ProductResponse
// @Router       /product/{id} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.service.UpdateProduct(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// CountProducts godoc
// @Summary      Conta produtos da empresa
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]int64
// @Router       /product/count [get]
func (h *Handler) CountProducts(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	count, err := h.service.CountProducts(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetProductsPerformanceSummary godoc
// @Summary      Resumo de performance dos produtos
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]float64
// @Router       /product/percentage [get]
func (h *Handler) GetProductsPerformanceSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	percentage, err := h.service.GetProductsPerformanceSummary(c.Request.Context(), companyId)
	if err != nil {
		log.Error().Err(err).Msg("Debug para error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"percentage": percentage})
}

// GetCostTotalStock godoc
// @Summary      Custo total do estoque
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /product/cost-total [get]
func (h *Handler) GetCostTotalStock(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	total, err := h.service.GetCostTotalStock(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cost_total": total})
}

// GetTop5BestSellingProducts godoc
// @Summary      Top 5 produtos mais vendidos
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.ProductResponse
// @Router       /product/top-products [get]
func (h *Handler) GetTop5BestSellingProducts(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	products, err := h.service.GetTop5BestSellingProducts(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"top_products": products})
}
