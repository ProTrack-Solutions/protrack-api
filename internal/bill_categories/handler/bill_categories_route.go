package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	billCategories := r.Group("/bill-categories").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		billCategories.POST("", h.CreateBillCategories)
		billCategories.DELETE("/:id", h.DeleteBillCategories)
		billCategories.GET("/:id", h.GetBillCategoriesById)
		billCategories.GET("", h.ListBillCategories)
		billCategories.GET("/active", h.ListBillCategoriesActive)
		billCategories.PUT("/toggle/:id", h.ToggleBillCategoriesActive)
	}
}
