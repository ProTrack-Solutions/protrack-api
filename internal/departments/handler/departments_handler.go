package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// CreateDepartment godoc
// @Summary      Cria um departamento
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        department body domain.CreateDepartmentParams true "Departamento"
// @Success      201 {object} domain.DepartmentResponse
// @Router       /departments [post]
func (h *Handler) CreateDepartment(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdStr := c.GetString("sub")
	if userIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.CreateDepartmentParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	department, err := h.service.CreateDepartment(c.Request.Context(), req, companyId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"department": department})
}

// DeleteDepartment godoc
// @Summary      Remove um departamento
// @Tags         departments
// @Produce      json
// @Param        id path string true "ID do departamento"
// @Success      204
// @Router       /departments/{id} [delete]
func (h *Handler) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	userIdStr := c.GetString("sub")
	if userIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteDepartment(c.Request.Context(), id, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDepartmentById godoc
// @Summary      Busca departamento por ID
// @Tags         departments
// @Produce      json
// @Param        id path string true "ID do departamento"
// @Success      200 {object} domain.DepartmentResponse
// @Router       /departments/{id} [get]
func (h *Handler) GetDepartmentById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	department, err := h.service.GetDepartmentById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"department": department})
}

// ListDepartmentsByCompanyId godoc
// @Summary      Lista departamentos por empresa
// @Tags         departments
// @Produce      json
// @Success      200 {array} domain.DepartmentResponse
// @Router       /departments/list [get]
func (h *Handler) ListDepartmentsByCompanyId(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	departments, err := h.service.ListDepartmentsByCompanyId(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, departments)
}

// SetStatusDepartment godoc
// @Summary      Altera o status de um departamento
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        status body domain.SetStatusDepartmentParams true "Status"
// @Success      200 {object} map[string]int64
// @Router       /departments/status [put]
func (h *Handler) SetStatusDepartment(c *gin.Context) {
	userIdStr := c.GetString("sub")
	if userIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	departmentIdStr := c.Param("departmentId")

	departmentId, err := uuid.Parse(departmentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.SetStatusDepartmentParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.service.SetStatusDepartment(c.Request.Context(), req, userId, departmentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// UpdateDepartment godoc
// @Summary      Atualiza um departamento
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        id path string true "ID do departamento"
// @Param        department body domain.UpdateDepartmentParams true "Departamento"
// @Success      200 {object} domain.DepartmentResponse
// @Router       /departments/{id} [put]
func (h *Handler) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	userIdStr := c.GetString("sub")
	if userIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateDepartmentParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	department, err := h.service.UpdateDepartment(c.Request.Context(), id, userId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"department": department})
}
