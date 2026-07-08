package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
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

// CreateSale godoc
// @Summary      Cria uma venda
// @Tags         sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sale body domain.CreateSaleRequest true "Venda"
// @Success      201 {object} map[string]string
// @Router       /sales [post]
func (h *Handler) CreateSale(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is null"})
	}

	var req domain.CreateSaleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.service.CreateSale(c.Request.Context(), userId, companyId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// DeleteSale godoc
// @Summary      Remove uma venda
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da venda"
// @Success      204
// @Router       /sales/{id} [delete]
func (h *Handler) DeleteSale(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is null"})
	}

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.DeleteSaleRequest

	req.CompanyID = companyId

	req.DeletedBy = userId

	if err := h.service.DeleteSale(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetSaleById godoc
// @Summary      Busca venda por ID
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da venda"
// @Success      200 {object} domain.ListSalesResponse
// @Router       /sales/{id} [get]
func (h *Handler) GetSaleById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.GetSaleByIdRequest

	req.CompanyID = companyId

	req.ID = id

	sale, err := h.service.GetSaleById(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sale": sale})
}

// ListSales godoc
// @Summary      Lista vendas da empresa
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.ListSalesResponse
// @Router       /sales/list/ [get]
func (h *Handler) ListSales(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	sales, err := h.service.ListSales(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

// UpdateSaleStatus godoc
// @Summary      Atualiza o status de uma venda
// @Tags         sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da venda"
// @Param        status body domain.UpdateSaleStatusRequest true "Status"
// @Success      200
// @Router       /sales/{id}/status [put]
func (h *Handler) UpdateSaleStatus(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is null"})
	}

	var req domain.UpdateSaleStatusRequest

	req.CompanyID = companyId

	req.UpdatedBy = userId

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	if err := h.service.UpdateSaleStatus(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// ListSalesByCompanyAndStatus godoc
// @Summary      Lista vendas por empresa e status
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Status da venda"
// @Param        customer_id query string false "ID do cliente"
// @Success      200 {array} domain.ListSalesResponse
// @Router       /sales/list/company [get]
func (h *Handler) ListSalesByCompanyAndStatus(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.ListSalesByCompanyAndStatusRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	sales, err := h.service.ListSalesByCustomerAndStatus(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

// CountSales godoc
// @Summary      Conta vendas da empresa
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]int64
// @Router       /sales/count [get]
func (h *Handler) CountSales(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	count, err := h.service.CountSales(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetSalesPerformanceSummary godoc
// @Summary      Resumo de performance das vendas
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]float64
// @Router       /sales/percentage [get]
func (h *Handler) GetSalesPerformanceSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	percentage, err := h.service.GetSalesPerformanceSummary(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"percentage": percentage})
}

// GetTotalAmountSummary godoc
// @Summary      Total geral de vendas
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /sales/total-amount [get]
func (h *Handler) GetTotalAmountSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	totalAmount, err := h.service.GetTotalAmountSummary(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, totalAmount)
}

// GetTotalAmountIsPending godoc
// @Summary      Total de vendas pendentes
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /sales/total-pending [get]
func (h *Handler) GetTotalAmountIsPending(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	total, err := h.service.GetTotalAmountIsPending(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_pending": total})
}

// GetTotalAmountIsOverdue godoc
// @Summary      Total de vendas em atraso
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /sales/total-overdue [get]
func (h *Handler) GetTotalAmountIsOverdue(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.GetTotalAmountByStatusRequest

	req.CompanyID = companyId

	total, err := h.service.GetTotalAmountIsOverdue(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_overdue": total})
}

// ContSalesPendingAndOverdue godoc
// @Summary      Conta vendas pendentes e em atraso
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]int64
// @Router       /sales/count/pending-overdue [get]
func (h *Handler) ContSalesPendingAndOverdue(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	count, err := h.service.ContSalesPendingAndOverdue(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cont_sales": count})
}

// ListSalesWithDetails godoc
// @Summary      Lista vendas com detalhes paginadas
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        Page header int false "Página"
// @Param        PerPage header int false "Itens por página"
// @Success      200 {object} domain.SaleResponsePaginate
// @Router       /sales/complete [get]
func (h *Handler) ListSalesWithDetails(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globalDomain.PaginationParams
	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sales, err := h.service.ListSalesWithDetailsPaginate(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// ListSalesWithDetailsPendingOverdue godoc
// @Summary      Lista vendas pendentes e em atraso com detalhes
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.ListSalesWithInstallmentsResponse
// @Router       /sales/complete/pending-overdue [get]
func (h *Handler) ListSalesWithDetailsPendingOverdue(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	sales, err := h.service.ListSalesWithDetailsPendingOverdue(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales_completed": sales})
}

// GetRealProfitItem godoc
// @Summary      Lucro real por item
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /sales/real-profit [get]
func (h *Handler) GetRealProfitItem(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	realProfit, err := h.service.GetRealProfitItem(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"real_profit": realProfit})
}

// GetTop5RealProfitItem godoc
// @Summary      Top 5 produtos por lucro real
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.GetTop5RealProfitItemResponse
// @Router       /sales/top5-products [get]
func (h *Handler) GetTop5RealProfitItem(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	topProducts, err := h.service.GetTop5RealProfitItem(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topProducts)
}

// GetPerformanceMonth godoc
// @Summary      Performance mensal de vendas
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.GetPerformanceMonthResponse
// @Router       /sales/performance-mounts [get]
func (h *Handler) GetPerformanceMonth(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	performance, err := h.service.GetPerformanceMonth(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"performance": performance})
}

// GetTotalInvestmentCategory godoc
// @Summary      Investimento total por categoria
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.GetTotalInvestmentCategoryResponse
// @Router       /sales/investment-categories [get]
func (h *Handler) GetTotalInvestmentCategory(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	investmentCategory, err := h.service.GetTotalInvestmentCategory(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"investment_category": investmentCategory})
}

// MarginDistribution godoc
// @Summary      Distribuição de margem de lucro
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.MarginDistributionResponse
// @Router       /sales/margin-distribution [get]
func (h *Handler) MarginDistribution(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	distribution, err := h.service.MarginDistribution(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"margin_distribution": distribution})
}

// MarginDistribution godoc
// @Summary      Atualizar dados da venda
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        saleId path string true "ID da venda"
// @Param        Sale body domain.UpdateSaleParams true "Data"
// @Success      200 {object} domain.UpdateSaleParams
// @Router       /sales/{saleId} [put]
func (h *Handler) UpdateSale(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is null"})
		return
	}

	saleIdStr := c.Param("saleId")

	saleId, err := uuid.Parse(saleIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateSaleParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.UpdateSale(c.Request.Context(), userId, companyId, saleId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// MarginDistribution godoc
// @Summary      Busca o giro de estoque da empresa
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetInventoryTurnoverResponse
// @Router       /sales/stock-turnover [get]
func (h *Handler) GetInventoryTurnover(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	stockTurnover, err := h.service.GetInventoryTurnover(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stockTurnover)
}
