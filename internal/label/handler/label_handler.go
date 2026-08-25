package handler

import (
	"fmt"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/label/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/label/service"
	"github.com/gin-gonic/gin"
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

// DownloadLabels godoc
// @Summary      Gera PDF de etiquetas de produtos
// @Tags         label
// @Accept       application/json
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        request body []domain.GenetareTagProductRequest true "Lista de produtos para gerar etiquetas"
// @Success      200 {file} file
// @Failure      400 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /label/download [post]
func (h *Handler) DownloadLabels(c *gin.Context) {
	var req []domain.GenetareTagProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pdfBuffer, err := h.service.GenerateProductsLabelPDF(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erro ao gerar PDF de etiquetas",
		})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\"etiquetas_produtos.pdf\"")
	c.Header("Content-Length", fmt.Sprintf("%d", pdfBuffer.Len()))

	c.Data(http.StatusOK, "application/pdf", pdfBuffer.Bytes())
}
