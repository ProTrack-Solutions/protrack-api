package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	companies := r.Group("/companies")
	companies.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{

		companies.GET("/me", h.GetCompanyByID)

		admin := companies.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.DELETE("/:id", h.DeleteCompany)
			admin.POST("/set/", h.SetCompanyStatus)
			admin.PUT("/:id", h.UpdateCompany)
		}
		superadmin := companies.Group("")
		superadmin.Use(middleware.RequireRole("SUPER_ADMIN"))
		{
			superadmin.GET("/list", h.ListCompanies)
			superadmin.GET("/count", h.CountCompanies)
		}
	}
}
