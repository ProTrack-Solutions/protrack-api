package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/modules/domain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service domain.ServiceInterface
}

func NewHandler(service domain.ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ListModules(c *gin.Context) {
	modules, err := h.service.ListModules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, modules)
}

func (h *Handler) GetModule(c *gin.Context) {
	code := c.Param("code")

	module, err := h.service.GetModule(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, module)
}
