package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	methods := r.Group("/payment-methods").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		methods.POST("", h.CreatePaymentMethod)
        methods.GET("", h.ListPaymentMethod)

        methods.GET("/is-active", h.ListPaymentMethodIsActive)
        methods.GET("/stats", h.GetPaymentMethodsStats)

        methods.GET("/:id", h.GetPaymentMethodById)
        methods.PUT("/:id", h.TogglePaymentMethodActive)
	}
}
