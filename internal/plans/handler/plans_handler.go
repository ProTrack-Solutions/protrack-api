package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
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

// CreatePlans godoc
// @Summary      Cria um plano
// @Tags         plans
// @Accept       json
// @Produce      json
// @Param        plan body domain.CreatePlanRequest true "Plano"
// @Success      201
// @Failure      400 {object} map[string]string "Requisição inválida"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans [post]
func (h *Handler) CreatePlans(c *gin.Context) {
	var req domain.CreatePlanRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreatePlans(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// GetPlanByID godoc
// @Summary      Busca um plano por ID
// @Tags         plans
// @Produce      json
// @Param        id path string true "ID do plano"
// @Success      200 {object} domain.PlanResponse
// @Failure      400 {object} map[string]string "ID do plano inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans/{id} [get]
func (h *Handler) GetPlanByID(c *gin.Context) {
	planIdStr := c.Param("id")

	planId, err := uuid.Parse(planIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	plan, err := h.service.GetPlanByID(c.Request.Context(), planId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// ListPlans godoc
// @Summary      Lista todos os planos
// @Tags         plans
// @Produce      json
// @Success      200 {array} domain.PlanResponse
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans [get]
func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.service.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plans)
}

// ListPlansByActiveStatus godoc
// @Summary      Lista planos por status ativo
// @Tags         plans
// @Produce      json
// @Param        active query boolean false "Ativo"
// @Success      200 {array} domain.PlanResponse
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans [get]
func (h *Handler) ListPlansByActiveStatus(c *gin.Context) {
	activeStr := c.Query("active")
	active := activeStr == "true"

	plans, err := h.service.ListPlansByActiveStatus(c.Request.Context(), active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plans)
}

// UpdatePlan godoc
// @Summary      Atualiza um plano
// @Tags         plans
// @Accept       json
// @Produce      json
// @Param        id path string true "ID do plano"
// @Param        plan body domain.UpdatePlanParams true "Plano"
// @Success      200
// @Failure      400 {object} map[string]string "Requisição inválida ou ID inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans/{id} [put]
func (h *Handler) UpdatePlan(c *gin.Context) {
	planIdStr := c.Param("id")

	planId, err := uuid.Parse(planIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	var req domain.UpdatePlanParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePlan(c.Request.Context(), planId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// TogglePlanActiveStatus godoc
// @Summary      Ativa ou desativa um plano
// @Tags         plans
// @Produce      json
// @Param        id path string true "ID do plano"
// @Param        active query boolean true "Ativo"
// @Success      200
// @Failure      400 {object} map[string]string "ID do plano inválido"
// @Failure      500 {object} map[string]string "Erro interno do servidor"
// @Router       /plans/{id}/active [patch]
func (h *Handler) TogglePlanActiveStatus(c *gin.Context) {
	planIdStr := c.Param("id")

	planId, err := uuid.Parse(planIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	activeStr := c.Query("active")
	active := activeStr == "true"

	if err := h.service.TogglePlanActiveStatus(c.Request.Context(), planId, active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
