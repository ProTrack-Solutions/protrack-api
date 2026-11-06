package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func respondDocumentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrSaleNotFound), errors.Is(err, domain.ErrDocumentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrSaleCanceled),
		errors.Is(err, domain.ErrDocumentNotPrintable),
		errors.Is(err, domain.ErrDocumentNotCancellable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidCancelReason):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.As(err, new(*service.ManualProblemsError)):
		var pe *service.ManualProblemsError
		errors.As(err, &pe)
		c.JSON(http.StatusBadRequest, gin.H{"error": "corrija a nota antes de emitir", "problems": pe.Problems})
	default:
		respondError(c, err)
	}
}

// ListDocuments godoc
// @Summary      Notas fiscais (NFC-e e NF-e) da empresa
// @Description  Última tentativa de cada venda e as notas manuais. problems=true traz só rejeitadas, com falha ou pendentes.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Situação (AUTHORIZED, REJECTED, FAILED, PENDING, CANCELLED...)"
// @Param        model query int false "55 = NF-e, 65 = NFC-e"
// @Param        source query string false "sale (vendas) ou manual"
// @Param        problems query bool false "Só as que precisam de atenção"
// @Param        page query int false "Página"
// @Param        per_page query int false "Itens por página (máx. 100)"
// @Success      200 {object} domain.FiscalDocumentList
// @Router       /fiscal/documents [get]
func (h *Handler) ListDocuments(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	problems, _ := strconv.ParseBool(c.Query("problems"))
	model, _ := strconv.Atoi(c.Query("model"))
	out, err := h.service.ListDocuments(c.Request.Context(), companyId, domain.DocumentFilter{
		Status: c.Query("status"), Model: model, Source: c.Query("source"), ProblemsOnly: problems,
	}, page, perPage)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// maxSalesPerLookup: uma página da lista de vendas
const maxSalesPerLookup = 100

// SalesDocuments godoc
// @Summary      Última NFC-e de várias vendas
// @Description  Para a lista de vendas: mapa venda -> nota (vendas sem nota ficam de fora).
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Param        ids query string true "IDs das vendas separados por vírgula (até 100)"
// @Success      200 {object} map[string]domain.FiscalDocument
// @Router       /fiscal/sales [get]
func (h *Handler) SalesDocuments(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	var ids []uuid.UUID
	for _, raw := range strings.Split(c.Query("ids"), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id de venda inválido"})
			return
		}
		ids = append(ids, id)
	}
	if len(ids) > maxSalesPerLookup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no máximo 100 vendas por consulta"})
		return
	}
	out, err := h.service.SalesDocuments(c.Request.Context(), companyId, ids)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"documents": out})
}

// GetSaleDocument godoc
// @Summary      NFC-e de uma venda
// @Description  Última tentativa de nota da venda; document null quando a venda não tem nota.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Param        saleId path string true "ID da venda"
// @Success      200 {object} domain.FiscalDocument
// @Router       /fiscal/sales/{saleId} [get]
func (h *Handler) GetSaleDocument(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	saleId, ok := uuidParam(c, "saleId")
	if !ok {
		return
	}
	out, err := h.service.GetSaleDocument(c.Request.Context(), companyId, saleId)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": out})
}

// EmitSaleDocument godoc
// @Summary      Emite (ou reemite) a NFC-e de uma venda
// @Description  Se a venda já tem nota autorizada/em andamento, devolve essa. Rejeitada ou com falha: nova tentativa com os dados atuais dos produtos.
// @Tags         fiscal
// @Produce      json
// @Security     BearerAuth
// @Param        saleId path string true "ID da venda"
// @Success      200 {object} domain.FiscalDocument
// @Router       /fiscal/sales/{saleId}/nfce [post]
func (h *Handler) EmitSaleDocument(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	saleId, ok := uuidParam(c, "saleId")
	if !ok {
		return
	}
	out, err := h.service.EmitForSale(c.Request.Context(), companyId, userId, saleId, false)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": out})
}

