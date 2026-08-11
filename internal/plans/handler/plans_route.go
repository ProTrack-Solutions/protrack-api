package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	plans := r.Group("/plans")
	{

		plans.GET("", h.ListPlans)
		plans.GET("/:id", h.GetPlanByID)

		admin := plans.Group("")
		admin.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
		admin.Use(middleware.RequireRole("SUPER_ADMIN"))
		{
			admin.POST("", h.CreatePlans)
			admin.PUT("/:id", h.UpdatePlan)
			admin.PATCH("/:id/active", h.TogglePlanActiveStatus)
		}
	}
}
