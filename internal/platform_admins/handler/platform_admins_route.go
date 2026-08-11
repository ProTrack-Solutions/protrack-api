package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/auth/platform/login", h.Login) // sem AuthMiddleware, é login
}
