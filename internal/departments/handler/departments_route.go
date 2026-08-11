package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	departments := r.Group("/departments")
	departments.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// leitura: ADMIN e USER podem ver
		// TODO(ownership): USER deveria só poder ver o próprio departamento;
		// o handler hoje aceita qualquer :id sem validar contra o usuário autenticado.
		departments.GET("/:id", h.GetDepartmentById)
		departments.GET("/list", h.ListDepartmentsByCompanyId)

		// escrita: só ADMIN
		admin := departments.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.POST("", h.CreateDepartment)
			admin.DELETE("/:id", h.DeleteDepartment)
			admin.PUT("/status/:departmentId", h.SetStatusDepartment)
			admin.PUT("/:id", h.UpdateDepartment)
		}
	}
}
