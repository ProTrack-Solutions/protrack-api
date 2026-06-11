package handler

import (
	"net/http"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/cache"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/auth/adapters/jwt"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/cash_flow/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/cash_flow/service"
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
