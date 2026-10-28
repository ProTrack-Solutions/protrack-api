package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

// Rotas da tela de clientes (indicadores, listagem com compras e perfil),
// no mesmo grupo /customers e com o mesmo módulo do cadastro.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	customers := r.Group("/customers")
	customers.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	customers.Use(middleware.RequireModule("customers", h.queries))
	{
		customers.GET("/overview", h.Overview)
		customers.GET("/browse", h.Browse)
		customers.GET("/:id/profile", h.Profile)
		customers.POST("/:id/deactivate", h.Deactivate)
		customers.POST("/:id/reactivate", h.Reactivate)
	}
}
