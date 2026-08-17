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
	extratorcontext "github.com/ProTrack-Solutions/protrack-api/internal/pkg/extratorContext"
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

func (h *Handler) SendMessage(c *gin.Context) {
	companyId, err := extratorcontext.ExtratorCompanyID(c)
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

func (h *Handler) GetCompanyConfig(c *gin.Context) {
	companyId, err := extratorcontext.ExtratorCompanyID(c)
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

func (h *Handler) UpsertCompanyConfig(c *gin.Context) {
	companyId, err := extratorcontext.ExtratorCompanyID(c)
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

func (h *Handler) ListApprovedTemplates(c *gin.Context) {
	template, err := h.service.ListApprovedTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

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
