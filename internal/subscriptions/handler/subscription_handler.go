package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
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

// GetSubscriptionById godoc
// @Summary      Busca assinatura por ID
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da assinatura"
// @Success      200 {object} domain.SubscriptionResponse
// @Failure      400 {object} map[string]string "ID da assinatura inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription/{id} [get]
func (h *Handler) GetSubscriptionById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription, err := h.service.GetSubscriptionById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// UpdateSubscriptionPlan godoc
// @Summary      Atualiza o plano da assinatura
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da assinatura"
// @Param        plan body domain.UpdateSubscriptionPlanRequest true "Dados do novo plano"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription/plan/{id} [put]
func (h *Handler) UpdateSubscriptionPlan(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateSubscriptionPlanRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateSubscriptionPlan(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UpdateSubscriptionMethod godoc
// @Summary      Atualiza método de pagamento da assinatura
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da assinatura"
// @Param        method body domain.UpdateSubscriptionMethodRequest true "Dados do novo método de pagamento"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription/method/{id} [put]
func (h *Handler) UpdateSubscriptionMethod(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateSubscriptionMethodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateSubscriptionMethod(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UpdateSubscriptionStatus godoc
// @Summary      Atualiza status da assinatura
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da assinatura"
// @Param        status body domain.UpdateSubscriptionStatusRequest true "Novo status"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription/status/{id} [put]
func (h *Handler) UpdateSubscriptionStatus(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateSubscriptionStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateSubscriptionStatus(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// CancelSubscription godoc
// @Summary      Cancela uma assinatura
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da assinatura"
// @Success      200
// @Failure      400 {object} map[string]string "ID da assinatura inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /subscription/cancel/{id} [put]
func (h *Handler) CancelSubscription(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CancelSubscription(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
