package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	plans := r.Group("/plans")
	{
		plans.POST("", h.CreatePlans)
		plans.GET("/:id", h.GetPlanByID)
		plans.GET("", h.ListPlans)
		plans.PUT("/:id", h.UpdatePlan)
		plans.PATCH("/:id/active", h.TogglePlanActiveStatus)
	}
}
