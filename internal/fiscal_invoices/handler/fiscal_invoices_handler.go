package handler

import (
	"crypto/subtle"
	"io"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/service"
	nfeemissionDomain "github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service    *service.Service
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
	cfg        *config.Config
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist, cfg *config.Config) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
		cfg:        cfg,
	}
}

// helper interno pra extrair company_id do contexto autenticado, mesmo
// padrão repetido em todos os handlers do repo (sales, products, etc.)
func getCompanyID(c *gin.Context) (uuid.UUID, bool) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
		return uuid.Nil, false
	}
	return companyIdAny.(uuid.UUID), true
}

// EmitFiscalInvoice godoc
// @Summary      Emite um documento fiscal (NF-e ou NFC-e) para uma venda
// @Tags         fiscal-invoices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.EmitFiscalInvoiceRequest true "Emissão"
// @Success      201 {object} domain.FiscalInvoiceResponse
// @Router       /fiscal-invoices [post]
func (h *Handler) EmitFiscalInvoice(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	var req domain.EmitFiscalInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.EmitFiscalInvoice(c.Request.Context(), req, companyId)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetFiscalInvoiceByID godoc
// @Summary      Consulta um documento fiscal por ID
// @Tags         fiscal-invoices
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do documento fiscal"
// @Success      200 {object} domain.FiscalInvoiceResponse
// @Router       /fiscal-invoices/{id} [get]
func (h *Handler) GetFiscalInvoiceByID(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	resp, err := h.service.GetFiscalInvoiceByID(c.Request.Context(), id, companyId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListFiscalInvoicesBySale godoc
// @Summary      Lista documentos fiscais de uma venda
// @Tags         fiscal-invoices
// @Produce      json
// @Security     BearerAuth
// @Param        sale_id query string true "ID da venda"
// @Success      200 {array} domain.FiscalInvoiceResponse
// @Router       /fiscal-invoices [get]
func (h *Handler) ListFiscalInvoicesBySale(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	saleId, err := uuid.Parse(c.Query("sale_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sale_id inválido"})
		return
	}

	resp, err := h.service.ListFiscalInvoicesBySale(c.Request.Context(), saleId, companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CancelFiscalInvoice godoc
// @Summary      Cancela um documento fiscal autorizado
// @Tags         fiscal-invoices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do documento fiscal"
// @Param        request body domain.CancelFiscalInvoiceRequest true "Cancelamento"
// @Success      200 {object} domain.FiscalInvoiceResponse
// @Router       /fiscal-invoices/{id}/cancel [post]
func (h *Handler) CancelFiscalInvoice(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req domain.CancelFiscalInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	req.CompanyID = companyId

	resp, err := h.service.CancelFiscalInvoice(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UploadCertificate godoc
// @Summary      Envia o certificado digital A1 da empresa
// @Tags         fiscal-invoices
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Certificado .pfx/.p12"
// @Param        password formData string true "Senha do certificado"
// @Success      200 {object} map[string]string
// @Router       /company-certificates [post]
// maxCertificateFileBytes limita o upload do certificado A1 (.pfx/.p12).
// Certificados A1 reais têm poucos KB — 2 MiB já é uma folga generosa e
// evita que um upload gigante segure memória/handler à toa.
const maxCertificateFileBytes = 2 << 20 // 2 MiB

func (h *Handler) UploadCertificate(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCertificateFileBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo do certificado é obrigatório"})
		return
	}
	if fileHeader.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo do certificado está vazio"})
		return
	}
	if fileHeader.Size > maxCertificateFileBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "arquivo do certificado excede o tamanho máximo permitido (2 MiB)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "falha ao abrir arquivo"})
		return
	}
	defer file.Close()

	certBytes, err := io.ReadAll(io.LimitReader(file, maxCertificateFileBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "falha ao ler arquivo"})
		return
	}

	password := c.PostForm("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "senha do certificado é obrigatória"})
		return
	}

	req := domain.UploadCertificateRequest{
		CompanyID: companyId,
		CertFile:  certBytes,
		Password:  password,
	}

	if err := h.service.UploadCertificate(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "certificado enviado com sucesso"})
}

// DeleteCompanyCertificate godoc
// @Summary      Revoga o certificado digital A1 da empresa
// @Tags         fiscal-invoices
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Router       /company-certificates [delete]
func (h *Handler) DeleteCompanyCertificate(c *gin.Context) {
	companyId, ok := getCompanyID(c)
	if !ok {
		return
	}

	userIdAny, exists := c.Get("sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
		return
	}
	userID, err := uuid.Parse(userIdAny.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteCompanyCertificate(c.Request.Context(), companyId, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "certificado revogado com sucesso"})
}

// HandlePlugNotasWebhook godoc
// @Summary      Recebe notificações assíncronas do PlugNotas
// @Tags         fiscal-invoices
// @Accept       json
// @Success      200
// @Router       /webhooks/plugnotas [post]
func (h *Handler) HandlePlugNotasWebhook(c *gin.Context) {
	const MaxBodyBytes = 65536
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	// Validação do segredo compartilhado configurado em
	// POST /empresa/{cnpj}/webhook (header customizado, não HMAC — ver
	// internal/plugnotas/empresa.go ConfigurarWebhook).
	authHeader := c.GetHeader("Authorization")
	if h.cfg.FiscalWebhookSecret == "" || authHeader == "" ||
		subtle.ConstantTimeCompare([]byte(authHeader), []byte(h.cfg.FiscalWebhookSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "segredo do webhook inválido"})
		return
	}

	var notification nfeemissionDomain.WebhookNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}

	if err := h.service.HandlePlugNotasWebhook(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
