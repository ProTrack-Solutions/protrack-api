package handler

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/stripe/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

type Handler struct {
	service       *service.Service
	cfg           *config.Config
	webhookLogger *discord.DiscordLogger
}

func NewHandler(service *service.Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
		// Logs do webhook do Stripe vão para um canal próprio no Discord.
		webhookLogger: discord.NewDiscordLoggerWithURL(cfg.StripeWebhookURLDiscord),
	}
}

func (h *Handler) SyncSubscriptionWebhook(c *gin.Context) {
	// Captura panics aqui para que caiam no canal do webhook do Stripe e não
	// no canal geral (via GinDiscordErrorMiddleware).
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("Stripe webhook: panic ao processar evento")
			h.webhookLogger.Send(domain.LevelError, "Stripe Webhook: PANIC",
				fmt.Sprintf("**Erro:** %v\n\n```%s```", r, truncate(string(debug.Stack()), 1500)))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
	}()

	const MaxBodyBytes = 65536
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Stripe webhook: erro ao ler payload")
		h.webhookLogger.Send(domain.LevelError, "Stripe Webhook: erro ao ler payload", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao ler payload"})
		return
	}

	// 1. Validação de Segurança (Garante que a chamada REALMENTE veio do Stripe)
	sigHeader := c.GetHeader("Stripe-Signature")
	endpointSecret := h.cfg.StripeWebhookSecret

	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		// O SDK também rejeita eventos de outra versão da API; separa os casos
		// para não confundir com segredo do webhook errado.
		title := "Stripe Webhook: assinatura inválida"
		if strings.Contains(err.Error(), "API version") {
			title = "Stripe Webhook: versão da API incompatível"
		}

		log.Error().Err(err).Msg(title)
		h.webhookLogger.Send(domain.LevelError, title, fmt.Sprintf("**Erro:** %s\n**Versão esperada pelo SDK:** %s", err.Error(), stripe.APIVersion))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assinatura do webhook inválida"})
		return
	}

	// 2. Processa os Eventos Chave
	if err := h.service.SyncSubscriptionWebhook(c.Request.Context(), event); err != nil {
		log.Error().Err(err).Str("event_id", event.ID).Str("event_type", string(event.Type)).Msg("Stripe webhook: erro ao processar evento")
		h.webhookLogger.Send(domain.LevelError, "Stripe Webhook: erro ao processar evento",
			fmt.Sprintf("**Evento:** %s\n**ID:** %s\n**Erro:** %s", event.Type, event.ID, err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("event_id", event.ID).Str("event_type", string(event.Type)).Msg("Stripe webhook: evento processado")
	h.webhookLogger.Send(domain.LevelInfo, "Stripe Webhook: evento processado",
		fmt.Sprintf("**Evento:** %s\n**ID:** %s", event.Type, event.ID))

	// Sempre retorne 200 OK para o Stripe saber que o evento foi entregue com sucesso
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n... [stack truncado]"
	}
	return s
}
