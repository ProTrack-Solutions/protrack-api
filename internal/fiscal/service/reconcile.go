package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

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

// ---------- Cancelamento ----------

// CancelDocument cancela a nota (NFC-e ou NF-e) autorizada na SEFAZ (dentro do prazo da UF).
func (s *Service) CancelDocument(ctx context.Context, companyId, id uuid.UUID, reason string) (domain.FiscalDocument, error) {
	api, ok := s.emission()
	if !ok {
		return domain.FiscalDocument{}, domain.ErrFiscalNotConfigured
	}
	reason = strings.Join(strings.Fields(reason), " ")
	if n := utf8.RuneCountInString(reason); n < 15 || n > 255 {
		return domain.FiscalDocument{}, domain.ErrInvalidCancelReason
	}
	d, err := s.getDocument(ctx, companyId, id)
	if err != nil {
		return domain.FiscalDocument{}, err
	}
	if d.Status == domain.StatusCancelled {
		return toDocument(d), nil
	}
	if d.Status != domain.StatusAuthorized || !d.ExternalID.Valid {
		return domain.FiscalDocument{}, domain.ErrDocumentNotCancellable
	}
	result, err := api.CancelDocument(ctx, client.Environment(d.Environment), int(d.Model), pgconv.PgUUIDToUUID(d.ExternalID).String(), reason)
	if err != nil {
		return domain.FiscalDocument{}, translate(err)
	}
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	switch {
	case result.DocumentStatus == domain.StatusCancelled:
		d, err = s.repo.MarkFiscalDocumentCancelled(saveCtx, db.MarkFiscalDocumentCancelledParams{
			ID: d.ID, CancelReason: pgtype.Text{String: reason, Valid: true},
		})
	case result.Event.Status == client.StatusRejected:
		msg := "a SEFAZ recusou o cancelamento"
		if result.Event.RejectReason != nil {
			msg += ": " + *result.Event.RejectReason
		}
		return domain.FiscalDocument{}, &APIError{Status: 422, Message: msg}
	default:
		// Ainda processando: o webhook *.cancelled (ou o worker) conclui
		err = s.repo.SetFiscalDocumentError(saveCtx, db.SetFiscalDocumentErrorParams{
			ID: d.ID, ErrorMessage: pgtype.Text{String: "cancelamento enviado; aguardando a SEFAZ", Valid: true},
		})
		if err == nil {
			d, err = s.getDocument(saveCtx, companyId, id)
		}
	}
	if err != nil {
		return domain.FiscalDocument{}, err
	}
	return toDocument(d), nil
}

// ---------- Conferência (worker) ----------

const reconcileBatch = 20

// ReconcileOnce reenvia notas PENDING e consulta as que aguardam a SEFAZ e não
// mudaram há minAge. Devolve quantas foram processadas.
func (s *Service) ReconcileOnce(ctx context.Context, minAge time.Duration) (int, error) {
	api, ok := s.emission()
	if !ok {
		return 0, nil
	}
	docs, err := s.repo.ClaimFiscalDocumentsToReconcile(ctx, db.ClaimFiscalDocumentsToReconcileParams{
		MinAgeSeconds: int32(minAge.Seconds()), BatchSize: reconcileBatch,
	})
	if err != nil {
		return 0, err
	}
	for _, d := range docs {
		if err := s.reconcile(ctx, api, d); err != nil {
			log.Warn().Err(err).Str("fiscal_document", pgconv.PgUUIDToUUID(d.ID).String()).Msg("nota fiscal: conferência falhou")
		}
	}
	return len(docs), nil
}

func (s *Service) reconcile(ctx context.Context, api EmissionAPI, d db.FiscalDocument) error {
	env := client.Environment(d.Environment)
	if !s.api.Configured(env) {
		return nil
	}
	if !d.ExternalID.Valid {
		// Nunca chegou à API fiscal (ou a resposta se perdeu): reenvia o mesmo pedido
		settings, err := s.loadSettings(ctx, s.repo, pgconv.PgUUIDToUUID(d.CompanyID))
		if err != nil {
			return err
		}
		issuer := settings.IssuerIDTest
		if env == client.EnvLive {
			issuer = settings.IssuerIDLive
		}
		if !issuer.Valid {
			return nil
		}
		// Reenvia exatamente o corpo gravado (a mesma Idempotency-Key exige o mesmo pedido)
		var probe struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal([]byte(d.RequestPayload), &probe); err != nil || len(probe.Items) == 0 {
			_, err = s.repo.ApplyFiscalDocumentResult(ctx, db.ApplyFiscalDocumentResultParams{
				ID: d.ID, Status: domain.StatusFailed,
				ErrorMessage: pgtype.Text{String: "pedido da nota perdido; emita de novo", Valid: true},
			})
			return err
		}
		_, err = s.send(ctx, api, env, pgconv.PgUUIDToUUID(issuer).String(), d, json.RawMessage(d.RequestPayload))
		return err
	}
	doc, err := api.GetDocument(ctx, env, int(d.Model), pgconv.PgUUIDToUUID(d.ExternalID).String())
	if err != nil {
		_, markErr := s.repo.MarkFiscalDocumentRetry(ctx, db.MarkFiscalDocumentRetryParams{
			ID: d.ID, ErrorMessage: d.ErrorMessage,
		})
		return errors.Join(err, markErr)
	}
	// Contingência com resultado incerto: a API fiscal emitiu outra nota no lugar
	if doc.Status == client.StatusSuperseded && doc.ReplacedByID != nil {
		replacement, err := api.GetDocument(ctx, env, int(d.Model), *doc.ReplacedByID)
		if err != nil {
			return err
		}
		doc = replacement
	}
	_, err = s.applyIfNewer(ctx, d, doc)
	return err
}

