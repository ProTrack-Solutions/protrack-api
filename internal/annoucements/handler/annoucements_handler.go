package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// CreateAnnoucements godoc
// @Summary      Cria um novo aviso
// @Description  Cria um comunicado/aviso para a empresa vinculada ao usuário autenticado
// @Tags         announcements
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.CreateAnnouncementsRequest true "Payload de criação do aviso"
// @Success      200 {object} map[string]string "{"message": "success"}"
// @Failure      400 {object} map[string]string "Erro de validação no JSON ou UUID inválido"
// @Failure      401 {object} map[string]string "Não autorizado se o company_id ou sub estiverem ausentes"
// @Failure      500 {object} map[string]string "Erro interno no servidor"
// @Router       /announcements [post]
func (h *Handler) CreateAnnoucements(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIDAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	userIdStr := userIDAny.(string)

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var req domain.CreateAnnouncementsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = h.service.CreateAnnoucements(c.Request.Context(), userID, companyId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "success"})
}

// ListAnnoucements godoc
// @Summary      Lista os avisos da empresa
// @Description  Retorna a lista de avisos ativos baseada nos parâmetros de paginação enviados via Header
// @Tags         announcements
// @Produce      json
// @Security     BearerAuth
// @Param        page header int false "Número da página (padrão: 1)"
// @Param        per_page header int false "Quantidade de registros por página (padrão: 10)"
// @Success      200 {object} globalDomain.PaginatedResponse[domain.ListAnnoucementsPaginateResponse] "Lista paginada de avisos"
// @Failure      401 {object} map[string]string "Não autorizado"
// @Failure      500 {object} map[string]string "Erro interno no servidor"
// @Router       /announcements [get]
func (h *Handler) ListAnnoucements(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var pagination globalDomain.PaginationParams
	if err := c.ShouldBindHeader(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	annoucements, err := h.service.ListAnnoucements(c.Request.Context(), companyId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, annoucements)
}

// DeleteAnnoucements godoc
// @Summary      Remove um aviso (Soft Delete)
// @Description  Desativa um aviso específico utilizando o ID enviado como parâmetro de rota
// @Tags         announcements
// @Security     BearerAuth
// @Param        id path string true "ID do aviso (UUID)"
// @Success      204 "No Content (Aviso removido com sucesso)"
// @Failure      400 {object} map[string]string "ID inválido informado na rota"
// @Failure      401 {object} map[string]string "Não autorizado"
// @Failure      500 {object} map[string]string "Erro interno no servidor"
// @Router       /announcements/{id} [delete]
func (h *Handler) DeleteAnnoucements(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	userIDAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return
	}

	userIdStr := userIDAny.(string)

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = h.service.DeleteAnnoucements(c.Request.Context(), id, companyId, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
