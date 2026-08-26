package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	extractorcontext "github.com/ProTrack-Solutions/protrack-api/internal/pkg/extractorContext"
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

// CreateUser cria um usuário diretamente pelo serviço.
//
// Não exposto como rota HTTP: o cadastro de usuários é feito pelo
// fluxo de registro (auth), via Service.CreateUserTx. Mantido para
// uso interno/testes — sem anotações @Router para não aparecer no
// Swagger como um endpoint disponível.
func (h *Handler) CreateUser(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}

	companyId := companyIdAny.(uuid.UUID)

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is null"})
	}

	var req domain.CreateUserParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), userId, companyId, req)
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
// @Router       /user/{id} [delete]
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

// GetUserByEmail busca um usuário pelo e-mail.
//
// Não exposto como rota HTTP no momento — usado internamente pelo
// serviço (ex.: validação de login e checagem de e-mail duplicado em
// UpdateUser). Sem anotações @Router para não aparecer no Swagger.
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
// @Router       /user/{id} [get]
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

// ListUsers lista todos os usuários.
//
// Não exposto como rota HTTP no momento. Sem anotações @Router para
// não aparecer no Swagger como um endpoint disponível.
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// UpdatePasswordHash godoc
// @Summary      Atualiza a senha do usuário autenticado
// @Description  Requer a senha atual para confirmação; o usuário é identificado pelo token JWT (claim "sub"), não pelo corpo da requisição.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        password body domain.UpdatePasswordParams true "Senha atual e nova senha"
// @Success      204
// @Router       /user/password [put]
func (h *Handler) UpdatePasswordHash(c *gin.Context) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdatePasswordParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePasswordHash(c.Request.Context(), userId, req); err != nil {
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
// @Router       /user/{id} [put]
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

func (h *Handler) UpdateOwnProfile(c *gin.Context) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIdStr := userIdAny.(string)

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var req domain.UpdateOwnProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateOwnProfile(c.Request.Context(), userId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (h *Handler) CountUsers(c *gin.Context) {
	users, err := h.service.CountUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// ListUsersByCompany godoc
// @Summary      Lista os usuários da empresa do usuário autenticado
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.UserResponse
// @Router       /user/list-company [get]
func (h *Handler) ListUsersByCompany(c *gin.Context) {
	companyId, err := extractorcontext.ExtratorCompanyID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	users, err := h.service.ListUsersByCOmpany(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}
