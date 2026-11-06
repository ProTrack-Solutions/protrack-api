package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/client"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

// EmissionAPI é o que a emissão usa do cliente da API fiscal (NFC-e e NF-e:
// model 65 ou 55 escolhe o caminho).
type EmissionAPI interface {
	EmitDocument(ctx context.Context, env client.Environment, model int, issuerID, idempotencyKey string, in any) (client.Document, error)
	GetDocument(ctx context.Context, env client.Environment, model int, documentID string) (client.Document, error)
	Printout(ctx context.Context, env client.Environment, model int, documentID string) ([]byte, error)
	Xml(ctx context.Context, env client.Environment, model int, documentID string) ([]byte, error)
	CancelDocument(ctx context.Context, env client.Environment, model int, documentID, reason string) (client.CancellationResult, error)
	NfeTotals(ctx context.Context, env client.Environment, in client.NfeTotalsRequest) (client.NfeTotals, error)
}

// EmitTimeout: quanto o PDV espera a nota antes de seguir (o worker termina depois).
const EmitTimeout = 20 * time.Second

func (s *Service) emission() (EmissionAPI, bool) {
	e, ok := s.api.(EmissionAPI)
	return e, ok
}

func toDocument(d db.FiscalDocument) domain.FiscalDocument {
	var total *float64
	if d.Total.Valid {
		v := pgconv.PgNumericToFloat64(d.Total)
		total = &v
	}
	return domain.FiscalDocument{
		ID:                pgconv.PgUUIDToUUID(d.ID),
		SaleID:            pgconv.PgUUIDToUUIDPtr(d.SaleID),
		Model:             d.Model,
		Source:            d.Source,
		OperationNature:   pgconv.ParsePgTextToStringPtr(d.OperationNature),
		RecipientName:     pgconv.ParsePgTextToStringPtr(d.RecipientName),
		RecipientDocument: pgconv.ParsePgTextToStringPtr(d.RecipientDocument),
		Total:             total,
		Environment:       d.Environment,
		Attempt:           d.Attempt,
		Status:            d.Status,
		Number:            pgconv.PgInt4ToIntPtr(d.Number),
		AccessKey:         pgconv.ParsePgTextToStringPtr(d.AccessKey),
		Protocol:          pgconv.ParsePgTextToStringPtr(d.Protocol),
		StatusCode:        pgconv.ParsePgTextToStringPtr(d.StatusCode),
		RejectReason:      pgconv.ParsePgTextToStringPtr(d.RejectReason),
		ErrorMessage:      pgconv.ParsePgTextToStringPtr(d.ErrorMessage),
		Contingency:       d.EmissionType.Valid && d.EmissionType.Int16 == 9,
		AuthorizedAt:      pgconv.PgTimestamptzToTimePtr(d.AuthorizedAt),
		CancelReason:      pgconv.ParsePgTextToStringPtr(d.CancelReason),
		CancelledAt:       pgconv.PgTimestamptzToTimePtr(d.CancelledAt),
		CreatedAt:         pgconv.PgTimestamptzToTime(d.CreatedAt),
	}
}

