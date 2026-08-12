package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/repository"
	productsService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	saleItemsService "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/service"
	salesDomain "github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	salesService "github.com/ProTrack-Solutions/protrack-api/internal/sales/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// isRetryableFiscalInvoiceStatus indica se um documento fiscal existente pode
// ser reaproveitado numa nova tentativa de emissão. 'rejected' (recusado
// pela SEFAZ/provedor) e 'cancelled' (cancelado, pode-se emitir um novo
// documento para a mesma venda) liberam retry; qualquer outro status
// significa que já existe uma emissão em andamento ou concluída.
func isRetryableFiscalInvoiceStatus(status interface{}) bool {
	s := fmt.Sprintf("%v", status)
	return s == "rejected" || s == "cancelled"
}

type Service struct {
	repo             *repository.Repository
	pool             *pgxpool.Pool
	fiscalProvider   FiscalProvider
	salesService     *salesService.Service
	saleItemsService *saleItemsService.Service
	companiesService *companiesService.Service
	productsService  *productsService.Service
	certCipher       *crypto.CertCipher
	cfg              *config.Config
}

func NewService(
	repo *repository.Repository,
	pool *pgxpool.Pool,
	fiscalProvider FiscalProvider,
	salesService *salesService.Service,
	saleItemsService *saleItemsService.Service,
	companiesService *companiesService.Service,
	productsService *productsService.Service,
	certCipher *crypto.CertCipher,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:             repo,
		pool:             pool,
		fiscalProvider:   fiscalProvider,
		salesService:     salesService,
		saleItemsService: saleItemsService,
		companiesService: companiesService,
		productsService:  productsService,
		certCipher:       certCipher,
		cfg:              cfg,
	}
}

func (s *Service) EmitFiscalInvoice(ctx context.Context, req domain.EmitFiscalInvoiceRequest, companyID uuid.UUID) (domain.FiscalInvoiceResponse, error) {
	// Carrega a venda primeiro (já escopada por companyID) — garante que o
	// sale_id pertence à empresa autenticada antes de qualquer outra checagem.
	sale, err := s.salesService.GetSaleById(ctx, salesDomain.GetSaleByIdRequest{ID: req.SaleID, CompanyID: companyID})
	if err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao carregar venda: %w", err)
	}

	// Reaproveita um documento existente se ele já falhou (rejected/cancelled)
	// — sem isso, uma nota rejeitada pela SEFAZ bloquearia essa venda para
	// sempre, já que UNIQUE(sale_id, type) impede criar outra linha. Erros
	// reais de banco (não "não encontrado") são propagados em vez de tratados
	// como "pode emitir".
	existing, err := s.repo.GetFiscalInvoiceBySaleAndType(ctx, db.GetFiscalInvoiceBySaleAndTypeParams{
		SaleID: pgconv.ParseUUIDToPgType(req.SaleID),
		Type:   req.Type,
	})
	var retryInvoiceID pgtype.UUID
	switch {
	case err == nil:
		if !isRetryableFiscalInvoiceStatus(existing.Status) {
			return domain.FiscalInvoiceResponse{}, domain.ErrInvoiceAlreadyExists
		}
		retryInvoiceID = existing.ID
	case errors.Is(err, pgx.ErrNoRows):
		// não existe ainda — segue para criar um novo documento.
	default:
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao verificar documento fiscal existente: %w", err)
	}

	items, err := s.saleItemsService.GetSaleItemsBySaleID(ctx, req.SaleID)
	if err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao carregar itens da venda: %w", err)
	}
	if len(items) == 0 {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: venda sem itens")
	}

	company, err := s.companiesService.GetCompanyByID(ctx, companyID)
	if err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao carregar empresa: %w", err)
	}

	cert, err := s.repo.GetCompanyCertificateByCompanyID(ctx, pgconv.ParseUUIDToPgType(companyID))
	if err != nil {
		return domain.FiscalInvoiceResponse{}, domain.ErrCertificateNotFound
	}
	if cert.ExpiresAt.Valid && cert.ExpiresAt.Time.Before(time.Now()) {
		return domain.FiscalInvoiceResponse{}, domain.ErrCertificateExpired
	}

	missing := s.validateFiscalFields(ctx, company, items)
	if missing.HasMissing() {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("%w: empresa=%v produtos=%v",
			domain.ErrMissingFiscalFields, missing.CompanyFields, missing.ProductFields)
	}

	var invoiceRow db.FiscalInvoice
	if retryInvoiceID.Valid {
		invoiceRow, err = s.repo.ResetFiscalInvoiceForRetry(ctx, retryInvoiceID)
		if err != nil {
			return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao reabrir documento fiscal para nova tentativa: %w", err)
		}
		s.recordEvent(ctx, invoiceRow.ID, "created", map[string]any{"retry": true})
	} else {
		invoiceRow, err = s.repo.CreateFiscalInvoice(ctx, db.CreateFiscalInvoiceParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			SaleID:    pgconv.ParseUUIDToPgType(req.SaleID),
			Type:      req.Type,
			Status:    "processing",
		})
		if err != nil {
			return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao criar registro do documento fiscal: %w", err)
		}
		s.recordEvent(ctx, invoiceRow.ID, "created", nil)
	}

	idIntegracao := invoiceRow.ID.String()
	var providerInvoiceID string

	if req.Type == "nfe" {
		payload, buildErr := s.buildNFePayload(idIntegracao, company, sale, items)
		if buildErr != nil {
			return domain.FiscalInvoiceResponse{}, buildErr
		}
		providerInvoiceID, err = s.fiscalProvider.EmitirNFe(ctx, payload)
	} else {
		payload, buildErr := s.buildNFCePayload(idIntegracao, company, sale, items)
		if buildErr != nil {
			return domain.FiscalInvoiceResponse{}, buildErr
		}
		providerInvoiceID, err = s.fiscalProvider.EmitirNFCe(ctx, payload)
	}

	if err != nil {
		s.recordEvent(ctx, invoiceRow.ID, "error", map[string]any{"error": err.Error()})
		_, _ = s.repo.UpdateFiscalInvoiceRejected(ctx, db.UpdateFiscalInvoiceRejectedParams{
			ID:           invoiceRow.ID,
			ErrorMessage: pgconv.ParseStringToPgType(err.Error()),
		})
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao emitir junto ao provedor: %w", err)
	}

	updated, err := s.repo.UpdateFiscalInvoiceProcessing(ctx, db.UpdateFiscalInvoiceProcessingParams{
		ID:                invoiceRow.ID,
		ProviderInvoiceID: pgconv.ParseStringToPgType(providerInvoiceID),
		Status:            "processing",
	})
	if err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao atualizar status: %w", err)
	}
	s.recordEvent(ctx, invoiceRow.ID, "sent_to_provider", map[string]any{"providerInvoiceId": providerInvoiceID})

	return domain.ToFiscalInvoiceResponse(updated), nil
}

