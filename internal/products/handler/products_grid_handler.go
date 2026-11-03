package handler

import (
	"errors"
	"net/http"

	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// respondGridError mapeia os erros da grade; o resto segue respondProductError
func respondGridError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidGrid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrModelNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrModelAlreadyExists),
		errors.Is(err, domain.ErrVariationsExist),
		errors.Is(err, domain.ErrVariationHasStock),
		errors.Is(err, domain.ErrBarcodeAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrProductLimitReached):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		respondProductError(c, err)
	}
}

// gridAuth lê empresa e usuário do token
func gridAuth(c *gin.Context) (companyId, userId uuid.UUID, ok bool) {
	companyAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return uuid.Nil, uuid.Nil, false
	}
	userId, err := uuid.Parse(c.GetString("sub"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return uuid.Nil, uuid.Nil, false
	}
	return companyAny.(uuid.UUID), userId, true
}

func modelName(c *gin.Context) (string, bool) {
	var q domain.ModelQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "informe o modelo em ?name="})
		return "", false
	}
	return q.Name, true
}

// ListStockEntries godoc
// @Summary      Estoque agrupado por modelo
// @Description  Cada modelo de grade vira uma linha com os totais das variações; produtos sem grade continuam como linha própria.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        Pagination-Query query globalDomain.PaginationParams false "Paginação"
// @Success      200 {object} domain.StockEntriesResponse
// @Router       /product/stock [get]
func (h *Handler) ListStockEntries(c *gin.Context) {
	companyId, _, ok := gridAuth(c)
	if !ok {
		return
	}
	var pagination globalDomain.PaginationParams
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.ListStockEntries(c.Request.Context(), companyId, pagination)
	if err != nil {
		respondGridError(c, err)
		return
	}
	hideCostInStock(c, &resp)
	c.JSON(http.StatusOK, resp)
}

// ListModelVariations godoc
// @Summary      Variações de um modelo
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        name query string true "Nome do modelo"
// @Success      200 {array} domain.GridVariation
// @Router       /product/models [get]
func (h *Handler) ListModelVariations(c *gin.Context) {
	companyId, _, ok := gridAuth(c)
	if !ok {
		return
	}
	name, ok := modelName(c)
	if !ok {
		return
	}
	out, err := h.service.ListModelVariations(c.Request.Context(), companyId, name)
	if err != nil {
		respondGridError(c, err)
		return
	}
	hideCostInVariations(c, out)
	c.JSON(http.StatusOK, gin.H{"variations": out})
}

// CreateModel godoc
// @Summary      Cadastra um modelo com grade de cores e tamanhos
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        model body domain.CreateModelRequest true "Modelo"
// @Success      201 {array} domain.GridVariation
// @Router       /product/models [post]
func (h *Handler) CreateModel(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	var req domain.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.CreateModel(c.Request.Context(), userId, companyId, req)
	if err != nil {
		respondGridError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"variations": out})
}

// UpdateModel godoc
// @Summary      Edita o modelo (nome, categoria, descrição e preços das variações)
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        name query string true "Nome atual do modelo"
// @Param        model body domain.UpdateModelRequest true "Modelo"
// @Success      200 {array} domain.GridVariation
// @Router       /product/models [put]
func (h *Handler) UpdateModel(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	name, ok := modelName(c)
	if !ok {
		return
	}
	var req domain.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.UpdateModel(c.Request.Context(), userId, companyId, name, req)
	if err != nil {
		respondGridError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"variations": out})
}

// AddVariations godoc
// @Summary      Adiciona cores/tamanhos a um modelo
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        name query string true "Nome do modelo"
// @Param        variations body domain.AddVariationsRequest true "Variações"
// @Success      201 {array} domain.GridVariation
// @Router       /product/models/variations [post]
func (h *Handler) AddVariations(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	name, ok := modelName(c)
	if !ok {
		return
	}
	var req domain.AddVariationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.AddVariations(c.Request.Context(), userId, companyId, name, req)
	if err != nil {
		respondGridError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"variations": out})
}

// RemoveVariation godoc
// @Summary      Desativa uma variação sem saldo
// @Tags         products
// @Security     BearerAuth
// @Param        id path string true "ID da variação"
// @Success      204
// @Router       /product/models/variations/{id} [delete]
func (h *Handler) RemoveVariation(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.RemoveVariation(c.Request.Context(), userId, companyId, id); err != nil {
		respondGridError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GridStockMovement godoc
// @Summary      Entrada ou ajuste de estoque pela grade
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        name query string true "Nome do modelo"
// @Param        movement body domain.GridStockMovementRequest true "Movimentação"
// @Success      201 {array} domain.StockMovementResponse
// @Router       /product/models/stock-movements [post]
func (h *Handler) GridStockMovement(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	name, ok := modelName(c)
	if !ok {
		return
	}
	var req domain.GridStockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.GridStockMovement(c.Request.Context(), userId, companyId, name, req)
	if err != nil {
		respondGridError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"stock_movements": out})
}

// GroupProducts godoc
// @Summary      Agrupa produtos soltos num modelo de grade
// @Description  Os produtos continuam os mesmos (saldo, histórico e vendas preservados); ganham modelo, cor e tamanho.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        group body domain.GroupProductsRequest true "Agrupamento"
// @Success      200 {array} domain.GridVariation
// @Router       /product/models/group [post]
func (h *Handler) GroupProducts(c *gin.Context) {
	companyId, userId, ok := gridAuth(c)
	if !ok {
		return
	}
	var req domain.GroupProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.GroupProducts(c.Request.Context(), userId, companyId, req)
	if err != nil {
		respondGridError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"variations": out})
}
