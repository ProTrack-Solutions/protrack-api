package handler

import (
	"errors"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/service"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
	queries    departmentModulesDomain.ServiceInterface
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist, queries departmentModulesDomain.ServiceInterface) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
		queries:    queries,
	}
}

// identity lê empresa, usuário e se é administrador a partir do token
func identity(c *gin.Context) (companyId, userId uuid.UUID, isAdmin bool, ok bool) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false, false
	}

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return uuid.Nil, uuid.Nil, false, false
	}

	userId, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, false, false
	}

	role, _ := c.Get("role")
	return companyIdAny.(uuid.UUID), userId, role == "ADMIN", true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCashSessionAlreadyOpen), errors.Is(err, domain.ErrNoOpenCashSession):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInsufficientCash):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrCashSessionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Current godoc
// @Summary      Caixa aberto do operador
// @Description  Devolve o caixa aberto do usuário logado com o resumo do turno, ou session nulo quando o caixa está fechado
// @Tags         cash-register
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.CashSessionDetail
// @Router       /cash-register/current [get]
func (h *Handler) Current(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	detail, err := h.service.Current(c.Request.Context(), companyId, userId)
	if err != nil {
		writeError(c, err)
		return
	}
	if detail == nil {
		c.JSON(http.StatusOK, gin.H{"session": nil})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Open godoc
// @Summary      Abrir caixa
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.OpenCashSessionRequest true "Valor inicial (troco)"
// @Success      201 {object} domain.CashSessionDetail
// @Failure      409 {object} map[string]string "Caixa já aberto"
// @Router       /cash-register/open [post]
func (h *Handler) Open(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	var req domain.OpenCashSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.Open(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// AddMovement godoc
// @Summary      Sangria ou suprimento
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.CreateMovementRequest true "Movimento"
// @Success      201 {object} domain.CashSessionDetail
// @Failure      409 {object} map[string]string "Nenhum caixa aberto"
// @Failure      422 {object} map[string]string "Sangria maior que o dinheiro em caixa"
// @Router       /cash-register/movements [post]
func (h *Handler) AddMovement(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	var req domain.CreateMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.AddMovement(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// Close godoc
// @Summary      Fechar caixa
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.CloseCashSessionRequest true "Dinheiro contado na gaveta"
// @Success      200 {object} domain.CashSessionDetail
// @Failure      409 {object} map[string]string "Nenhum caixa aberto"
// @Router       /cash-register/close [post]
func (h *Handler) Close(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	var req domain.CloseCashSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.Close(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}

// List godoc
// @Summary      Histórico de caixas
// @Description  O administrador vê os caixas de todos os operadores; os demais, só os próprios
// @Tags         cash-register
// @Produce      json
// @Security     BearerAuth
// @Param        query query domain.ListCashSessionsParams false "Paginação"
// @Success      200 {object} domain.ListCashSessionsResponse
// @Router       /cash-register/sessions [get]
func (h *Handler) List(c *gin.Context) {
	companyId, userId, isAdmin, ok := identity(c)
	if !ok {
		return
	}

	var params domain.ListCashSessionsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.List(c.Request.Context(), companyId, userId, isAdmin, params)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetById godoc
// @Summary      Detalhe de um caixa
// @Tags         cash-register
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do caixa"
// @Success      200 {object} domain.CashSessionDetail
// @Failure      404 {object} map[string]string "Caixa não encontrado"
// @Router       /cash-register/sessions/{id} [get]
func (h *Handler) GetById(c *gin.Context) {
	companyId, userId, isAdmin, ok := identity(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	detail, err := h.service.GetById(c.Request.Context(), companyId, userId, isAdmin, id)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}
