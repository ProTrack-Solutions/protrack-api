package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	departments := r.Group("/departments").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		departments.POST("", h.CreateDepartment)
		departments.DELETE("/:id", h.DeleteDepartment)
		departments.GET("/:id", h.GetDepartmentById)
		departments.GET("/list", h.ListDepartmentsByCompanyId)
		departments.PUT("/status/:departmentId", h.SetStatusDepartment)
		departments.PUT("/:id", h.UpdateDepartment)

	}
}
