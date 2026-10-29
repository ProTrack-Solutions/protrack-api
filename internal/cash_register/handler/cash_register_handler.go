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

// RegisterContext lê o header CashRegisterId (caixa em que o computador
// trabalha) e guarda no contexto da requisição. Vale para todas as rotas: venda,
// recebimento e devolução usam para achar o turno aberto. Header inválido é
// ignorado, como se não tivesse vindo.
func RegisterContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		if id, err := uuid.Parse(c.GetHeader(domain.RegisterHeader)); err == nil {
			c.Request = c.Request.WithContext(domain.WithRegisterID(c.Request.Context(), id))
		}
		c.Next()
	}
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCashSessionAlreadyOpen), errors.Is(err, domain.ErrNoOpenCashSession),
		errors.Is(err, domain.ErrCashRegisterNotSelected), errors.Is(err, domain.ErrCashRegisterInUse),
		errors.Is(err, domain.ErrTransferAlreadyResolved), errors.Is(err, domain.ErrTransferTargetClosed),
		errors.Is(err, domain.ErrPendingTransfers):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInsufficientCash), errors.Is(err, domain.ErrTransferSameRegister):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrTransferWrongRegister):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrCashSessionNotFound), errors.Is(err, domain.ErrCashRegisterNotFound),
		errors.Is(err, domain.ErrTransferNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Current godoc
// @Summary      Turno aberto do caixa
// @Description  Devolve o turno aberto do caixa do computador (header CashRegisterId; sem header, o único caixa ativo) com o resumo, ou session nulo quando o caixa está fechado
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
// @Description  O administrador vê todos os turnos; os demais, os turnos de que participaram (abriram, fecharam, venderam ou receberam)
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

// ListRegisters godoc
// @Summary      Caixas da empresa
// @Description  Caixas cadastrados com o turno aberto de cada um
// @Tags         cash-register
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.CashRegister
// @Router       /cash-register/registers [get]
func (h *Handler) ListRegisters(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}

	registers, err := h.service.ListRegisters(c.Request.Context(), companyId)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, registers)
}

// CreateRegister godoc
// @Summary      Cadastrar caixa
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.SaveCashRegisterRequest true "Caixa"
// @Success      201 {object} domain.CashRegister
// @Router       /cash-register/registers [post]
func (h *Handler) CreateRegister(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}

	var req domain.SaveCashRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	register, err := h.service.CreateRegister(c.Request.Context(), companyId, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, register)
}

// UpdateRegister godoc
// @Summary      Renomear, ativar ou desativar caixa
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do caixa"
// @Param        body body domain.SaveCashRegisterRequest true "Caixa"
// @Success      200 {object} domain.CashRegister
// @Failure      409 {object} map[string]string "Caixa com turno aberto"
// @Router       /cash-register/registers/{id} [put]
func (h *Handler) UpdateRegister(c *gin.Context) {
	companyId, _, _, ok := identity(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req domain.SaveCashRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	register, err := h.service.UpdateRegister(c.Request.Context(), companyId, id, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, register)
}

// CreateTransfer godoc
// @Summary      Transferir troco para outro caixa
// @Description  Tira o dinheiro da gaveta do caixa deste computador agora; o caixa de destino precisa aceitar para receber
// @Tags         cash-register
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.CreateTransferRequest true "Transferência"
// @Success      201 {object} domain.CashSessionDetail
// @Failure      409 {object} map[string]string "Caixa de destino fechado"
// @Failure      422 {object} map[string]string "Valor maior que o dinheiro em caixa"
// @Router       /cash-register/transfers [post]
func (h *Handler) CreateTransfer(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	var req domain.CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := h.service.CreateTransfer(c.Request.Context(), companyId, userId, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// ResolveTransfer godoc
// @Summary      Aceitar, recusar ou cancelar transferência de troco
// @Description  accept e reject no computador do caixa de destino; cancel no do caixa de origem. Recusada ou cancelada, o dinheiro volta para a origem
// @Tags         cash-register
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da transferência"
// @Param        action path string true "accept, reject ou cancel"
// @Success      200 {object} domain.CashSessionDetail
// @Failure      403 {object} map[string]string "Transferência de outro caixa"
// @Failure      409 {object} map[string]string "Já resolvida"
// @Router       /cash-register/transfers/{id}/{action} [post]
func (h *Handler) ResolveTransfer(c *gin.Context) {
	companyId, userId, _, ok := identity(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var detail domain.CashSessionDetail
	switch c.Param("action") {
	case "accept":
		detail, err = h.service.AcceptTransfer(c.Request.Context(), companyId, userId, id)
	case "reject":
		detail, err = h.service.RejectTransfer(c.Request.Context(), companyId, userId, id)
	case "cancel":
		detail, err = h.service.CancelTransfer(c.Request.Context(), companyId, userId, id)
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "ação inválida"})
		return
	}
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}
