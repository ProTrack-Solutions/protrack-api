package handler

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/service"
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

// CreateInstance godoc
// @Summary      Cria instância WhatsApp e retorna QR Code
// @Tags         whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      201 {object} map[string]interface{}
// @Router       /whatsapp/instance/create [post]
func (h *Handler) CreateInstance(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	resp, err := h.service.CreateInstance(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"qr_code": resp})
}

// ConnectionState godoc
// @Summary      Obtém o estado de conexão do WhatsApp
// @Description  Consulta o status da instância na Evolution API (ex: open, connecting, close) e retorna os dados da conexão.
// @Tags         whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{} 					"Estado atual do WhatsApp obtido com sucesso"
// @Failure      401  {object}  map[string]interface{}                  "Empresa não autenticada no contexto"
// @Failure      500  {object}  map[string]interface{}                  "Falha na comunicação com o serviço ou Evolution API"
// @Router       /whatsapp/instance/connection-state [get]
func (h *Handler) ConnectonState(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	state, err := h.service.ConnectonState(c.Request.Context(), companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// DeleteInstance godoc
// @Summary      Deleta a instância do WhatsApp
// @Description  Remove/desconecta a instância do WhatsApp associada à empresa do usuário autenticado.
// @Tags         whatsapp
// @Produce      json
// @Security     BearerAuth
// @Success      204      "Instância deletada com sucesso"
// @Failure      401      {object}  map[string]interface{}  "Empresa não autenticada no contexto"
// @Failure      500      {object}  map[string]interface{}  "Falha ao deletar a instância no serviço"
// @Router       /whatsapp/instance/delete [delete]
func (h *Handler) Deleteinstance(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company id is null"})
		return
	}

	companyId := companyIdAny.(uuid.UUID)

	if err := h.service.DeleteInstance(c.Request.Context(), companyId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
