package handler

import (
	"net/http"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/cache"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/auth/adapters/jwt"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/payment_history/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/payment_history/service"
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

func (h *Handler) CreatePaymentHistory(c *gin.Context) {
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

	var req domain.CreatePaymentHistoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	req.UserID = userId

	if err := h.service.CreatePaymentHistory(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) GetPaymentsByCustomer(c *gin.Context) {
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

	var req domain.GetPaymentsByCustomerRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	req.CustomerID = customerId

	paymentsHistory, err := h.service.GetPaymentsByCustomer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments_history": paymentsHistory})
}

func (h *Handler) GetPaymentsBySale(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	saleIdStr := c.Param("saleId")

	saleId, err := uuid.Parse(saleIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.GetPaymentsBySaleRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	req.SaleID = saleId

	paymentsHistory, err := h.service.GetPaymentsBySale(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments_history": paymentsHistory})
}

func (h *Handler) GetTotalReceivedByPeriod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.GetTotalReceivedByPeriodRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	total, err := h.service.GetTotalReceivedByPeriod(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	c.JSON(http.StatusOK, gin.H{"total_payment_period": total})
}

func (h *Handler) ListPaymentHistory(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	paymentsHistory, err := h.service.ListPaymentHistory(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments_history": paymentsHistory})
}

// func (h *Handler) ExportExcel(c *gin.Context) {
// 	companyIdAny, exists := c.Get("company_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
// 		return
// 	}

// 	companyId := companyIdAny.(uuid.UUID)

// 	history, err := h.service.ListPaymentHistory(context.Background(), companyId)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados"})
// 		return
// 	}

// 	headers := []string{
// 		"Data do Pagamento",
// 		"Cliente",
// 		"Valor Pago",
// 		"Método",
// 		"Recebido por",
// 		"Observações",
// 		"ID da Venda",
// 	}

// 	var rows [][]any
// 	for _, p := range history {
// 		rows = append(rows, []any{
// 			p.PaymentDate.Format("02/01/2006 15:04"),
// 			p.CustomerName,
// 			p.AmountPaid,
// 			p.PaymentMethodName,
// 			p.UserName,
// 			p.Notes,
// 			p.SaleID.String(),
// 		})
// 	}

// 	fileName := fmt.Sprintf("historico_pagamentos_%s.xlsx", time.Now().Format("2006-01-02"))
// 	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
// 	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

// 	err = reports.GenerateExcel(c.Writer, "Histórico de Pagamentos", headers, rows)
// 	if err != nil {
// 		return
// 	}
// }
