package handler

import (
	"net/http"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/accounts_receivable/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/accounts_receivable/service"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/cache"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/auth/adapters/jwt"
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
