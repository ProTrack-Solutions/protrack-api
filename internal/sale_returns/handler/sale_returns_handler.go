package handler

import (
	"errors"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	cashRegisterDomain "github.com/ProTrack-Solutions/protrack-api/internal/cash_register/domain"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sale_returns/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sale_returns/service"
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

// identity lê empresa, usuário e papel a partir do token
func identity(c *gin.Context) (companyId, userId uuid.UUID, role string, ok bool) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, "", false
	}

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return uuid.Nil, uuid.Nil, "", false
	}

	userId, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, "", false
	}

	roleAny, _ := c.Get("role")
	role, _ = roleAny.(string)

	return companyIdAny.(uuid.UUID), userId, role, true
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
	case errors.Is(err, domain.ErrSaleNotFound), errors.Is(err, domain.ErrReturnNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrOutOfWindow):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrSaleCanceled), errors.Is(err, domain.ErrInsufficientStock),
		errors.Is(err, cashRegisterDomain.ErrCashRegisterRequired):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidReturn), errors.Is(err, domain.ErrQuantityExceeded),
		errors.Is(err, domain.ErrOverrideReasonRequired), errors.Is(err, domain.ErrCustomerRequired),
		errors.Is(err, domain.ErrCustomerNotFound), errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		log.Error().Err(err).Str("path", c.FullPath()).Msg("erro em trocas e devoluções")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno ao processar a devolução"})
	}
}

// List godoc
// @Summary      Lista trocas e devoluções
// @Tags         returns
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.ListReturnsParams false "Filtros e paginação"
// @Success      200 {object} domain.ListReturnsResponse
// @Router       /returns [get]
func (h *Handler) List(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}

	var params domain.ListReturnsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.List(c.Request.Context(), companyId, params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Create godoc
// @Summary      Registra uma devolução ou troca
// @Description  Devolve o estoque, abate parcelas em aberto, gera vale-troca ou reembolso e, na troca, cria a venda dos itens novos
// @Tags         returns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.CreateReturnRequest true "Devolução"
// @Success      201 {object} domain.ReturnDetail
// @Router       /returns [post]
func (h *Handler) Create(c *gin.Context) {
	companyId, userId, role, ok := identity(c)
	if !ok {
		return
	}

	var req domain.CreateReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.Create(c.Request.Context(), companyId, userId, role, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, detail)
}

// Get godoc
// @Summary      Detalhe da troca/devolução
// @Tags         returns
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da devolução"
// @Success      200 {object} domain.ReturnDetail
// @Router       /returns/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	detail, err := h.service.Get(c.Request.Context(), companyId, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// SearchSales godoc
// @Summary      Busca vendas para trocar ou devolver
// @Tags         returns
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.SearchSalesParams false "Busca (cliente, produto ou início do código da venda)"
// @Success      200 {object} domain.SaleSearchResponse
// @Router       /returns/sales [get]
func (h *Handler) SearchSales(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}

	var params domain.SearchSalesParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.SearchSales(c.Request.Context(), companyId, params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ReturnableSale godoc
// @Summary      Itens da venda que ainda podem ser devolvidos
// @Tags         returns
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da venda"
// @Success      200 {object} domain.ReturnableSale
// @Router       /returns/sales/{id} [get]
func (h *Handler) ReturnableSale(c *gin.Context) {
	companyId, _, role, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	resp, err := h.service.ReturnableSale(c.Request.Context(), companyId, role, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CustomerCredit godoc
// @Summary      Saldo e extrato do vale-troca do cliente
// @Tags         returns
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      200 {object} domain.CustomerCredit
// @Router       /returns/customers/{id}/credit [get]
func (h *Handler) CustomerCredit(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}

	resp, err := h.service.CustomerCredit(c.Request.Context(), companyId, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
