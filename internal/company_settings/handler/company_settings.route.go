package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	companySettings := r.Group("/company-settings")
	companySettings.Use(middleware.AuthMiddleware(h.jwt, h.blacklist))
	{

		admin := companySettings.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			companySettings.GET("", h.ListCompanySettings)
			companySettings.PUT("", h.RestorDefaultSettings)
			companySettings.POST("", h.UpsertCompanySetting)

		}

	}
}