// applyIfNewer grava a situação vinda da API sem regredir nota já cancelada
// (webhooks e consultas podem chegar fora de ordem).
func (s *Service) applyIfNewer(ctx context.Context, current db.FiscalDocument, doc client.Document) (db.FiscalDocument, error) {
	if current.Status == domain.StatusCancelled && doc.Status != client.StatusCancelled {
		return current, nil
	}
	if doc.Status == client.StatusCancelled {
		return s.repo.MarkFiscalDocumentCancelled(ctx, db.MarkFiscalDocumentCancelledParams{ID: current.ID})
	}
	return applyDocument(ctx, s.repo, current.ID, doc)
}

// StartReconciler roda a conferência a cada intervalo até o contexto acabar.
func (s *Service) StartReconciler(ctx context.Context, every time.Duration) {
	if _, ok := s.emission(); !ok {
		return
	}
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, every)
				if n, err := s.ReconcileOnce(runCtx, every); err != nil {
					log.Error().Err(err).Msg("NFC-e: worker de conferência")
				} else if n > 0 {
					log.Info().Int("notas", n).Msg("NFC-e: notas conferidas")
				}
				cancel()
			}
		}
	}()
}

// ---------- Webhook ----------

var ErrInvalidSignature = errors.New("assinatura do webhook inválida")

// HandleWebhook confere a assinatura e aplica o evento. Evento repetido (mesmo id)
// e nota desconhecida são aceitos sem efeito.
func (s *Service) HandleWebhook(ctx context.Context, signature string, body []byte, secrets ...string) error {
	valid := false
	for _, secret := range secrets {
		if client.VerifyWebhookSignature(secret, signature, body, s.now(), client.DefaultSignatureTolerance) {
			valid = true
			break
		}
	}
	if !valid {
		return ErrInvalidSignature
	}
	var payload client.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	eventId, err := uuid.Parse(payload.ID)
	if err != nil {
		return err
	}
	kind, event, _ := strings.Cut(payload.Type, ".")
	if (kind != "nfce" && kind != "nfe") || payload.Data.Document == nil {
		return nil // certificado, cobrança e carta de correção não interessam aqui
	}
	return s.inTx(ctx, func(q *repository.Repository) error {
		n, err := q.InsertFiscalWebhookEvent(ctx, db.InsertFiscalWebhookEventParams{
			ID: pgconv.ParseUUIDToPgType(eventId), Type: payload.Type,
		})
		if err != nil || n == 0 {
			return err
		}
		ext, err := uuid.Parse(payload.Data.Document.ID)
		if err != nil {
			return nil
		}
		current, err := q.GetFiscalDocumentByExternalId(ctx, pgconv.ParseUUIDToPgType(ext))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // nota substituída em contingência: o worker acha pela original
		}
		if err != nil {
			return err
		}
		switch event {
		case "cancelled":
			var reason pgtype.Text
			if c := payload.Data.Cancellation; c != nil && c.Reason != nil {
				reason = pgtype.Text{String: *c.Reason, Valid: true}
			}
			_, err = q.MarkFiscalDocumentCancelled(ctx, db.MarkFiscalDocumentCancelledParams{ID: current.ID, CancelReason: reason})
		case "cancellation_rejected":
			msg := "a SEFAZ recusou o cancelamento"
			if c := payload.Data.Cancellation; c != nil && c.RejectReason != nil {
				msg += ": " + *c.RejectReason
			}
			err = q.SetFiscalDocumentError(ctx, db.SetFiscalDocumentErrorParams{
				ID: current.ID, ErrorMessage: pgtype.Text{String: msg, Valid: true},
			})
		default:
			if current.Status == domain.StatusCancelled {
				return nil
			}
			_, err = applyDocument(ctx, q, current.ID, *payload.Data.Document)
		}
		return err
	})
}
