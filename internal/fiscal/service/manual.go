package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/client"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ManualProblemsError: a nota manual não pode ser emitida como está (a tela mostra a lista).
type ManualProblemsError struct{ Problems []string }

func (e *ManualProblemsError) Error() string { return strings.Join(e.Problems, "; ") }

// ManualPreview é a conferência da nota antes de emitir: problemas, CFOP de cada
// item e os totais (na NF-e, calculados pela API fiscal com IPI/ST/DIFAL/IBS/CBS).
type ManualPreview struct {
	Problems   []string            `json:"problems"`
	Interstate bool                `json:"interstate"`
	Items      []ManualPreviewItem `json:"items"`
	// Total a pagar: vNF na NF-e, soma dos itens na NFC-e
	Total  float64           `json:"total"`
	Totals *client.NfeTotals `json:"totals"`
}

type ManualPreviewItem struct {
	Description string `json:"description"`
	CFOP        string `json:"cfop"`
}

type manualContext struct {
	build  domain.ManualBuild
	env    client.Environment
	issuer string
}

// prepareManual lê produtos, perfis e configuração e monta o pedido.
func (s *Service) prepareManual(ctx context.Context, companyId uuid.UUID, in domain.ManualDocumentInput) (manualContext, []string, error) {
	var mc manualContext
	settings, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return mc, nil, err
	}
	env, issuer, registered := IssuerFor(settings.CompanyFiscalSetting)
	if !registered {
		return mc, nil, domain.ErrIssuerNotRegistered
	}
	if !s.api.Configured(env) {
		return mc, nil, domain.ErrFiscalNotConfigured
	}
	mc.env, mc.issuer = env, issuer

	company, err := s.repo.GetCompanyForFiscal(ctx, pgconv.ParseUUIDToPgType(companyId))
	if errors.Is(err, pgx.ErrNoRows) {
		return mc, nil, domain.ErrCompanyNotFound
	}
	if err != nil {
		return mc, nil, err
	}

	var productIds []pgtype.UUID
	seen := map[uuid.UUID]bool{}
	for _, it := range in.Items {
		if it.ProductID != uuid.Nil && !seen[it.ProductID] {
			seen[it.ProductID] = true
			productIds = append(productIds, pgconv.ParseUUIDToPgType(it.ProductID))
		}
	}
	rows := []db.ListProductsForManualDocumentRow{}
	if len(productIds) > 0 {
		rows, err = s.repo.ListProductsForManualDocument(ctx, db.ListProductsForManualDocumentParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyId), Ids: productIds,
		})
		if err != nil {
			return mc, nil, err
		}
	}

	// Perfis: o resolvido do produto e os escolhidos na tela
	var profileIds []pgtype.UUID
	seenProfile := map[pgtype.UUID]bool{}
	addProfile := func(id pgtype.UUID) {
		if id.Valid && !seenProfile[id] {
			seenProfile[id] = true
			profileIds = append(profileIds, id)
		}
	}
	for _, r := range rows {
		addProfile(r.TaxProfileID)
	}
	for _, it := range in.Items {
		if it.TaxProfileID != nil {
			addProfile(pgconv.ParseUUIDToPgType(*it.TaxProfileID))
		}
	}
	profiles := map[uuid.UUID]domain.TaxProfile{}
	if len(profileIds) > 0 {
		list, err := s.repo.ListTaxProfilesByIds(ctx, db.ListTaxProfilesByIdsParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyId), Ids: profileIds,
		})
		if err != nil {
			return mc, nil, err
		}
		for _, p := range list {
			profiles[pgconv.PgUUIDToUUID(p.ID)] = toTaxProfile(p)
		}
	}

	catalog := map[uuid.UUID]domain.ManualProduct{}
	for _, r := range rows {
		p := domain.ManualProduct{
			ID:      pgconv.PgUUIDToUUID(r.ID),
			Name:    r.Name,
			Barcode: pgconv.ParsePgTextToString(r.Barcode),
			NCM:     pgconv.ParsePgTextToString(r.Ncm),
			CEST:    pgconv.ParsePgTextToString(r.Cest),
			Unit:    string(r.Unit),
			Origin:  r.OrigemMercadoria,
		}
		if prof, ok := profiles[pgconv.PgUUIDToUUID(r.TaxProfileID)]; ok && r.TaxProfileID.Valid {
			p.Profile = &prof
		}
		catalog[p.ID] = p
	}

	// Perfil escolhido no item vale só para aquele item: monta um produto por item
	var problems []string
	items := make([]domain.ManualItemInput, len(in.Items))
	copy(items, in.Items)
	perItem := map[uuid.UUID]domain.ManualProduct{}
	for i, it := range items {
		p, ok := catalog[it.ProductID]
		if !ok {
			continue
		}
		if it.TaxProfileID != nil {
			if prof, ok := profiles[*it.TaxProfileID]; ok {
				p.Profile = &prof
			} else {
				problems = append(problems, "item "+strconv.Itoa(i+1)+": perfil tributário não encontrado")
			}
		}
		key := uuid.New()
		p.ID = it.ProductID
		perItem[key] = p
		items[i].ProductID = key
	}
	in.Items = items
	build, buildProblems := domain.BuildManual(in, perItem, pgconv.ParsePgTextToString(company.AddressState),
		pgconv.ParsePgTextToString(settings.TaxRegime))
	mc.build = build
	return mc, append(problems, buildProblems...), nil
}

