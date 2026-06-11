package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	companies := r.Group("/companies")
	companies.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		companies.POST("", h.CreateCompany)
		companies.DELETE("/:id", h.DeleteCompany)
		companies.GET("/document/:document", h.GetCompanyByDocument)
		companies.GET("/:id", h.GetCompanyByID)
		companies.GET("", h.ListCompanies)
		companies.POST("/set/", h.SetCompanyStatus)
		companies.PUT("/:id", h.UpdateCompany)
	}
}