// Danfce godoc
// @Summary      PDF da nota: DANFC-e 80 mm (NFC-e) ou DANFE A4 (NF-e)
// @Tags         fiscal
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        id path string true "ID da nota"
// @Success      200 {file} binary
// @Router       /fiscal/documents/{id}/danfce [get]
func (h *Handler) Danfce(c *gin.Context) { h.printout(c, false) }

// Xml godoc
// @Summary      XML da nota
// @Tags         fiscal
// @Produce      application/xml
// @Security     BearerAuth
// @Param        id path string true "ID da nota"
// @Success      200 {file} binary
// @Router       /fiscal/documents/{id}/xml [get]
func (h *Handler) Xml(c *gin.Context) { h.printout(c, true) }

func (h *Handler) printout(c *gin.Context, xml bool) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	data, filename, err := h.service.Printout(c.Request.Context(), companyId, id, xml)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	ctype, disposition := "application/pdf", "inline"
	if xml {
		ctype, disposition = "application/xml; charset=utf-8", "attachment"
	}
	c.Header("Content-Disposition", disposition+`; filename="`+filename+`"`)
	c.Data(http.StatusOK, ctype, data)
}

// CancelDocument godoc
// @Summary      Cancela a NFC-e na SEFAZ
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID da nota"
// @Param        body body domain.CancelInput true "Justificativa (15 a 255 caracteres)"
// @Success      200 {object} domain.FiscalDocument
// @Router       /fiscal/documents/{id}/cancel [post]
func (h *Handler) CancelDocument(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var in domain.CancelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.CancelDocument(c.Request.Context(), companyId, id, in.Reason)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// Webhook godoc
// @Summary      Webhook da protrack-fiscal-api (situação das NFC-e)
// @Description  Público; autenticado pela assinatura X-ProTrack-Signature (HMAC-SHA256).
// @Tags         fiscal
// @Accept       json
// @Success      200
// @Router       /webhooks/fiscal [post]
func (h *Handler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo inválido"})
		return
	}
	err = h.service.HandleWebhook(c.Request.Context(), c.GetHeader("X-ProTrack-Signature"), body, h.webhookSecrets...)
	switch {
	case errors.Is(err, service.ErrInvalidSignature):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case err != nil:
		// 500: a API fiscal tenta de novo mais tarde
		log.Error().Err(err).Str("event", c.GetHeader("X-ProTrack-Event")).Msg("webhook fiscal")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao processar o evento"})
	default:
		c.Status(http.StatusOK)
	}
}

// PreviewManualDocument godoc
// @Summary      Confere a NF-e/NFC-e manual antes de emitir
// @Description  Devolve os problemas que impedem a emissão, o CFOP de cada item e os totais (na NF-e, calculados pela API fiscal).
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.ManualDocumentInput true "Nota manual"
// @Success      200 {object} service.ManualPreview
// @Router       /fiscal/documents/manual/preview [post]
func (h *Handler) PreviewManualDocument(c *gin.Context) {
	companyId, _, ok := identity(c)
	if !ok {
		return
	}
	var in domain.ManualDocumentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo inválido: " + err.Error()})
		return
	}
	out, err := h.service.PreviewManual(c.Request.Context(), companyId, in)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// EmitManualDocument godoc
// @Summary      Emite NF-e ou NFC-e manual (avulsa)
// @Description  Nota sem venda e sem baixa de estoque. idempotency_key (UUID gerado pela tela) evita nota duplicada. Problemas no pedido voltam 400 com a lista em "problems".
// @Tags         fiscal
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.ManualDocumentInput true "Nota manual"
// @Success      200 {object} domain.FiscalDocument
// @Router       /fiscal/documents/manual [post]
func (h *Handler) EmitManualDocument(c *gin.Context) {
	companyId, userId, ok := identity(c)
	if !ok {
		return
	}
	var in domain.ManualDocumentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo inválido: " + err.Error()})
		return
	}
	out, err := h.service.EmitManual(c.Request.Context(), companyId, userId, in)
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": out})
}
