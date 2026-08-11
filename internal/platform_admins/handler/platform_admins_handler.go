package handler

import (
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/platform_admins/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/platform_admins/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service     *service.Service
	rateLimiter *cache.RateLimiter
	cfg         *config.Config
}

func NewHandler(service *service.Service, cfg *config.Config, rateLimiter *cache.RateLimiter) *Handler {
	return &Handler{
		service:     service,
		rateLimiter: rateLimiter,
		cfg:         cfg,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req domain.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowed, err := h.rateLimiter.Allow(c.Request.Context(), "platform_login:"+req.Email, 5, time.Minute*15)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
		return
	}

	response, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)

	c.SetCookie(
		"platform_access_token", // nome distinto do cookie de tenant
		response.AccessToken,
		int(response.ExpiresIn), // TTL correto: é o do access token mesmo, aqui está certo
		"/",
		"",
		h.cfg.IsProduction, // vem de config, não hardcoded
		true,
	)

	c.SetCookie(
		"platform_refresh_token",
		response.RefreshToken,
		7*24*60*60, // TTL do refresh de verdade (7 dias em segundos)
		"/",
		"",
		h.cfg.IsProduction,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "login successful"}) // sem tokens no body
}
