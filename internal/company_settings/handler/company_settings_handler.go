package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	extractorcontext "github.com/ProTrack-Solutions/protrack-api/internal/pkg/extractorContext"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service   domain.ServiceInterface
	jwt       *jwt.JWTManager
	blacklist *cache.TokenBlacklist
}

func NewHandler(service domain.ServiceInterface, jwt *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:   service,
		jwt:       jwt,
		blacklist: blacklist,
	}
}

func (h *Handler) ListCompanySettings(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	settings, err := h.service.ListCompanySettings(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func (h *Handler) UpsertCompanySetting(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpsertCompanySettingRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setting, err := h.service.UpsertCompanySetting(c.Request.Context(), companyId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if setting.CreatedAt.Equal(setting.UpdatedAt) {
		c.Status(http.StatusCreated)
	} else {
		c.Status(http.StatusOK)
	}
}

func (h *Handler) RestorDefaultSettings(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RestorDefaultSettings(c.Request.Context(), companyId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.Status(http.StatusOK)
}