func textOrNull(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// applyDocument grava o que a API fiscal disse da nota.
func applyDocument(ctx context.Context, repo *repository.Repository, id pgtype.UUID, doc client.Document) (db.FiscalDocument, error) {
	params := db.ApplyFiscalDocumentResultParams{
		ID:           id,
		Status:       doc.Status,
		AccessKey:    textOrNull(doc.AccessKey),
		Protocol:     textOrNull(doc.Protocol),
		StatusCode:   textOrNull(doc.StatusCode),
		RejectReason: textOrNull(doc.RejectReason),
	}
	if ext, err := uuid.Parse(doc.ID); err == nil {
		params.ExternalID = pgconv.ParseUUIDToPgType(ext)
	}
	if doc.Number > 0 {
		params.Number = pgtype.Int4{Int32: int32(doc.Number), Valid: true}
	}
	if doc.EmissionType > 0 {
		params.EmissionType = pgtype.Int2{Int16: int16(doc.EmissionType), Valid: true}
	}
	if doc.AuthorizedAt != nil {
		params.AuthorizedAt = pgconv.TimeToPgTimestamptz(*doc.AuthorizedAt)
	}
	switch doc.Status {
	case client.StatusRejected:
		params.ErrorMessage = textOrNull(doc.RejectReason)
	case client.StatusError:
		params.ErrorMessage = pgtype.Text{String: "a API fiscal não conseguiu emitir a nota; tente de novo", Valid: true}
	}
	return repo.ApplyFiscalDocumentResult(ctx, params)
}

// loadSaleForNfce lê venda, itens e perfis e monta o pedido.
func (s *Service) loadSaleForNfce(ctx context.Context, repo *repository.Repository, companyId, saleId uuid.UUID, taxRegime string) (client.NfceRequest, []string, error) {
	sale, err := repo.GetSaleForNfce(ctx, db.GetSaleForNfceParams{
		ID: pgconv.ParseUUIDToPgType(saleId), CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return client.NfceRequest{}, nil, domain.ErrSaleNotFound
	}
	if err != nil {
		return client.NfceRequest{}, nil, err
	}
	if sale.DeletedAt.Valid || sale.Status == "canceled" {
		return client.NfceRequest{}, nil, domain.ErrSaleCanceled
	}
	items, err := repo.ListSaleItemsForNfce(ctx, pgconv.ParseUUIDToPgType(saleId))
	if err != nil {
		return client.NfceRequest{}, nil, err
	}
	var ids []pgtype.UUID
	seen := map[pgtype.UUID]bool{}
	for _, it := range items {
		if it.TaxProfileID.Valid && !seen[it.TaxProfileID] {
			seen[it.TaxProfileID] = true
			ids = append(ids, it.TaxProfileID)
		}
	}
	profiles := map[pgtype.UUID]domain.TaxProfile{}
	if len(ids) > 0 {
		rows, err := repo.ListTaxProfilesByIds(ctx, db.ListTaxProfilesByIdsParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyId), Ids: ids,
		})
		if err != nil {
			return client.NfceRequest{}, nil, err
		}
		for _, r := range rows {
			profiles[r.ID] = toTaxProfile(r)
		}
	}
	in := domain.NfceSale{
		Subtotal:      pgconv.PgNumericToFloat64(sale.Subtotal),
		Discount:      pgconv.PgNumericToFloat64(sale.DiscountAmount),
		Total:         pgconv.PgNumericToFloat64(sale.TotalAmount),
		DownPayment:   pgconv.PgNumericToFloat64(sale.DownPayment),
		Credit:        pgconv.PgNumericToFloat64(sale.CreditAmount),
		PaymentMethod: sale.PaymentMethod,
		BuyerDocument: pgconv.ParsePgTextToString(sale.BuyerDocument),
	}
	for _, it := range items {
		item := domain.NfceSaleItem{
			ProductID: pgconv.PgUUIDToUUID(it.ProductID),
			Name:      it.Name,
			Barcode:   pgconv.ParsePgTextToString(it.Barcode),
			NCM:       pgconv.ParsePgTextToString(it.Ncm),
			CEST:      pgconv.ParsePgTextToString(it.Cest),
			Unit:      string(it.Unit),
			Origin:    it.OrigemMercadoria,
			Quantity:  it.Quantity,
			UnitPrice: pgconv.PgNumericToFloat64(it.UnitPrice),
			Discount:  pgconv.PgNumericToFloat64(it.Discount),
		}
		if p, ok := profiles[it.TaxProfileID]; ok {
			item.Profile = &p
		}
		in.Items = append(in.Items, item)
	}
	req, problems := domain.BuildNfce(in, taxRegime)
	return req, problems, nil
}

