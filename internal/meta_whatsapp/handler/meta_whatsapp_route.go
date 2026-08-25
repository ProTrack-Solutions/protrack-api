package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	publicMetaWhatsapp := r.Group("/meta-whatsapp")
	{
		publicMetaWhatsapp.GET("/webhook", h.VerifyWebhook)
		publicMetaWhatsapp.POST("/webhook", middleware.MetaWhatsappMiddleware(h.AppSecret), h.HandleWebhookEvent)
	}

	metaWhatsapp := r.Group("/meta-whatsapp").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	metaWhatsapp.Use(middleware.RequireModule("whatsapp", h.queries))
	{
		metaWhatsapp.POST("/send-message", h.SendMessage)
		metaWhatsapp.GET("/config", h.GetCompanyConfig)
		metaWhatsapp.GET("/list-templates", h.ListApprovedTemplates)
		metaWhatsapp.POST("/upsert-config", h.UpsertCompanyConfig)
	}
}
