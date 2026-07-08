package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// CreateUser godoc
// @Summary      Cria um usuário
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user body domain.CreateUserParams true "Usuário"
// @Success      201 {object} domain.UserResponse
// @Router       /users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req domain.CreateUserParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// DeleteUser godoc
// @Summary      Remove um usuário
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do usuário"
// @Success      204
// @Router       /{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")

	idUUID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := pgtype.UUID{Bytes: idUUID, Valid: true}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetUserByEmail godoc
// @Summary      Busca usuário por e-mail
// @Tags         users
// @Produce      json
// @Param        email path string true "E-mail"
// @Success      200 {object} domain.UserResponse
// @Router       /users/email/{email} [get]
func (h *Handler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	user, err := h.service.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetUserById godoc
// @Summary      Busca usuário por ID
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do usuário"
// @Success      200 {object} domain.UserResponse
// @Router       /{id} [get]
func (h *Handler) GetUserById(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// ListUsers godoc
// @Summary      Lista todos os usuários
// @Tags         users
// @Produce      json
// @Success      200 {array} domain.UserResponse
// @Router       /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// UpdatePasswordHash godoc
// @Summary      Atualiza a senha do usuário
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        password body domain.UpdatePasswordHashParams true "Senha"
// @Success      204
// @Router       /users/password [put]
func (h *Handler) UpdatePasswordHash(c *gin.Context) {
	var req domain.UpdatePasswordHashParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePasswordHash(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	c.Status(http.StatusNoContent)
}

// UpdateUser godoc
// @Summary      Atualiza um usuário
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do usuário"
// @Param        user body domain.UpdateUserRequest true "Usuário"
// @Success      200 {object} domain.UserResponse
// @Router       /{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
