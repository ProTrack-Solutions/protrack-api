package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/mail"
	"strings"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/generator"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}

// SetPublisher liga a fila de e-mail (opcional: sem ela o envio responde indisponível)
func (s *Service) SetPublisher(p Publisher) {
	s.publisher = p
}

// EmailOrder envia o pedido ao fornecedor: corpo com os itens e o PDF anexo.
// A resposta do fornecedor vai para o e-mail da empresa, com cópia para quem
// enviou. A entrega é assíncrona (fila); o histórico registra a solicitação.
func (s *Service) EmailOrder(ctx context.Context, companyId, userId, orderId uuid.UUID, req domain.EmailOrderRequest) (domain.OrderDetail, error) {
	if s.publisher == nil {
		return domain.OrderDetail{}, domain.ErrEmailUnavailable
	}

	info, err := s.repo.GetPurchaseEmailContext(ctx, db.GetPurchaseEmailContextParams{
		OrderID:   pgconv.ParseUUIDToPgType(orderId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    pgconv.ParseUUIDToPgType(userId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrderDetail{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.OrderDetail{}, err
	}
	if !domain.ActionsFor(info.Status, 0).CanEmail {
		return domain.OrderDetail{}, fmt.Errorf("%w: só pedidos enviados e ainda não recebidos por completo podem ir por e-mail", domain.ErrOrderStatus)
	}

	vendorEmail, ok := validEmail(info.VendorEmail)
	if !ok {
		return domain.OrderDetail{}, domain.ErrVendorWithoutEmail
	}

	order, err := s.GetOrder(ctx, companyId, orderId)
	if err != nil {
		return domain.OrderDetail{}, err
	}

	updated := req.Kind == domain.EmailKindUpdated
	pdfData := purchaseOrderPDFData(order, info, updated)
	pdf, err := generator.GeneratePurchaseOrderPDF(pdfData)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	html, err := renderOrderEmail(order, info, updated)
	if err != nil {
		return domain.OrderDetail{}, err
	}

	// Respostas vão para a empresa; sem e-mail da empresa, para quem enviou
	companyEmail, hasCompanyEmail := validEmail(info.CompanyEmail)
	userEmail, hasUserEmail := validEmail(info.UserEmail)
	replyTo := companyEmail
	if !hasCompanyEmail && hasUserEmail {
		replyTo = userEmail
	}
	var cc []string
	if hasUserEmail && !strings.EqualFold(userEmail, vendorEmail) {
		cc = append(cc, userEmail)
	}

	subject := fmt.Sprintf("Pedido de compra nº %d — %s", order.Number, info.CompanyName)
	if updated {
		subject = fmt.Sprintf("Pedido de compra nº %d ATUALIZADO — %s", order.Number, info.CompanyName)
	}

	body, err := json.Marshal(events.PurchaseOrderEmail{
		CompanyID: companyId,
		OrderID:   orderId,
		To:        []string{vendorEmail},
		Cc:        cc,
		ReplyTo:   replyTo,
		FromName:  info.CompanyName + " via ProTrack",
		Subject:   subject,
		HTML:      html,
		Attachments: []events.EmailAttachment{{
			Filename:    fmt.Sprintf("pedido-de-compra-%d.pdf", order.Number),
			ContentType: "application/pdf",
			Data:        pdf.Bytes(),
		}},
	})
	if err != nil {
		return domain.OrderDetail{}, err
	}

	if err := s.publisher.Publish(ctx, events.RoutingKeyPurchaseOrderEmail, body); err != nil {
		log.Error().Err(err).Str("order_id", orderId.String()).Msg("falha ao publicar e-mail do pedido de compra")
		return domain.OrderDetail{}, domain.ErrEmailUnavailable
	}

	if err := s.repo.CreatePurchaseOrderEmail(ctx, db.CreatePurchaseOrderEmailParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		OrderID:   pgconv.ParseUUIDToPgType(orderId),
		Kind:      req.Kind,
		Recipient: vendorEmail,
		Cc:        pgconv.ParseStringToPgType(strings.Join(cc, ", ")),
		SentBy:    pgconv.ParseUUIDToPgType(userId),
	}); err != nil {
		// O e-mail já está na fila: só o histórico se perdeu
		log.Error().Err(err).Str("order_id", orderId.String()).Msg("e-mail do pedido enfileirado mas não registrado")
	}

	return s.GetOrder(ctx, companyId, orderId)
}

func validEmail(value string) (string, bool) {
	addr, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil || !strings.Contains(addr.Address, ".") {
		return "", false
	}
	return addr.Address, true
}

func purchaseOrderPDFData(order domain.OrderDetail, info db.GetPurchaseEmailContextRow, updated bool) generator.PurchaseOrderPDFData {
	data := generator.PurchaseOrderPDFData{
		Number:          order.Number,
		Updated:         updated,
		IssuedAt:        time.Now(),
		Notes:           derefString(order.Notes),
		CompanyName:     info.CompanyName,
		CompanyDocument: info.CompanyDocument,
		CompanyEmail:    info.CompanyEmail,
		CompanyPhone:    info.CompanyPhone,
		VendorName:      info.VendorName,
		BuyerName:       info.UserName,
		Total:           order.Total,
	}
	if order.ExpectedDate != nil {
		if t, err := time.Parse(dateLayout, *order.ExpectedDate); err == nil {
			data.ExpectedDate = t.Format("02/01/2006")
		}
	}
	for _, it := range order.Items {
		data.Items = append(data.Items, generator.PurchaseOrderPDFItem{
			Product:  it.ProductName,
			Barcode:  it.Barcode,
			Quantity: it.Quantity,
			UnitCost: it.UnitCost,
			Total:    it.Total,
		})
	}
	return data
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var orderEmailTemplate = template.Must(template.New("order").Funcs(template.FuncMap{
	"money": func(v float64) string { return brl(v) },
}).Parse(`<!doctype html>
<html lang="pt-BR"><body style="margin:0;padding:24px;background:#f4f5f7;font-family:Arial,Helvetica,sans-serif;color:#1f2937">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:640px;margin:0 auto;background:#ffffff;border-radius:12px;overflow:hidden">
<tr><td style="background:#4338ca;color:#ffffff;padding:20px 24px">
<div style="font-size:12px;opacity:.85">{{.Company}}</div>
<div style="font-size:20px;font-weight:bold;margin-top:4px">Pedido de compra nº {{.Number}}{{if .Updated}} (atualizado){{end}}</div>
</td></tr>
<tr><td style="padding:20px 24px;font-size:14px;line-height:1.5">
<p style="margin:0 0 12px">Olá, {{.Vendor}}.</p>
{{if .Updated}}<p style="margin:0 0 12px"><strong>Este pedido foi alterado</strong> e substitui a versão enviada anteriormente.</p>{{end}}
<p style="margin:0 0 12px">Segue o pedido de compra da <strong>{{.Company}}</strong>. O PDF está anexado a este e-mail.{{if .Expected}} Entrega prevista para <strong>{{.Expected}}</strong>.{{end}}</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;font-size:13px;margin:16px 0">
<tr style="background:#eef2ff;color:#374151">
<th align="left" style="padding:8px">Produto</th><th align="right" style="padding:8px">Qtd.</th><th align="right" style="padding:8px">Custo unit.</th><th align="right" style="padding:8px">Total</th></tr>
{{range .Items}}<tr style="border-top:1px solid #e5e7eb">
<td style="padding:8px">{{.ProductName}}</td><td align="right" style="padding:8px">{{.Quantity}}</td><td align="right" style="padding:8px">{{money .UnitCost}}</td><td align="right" style="padding:8px">{{money .Total}}</td></tr>
{{end}}<tr style="border-top:2px solid #4338ca;font-weight:bold">
<td style="padding:8px" colspan="3">Total</td><td align="right" style="padding:8px">{{money .Total}}</td></tr>
</table>
{{if .Notes}}<p style="margin:0 0 12px"><strong>Observações:</strong> {{.Notes}}</p>{{end}}
<p style="margin:16px 0 0">Para confirmar o pedido ou tirar dúvidas, basta responder este e-mail.</p>
<p style="margin:12px 0 0">Atenciosamente,<br>{{.Buyer}}<br>{{.Company}}{{if .Phone}} · {{.Phone}}{{end}}</p>
</td></tr>
<tr><td style="padding:12px 24px;background:#f9fafb;font-size:11px;color:#6b7280">Enviado pelo ProTrack em nome de {{.Company}}.</td></tr>
</table></body></html>`))

func renderOrderEmail(order domain.OrderDetail, info db.GetPurchaseEmailContextRow, updated bool) (string, error) {
	expected := ""
	if order.ExpectedDate != nil {
		if t, err := time.Parse(dateLayout, *order.ExpectedDate); err == nil {
			expected = t.Format("02/01/2006")
		}
	}
	buyer := info.UserName
	if buyer == "" {
		buyer = "Equipe de compras"
	}

	var buf bytes.Buffer
	err := orderEmailTemplate.Execute(&buf, map[string]any{
		"Company":  info.CompanyName,
		"Vendor":   info.VendorName,
		"Number":   order.Number,
		"Updated":  updated,
		"Expected": expected,
		"Items":    order.Items,
		"Total":    order.Total,
		"Notes":    derefString(order.Notes),
		"Buyer":    buyer,
		"Phone":    info.CompanyPhone,
	})
	return buf.String(), err
}

// brl 1234.5 → "R$ 1.234,50"
func brl(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	intPart, dec, _ := strings.Cut(s, ".")
	negative := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := "R$ " + b.String() + "," + dec
	if negative {
		out = "-" + out
	}
	return out
}
