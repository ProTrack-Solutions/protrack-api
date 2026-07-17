package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/service"
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

// CashFlowSummary godoc
// @Summary      Resumo do fluxo de caixa
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Param        start_at query string true "Data inicial"
// @Param        end_at query string true "Data final"
// @Success      200 {object} domain.CashFlowSummaryResponse
// @Router       /cash-flow/summary [get]
func (h *Handler) CashFlowSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CashFlowSummaryRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.service.CashFlowSummary(c.Request.Context(), companyId, req.StartAt, req.EndAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cash_flow_summary": summary})
}

// GetCashFlowHistoryProjections godoc
// @Summary      Histórico e projeções do fluxo de caixa
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetCashFlowHistoryProjectionsResponse
// @Router       /cash-flow/history-projection [get]
func (h *Handler) GetCashFlowHistoryProjections(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	cashFlow, err := h.service.GetCashFlowHistoryProjections(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cash_flow_history": cashFlow})
}

// GetCashInFlowByCategory godoc
// @Summary      Entradas de caixa por categoria
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.GetCashInFlowByCategoryResponse
// @Router       /cash-flow/inflow-category [get]
func (h *Handler) GetCashInFlowByCategory(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	cashFlowCategories, err := h.service.GetCashInFlowByCategory(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cash_inflow_categories": cashFlowCategories})
}

// GetCashOutFlowByCategory godoc
// @Summary      Saídas de caixa por categoria
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.GetCashOutFlowByCategoryResponse
// @Router       /cash-flow/outflow-category [get]
func (h *Handler) GetCashOutFlowByCategory(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	cashFlowCategories, err := h.service.GetCashOutFlowByCategory(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cash_outflow_categories": cashFlowCategories})
}

// GetCashFlowPeriod godoc
// @Summary      Fluxo de caixa do período mensal
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetCashFlowPeriodResponse
// @Router       /cash-flow/summary-month [get]
func (h *Handler) GetCashFlowPeriod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	cashFlowMonth, err := h.service.GetCashFlowPeriod(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cash_flow_month": cashFlowMonth})
}

// GetCashFlow godoc
// @Summary      Fluxo de caixa
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetCashFlowResponse
// @Router       /cash-flow [get]
func (h *Handler) GetCashFlow(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	cashFlow, err := h.service.GetCashFlow(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cashFlow)
}

// GetTotalSummary godoc
// @Summary      Resumo total
// @Tags         cash-flow
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.GetTotalSummaryParams true "Parâmetros de busca"
// @Success      200 {object} []domain.GetTotalSummaryResponse
// @Failure      400 {object} map[string]string "Erro de validação"
// @Failure      401 {object} map[string]string "Não autorizado"
// @Failure      500 {object} map[string]string "Erro interno"
// @Router       /cash-flow/total-summary [get]
func (h *Handler) GetTotalSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var params domain.GetTotalSummaryParams

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	totalSummary, err := h.service.GetTotalSummary(c.Request.Context(), params, companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, totalSummary)
}
