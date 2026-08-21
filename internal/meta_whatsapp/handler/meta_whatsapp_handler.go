package handler

import (
	"io"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	discordDomain "github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	extractorcontext "github.com/ProTrack-Solutions/protrack-api/internal/pkg/extractorContext"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       domain.ServiceInterface
	discordLogger *discord.DiscordLogger
	VerifyToken   string
	AppSecret     string
	jwtManager    *jwt.JWTManager
	blacklist     *cache.TokenBlacklist
}

func NewHandler(service domain.ServiceInterface, cfg *config.Config, discordLogger *discord.DiscordLogger, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:       service,
		VerifyToken:   cfg.MetaWebhookVerifyToken,
		discordLogger: discordLogger,
		jwtManager:    jwtManager,
		blacklist:     blacklist,
		AppSecret:     cfg.MetaAppSecret,
	}
}

// SendMessage godoc
// @Summary      Envia uma mensagem de template via WhatsApp
// @Description  Dispara uma mensagem de template aprovado para o número informado, respeitando o modo de envio (platform_shared ou own_waba) configurado para a empresa
// @Tags         meta-whatsapp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        message body domain.SendMessageRequest true "Dados da mensagem"
// @Success      200 {object} domain.Message
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /meta-whatsapp/send-message [post]
func (h *Handler) SendMessage(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req domain.SendMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.service.SendMessage(c.Request.Context(), companyId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

// GetCompanyConfig godoc
// @Summary      Busca a configuração de WhatsApp da empresa
// @Tags         meta-whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.CompanyWhatsAppConfig
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /meta-whatsapp/config [get]
func (h *Handler) GetCompanyConfig(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.GetCompanyConfig(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpsertCompanyConfig godoc
// @Summary      Cria ou atualiza a configuração de WhatsApp da empresa
// @Tags         meta-whatsapp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        config body domain.UpsertCompanyConfigRequest true "Configuração de WhatsApp"
// @Success      200 {object} domain.CompanyWhatsAppConfig
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /meta-whatsapp/upsert-config [post]
func (h *Handler) UpsertCompanyConfig(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpsertCompanyConfigRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.UpsertCompanyConfig(c.Request.Context(), companyId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListApprovedTemplates godoc
// @Summary      Lista os templates de WhatsApp aprovados
// @Tags         meta-whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.Template
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /meta-whatsapp/list-templates [get]
func (h *Handler) ListApprovedTemplates(c *gin.Context) {
	template, err := h.service.ListApprovedTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// VerifyWebhook godoc
// @Summary      Verifica o webhook do WhatsApp junto à Meta
// @Description  Endpoint de handshake exigido pela Meta ao configurar o webhook: valida hub.verify_token e ecoa hub.challenge quando hub.mode é "subscribe"
// @Tags         meta-whatsapp
// @Produce      plain
// @Param        hub.mode query string true "Modo do handshake (subscribe)"
// @Param        hub.verify_token query string true "Token de verificação configurado na Meta"
// @Param        hub.challenge query string true "Valor a ser ecoado quando a verificação for bem-sucedida"
// @Success      200 {string} string "challenge"
// @Failure      403
// @Router       /meta-whatsapp/webhook [get]
func (h *Handler) VerifyWebhook(c *gin.Context) {
	mode := c.Query("hub.mode")
	verifyToken := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && verifyToken == h.VerifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.Status(http.StatusForbidden)
}

// HandleWebhookEvent godoc
// @Summary      Recebe eventos do webhook do WhatsApp
// @Description  Endpoint chamado pela Meta para notificar status de mensagens e eventos da conta. A assinatura da requisição é validada pelo MetaWhatsappMiddleware via header X-Hub-Signature-256. Sempre responde 200 para evitar reenvios da Meta, mesmo em caso de falha no processamento (o erro é registrado no Discord)
// @Tags         meta-whatsapp
// @Accept       json
// @Success      200
// @Router       /meta-whatsapp/webhook [post]
func (h *Handler) HandleWebhookEvent(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.discordLogger.Send(discordDomain.LevelWarning, "HandleWebhookEvent: ReadAll", err.Error())
		c.Status(http.StatusOK)
		return
	}
	if err := h.service.HandleWebhookEvent(c.Request.Context(), payload); err != nil {
		h.discordLogger.Send(discordDomain.LevelWarning, "HandleWebhookEvent: HandleWebhookEvent", err.Error())
	}

	c.Status(http.StatusOK)
}
