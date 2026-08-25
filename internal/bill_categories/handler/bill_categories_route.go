package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	billCategories := r.Group("/bill-categories")
	billCategories.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	billCategories.Use(middleware.RequireModule("financial", h.queries))
	{
		// leitura: ADMIN e USER podem consultar as categorias ao lançar/ver contas
		billCategories.GET("/:id", h.GetBillCategoriesById)
		billCategories.GET("", h.ListBillCategories)
		billCategories.GET("/active", h.ListBillCategoriesActive)

		// criar/remover/ativar categoria é configuração financeira da empresa -> ADMIN
		admin := billCategories.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.POST("", h.CreateBillCategories)
			admin.DELETE("/:id", h.DeleteBillCategories)
			admin.PUT("/toggle/:id", h.ToggleBillCategoriesActive)
		}
	}
}
