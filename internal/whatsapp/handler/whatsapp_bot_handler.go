package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// CreateWhatsAppBot godoc
// @Summary      Cria a configuração do bot do WhatsApp
// @Description  Cria a configuração do bot (mensagem de boas-vindas e opções de menu) para a empresa autenticada.
// @Tags         whatsapp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      domain.CreateWhatsAppBotConfigRequest  true  "Dados de criação"
// @Success      201  {object}  map[string]interface{}  "Configuração criada com sucesso"
// @Failure      400  {object}  map[string]interface{}  "Payload inválido"
// @Failure      401  {object}  map[string]interface{}  "Empresa não autenticada no contexto"
// @Failure      500  {object}  map[string]interface{}  "Falha ao criar a configuração do bot"
// @Router       /whatsapp/bot [post]
func (h *Handler) CreateWhatsAppBot(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	var req domain.CreateWhatsAppBotConfigRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateWhatsAppBot(c.Request.Context(), companyId, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "created"})
}

// GetWhatsAppBot godoc
// @Summary      Obtém a configuração do bot do WhatsApp
// @Description  Retorna a configuração do bot (mensagem de boas-vindas, status e opções de menu) da empresa autenticada.
// @Tags         whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  domain.WhatsappBotConfig
// @Failure      401  {object}  map[string]interface{}  "Empresa não autenticada no contexto"
// @Failure      404  {object}  map[string]interface{}  "Configuração do bot não encontrada"
// @Failure      500  {object}  map[string]interface{}  "Falha ao obter a configuração do bot"
// @Router       /whatsapp/bot [get]
func (h *Handler) GetWhatsAppBot(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	config, err := h.service.GetWhatsAppBotConfigWithOptionsByCompanyID(c.Request.Context(), companyId)
	if err != nil {
		if errors.Is(err, domain.ErrWhatsAppBotConfigNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateWhatsAppBot godoc
// @Summary      Atualiza a configuração do bot do WhatsApp
// @Description  Atualiza a mensagem de boas-vindas, status e opções de menu da configuração do bot da empresa autenticada.
// @Tags         whatsapp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                true  "ID da configuração do bot"
// @Param        body  body      domain.UpdateWhatsAppBotConfigRequest  true  "Dados de atualização"
// @Success      200  {object}  map[string]interface{}  "Configuração atualizada com sucesso"
// @Failure      400  {object}  map[string]interface{}  "Id inválido ou payload inválido"
// @Failure      401  {object}  map[string]interface{}  "Empresa não autenticada no contexto"
// @Failure      404  {object}  map[string]interface{}  "Configuração do bot não encontrada"
// @Failure      500  {object}  map[string]interface{}  "Falha ao atualizar a configuração do bot"
// @Router       /whatsapp/bot/{id} [put]
func (h *Handler) UpdateWhatsAppBot(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req domain.UpdateWhatsAppBotConfigRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateWhatsAppBotConfig(c.Request.Context(), id, companyId, req); err != nil {
		if errors.Is(err, domain.ErrWhatsAppBotConfigNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// HandleWebhook godoc
// @Summary      Recebe eventos da Evolution API (webhook do bot do WhatsApp)
// @Description  Endpoint público chamado pela Evolution API a cada mensagem recebida na instância da empresa. Resolve a resposta do bot e a envia de volta.
// @Tags         whatsapp
// @Accept       json
// @Produce      json
// @Param        companyId  path  string  true  "ID da empresa"
// @Success      200  {object}  map[string]interface{}
// @Router       /whatsapp/webhook/{companyId} [post]
func (h *Handler) HandleWebhook(c *gin.Context) {
	companyId, err := uuid.Parse(c.Param("companyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid companyId"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var payload domain.EvolutionWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if !strings.EqualFold(payload.Event, "messages.upsert") ||
		payload.Data.Key.FromMe ||
		strings.HasSuffix(payload.Data.Key.RemoteJid, "@g.us") ||
		payload.Data.MessageType != "conversation" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// A partir da migração da Meta para LID (Linked ID), contatos com a privacidade
	// de número ativada chegam com remoteJid = "<id>@lid" em vez de um número de
	// telefone real. Enviar esse "número" para a Evolution API sempre falha com
	// 400 (exists:false), pois não é um MSISDN válido. Nesses casos a Evolution
	// informa o JID baseado em telefone no campo raiz "sender" do payload
	// (confirmado em produção em 2026-08-13), então usamos ele como destino.
	targetJid := payload.Data.Key.RemoteJid
	if strings.HasSuffix(targetJid, "@lid") {
		if payload.Sender != "" && !strings.HasSuffix(payload.Sender, "@lid") {
			targetJid = payload.Sender
		} else {
			log.Warn().
				Str("company_id", companyId.String()).
				Str("remote_jid", payload.Data.Key.RemoteJid).
				Str("sender", payload.Sender).
				RawJSON("raw_payload", rawBody).
				Msg("mensagem recebida de contato LID sem número de telefone resolvível")
			c.JSON(http.StatusOK, gin.H{"status": "ignored: lid contact without resolvable phone number"})
			return
		}
	}

	ctx := c.Request.Context()

	config, err := h.service.GetWhatsAppBotConfigWithOptionsByCompanyID(ctx, companyId)
	if err != nil {
		if errors.Is(err, domain.ErrWhatsAppBotConfigNotFound) {
			c.JSON(http.StatusOK, gin.H{"status": "bot not configured"})
			return
		}
		log.Error().Err(err).Str("company_id", companyId.String()).Msg("failed to load whatsapp bot config")
		c.JSON(http.StatusOK, gin.H{"status": "error"})
		return
	}

	if !config.IsActive {
		c.JSON(http.StatusOK, gin.H{"status": "bot inactive"})
		return
	}

	responseText := ResolveIncomingMessage(config, payload.Data.Message.Conversation)

	if err := h.service.SendBotReply(ctx, config.InstanceName, targetJid, responseText); err != nil {
		log.Error().Err(err).Str("company_id", companyId.String()).Msg("failed to send whatsapp bot response")
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}
