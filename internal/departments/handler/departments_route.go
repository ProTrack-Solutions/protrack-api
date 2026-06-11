package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	departments := r.Group("/departments")
	{
		departments.POST("", h.CreateDepartment)
		departments.DELETE("/:id", h.DeleteDepartment)
		departments.GET("/:id", h.GetDepartmentById)
		departments.GET("/list/:id", h.ListDepartmentsByCompanyId)
		departments.PUT("/status", h.SetStatusDepartment)
		departments.PUT("/:id", h.UpdateDepartment)

	}
}