// EmitForSale emite (ou devolve a já emitida) NFC-e da venda. auto=true é a
// emissão automática depois da venda: só acontece com a emissão ligada.
// A chamada à API fiscal fica fora da transação; falha transitória deixa a nota
// PENDING para o worker reenviar o mesmo pedido com a mesma Idempotency-Key.
func (s *Service) EmitForSale(ctx context.Context, companyId, userId, saleId uuid.UUID, auto bool) (*domain.FiscalDocument, error) {
	api, ok := s.emission()
	if !ok {
		return nil, domain.ErrFiscalNotConfigured
	}
	settings, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return nil, err
	}
	if auto && !settings.Enabled {
		return nil, nil
	}
	env, issuer, registered := IssuerFor(settings.CompanyFiscalSetting)
	if !registered {
		return nil, domain.ErrIssuerNotRegistered
	}
	if !s.api.Configured(env) {
		return nil, domain.ErrFiscalNotConfigured
	}

	var doc db.FiscalDocument
	var send bool
	var req client.NfceRequest
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockSaleFiscal(ctx, saleId.String()); err != nil {
			return err
		}
		existing, err := repo.GetLiveFiscalDocumentBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
		if err == nil {
			if pgconv.PgUUIDToUUID(existing.CompanyID) != companyId {
				return domain.ErrSaleNotFound
			}
			doc = existing
			return nil // já existe nota viva: idempotente
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var problems []string
		req, problems, err = s.loadSaleForNfce(ctx, repo, companyId, saleId, pgconv.ParsePgTextToString(settings.TaxRegime))
		if err != nil {
			return err
		}
		attempt, err := repo.NextFiscalAttempt(ctx, pgconv.ParseUUIDToPgType(saleId))
		if err != nil {
			return err
		}
		payload, err := json.Marshal(req)
		if err != nil {
			return err
		}
		params := db.CreateFiscalDocumentParams{
			CompanyID:      pgconv.ParseUUIDToPgType(companyId),
			SaleID:         pgconv.ParseUUIDToPgType(saleId),
			Environment:    string(env),
			Attempt:        attempt,
			IdempotencyKey: fmt.Sprintf("sale-%s-%s-%d", saleId, env, attempt),
			Status:         domain.StatusPending,
			RequestPayload: string(payload),
			CreatedBy:      pgconv.ParseUUIDToPgType(userId),
		}
		if len(problems) > 0 {
			// Não vira nota: fica registrada para o lojista corrigir e reenviar
			params.Status = domain.StatusFailed
			params.ErrorMessage = pgtype.Text{String: strings.Join(problems, "; "), Valid: true}
			params.RequestPayload = "{}"
		}
		doc, err = repo.CreateFiscalDocument(ctx, params)
		send = len(problems) == 0
		return err
	})
	if err != nil {
		return nil, err
	}
	if !send {
		out := toDocument(doc)
		return &out, nil
	}
	doc, err = s.send(ctx, api, env, issuer, doc, req)
	if err != nil {
		return nil, err
	}
	out := toDocument(doc)
	return &out, nil
}

// send chama a API fiscal e grava o resultado. Erros transitórios não voltam
// como erro: a nota fica PENDING com a mensagem e o worker tenta de novo.
// req é o pedido (NfceRequest, NfeRequest ou o JSON gravado); o modelo vem da nota.
func (s *Service) send(ctx context.Context, api EmissionAPI, env client.Environment, issuer string, doc db.FiscalDocument, req any) (db.FiscalDocument, error) {
	result, callErr := api.EmitDocument(ctx, env, int(doc.Model), issuer, doc.IdempotencyKey, req)
	// Grava mesmo se o contexto da requisição já acabou (timeout do PDV)
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if callErr == nil {
		return applyDocument(saveCtx, s.repo, doc.ID, result)
	}
	apiErr, isAPI := client.AsAPIError(callErr)
	permanent := isAPI && apiErr.Status >= 400 && apiErr.Status < 500 &&
		apiErr.Status != 408 && apiErr.Status != 429
	if permanent {
		msg := apiErr.UserMessage()
		if msg == "" {
			msg = fmt.Sprintf("a API fiscal recusou a nota (%d)", apiErr.Status)
		}
		return s.repo.ApplyFiscalDocumentResult(saveCtx, db.ApplyFiscalDocumentResultParams{
			ID: doc.ID, Status: domain.StatusFailed,
			ErrorMessage: pgtype.Text{String: msg, Valid: true},
		})
	}
	msg := "serviço fiscal indisponível; a nota será enviada automaticamente"
	if isAPI && apiErr.SefazUnavailable() {
		msg = "SEFAZ sem resposta; a nota será enviada automaticamente"
	}
	log.Warn().Err(callErr).Str("fiscal_document", pgconv.PgUUIDToUUID(doc.ID).String()).Int16("model", doc.Model).Msg("nota fiscal: falha transitória, worker vai reenviar")
	return s.repo.MarkFiscalDocumentRetry(saveCtx, db.MarkFiscalDocumentRetryParams{ID: doc.ID, ErrorMessage: pgtype.Text{String: msg, Valid: true}})
}

