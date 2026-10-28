package handler

import (
	"errors"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/customer_profile/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/customer_profile/service"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
	queries    departmentModulesDomain.ServiceInterface
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist, queries departmentModulesDomain.ServiceInterface) *Handler {
	return &Handler{service: service, jwtManager: jwtManager, blacklist: blacklist, queries: queries}
}

func identity(c *gin.Context) (companyId, userId uuid.UUID, ok bool) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false
	}
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return uuid.Nil, uuid.Nil, false
	}
	userId, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, false
	}
	return companyIdAny.(uuid.UUID), userId, true
}

func pathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return uuid.Nil, false
	}
	return id, true
}

func writeError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrCustomerNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	log.Error().Err(err).Str("path", c.FullPath()).Msg("erro na tela de clientes")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno ao carregar os clientes"})
}

// Overview godoc
// @Summary      Indicadores da tela de clientes
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.Overview
// @Router       /customers/overview [get]
func (h *Handler) Overview(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	resp, err := h.service.Overview(c.Request.Context(), companyId)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Browse godoc
// @Summary      Lista clientes com resumo de compras e financeiro
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.BrowseParams false "Filtros, ordenação e paginação"
// @Success      200 {object} domain.BrowseResponse
// @Router       /customers/browse [get]
func (h *Handler) Browse(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	var params domain.BrowseParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.Browse(c.Request.Context(), companyId, params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Profile godoc
// @Summary      Perfil do cliente: dados, compras, parcelas em aberto e vale-troca
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      200 {object} domain.Profile
// @Router       /customers/{id}/profile [get]
func (h *Handler) Profile(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.service.Profile(c.Request.Context(), companyId, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) setActive(c *gin.Context, active bool) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.service.SetActive(c.Request.Context(), companyId, userId, id, active)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Deactivate godoc
// @Summary      Inativa o cliente (mantém o histórico)
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      200 {object} domain.Profile
// @Router       /customers/{id}/deactivate [post]
func (h *Handler) Deactivate(c *gin.Context) { h.setActive(c, false) }

// Reactivate godoc
// @Summary      Reativa o cliente
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do cliente"
// @Success      200 {object} domain.Profile
// @Router       /customers/{id}/reactivate [post]
func (h *Handler) Reactivate(c *gin.Context) { h.setActive(c, true) }
