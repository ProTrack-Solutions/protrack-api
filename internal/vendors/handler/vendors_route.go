package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	vendors := r.Group("/vendors").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		vendors.POST("", h.CreateVendors)
		vendors.GET("/:id", h.GetVendorsById)
		vendors.GET("/list", h.ListVendors)
		vendors.GET("/list/is-active", h.ListVendorsIsActive)
		vendors.PUT("/toggle/:id", h.ToggleVendorsActive)
		vendors.PUT("/:id", h.UpdateVendors)
	}
}
