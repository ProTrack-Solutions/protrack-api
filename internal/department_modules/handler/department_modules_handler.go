package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// ListModulesByDepartment godoc
// @Summary      Lista os módulos de um departamento
// @Tags         department-modules
// @Produce      json
// @Security     BearerAuth
// @Param        departmentId path string true "ID do departamento"
// @Success      200 {array} domain.ModuleResponse
// @Router       /department-modules/{departmentId} [get]
func (h *Handler) ListModulesByDepartment(c *gin.Context) {
	departmentIdStr := c.Param("departmentId")

	departmentId, err := uuid.Parse(departmentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	modules, err := h.service.ListModulesByDepartment(c.Request.Context(), departmentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, modules)
}

// AddModuleToDepartment godoc
// @Summary      Adiciona um módulo a um departamento
// @Tags         department-modules
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        module body domain.AddModuleToDepartmentRequest true "Módulo"
// @Success      201
// @Router       /department-modules [post]
func (h *Handler) AddModuleToDepartment(c *gin.Context) {
	var req domain.AddModuleToDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddModuleToDepartment(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveModuleFromDepartment godoc
// @Summary      Remove um módulo de um departamento
// @Tags         department-modules
// @Produce      json
// @Security     BearerAuth
// @Param        departmentId path string true "ID do departamento"
// @Param        module_code query string true "Código do módulo"
// @Success      204
// @Router       /department-modules/remove/{departmentId} [delete]
func (h *Handler) RemoveModuleFromDepartment(c *gin.Context) {
	departmentIdStr := c.Param("departmentId")

	departmentId, err := uuid.Parse(departmentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.RemoveModuleFromDepartmentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RemoveModuleFromDepartment(c.Request.Context(), departmentId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ReplaceDepartmentModules godoc
// @Summary      Restaura os módulos padrão de um departamento
// @Tags         department-modules
// @Produce      json
// @Security     BearerAuth
// @Param        departmentId path string true "ID do departamento"
// @Success      204
// @Router       /department-modules/replace/{departmentId} [delete]
func (h *Handler) ReplaceDepartmentModules(c *gin.Context) {
	departmentIdStr := c.Param("departmentId")

	departmentId, err := uuid.Parse(departmentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ReplaceDepartmentModules(c.Request.Context(), departmentId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
