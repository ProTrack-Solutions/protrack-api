package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(router *gin.RouterGroup) {
	whatsappGroup := router.Group("/whatsapp").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	// integração de WhatsApp é configuração da empresa como um todo -> ADMIN
	whatsappGroup.Use(middleware.RequireRole("ADMIN"))
	{
		whatsappGroup.POST("/instance/create", h.CreateInstance)
		whatsappGroup.GET("/instance/connection-state", h.ConnectonState)
		whatsappGroup.DELETE("/instance/delete", h.Deleteinstance)
		whatsappGroup.GET("/instance/connect", h.ConnectInstance)
	}
}
