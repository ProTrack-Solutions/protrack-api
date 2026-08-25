package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/modules/domain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service   domain.ServiceInterface
	jwt       *jwt.JWTManager
	blackList *cache.TokenBlacklist
}

func NewHandler(service domain.ServiceInterface, jwt *jwt.JWTManager, blackList *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:   service,
		jwt:       jwt,
		blackList: blackList,
	}
}

// ListModules godoc
// @Summary      Lista todos os módulos
// @Tags         modules
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.ModuleResponse
// @Router       /modules/list [get]
func (h *Handler) ListModules(c *gin.Context) {
	modules, err := h.service.ListModules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, modules)
}

// GetModule godoc
// @Summary      Busca módulo por código
// @Tags         modules
// @Produce      json
// @Security     BearerAuth
// @Param        code path string true "Código do módulo"
// @Success      200 {object} domain.ModuleResponse
// @Router       /modules/{code} [get]
func (h *Handler) GetModule(c *gin.Context) {
	code := c.Param("code")

	module, err := h.service.GetModule(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, module)
}
