package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/generator"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	jwtManager *jwt.JWTManager
	blacklist  *cache.TokenBlacklist
	service    *service.Service
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

// GenerateReport godoc
// @Summary      Gera um relatório
// @Description  Monta o relatório do tipo pedido para o período (datas locais da empresa, inclusivas; sem datas usa o mês corrente). format=json devolve colunas, linhas, totais e indicadores para exibir na tela; xlsx e pdf devolvem o arquivo para download.
// @Description  Tipos: sales, sales_by_day, sales_by_payment_method, sales_by_seller, sales_by_customer, products_abc, profit_category, inventory, stock_movements, dre, payments_received, open_installments, bills_payable, cash_sessions
// @Tags         reports
// @Produce      json
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        query query domain.ReportRequest true "Parâmetros do relatório"
// @Success      200 {object} domain.Report
// @Failure      400 {object} map[string]string "Parâmetros inválidos"
// @Failure      401 {object} map[string]string "Não autorizado"
// @Failure      404 {object} map[string]string "Tipo de relatório inexistente"
// @Failure      500 {object} map[string]string "Erro interno"
// @Router       /reports [get]
func (h *Handler) GenerateReport(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	companyId := companyIdAny.(uuid.UUID)

	var req domain.ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	format := req.Format
	if format == "" {
		format = domain.FormatJSON
	}
	if format != domain.FormatJSON && format != domain.FormatXLSX && format != domain.FormatPDF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format deve ser json, xlsx ou pdf"})
		return
	}

	report, err := h.service.GenerateReport(c.Request.Context(), companyId, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidReportParams):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrReportNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Str("type", string(req.Type)).Msg("erro ao gerar relatório")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao gerar relatório"})
		}
		return
	}

	switch format {
	case domain.FormatXLSX:
		// Gera em memória antes de escrever: se falhar no meio, ainda dá para responder com erro
		var buf bytes.Buffer
		if err := generator.GenerateReportExcel(&buf, report); err != nil {
			log.Error().Err(err).Str("type", string(req.Type)).Msg("erro ao gerar Excel do relatório")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao gerar o arquivo Excel"})
			return
		}
		attachment(c, generator.ReportFileName(report, "xlsx"))
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
	case domain.FormatPDF:
		buf, err := generator.GenerateReportPDF(report)
		if err != nil {
			log.Error().Err(err).Str("type", string(req.Type)).Msg("erro ao gerar PDF do relatório")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao gerar o PDF"})
			return
		}
		attachment(c, generator.ReportFileName(report, "pdf"))
		c.Data(http.StatusOK, "application/pdf", buf.Bytes())
	default:
		c.JSON(http.StatusOK, report)
	}
}

func attachment(c *gin.Context, fileName string) {
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
}
