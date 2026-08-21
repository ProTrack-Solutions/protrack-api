package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	extractorcontext "github.com/ProTrack-Solutions/protrack-api/internal/pkg/extractorContext"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/service"
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

// GetSubscriptionDetails godoc
// @Summary      Busca os dados completos da assinatura da empresa autenticada
// @Description  Retorna a assinatura, o plano contratado (com valor e features) e o método de pagamento vinculado
// @Tags         subscription-management
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.SubscriptionDetailsResponse
// @Failure      401 {object} map[string]string "Não autenticado"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription-management/subscription [get]
func (h *Handler) GetSubscriptionDetails(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	subscription, err := h.service.GetSubscriptionDetails(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// AddPaymentMethod godoc
// @Summary      Adiciona um método de pagamento à assinatura
// @Tags         subscription-management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payment_method body domain.AddPaymentMethodRequest true "Dados do novo método de pagamento"
// @Success      201
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      401 {object} map[string]string "Não autenticado"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription-management/payment-methods [post]
func (h *Handler) AddPaymentMethod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.AddPaymentMethodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddPaymentMethod(c.Request.Context(), companyId, userId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Método de pagamento adicionado com sucesso"})
}

// UpdateDefaultPaymentMethod godoc
// @Summary      Define o método de pagamento padrão da assinatura
// @Tags         subscription-management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do método de pagamento"
// @Param        payment_method body domain.UpdateDefaultPaymentMethodRequest true "Dados da atualização"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      401 {object} map[string]string "Não autenticado"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription-management/payment-methods/{id}/default [put]
func (h *Handler) UpdateDefaultPaymentMethod(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	paymentMethodIdStr := c.Param("id")

	paymentMethodId, err := uuid.Parse(paymentMethodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method ID format"})
		return
	}

	var req domain.UpdateDefaultPaymentMethodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateDefaultPaymentMethod(c.Request.Context(), companyId, paymentMethodId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// CancelSubscription godoc
// @Summary      Cancela a assinatura da empresa autenticada
// @Tags         subscription-management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        cancel body domain.CancelSubscriptionRequest true "Dados do cancelamento"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      401 {object} map[string]string "Não autenticado"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription-management/subscription/cancel [put]
func (h *Handler) CancelSubscription(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CancelSubscriptionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CancelSubscription(c.Request.Context(), companyId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// CountMenssageInvitAmount godoc
// @Summary      Consulta o uso de mensagens de WhatsApp da assinatura autenticada
// @Description  Retorna o limite de mensagens do plano contratado, a quantidade de mensagens de convite enviadas no período de cobrança vigente e o percentual já utilizado
// @Tags         subscription-management
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.CountMenssageInvitAmountResponse
// @Failure      401 {object} map[string]string "Não autenticado"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription-management/whatsapp-use [get]
func (h *Handler) CountMenssageInvitAmount(c *gin.Context) {
	companyID, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.service.CountMenssageInvitAmount(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}
