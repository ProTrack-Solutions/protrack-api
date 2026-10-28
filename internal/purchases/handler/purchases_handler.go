package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
	queries    departmentModulesDomain.ServiceInterface
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist, queries departmentModulesDomain.ServiceInterface) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
		queries:    queries,
	}
}

// identity lê empresa e usuário a partir do token
func identity(c *gin.Context) (companyId, userId uuid.UUID, ok bool) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false
	}

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return uuid.Nil, uuid.Nil, false
	}

	userId, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, false
	}

	return companyIdAny.(uuid.UUID), userId, true
}

func pathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return uuid.Nil, false
	}
	return id, true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound), errors.Is(err, domain.ErrReceiptNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrOrderNotEditable), errors.Is(err, domain.ErrOrderStatus),
		errors.Is(err, domain.ErrReceiptAlreadyReversed), errors.Is(err, domain.ErrReceiptHasPaidBills),
		errors.Is(err, domain.ErrInsufficientStock), errors.Is(err, domain.ErrVariationsExist):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidPurchase), errors.Is(err, domain.ErrVendorNotFound),
		errors.Is(err, domain.ErrVendorInactive), errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrQuantityExceeded), errors.Is(err, domain.ErrInstallmentsMismatch),
		errors.Is(err, domain.ErrCategoryNotFound), errors.Is(err, domain.ErrProductLimitReached),
		errors.Is(err, domain.ErrVendorWithoutEmail):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrEmailUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		log.Error().Err(err).Str("path", c.FullPath()).Msg("erro em compras")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno ao processar a compra"})
	}
}

