package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	methods := r.Group("/payment-methods")
	methods.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// leitura: ADMIN e USER precisam consultar ao registrar uma venda/pagamento
		methods.GET("", h.ListPaymentMethod)
		methods.GET("/is-active", h.ListPaymentMethodIsActive)
		methods.GET("/stats", h.GetPaymentMethodsStats)
		methods.GET("/:id", h.GetPaymentMethodById)

		// criar/ativar-desativar método de pagamento é configuração da empresa -> ADMIN
		admin := methods.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.POST("", h.CreatePaymentMethod)
			admin.PUT("/:id", h.TogglePaymentMethodActive)
		}
	}
}
