package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	cashRegister := r.Group("/cash-register")
	cashRegister.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	cashRegister.Use(middleware.RequireModule("sales", h.queries))
	{
		cashRegister.GET("/current", h.Current)
		cashRegister.POST("/open", h.Open)
		cashRegister.POST("/movements", h.AddMovement)
		cashRegister.POST("/close", h.Close)
		cashRegister.GET("/sessions", h.List)
		cashRegister.GET("/sessions/:id", h.GetById)
	}
}
