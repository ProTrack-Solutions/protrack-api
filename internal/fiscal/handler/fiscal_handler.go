package handler

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Tamanho máximo do .pfx aceito no upload (um A1 tem poucos KB).
const maxCertificateBytes = 64 << 10

type Handler struct {
	service        *service.Service
	jwtManager     *jwt.JWTManager
	blacklist      *cache.TokenBlacklist
	queries        departmentModulesDomain.ServiceInterface
	webhookSecrets []string
}

// NewHandler: webhookSecrets são os segredos whsec_ aceitos (o atual e, durante a
// troca, o anterior); vazio recusa todo webhook.
func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist, queries departmentModulesDomain.ServiceInterface, webhookSecrets ...string) *Handler {
	var secrets []string
	for _, s := range webhookSecrets {
		if s != "" {
			secrets = append(secrets, s)
		}
	}
	return &Handler{service: service, jwtManager: jwtManager, blacklist: blacklist, queries: queries, webhookSecrets: secrets}
}

func identity(c *gin.Context) (companyId, userId uuid.UUID, ok bool) {
	companyAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
		return uuid.Nil, uuid.Nil, false
	}
	userId, err := uuid.Parse(c.GetString("sub"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return uuid.Nil, uuid.Nil, false
	}
	return companyAny.(uuid.UUID), userId, true
}

func respondError(c *gin.Context, err error) {
	var apiErr *service.APIError
	switch {
	case errors.As(err, &apiErr):
		status := http.StatusUnprocessableEntity
		if apiErr.Status >= 500 || apiErr.Status == 0 {
			status = http.StatusBadGateway
		} else if apiErr.Status == http.StatusConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": apiErr.Message})
	case errors.Is(err, domain.ErrInvalidTaxProfile),
		errors.Is(err, domain.ErrInvalidFiscalData),
		errors.Is(err, domain.ErrNotReadyToEnable):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrTaxProfileNotFound),
		errors.Is(err, domain.ErrCategoryNotFound),
		errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrCompanyNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrTaxProfileNameExists),
		errors.Is(err, domain.ErrIssuerAlreadyExists),
		errors.Is(err, domain.ErrAlreadyLive):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrIssuerNotRegistered):
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrFiscalNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno"})
	}
}

func uuidParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return uuid.Nil, false
	}
	return id, true
}

// ---------- Configuração ----------

// GetSettings godoc
// @Summary      Configuração fiscal da empresa
// @Description  Situação da NFC-e: ambiente, emitente, A1, regime, perfil padrão e produtos sem NCM.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.FiscalSettings
// @Router       /fiscal/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.GetSettings(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// UpdateSettings godoc
// @Summary      Liga/desliga a NFC-e e define o perfil tributário padrão
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        settings body domain.UpdateSettingsInput true "Configuração"
// @Success      200 {object} domain.FiscalSettings
// @Router       /fiscal/settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var in domain.UpdateSettingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.UpdateSettings(c.Request.Context(), companyId, userId, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// RegisterIssuer godoc
// @Summary      Cadastra a empresa como emitente de NFC-e
// @Description  Usa o CNPJ e a razão social do cadastro da empresa; cria o emitente em homologação e produção.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      201 {object} domain.FiscalSettings
// @Router       /fiscal/issuer [post]
func (h *Handler) RegisterIssuer(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.RegisterIssuer(c.Request.Context(), companyId, userId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

// UploadCertificate godoc
// @Summary      Envia o certificado A1
// @Description  O arquivo vai direto para o serviço fiscal; o ERP não guarda o certificado nem a senha.
// @Tags         fiscal
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Certificado .pfx/.p12"
// @Param        password formData string true "Senha do certificado"
// @Success      201 {object} domain.CertificateInfo
// @Router       /fiscal/certificate [post]
func (h *Handler) UploadCertificate(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCertificateBytes+16<<10)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "envie o arquivo do certificado (.pfx ou .p12)"})
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pfx" && ext != ".p12" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "o certificado A1 deve ser um arquivo .pfx ou .p12"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxCertificateBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxCertificateBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo de certificado inválido"})
		return
	}
	password := c.Request.FormValue("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "informe a senha do certificado"})
		return
	}
	out, err := h.service.UploadCertificate(c.Request.Context(), companyId, userId, filepath.Base(header.Filename), data, password)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

