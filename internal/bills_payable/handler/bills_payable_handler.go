package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/service"
	globaldomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
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

// CreateBillPayable godoc
// @Summary      Cria uma conta a pagar
// @Tags         bills-payable
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        bill body domain.CreateBillPayableRequest true "Conta a pagar"
// @Success      201
// @Router       /bills-payable [post]
func (h *Handler) CreateBillPayable(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreateBillPayableRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateBillPayable(c.Request.Context(), companyId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// GetBillsPayableById godoc
// @Summary      Busca conta a pagar por ID
// @Tags         bills-payable
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da conta"
// @Success      200 {object} domain.BillsPayableResponse
// @Router       /bills-payable/{id} [get]
func (h *Handler) GetBillsPayableById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.GetBillsByIdRequest

	req.ID = id

	req.CompanyID = companyId

	billPayable, err := h.service.GetBillsPayableById(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bill_payable": billPayable})
}

// GetBillsByStatus godoc
// @Summary      Lista contas a pagar por status
// @Tags         bills-payable
// @Produce      json
// @Security     BearerAuth
// @Param        status path string true "Status da conta"
// @Success      200 {array} domain.BillsPayableResponse
// @Router       /bills-payable/status/{status} [get]
func (h *Handler) GetBillsByStatus(c *gin.Context) {
	status := c.GetString("status")

	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.GetBillsByStatusRequest

	req.CompanyID = companyId

	req.Status = status

	billsPayable, err := h.service.GetBillsByStatus(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bills_payable": billsPayable})
}

// GetOverdueBills godoc
// @Summary      Lista contas a pagar em atraso
// @Tags         bills-payable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.BillsPayableResponse
// @Router       /bills-payable/overdue [get]
func (h *Handler) GetOverdueBills(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	billsPayable, err := h.service.GetOverdueBills(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bills_payable": billsPayable})
}

// ListBillsPayable godoc
// @Summary      Lista contas a pagar da empresa
// @Tags         bills-payable
// @Produce      json
// @Security     BearerAuth
// @Param        page header int false "Número da página (padrão: 1)"
// @Param        per_page header int false "Quantidade de registros por página (padrão: 10)"
// @Success      200 {array} domain.ListBillsPayableResponse
// @Router       /bills-payable/list [get]
func (h *Handler) ListBillsPayable(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globaldomain.PaginationParams

	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	billsPayable, err := h.service.ListBillsPayable(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bills_payable": billsPayable})
}

// PayBill godoc
// @Summary      Registra pagamento de uma conta
// @Tags         bills-payable
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da conta"
// @Param        payment body domain.PayBillRequest true "Pagamento"
// @Success      200
// @Router       /bills-payable/pay/{id} [put]
func (h *Handler) PayBill(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.PayBillRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = id

	req.CompanyID = companyId

	if err := h.service.PayBill(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.Status(http.StatusOK)
}

// ScheduleBill godoc
// @Summary      Agenda pagamento de uma conta
// @Tags         bills-payable
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da conta"
// @Param        schedule body domain.ScheduleBillRequest true "Agendamento"
// @Success      200
// @Router       /bills-payable/schedule/{id} [put]
func (h *Handler) ScheduleBill(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.ScheduleBillRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	req.ID = id

	if err := h.service.ScheduleBill(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UpdateBillPayable godoc
// @Summary      Atualiza uma conta a pagar
// @Tags         bills-payable
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da conta"
// @Param        bill body domain.UpdateBillPayableRequest true "Conta a pagar"
// @Success      200
// @Router       /bills-payable/{id} [put]
func (h *Handler) UpdateBillPayable(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateBillPayableRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyId

	req.ID = id

	if err := h.service.UpdateBillPayable(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// GetBillsPayableSummary godoc
// @Summary      Resumo das contas a pagar
// @Tags         bills-payable
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.GetBillsPayableSummaryResponse
// @Router       /bills-payable/summary [get]
func (h *Handler) GetBillsPayableSummary(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	billsSummary, err := h.service.GetBillsPayableSummary(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bills_summary": billsSummary})
}
