package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	globaldomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
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

// GetCustomerDebtSummary godoc
// @Summary      Resumo da dívida do cliente
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Param        customerId path string true "ID do cliente"
// @Success      200 {object} domain.AccountsReceivableResponse
// @Router       /accounts-receivable [get]
func (h *Handler) GetCustomerDebtSummary(c *gin.Context) {
	customerIdStr := c.Param("customerId")

	customerId, err := uuid.Parse(customerIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.service.GetCustomerDebtSummary(c.Request.Context(), customerId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"account_receivable": account})
}

// GetPendingReceivablesByCustomer godoc
// @Summary      Contas a receber pendentes por cliente
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Param        customerId path string true "ID do cliente"
// @Success      200 {array} domain.AccountsReceivableResponse
// @Router       /accounts-receivable/customer/{customerId} [get]
func (h *Handler) GetPendingReceivablesByCustomer(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	customerIdStr := c.Param("customerId")

	customerId, err := uuid.Parse(customerIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accounts, err := h.service.GetPendingReceivablesByCustomer(c.Request.Context(), companyId, customerId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts_receivable": accounts})
}

// GetReceivablesBySale godoc
// @Summary      Contas a receber por venda
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Param        saleId path string true "ID da venda"
// @Success      200 {array} domain.AccountsReceivableResponse
// @Router       /accounts-receivable/sale/{saleId} [get]
func (h *Handler) GetReceivablesBySale(c *gin.Context) {
	saleIdStr := c.Param("saleId")

	saleId, err := uuid.Parse(saleIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accounts, err := h.service.GetReceivablesBySale(c.Request.Context(), saleId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts_receivable": accounts})
}

// ListOverdueReceivables godoc
// @Summary      Lista contas a receber em atraso
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.AccountsReceivableResponse
// @Router       /accounts-receivable/list [get]
func (h *Handler) ListOverdueReceivables(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	accounts, err := h.service.ListOverdueReceivables(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts_receivable": accounts})
}

// GetTotalOpenAmountByCompany godoc
// @Summary      Total em aberto da empresa
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /accounts-receivable/total-pending [get]
func (h *Handler) GetTotalOpenAmountByCompany(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	total, err := h.service.GetTotalOpenAmountByCompany(c.Request.Context(), companyId)
	if err != nil {
		log.Error().Err(err).Msg("Erro na rota GetTotalOpenAmountByCompany")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_open": total})
}

// GetTotalOverdueAmountByCompany godoc
// @Summary      Total em atraso da empresa
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /accounts-receivable/total-overdue [get]
func (h *Handler) GetTotalOverdueAmountByCompany(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	total, err := h.service.GetTotalOverdueAmountByCompany(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_overdue": total})
}

// GetTotalPendingAndOverdue godoc
// @Summary      Total pendente e em atraso da empresa
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetTotalPendingAndOverdueResponse
// @Router       /accounts-receivable/total-pending-overdue [get]
func (h *Handler) GetTotalPendingAndOverdue(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var response domain.GetTotalPendingAndOverdueResponse

	totalOverdue, err := h.service.GetTotalOverdueAmountByCompany(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPending, err := h.service.GetTotalOpenAmountByCompany(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.TotalOverdue = totalOverdue

	response.TotalPending = totalPending

	c.JSON(http.StatusOK, gin.H{"totals": response})
}

// ListAccountsReceivables godoc
// @Summary      Lista as contas a receber
// @Description  Retorna a lista de contas a receber da empresa com base nos parâmetros de paginação enviados no cabeçalho.
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Param        Pagination-Headers header globaldomain.PaginationParams false "Parâmetros de paginação enviados no Header"
// @Success      200 {object} domain.ListAccountsReceivablesResponse
// @Failure      400 {object} map[string]interface{} "Falha na validação dos parâmetros de paginação"
// @Failure      401 {object} map[string]interface{} "company_id não encontrado na sessão"
// @Failure      500 {object} map[string]interface{} "Erro interno do servidor"
// @Router       /accounts-receivable/complete/list [get]
func (h *Handler) ListAccountsReceivables(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globaldomain.PaginationParams

	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	response, err := h.service.ListAccountsReceivables(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
