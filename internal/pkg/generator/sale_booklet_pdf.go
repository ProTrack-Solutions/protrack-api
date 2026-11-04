package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/translate"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/go-pdf/fpdf"
)

// Carnê em A4 retrato: 4 lâminas por página, cada uma com o canhoto da loja à
// esquerda e a via do cliente à direita, separados por linhas de corte.
const (
	bookletMargin       = 10.0
	bookletWidth        = 190.0 // A4 (210mm) menos as margens
	bookletSlipHeight   = 69.0
	bookletSlipsPerPage = 4
	bookletStubWidth    = 56.0
	bookletGap          = 6.0 // espaço da linha de corte entre canhoto e via do cliente
	bookletBoxHeight    = 10.0
)

func GenerateSaleBookletPDF(booklet domain.SaleBooklet) (*bytes.Buffer, error) {
	if len(booklet.Installments) == 0 {
		return nil, fmt.Errorf("carnê sem parcelas")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(bookletMargin, bookletMargin, bookletMargin)
	pdf.SetAutoPageBreak(false, 0)

	for i, inst := range booklet.Installments {
		slot := i % bookletSlipsPerPage
		if slot == 0 {
			pdf.AddPage()
		}
		top := bookletMargin + float64(slot)*bookletSlipHeight
		bookletSlip(pdf, booklet, inst, top)

		// linha de corte horizontal entre as lâminas
		if slot < bookletSlipsPerPage-1 && i < len(booklet.Installments)-1 {
			bookletCutLine(pdf, bookletMargin, top+bookletSlipHeight, bookletMargin+bookletWidth, top+bookletSlipHeight)
		}
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, fmt.Errorf("falha ao gerar buffer do PDF: %w", err)
	}
	return &buffer, nil
}

func bookletSlip(pdf *fpdf.Fpdf, b domain.SaleBooklet, inst domain.SaleBookletInstallment, top float64) {
	text := func(s string) string { return translate.PDFText(pdf, s) }

	slipTop := top + 3
	slipHeight := bookletSlipHeight - 6
	customerName := strings.ToUpper(b.Customer.Name)
	if customerName == "" {
		customerName = "CONSUMIDOR FINAL"
	}
	saleCode := strings.ToUpper(b.SaleID.String())
	installmentLabel := fmt.Sprintf("%d/%d", inst.Number, max(inst.Total, inst.Number))
	dueDate := inst.DueDate.Format("02/01/2006")
	amount := "R$ " + formatBRL(inst.Amount)
	paid := bookletPaidLabel(inst)

	// ---------- CANHOTO (VIA DA LOJA) ----------
	x := bookletMargin
	w := bookletStubWidth
	pdf.SetDrawColor(0, 0, 0)
	pdf.Rect(x, slipTop, w, slipHeight, "D")

	pdf.SetXY(x, slipTop+1)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(w, 4, text("CANHOTO - VIA DA LOJA"), "", 2, "C", false, 0, "")
	pdf.SetFont("Arial", "", 6.5)
	pdf.CellFormat(w, 3.5, text(translate.PrepareLabelText(b.Company.Name, 34)), "", 2, "C", false, 0, "")

	y := slipTop + 9
	half := w / 2
	bookletBox(pdf, x, y, half, "PARCELA", installmentLabel, 9)
	bookletBox(pdf, x+half, y, half, "VENCIMENTO", dueDate, 9)
	y += bookletBoxHeight
	bookletBox(pdf, x, y, w, "VALOR DA PARCELA", amount, 10)
	y += bookletBoxHeight
	bookletBox(pdf, x, y, w, "CLIENTE", translate.PrepareLabelText(customerName, 30), 7.5)
	y += bookletBoxHeight
	bookletBox(pdf, x, y, half, "VENDA", saleCode[:8], 8)
	bookletBox(pdf, x+half, y, half, "VALOR PAGO", paid, 6.5)
	y += bookletBoxHeight
	bookletBox(pdf, x, y, half, "DATA PGTO.", "", 8)
	bookletBox(pdf, x+half, y, half, "VISTO", "", 8)

	// linha de corte vertical entre canhoto e via do cliente
	cutX := x + w + bookletGap/2
	bookletCutLine(pdf, cutX, slipTop, cutX, slipTop+slipHeight)

	// ---------- VIA DO CLIENTE ----------
	x = bookletMargin + bookletStubWidth + bookletGap
	w = bookletWidth - bookletStubWidth - bookletGap
	pdf.Rect(x, slipTop, w, slipHeight, "D")

	// cabeçalho: empresa à esquerda, título à direita
	pdf.SetXY(x+2, slipTop+1.5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(w*0.62, 5, text(translate.PrepareLabelText(b.Company.Name, 40)), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(w*0.38-4, 5, text("CARNÊ DE PAGAMENTO"), "", 1, "R", false, 0, "")
	pdf.SetX(x + 2)
	pdf.SetFont("Arial", "", 7)
	companyLine := joinNonEmpty(" | ", b.Company.Document, b.Company.Phone)
	pdf.CellFormat(w*0.62, 3.5, text(companyLine), "", 0, "L", false, 0, "")
	pdf.CellFormat(w*0.38-4, 3.5, text("VIA DO CLIENTE"), "", 1, "R", false, 0, "")
	if b.Company.Address != "" {
		pdf.SetX(x + 2)
		pdf.CellFormat(w-4, 3.5, text(translate.PrepareLabelText(b.Company.Address, 80)), "", 1, "L", false, 0, "")
	}

	y = slipTop + 14
	third := w / 3
	bookletBox(pdf, x, y, third, "PARCELA", installmentLabel, 11)
	bookletBox(pdf, x+third, y, third, "VENCIMENTO", dueDate, 11)
	bookletBox(pdf, x+2*third, y, third, "VALOR DA PARCELA", amount, 11)
	y += bookletBoxHeight

	bookletBox(pdf, x, y, w*0.68, "CLIENTE", translate.PrepareLabelText(customerName, 48), 8.5)
	bookletBox(pdf, x+w*0.68, y, w*0.32, "CPF/CNPJ", b.Customer.Document, 8.5)
	y += bookletBoxHeight

	bookletBox(pdf, x, y, w*0.5, "CÓDIGO DA VENDA", saleCode, 7)
	bookletBox(pdf, x+w*0.5, y, w*0.25, "DATA DA COMPRA", b.SaleAt.Format("02/01/2006"), 8.5)
	bookletBox(pdf, x+w*0.75, y, w*0.25, "TOTAL DA COMPRA", "R$ "+formatBRL(b.TotalAmount), 8.5)
	y += bookletBoxHeight

	bookletBox(pdf, x, y, third, "VALOR PAGO", paid, 9)
	bookletBox(pdf, x+third, y, third, "DATA DO PAGAMENTO", "", 9)
	bookletBox(pdf, x+2*third, y, third, "AUTENTICAÇÃO / ASSINATURA", "", 9)
	y += bookletBoxHeight

	pdf.SetXY(x+2, y+0.8)
	pdf.SetFont("Arial", "I", 6.5)
	note := "Pague até a data de vencimento. Guarde este comprovante após o pagamento."
	if b.DownPayment > 0 {
		note = fmt.Sprintf("Entrada paga na compra: R$ %s. %s", formatBRL(b.DownPayment), note)
	}
	pdf.CellFormat(w-4, 3.5, text(note), "", 0, "L", false, 0, "")
}

// bookletBox desenha um campo com rótulo pequeno no topo e o valor abaixo.
func bookletBox(pdf *fpdf.Fpdf, x, y, w float64, label, value string, valueSize float64) {
	pdf.Rect(x, y, w, bookletBoxHeight, "D")
	pdf.SetXY(x+1, y+0.6)
	pdf.SetFont("Arial", "", 5.5)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(w-2, 2.8, translate.PDFText(pdf, label), "", 2, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	if value != "" {
		pdf.SetFont("Arial", "B", valueSize)
		pdf.CellFormat(w-2, 6, translate.PDFText(pdf, value), "", 0, "L", false, 0, "")
	}
}

// bookletPaidLabel preenche o "valor pago" quando o carnê é reimpresso depois
// de pagamentos; parcela em aberto fica em branco para anotação no balcão.
func bookletPaidLabel(inst domain.SaleBookletInstallment) string {
	switch strings.ToLower(inst.Status) {
	case "paid":
		return "R$ " + formatBRL(inst.Amount) + " (PAGA)"
	case "partial":
		return "R$ " + formatBRL(inst.Amount-inst.Balance) + " (PARCIAL)"
	}
	return ""
}

func bookletCutLine(pdf *fpdf.Fpdf, x1, y1, x2, y2 float64) {
	pdf.SetDrawColor(150, 150, 150)
	pdf.SetDashPattern([]float64{1.5, 1.2}, 0)
	pdf.Line(x1, y1, x2, y2)
	pdf.SetDashPattern([]float64{}, 0)
	pdf.SetDrawColor(0, 0, 0)
}

func joinNonEmpty(sep string, parts ...string) string {
	filled := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			filled = append(filled, p)
		}
	}
	return strings.Join(filled, sep)
}
