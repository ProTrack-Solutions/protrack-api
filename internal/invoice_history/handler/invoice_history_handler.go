package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/invoice_history/service"

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

// GetInvoiceById godoc
// @Summary      Busca fatura por ID
// @Tags         invoice-history
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da fatura"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "ID inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /invoice-history/{id} [get]
func (h *Handler) GetInvoiceById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	invoice, err := h.service.GetInvoceById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoice_history": invoice})
}

// GetInvoiceByMpPaymentId godoc
// @Summary      Busca fatura por mp_payment_id
// @Tags         invoice-history
// @Produce      json
// @Security     BearerAuth
// @Param        mp_payment_id path string true "MP Payment ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /invoice-history/mp/{mp_payment_id} [get]
func (h *Handler) GetInvoiceByMpPaymentId(c *gin.Context) {
	mpPaymentId := c.Param("mp_payment_id")

	if mpPaymentId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mp_payment_id is required"})
		return
	}

	invoice, err := h.service.GetInvoiceByMpPaymentId(c.Request.Context(), mpPaymentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoice_history": invoice})
}

// ListInvoiceByCompany godoc
// @Summary      Lista faturas da empresa com paginação
// @Tags         invoice-history
// @Produce      json
// @Security     BearerAuth
// @Param        Page header int false "Página"
// @Param        PerPage header int false "Itens por página"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "Parâmetros inválidos"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /invoice-history/company [get]
func (h *Handler) ListInvoiceByCompany(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globaldomain.PaginationParams
	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoices, err := h.service.ListInvoiceByCompany(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoices)
}