// AfterSale é chamado pelo PDV depois de gravar a venda: emite se a NFC-e estiver
// ligada e nunca falha a venda (erro vira log e a nota fica para o painel/worker).
func (s *Service) AfterSale(ctx context.Context, companyId, userId, saleId uuid.UUID) *domain.FiscalDocument {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), EmitTimeout)
	defer cancel()
	doc, err := s.EmitForSale(ctx, companyId, userId, saleId, true)
	if err != nil {
		if !errors.Is(err, domain.ErrIssuerNotRegistered) && !errors.Is(err, domain.ErrFiscalNotConfigured) {
			log.Error().Err(err).Str("sale_id", saleId.String()).Msg("NFC-e: falha ao emitir depois da venda")
		}
		return nil
	}
	return doc
}

// GetSaleDocument devolve a última tentativa de nota da venda (nil se nunca houve).
func (s *Service) GetSaleDocument(ctx context.Context, companyId, saleId uuid.UUID) (*domain.FiscalDocument, error) {
	d, err := s.repo.GetLatestFiscalDocumentBySale(ctx, db.GetLatestFiscalDocumentBySaleParams{
		SaleID: pgconv.ParseUUIDToPgType(saleId), CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toDocument(d)
	return &out, nil
}

func (s *Service) getDocument(ctx context.Context, companyId, id uuid.UUID) (db.FiscalDocument, error) {
	d, err := s.repo.GetFiscalDocument(ctx, db.GetFiscalDocumentParams{
		ID: pgconv.ParseUUIDToPgType(id), CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return d, domain.ErrDocumentNotFound
	}
	return d, err
}

// Printout devolve o PDF (DANFC-e da NFC-e ou DANFE da NF-e) ou o XML da nota autorizada.
func (s *Service) Printout(ctx context.Context, companyId, id uuid.UUID, xml bool) ([]byte, string, error) {
	api, ok := s.emission()
	if !ok {
		return nil, "", domain.ErrFiscalNotConfigured
	}
	d, err := s.getDocument(ctx, companyId, id)
	if err != nil {
		return nil, "", err
	}
	if !d.ExternalID.Valid || (d.Status != domain.StatusAuthorized && d.Status != domain.StatusContingency &&
		!(xml && d.Status == domain.StatusCancelled)) {
		return nil, "", domain.ErrDocumentNotPrintable
	}
	env := client.Environment(d.Environment)
	ext := pgconv.PgUUIDToUUID(d.ExternalID).String()
	key := pgconv.ParsePgTextToString(d.AccessKey)
	model := int(d.Model)
	prefix, pdf := "nfce-", "danfce-"
	if model == domain.ModelNfe {
		prefix, pdf = "nfe-", "danfe-"
	}
	if xml {
		data, err := api.Xml(ctx, env, model, ext)
		return data, prefix + key + ".xml", translate(err)
	}
	data, err := api.Printout(ctx, env, model, ext)
	return data, pdf + key + ".pdf", translate(err)
}

// ListDocuments: painel de notas (última tentativa de cada venda).
func (s *Service) ListDocuments(ctx context.Context, companyId uuid.UUID, f domain.DocumentFilter, page, perPage int) (domain.FiscalDocumentList, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	rows, err := s.repo.ListFiscalDocuments(ctx, db.ListFiscalDocumentsParams{
		CompanyID:    pgconv.ParseUUIDToPgType(companyId),
		Status:       optText(strings.ToUpper(strings.TrimSpace(f.Status))),
		Model:        modelParam(f.Model),
		Source:       optText(strings.ToLower(strings.TrimSpace(f.Source))),
		ProblemsOnly: f.ProblemsOnly,
		PageOffset:   int32((page - 1) * perPage),
		PageSize:     int32(perPage),
	})
	if err != nil {
		return domain.FiscalDocumentList{}, err
	}
	out := domain.FiscalDocumentList{Documents: make([]domain.FiscalDocumentListItem, 0, len(rows)), Page: page, PerPage: perPage}
	for _, r := range rows {
		d := toDocument(db.FiscalDocument{
			ID: r.ID, CompanyID: r.CompanyID, SaleID: r.SaleID, Model: r.Model, Environment: r.Environment,
			Attempt: r.Attempt, IdempotencyKey: r.IdempotencyKey, ExternalID: r.ExternalID, Status: r.Status,
			Number: r.Number, AccessKey: r.AccessKey, Protocol: r.Protocol, StatusCode: r.StatusCode,
			RejectReason: r.RejectReason, ErrorMessage: r.ErrorMessage, EmissionType: r.EmissionType,
			AuthorizedAt: r.AuthorizedAt, CancelReason: r.CancelReason, CancelledAt: r.CancelledAt,
			CreatedAt: r.CreatedAt, Source: r.Source, OperationNature: r.OperationNature,
			RecipientName: r.RecipientName, RecipientDocument: r.RecipientDocument, Total: r.Total,
		})
		customer := pgconv.ParsePgTextToStringPtr(r.CustomerName)
		if customer == nil {
			customer = d.RecipientName
		}
		out.Total = r.TotalCount
		out.Documents = append(out.Documents, domain.FiscalDocumentListItem{
			FiscalDocument: d,
			SaleTotal:      pgconv.PgNumericToFloat64(r.TotalAmount),
			SaleAt:         pgconv.PgTimestamptzToTime(r.SaleAt),
			CustomerName:   customer,
		})
	}
	return out, nil
}

// SalesDocuments: última nota de cada venda pedida (lista de vendas). Vendas sem
// nota não aparecem no mapa.
func (s *Service) SalesDocuments(ctx context.Context, companyId uuid.UUID, saleIds []uuid.UUID) (map[uuid.UUID]domain.FiscalDocument, error) {
	out := map[uuid.UUID]domain.FiscalDocument{}
	if len(saleIds) == 0 {
		return out, nil
	}
	ids := make([]pgtype.UUID, 0, len(saleIds))
	for _, id := range saleIds {
		ids = append(ids, pgconv.ParseUUIDToPgType(id))
	}
	rows, err := s.repo.ListFiscalDocumentsBySales(ctx, db.ListFiscalDocumentsBySalesParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId), SaleIds: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[pgconv.PgUUIDToUUID(r.SaleID)] = toDocument(r)
	}
	return out, nil
}

// Motivo usado quando a venda é cancelada no ERP (mínimo de 15 caracteres na SEFAZ).
const saleCancelReason = "Venda cancelada no sistema de gestão ProTrack"

// BeforeSaleCancel roda antes de cancelar a venda: cancela a NFC-e autorizada e
// bloqueia se a nota ainda está em andamento ou se a SEFAZ recusar o cancelamento.
func (s *Service) BeforeSaleCancel(ctx context.Context, companyId, saleId uuid.UUID) error {
	d, err := s.repo.GetLiveFiscalDocumentBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if pgconv.PgUUIDToUUID(d.CompanyID) != companyId {
		return nil // o cancelamento da venda devolve "não encontrada"
	}
	// Venda que não pode ser cancelada (devolução/troca): não mexe na nota
	returns, err := s.repo.CountSaleReturnsBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if err != nil {
		return err
	}
	if returns.ReturnsCount > 0 || returns.IsExchangeSale {
		return nil
	}
	switch d.Status {
	case domain.StatusCancelled:
		return nil
	case domain.StatusAuthorized:
		out, err := s.CancelDocument(ctx, companyId, pgconv.PgUUIDToUUID(d.ID), saleCancelReason)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				return &domain.SaleBlockedError{Reason: "não foi possível cancelar a NFC-e: " + apiErr.Message + ". Fora do prazo de cancelamento, registre uma devolução"}
			}
			return err
		}
		if out.Status != domain.StatusCancelled {
			return &domain.SaleBlockedError{Reason: "o cancelamento da NFC-e foi enviado e aguarda a SEFAZ; tente cancelar a venda em instantes"}
		}
		return nil
	default:
		return &domain.SaleBlockedError{Reason: "a NFC-e desta venda ainda está sendo processada; aguarde alguns minutos para cancelar"}
	}
}

// BeforeSaleUpdate bloqueia a alteração de venda que já tem NFC-e válida ou em andamento.
func (s *Service) BeforeSaleUpdate(ctx context.Context, companyId, saleId uuid.UUID) error {
	d, err := s.repo.GetLiveFiscalDocumentBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if pgconv.PgUUIDToUUID(d.CompanyID) != companyId || d.Status == domain.StatusCancelled {
		return nil
	}
	return &domain.SaleBlockedError{Reason: "a venda já tem NFC-e e não pode ser alterada; cancele a venda (a nota é cancelada junto) e faça uma nova"}
}

func modelParam(model int) pgtype.Int2 {
	if model != domain.ModelNfe && model != domain.ModelNfce {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(model), Valid: true}
}
