package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	companySettings := r.Group("/company-settings")
	{
		companySettings.GET("", h.ListCompanySettings)
		companySettings.PUT("", h.RestorDefaultSettings)
		companySettings.POST("", h.UpsertCompanySetting)
	}
}