func (s *Service) CancelFiscalInvoice(ctx context.Context, req domain.CancelFiscalInvoiceRequest) (domain.FiscalInvoiceResponse, error) {
	if err := domain.ValidateCancelFiscalInvoiceRequest(req); err != nil {
		return domain.FiscalInvoiceResponse{}, err
	}

	invoice, err := s.repo.GetFiscalInvoiceByID(ctx, db.GetFiscalInvoiceByIDParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
	})
	if err != nil {
		return domain.FiscalInvoiceResponse{}, domain.ErrInvoiceNotFound
	}
	if invoice.Status != "authorized" {
		return domain.FiscalInvoiceResponse{}, domain.ErrInvoiceNotAuthorized
	}
	if !invoice.AuthorizedAt.Valid || time.Since(invoice.AuthorizedAt.Time) > 24*time.Hour {
		return domain.FiscalInvoiceResponse{}, domain.ErrCancellationWindowExpired
	}

	if _, err := s.repo.UpdateFiscalInvoiceCancelProcessing(ctx, invoice.ID); err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao marcar cancelamento em processamento: %w", err)
	}
	s.recordEvent(ctx, invoice.ID, "cancel_requested", map[string]any{"justificativa": req.Justificativa})

	providerInvoiceID := pgconv.ParsePgTextToString(invoice.ProviderInvoiceID)

	var cancelErr error
	if invoice.Type == "nfe" {
		cancelErr = s.fiscalProvider.CancelarNFe(ctx, providerInvoiceID, req.Justificativa)
	} else {
		cancelErr = s.fiscalProvider.CancelarNFCe(ctx, providerInvoiceID, req.Justificativa)
	}
	if cancelErr != nil {
		s.recordEvent(ctx, invoice.ID, "cancel_rejected", map[string]any{"error": cancelErr.Error()})
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao solicitar cancelamento junto ao provedor: %w", cancelErr)
	}

	updated, err := s.repo.UpdateFiscalInvoiceCancelled(ctx, db.UpdateFiscalInvoiceCancelledParams{
		ID:              invoice.ID,
		CancelledReason: pgconv.ParseStringToPgType(req.Justificativa),
	})
	if err != nil {
		return domain.FiscalInvoiceResponse{}, fmt.Errorf("fiscal_invoices: falha ao atualizar status de cancelamento: %w", err)
	}
	s.recordEvent(ctx, invoice.ID, "cancelled", nil)

	return domain.ToFiscalInvoiceResponse(updated), nil
}

func (s *Service) GetFiscalInvoiceByID(ctx context.Context, id uuid.UUID, companyID uuid.UUID) (domain.FiscalInvoiceResponse, error) {
	invoice, err := s.repo.GetFiscalInvoiceByID(ctx, db.GetFiscalInvoiceByIDParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(companyID),
	})
	if err != nil {
		return domain.FiscalInvoiceResponse{}, domain.ErrInvoiceNotFound
	}
	return domain.ToFiscalInvoiceResponse(invoice), nil
}

func (s *Service) ListFiscalInvoicesBySale(ctx context.Context, saleID uuid.UUID, companyID uuid.UUID) ([]domain.FiscalInvoiceResponse, error) {
	rows, err := s.repo.ListFiscalInvoicesBySale(ctx, db.ListFiscalInvoicesBySaleParams{
		SaleID:    pgconv.ParseUUIDToPgType(saleID),
		CompanyID: pgconv.ParseUUIDToPgType(companyID),
	})
	if err != nil {
		return nil, err
	}
	var out []domain.FiscalInvoiceResponse
	for _, row := range rows {
		out = append(out, domain.ToFiscalInvoiceResponse(row))
	}
	return out, nil
}