// nfeTotals pede os totais à API fiscal. Recusa do pedido (4xx) vira problema.
func (s *Service) nfeTotals(ctx context.Context, api EmissionAPI, mc manualContext) (*client.NfeTotals, []string, error) {
	totals, err := api.NfeTotals(ctx, mc.env, client.NfeTotalsRequest{Items: mc.build.Nfe.Items, FinalConsumer: mc.build.Nfe.FinalConsumer})
	if err == nil {
		return &totals, nil, nil
	}
	if apiErr, ok := client.AsAPIError(err); ok && apiErr.Status >= 400 && apiErr.Status < 500 {
		msg := apiErr.UserMessage()
		if msg == "" || apiErr.Status == http.StatusBadRequest {
			msg = "a API fiscal recusou os itens: " + strings.TrimSpace(string(apiErr.Body))
		}
		return nil, []string{msg}, nil
	}
	return nil, nil, translate(err)
}

// PreviewManual confere a nota sem emitir.
func (s *Service) PreviewManual(ctx context.Context, companyId uuid.UUID, in domain.ManualDocumentInput) (ManualPreview, error) {
	out := ManualPreview{Problems: []string{}, Items: []ManualPreviewItem{}}
	api, ok := s.emission()
	if !ok {
		return out, domain.ErrFiscalNotConfigured
	}
	mc, problems, err := s.prepareManual(ctx, companyId, in)
	if err != nil {
		return out, err
	}
	out.Problems = append(out.Problems, problems...)
	out.Total = mc.build.ItemsTotal
	if n := mc.build.Nfe; n != nil {
		for _, it := range n.Items {
			out.Items = append(out.Items, ManualPreviewItem{Description: it.Description, CFOP: it.CFOP})
			if strings.HasPrefix(it.CFOP, "6") {
				out.Interstate = true
			}
		}
		if len(n.Items) > 0 && len(n.Items) == len(in.Items) {
			totals, more, err := s.nfeTotals(ctx, api, mc)
			if err != nil {
				return out, err
			}
			out.Problems = append(out.Problems, more...)
			if totals != nil {
				out.Totals = totals
				out.Total = totals.Total
				if msg := paymentShortfall(mc.build, totals.Total); msg != "" {
					out.Problems = append(out.Problems, msg)
				}
			}
		}
	} else if n := mc.build.Nfce; n != nil {
		for _, it := range n.Items {
			out.Items = append(out.Items, ManualPreviewItem{Description: it.Description, CFOP: it.CFOP})
		}
	}
	return out, nil
}

