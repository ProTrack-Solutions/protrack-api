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

// ListCompanySettings godoc
// @Summary      Lista as configurações da empresa
// @Tags         company-settings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.CompanySettingResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /company-settings [get]
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

// UpsertCompanySetting godoc
// @Summary      Cria ou atualiza uma configuração da empresa
// @Description  Retorna 201 quando a configuração é criada e 200 quando uma configuração existente é atualizada
// @Tags         company-settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        setting body domain.UpsertCompanySettingRequest true "Configuração"
// @Success      200
// @Success      201
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /company-settings [post]
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

// RestorDefaultSettings godoc
// @Summary      Restaura as configurações padrão da empresa
// @Tags         company-settings
// @Produce      json
// @Security     BearerAuth
// @Success      200
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /company-settings [put]
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
