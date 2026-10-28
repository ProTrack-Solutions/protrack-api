package handler

import (
	"errors"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/receivable_reminders/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/receivable_reminders/service"
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
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
		queries:    queries,
	}
}

// Mesmas regras de acesso do contas a receber: ADMIN com o módulo financeiro
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	reminders := r.Group("/accounts-receivable/reminders")
	reminders.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	reminders.Use(middleware.RequireRole("ADMIN"))
	reminders.Use(middleware.RequireModule("financial", h.queries))
	{
		reminders.POST("/:id", h.Send)
	}
}

// Send godoc
// @Summary      Envia lembrete de cobrança pelo WhatsApp
// @Description  Parcela vencida usa o template de venda vencida; parcela a vencer usa o template de lembrete configurado na empresa. Respeita WhatsApp ativo, plano e limite de mensagens, e permite um lembrete por parcela a cada 24 horas.
// @Tags         accounts-receivable
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da parcela (conta a receber)"
// @Success      201 {object} domain.ReminderResponse
// @Failure      404 {object} map[string]string "Parcela não encontrada"
// @Failure      409 {object} map[string]string "Parcela quitada ou cancelada"
// @Failure      422 {object} map[string]string "Sem template, cliente sem WhatsApp ou WhatsApp indisponível no plano"
// @Failure      429 {object} map[string]string "Lembrete enviado há menos de 24 horas"
// @Failure      502 {object} map[string]string "Falha no envio pela Meta"
// @Router       /accounts-receivable/reminders/{id} [post]
func (h *Handler) Send(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sub is null"})
		return
	}
	userId, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	accountId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	resp, err := h.service.Send(c.Request.Context(), companyIdAny.(uuid.UUID), userId, accountId)
	if err != nil {
		var tooSoon *domain.TooSoonError
		switch {
		case errors.As(err, &tooSoon):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error(), "next_reminder_at": tooSoon.NextAt})
		case errors.Is(err, domain.ErrReceivableNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrAlreadySettled):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrTemplateNotConfigured), errors.Is(err, domain.ErrCustomerWithoutWhatsApp),
			errors.Is(err, domain.ErrWhatsAppUnavailable):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrSendFailed):
			// O detalhe técnico da Meta fica no log; o usuário recebe a mensagem amigável
			c.JSON(http.StatusBadGateway, gin.H{"error": domain.ErrSendFailed.Error() + ". Confira o número do cliente e tente novamente."})
		default:
			log.Error().Err(err).Str("account_id", accountId.String()).Msg("erro ao enviar lembrete")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno ao enviar o lembrete"})
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}