// EmitManual emite a NF-e ou NFC-e avulsa. O mesmo idempotency_key devolve a nota
// já criada (duplo clique). Nota com problema não é gravada: a tela corrige e reenvia.
func (s *Service) EmitManual(ctx context.Context, companyId, userId uuid.UUID, in domain.ManualDocumentInput) (domain.FiscalDocument, error) {
	api, ok := s.emission()
	if !ok {
		return domain.FiscalDocument{}, domain.ErrFiscalNotConfigured
	}
	key, err := uuid.Parse(strings.TrimSpace(in.IdempotencyKey))
	if err != nil {
		return domain.FiscalDocument{}, &ManualProblemsError{Problems: []string{"idempotency_key deve ser um UUID gerado pela tela"}}
	}
	idemKey := "manual-" + key.String()
	if existing, err := s.repo.GetFiscalDocumentByIdempotencyKey(ctx, db.GetFiscalDocumentByIdempotencyKeyParams{
		IdempotencyKey: idemKey, CompanyID: pgconv.ParseUUIDToPgType(companyId),
	}); err == nil {
		return toDocument(existing), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.FiscalDocument{}, err
	}

	mc, problems, err := s.prepareManual(ctx, companyId, in)
	if err != nil {
		return domain.FiscalDocument{}, err
	}
	if len(problems) > 0 {
		return domain.FiscalDocument{}, &ManualProblemsError{Problems: problems}
	}
	total := mc.build.ItemsTotal
	if mc.build.Nfe != nil {
		totals, more, err := s.nfeTotals(ctx, api, mc)
		switch {
		case len(more) > 0:
			return domain.FiscalDocument{}, &ManualProblemsError{Problems: more}
		case err == nil && totals != nil:
			total = totals.Total
			if msg := paymentShortfall(mc.build, total); msg != "" {
				return domain.FiscalDocument{}, &ManualProblemsError{Problems: []string{msg}}
			}
		}
		// API fora: segue com a soma dos itens; a emissão fica PENDING e o worker reenvia
	}

	payload, err := json.Marshal(mc.build.Request())
	if err != nil {
		return domain.FiscalDocument{}, err
	}
	doc, err := s.repo.CreateManualFiscalDocument(ctx, db.CreateManualFiscalDocumentParams{
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		Model:             int16(mc.build.Model),
		Environment:       string(mc.env),
		IdempotencyKey:    idemKey,
		RequestPayload:    string(payload),
		OperationNature:   pgconv.ParseStringToPgText(mc.build.OperationNature),
		RecipientName:     pgconv.ParseStringToPgText(mc.build.RecipientName),
		RecipientDocument: pgconv.ParseStringToPgText(mc.build.RecipientDocument),
		Total:             pgconv.Float64ToPgNumeric(total),
		CreatedBy:         pgconv.ParseUUIDToPgType(userId),
	})
	if isUniqueViolation(err) {
		// Outro envio com a mesma chave ganhou a corrida
		existing, getErr := s.repo.GetFiscalDocumentByIdempotencyKey(ctx, db.GetFiscalDocumentByIdempotencyKeyParams{
			IdempotencyKey: idemKey, CompanyID: pgconv.ParseUUIDToPgType(companyId),
		})
		if getErr != nil {
			return domain.FiscalDocument{}, getErr
		}
		return toDocument(existing), nil
	}
	if err != nil {
		return domain.FiscalDocument{}, err
	}

	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), EmitTimeout)
	defer cancel()
	doc, err = s.send(sendCtx, api, mc.env, mc.issuer, doc, mc.build.Request())
	if err != nil {
		return domain.FiscalDocument{}, err
	}
	return toDocument(doc), nil
}

// paymentShortfall: na NF-e os pagamentos cobrem o vNF (a mais vira troco), salvo "sem pagamento".
func paymentShortfall(b domain.ManualBuild, total float64) string {
	if b.NoPayment || math.Round(b.Paid*100) >= math.Round(total*100) {
		return ""
	}
	return fmt.Sprintf("os pagamentos (R$ %.2f) não cobrem o total da nota (R$ %.2f)", b.Paid, total)
}
