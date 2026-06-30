package handler

import (
	"fmt"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports"
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
}

func NewHandler(service *service.Service, jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// GenerateReports godoc
// @Summary      Gera relatório em Excel
// @Tags         reports
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        type query string true "Tipo do relatório"
// @Param        start_date query string true "Data inicial"
// @Param        end_date query string true "Data final"
// @Param        format query string false "Formato (xlsx)"
// @Success      200 {file} file
// @Router       /reports [get]
func (h *Handler) GenerateReports(c *gin.Context) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
	}
	companyId := companyIdAny.(uuid.UUID)

	var req domain.ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.GenerateReports(c.Request.Context(), string(req.Type), companyId, req.StartDate, req.EndDate)
	if err != nil {
		log.Error().Err(err).Msg("erro ao gerar relatório")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("file", report.FileName).
		Int("headers_count", len(report.Headers)).
		Int("rows_count", len(report.Rows)).
		Msg("Relatório gerado com sucesso pelo service")

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", report.FileName))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	if req.Format == "xlsx" {
		err = reports.GenerateExcel(c.Writer, report.FileName, report.Headers, report.Rows)
		if err != nil {
			log.Error().Err(err).Msg("erro ao gerar relatório")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
}
