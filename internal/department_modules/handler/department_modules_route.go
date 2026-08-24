package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	departmentModules := r.Group("/department-modules")
	departmentModules.Use(middleware.AuthMiddleware(h.jwt, h.blackList))

	admin := departmentModules.Use(middleware.RequireRole("ADMIN"))
	{
		admin.GET("/:departmentId", h.ListModulesByDepartment)
		admin.POST("", h.AddModuleToDepartment)
		admin.DELETE("/remove/:departmentId", h.RemoveModuleFromDepartment)
		admin.DELETE("/replace/:departmentId", h.ReplaceDepartmentModules)
	}
}
