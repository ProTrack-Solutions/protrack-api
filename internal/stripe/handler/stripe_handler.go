package handler

import (
	"io"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/stripe/service"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v86/webhook"
)

type Handler struct {
	service *service.Service
	cfg     *config.Config
}

func NewHandler(service *service.Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) SyncSubscriptionWebhook(c *gin.Context) {
	const MaxBodyBytes = 65536
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao ler payload"})
		return
	}

	// 1. Validação de Segurança (Garante que a chamada REALMENTE veio do Stripe)
	sigHeader := c.GetHeader("Stripe-Signature")
	endpointSecret := h.cfg.StripeWebhookSecret

	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assinatura do webhook inválida"})
		return
	}

	// 2. Processa os Eventos Chave
	if err := h.service.SyncSubscriptionWebhook(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sempre retorne 200 OK para o Stripe saber que o evento foi entregue com sucesso
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
