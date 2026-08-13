package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(router *gin.RouterGroup) {
	// Webhook chamado pela Evolution API a cada mensagem recebida — não passa
	// pelo AuthMiddleware, pois a Evolution API não possui um token JWT.
	router.POST("/whatsapp/webhook/:companyId", h.HandleWebhook)

	whatsappGroup := router.Group("/whatsapp").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	// integração de WhatsApp é configuração da empresa como um todo -> ADMIN
	whatsappGroup.Use(middleware.RequireRole("ADMIN"))
	{
		whatsappGroup.POST("/instance/create", h.CreateInstance)
		whatsappGroup.GET("/instance/connection-state", h.ConnectonState)
		whatsappGroup.DELETE("/instance/delete", h.Deleteinstance)
		whatsappGroup.GET("/instance/connect", h.ConnectInstance)

		whatsappGroup.POST("/bot", h.CreateWhatsAppBot)
		whatsappGroup.GET("/bot", h.GetWhatsAppBot)
		whatsappGroup.PUT("/bot/:id", h.UpdateWhatsAppBot)
	}
}