// ListOrders godoc
// @Summary      Lista pedidos de compra
// @Description  Pedidos com totais e quanto já foi recebido, mais os indicadores da tela (a receber, recebido no mês, atrasados e contagem por status)
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.ListOrdersParams false "Filtros e paginação"
// @Success      200 {object} domain.ListOrdersResponse
// @Router       /purchases/orders [get]
func (h *Handler) ListOrders(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}

	var params domain.ListOrdersParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ListOrders(c.Request.Context(), companyId, params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetOrder godoc
// @Summary      Detalhe do pedido de compra
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Success      200 {object} domain.OrderDetail
// @Failure      404 {object} map[string]string
// @Router       /purchases/orders/{id} [get]
func (h *Handler) GetOrder(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	detail, err := h.service.GetOrder(c.Request.Context(), companyId, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// CreateOrder godoc
// @Summary      Cria pedido de compra
// @Description  Cria em rascunho; com send=true já fica como enviado ao fornecedor
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.SaveOrderRequest true "Pedido"
// @Success      201 {object} domain.OrderDetail
// @Failure      422 {object} map[string]string
// @Router       /purchases/orders [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}

	var req domain.SaveOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.CreateOrder(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, detail)
}

// UpdateOrder godoc
// @Summary      Altera pedido de compra
// @Description  Só em rascunho ou enviado, antes de qualquer recebimento. Os itens são substituídos pelos enviados.
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Param        body body domain.SaveOrderRequest true "Pedido"
// @Success      200 {object} domain.OrderDetail
// @Failure      409 {object} map[string]string
// @Router       /purchases/orders/{id} [put]
func (h *Handler) UpdateOrder(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req domain.SaveOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.UpdateOrder(c.Request.Context(), companyId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// SendOrder godoc
// @Summary      Marca o pedido como enviado ao fornecedor
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Success      200 {object} domain.OrderDetail
// @Router       /purchases/orders/{id}/send [post]
func (h *Handler) SendOrder(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	detail, err := h.service.SendOrder(c.Request.Context(), companyId, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// CloseOrder godoc
// @Summary      Encerra pedido recebido em parte
// @Description  Fecha o pedido com o que foi recebido; a quantidade pendente é descartada
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Param        body body domain.CloseOrderRequest false "Motivo"
// @Success      200 {object} domain.OrderDetail
// @Router       /purchases/orders/{id}/close [post]
func (h *Handler) CloseOrder(c *gin.Context) {
	h.finish(c, h.service.CloseOrder)
}

// CancelOrder godoc
// @Summary      Cancela pedido sem recebimentos
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Param        body body domain.CloseOrderRequest false "Motivo"
// @Success      200 {object} domain.OrderDetail
// @Router       /purchases/orders/{id}/cancel [post]
func (h *Handler) CancelOrder(c *gin.Context) {
	h.finish(c, h.service.CancelOrder)
}

type finishFunc func(ctx context.Context, companyId, userId, orderId uuid.UUID, req domain.CloseOrderRequest) (domain.OrderDetail, error)

func (h *Handler) finish(c *gin.Context, fn finishFunc) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req domain.CloseOrderRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	detail, err := fn(c.Request.Context(), companyId, userId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// Receive godoc
// @Summary      Recebe mercadoria do pedido
// @Description  Dá entrada no estoque, recalcula o custo médio com frete/despesas/desconto rateados, atualiza preço de venda quando informado e gera as contas a pagar das parcelas
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Param        body body domain.ReceiveRequest true "Recebimento"
// @Success      201 {object} domain.OrderDetail
// @Failure      409 {object} map[string]string
// @Failure      422 {object} map[string]string
// @Router       /purchases/orders/{id}/receipts [post]
func (h *Handler) Receive(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req domain.ReceiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.Receive(c.Request.Context(), companyId, userId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, detail)
}

// ReverseReceipt godoc
// @Summary      Estorna um recebimento
// @Description  Devolve o estoque, recalcula o custo médio e cancela as parcelas em aberto. Bloqueado se houver parcela paga ou se o estoque atual não cobrir a devolução.
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do recebimento"
// @Param        body body domain.ReverseReceiptRequest true "Motivo do estorno"
// @Success      200 {object} domain.OrderDetail
// @Failure      409 {object} map[string]string
// @Router       /purchases/receipts/{id}/reverse [post]
func (h *Handler) ReverseReceipt(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req domain.ReverseReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.ReverseReceipt(c.Request.Context(), companyId, userId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// SearchCatalog godoc
// @Summary      Busca produtos para o pedido de compra
// @Description  Produtos da empresa e, quando há texto, os modelos de grade que casam com ele (atalho de recompra). Não exige o módulo Estoque.
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.CatalogSearchParams false "Busca e paginação"
// @Success      200 {object} domain.CatalogSearchResponse
// @Router       /purchases/catalog/products [get]
func (h *Handler) SearchCatalog(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	var params domain.CatalogSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.SearchProducts(c.Request.Context(), companyId, params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ModelVariations godoc
// @Summary      Variações de um modelo de grade
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        name query string true "Nome do modelo"
// @Success      200 {array} domain.CatalogProduct
// @Router       /purchases/catalog/models [get]
func (h *Handler) ModelVariations(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "informe o nome do modelo"})
		return
	}
	variations, err := h.service.ModelVariations(c.Request.Context(), companyId, name)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, variations)
}

// ListCategories godoc
// @Summary      Categorias de produto ativas (cadastro rápido no pedido)
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.CatalogCategory
// @Router       /purchases/catalog/categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	categories, err := h.service.ListCategories(c.Request.Context(), companyId)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

// CreateVariations godoc
// @Summary      Cadastro rápido de produto com grade
// @Description  Cria um produto por combinação de cor e tamanho, sem estoque e com código de barras interno. Tudo ou nada: combinação já existente ou limite do plano recusam o lote inteiro.
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.CreateVariationsRequest true "Modelo e variações"
// @Success      201 {array} domain.CatalogProduct
// @Failure      409 {object} map[string]string "Variação já cadastrada"
// @Failure      422 {object} map[string]string "Limite do plano ou categoria inválida"
// @Router       /purchases/catalog/products [post]
func (h *Handler) CreateVariations(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var req domain.CreateVariationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.service.CreateVariations(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

// EmailOrder godoc
// @Summary      Envia o pedido de compra por e-mail ao fornecedor
// @Description  Corpo com os itens e o PDF anexo; resposta vai para o e-mail da empresa com cópia para quem enviou. kind: sent (pedido enviado), updated (pedido alterado) ou resend (reenvio). A entrega é assíncrona.
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do pedido"
// @Param        body body domain.EmailOrderRequest true "Tipo do e-mail"
// @Success      202 {object} domain.OrderDetail
// @Failure      409 {object} map[string]string "Pedido não está enviado"
// @Failure      422 {object} map[string]string "Fornecedor sem e-mail"
// @Failure      503 {object} map[string]string "Fila de e-mail indisponível"
// @Router       /purchases/orders/{id}/email [post]
func (h *Handler) EmailOrder(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req domain.EmailOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	detail, err := h.service.EmailOrder(c.Request.Context(), companyId, userId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, detail)
}