// GetCertificate godoc
// @Summary      Situação do certificado A1
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.CertificateInfo
// @Router       /fiscal/certificate [get]
func (h *Handler) GetCertificate(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.GetCertificate(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetFiscalProfile godoc
// @Summary      Perfil fiscal do emitente (IE, regime, endereço, CSC)
// @Description  O código do CSC nunca volta; csc_configured diz se já foi informado. null quando ainda não foi salvo.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.FiscalProfile
// @Router       /fiscal/profile [get]
func (h *Handler) GetFiscalProfile(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.GetFiscalProfile(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// SaveFiscalProfile godoc
// @Summary      Salva o perfil fiscal do emitente
// @Description  CSC em branco mantém o atual.
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        profile body domain.FiscalProfileInput true "Perfil fiscal"
// @Success      200 {object} domain.FiscalProfile
// @Router       /fiscal/profile [put]
func (h *Handler) SaveFiscalProfile(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var in domain.FiscalProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.SaveFiscalProfile(c.Request.Context(), companyId, userId, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// SefazStatus godoc
// @Summary      Testa a conexão com a SEFAZ usando o A1 da empresa
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.SefazStatus
// @Router       /fiscal/sefaz-status [get]
func (h *Handler) SefazStatus(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.SefazStatus(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// GoLive godoc
// @Summary      Passa a emitir NFC-e em produção
// @Description  Copia A1 e perfil fiscal de homologação para produção, com o CSC de produção.
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        csc body domain.GoLiveInput true "CSC de produção"
// @Success      200 {object} domain.FiscalSettings
// @Router       /fiscal/go-live [post]
func (h *Handler) GoLive(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var in domain.GoLiveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.GoLive(c.Request.Context(), companyId, userId, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// ---------- Perfis tributários ----------

// ListTaxProfiles godoc
// @Summary      Perfis tributários da empresa
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.TaxProfile
// @Router       /fiscal/tax-profiles [get]
func (h *Handler) ListTaxProfiles(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.ListTaxProfiles(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tax_profiles": out})
}

// CreateTaxProfile godoc
// @Summary      Cria um perfil tributário
// @Description  O primeiro perfil da empresa vira o padrão.
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        profile body domain.TaxProfileInput true "Perfil"
// @Success      201 {object} domain.TaxProfile
// @Router       /fiscal/tax-profiles [post]
func (h *Handler) CreateTaxProfile(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var in domain.TaxProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.CreateTaxProfile(c.Request.Context(), companyId, userId, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

// UpdateTaxProfile godoc
// @Summary      Edita um perfil tributário
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do perfil"
// @Param        profile body domain.TaxProfileInput true "Perfil"
// @Success      200 {object} domain.TaxProfile
// @Router       /fiscal/tax-profiles/{id} [put]
func (h *Handler) UpdateTaxProfile(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var in domain.TaxProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.UpdateTaxProfile(c.Request.Context(), companyId, userId, id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// DeleteTaxProfile godoc
// @Summary      Exclui um perfil tributário
// @Description  Categorias e produtos que usavam o perfil voltam a herdar o da categoria/empresa.
// @Tags         fiscal
// @Security     BearerAuth
// @Param        id path string true "ID do perfil"
// @Success      204
// @Router       /fiscal/tax-profiles/{id} [delete]
func (h *Handler) DeleteTaxProfile(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteTaxProfile(c.Request.Context(), companyId, userId, id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- Categorias e produtos ----------

// ListCategories godoc
// @Summary      Categorias com o perfil tributário de cada uma
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} domain.CategoryFiscal
// @Router       /fiscal/categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	out, err := h.service.ListCategories(c.Request.Context(), companyId)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": out})
}

// SetCategoryTaxProfile godoc
// @Summary      Define o perfil tributário de uma categoria
// @Description  tax_profile_id null: a categoria usa o perfil padrão da empresa.
// @Tags         fiscal
// @Accept       json
// @Security     BearerAuth
// @Param        id path string true "ID da categoria"
// @Param        body body domain.SetCategoryTaxProfileInput true "Perfil"
// @Success      204
// @Router       /fiscal/categories/{id}/tax-profile [put]
func (h *Handler) SetCategoryTaxProfile(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var in domain.SetCategoryTaxProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.SetCategoryTaxProfile(c.Request.Context(), companyId, id, in.TaxProfileID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetProductFiscal godoc
// @Summary      Dados fiscais de um produto
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID do produto"
// @Success      200 {object} domain.ProductFiscal
// @Router       /fiscal/products/{id} [get]
func (h *Handler) GetProductFiscal(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	out, err := h.service.GetProductFiscal(c.Request.Context(), companyId, id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// UpdateProductFiscal godoc
// @Summary      Grava NCM, CEST, unidade, origem e perfil de um produto
// @Tags         fiscal
// @Accept       json
// @Security     BearerAuth
// @Param        id path string true "ID do produto"
// @Param        body body domain.ProductFiscalInput true "Dados fiscais"
// @Success      200
// @Router       /fiscal/products/{id} [put]
func (h *Handler) UpdateProductFiscal(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var in domain.ProductFiscalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.service.UpdateProductFiscal(c.Request.Context(), companyId, userId, &id, "", in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": n})
}

// UpdateModelFiscal godoc
// @Summary      Grava os dados fiscais em todas as variações de um modelo de grade
// @Tags         fiscal
// @Accept       json
// @Security     BearerAuth
// @Param        name query string true "Nome do modelo"
// @Param        body body domain.ProductFiscalInput true "Dados fiscais"
// @Success      200
// @Router       /fiscal/models [put]
func (h *Handler) UpdateModelFiscal(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "informe o modelo em ?name="})
		return
	}
	var in domain.ProductFiscalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.service.UpdateProductFiscal(c.Request.Context(), companyId, userId, nil, name, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": n})
}
