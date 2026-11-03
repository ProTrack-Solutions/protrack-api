package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	fiscal := r.Group("/fiscal")
	fiscal.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))

	// Leitura da configuração: a tela de vendas mostra se a NFC-e está ligada
	read := fiscal.Group("")
	read.Use(middleware.RequireAnyModule(h.queries, "sales", "inventory"))
	{
		read.GET("/settings", h.GetSettings)
	}

	// Tributação de produtos e categorias: quem cuida do estoque
	products := fiscal.Group("")
	products.Use(middleware.RequireModule("inventory", h.queries))
	{
		products.GET("/tax-profiles", h.ListTaxProfiles)
		products.GET("/categories", h.ListCategories)
		products.PUT("/categories/:id/tax-profile", h.SetCategoryTaxProfile)
		products.GET("/products/:id", h.GetProductFiscal)
		products.PUT("/products/:id", h.UpdateProductFiscal)
		products.PUT("/models", h.UpdateModelFiscal)
	}

	// Emitente, A1, CSC, perfis tributários e produção: só administrador
	admin := fiscal.Group("")
	admin.Use(middleware.RequireRole("ADMIN"))
	{
		admin.PUT("/settings", h.UpdateSettings)
		admin.POST("/issuer", h.RegisterIssuer)
		admin.POST("/certificate", h.UploadCertificate)
		admin.GET("/certificate", h.GetCertificate)
		admin.GET("/profile", h.GetFiscalProfile)
		admin.PUT("/profile", h.SaveFiscalProfile)
		admin.GET("/sefaz-status", h.SefazStatus)
		admin.POST("/go-live", h.GoLive)
		admin.POST("/tax-profiles", h.CreateTaxProfile)
		admin.PUT("/tax-profiles/:id", h.UpdateTaxProfile)
		admin.DELETE("/tax-profiles/:id", h.DeleteTaxProfile)
	}
}
