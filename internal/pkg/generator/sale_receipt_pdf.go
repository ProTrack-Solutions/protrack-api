package generator

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/translate"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/go-pdf/fpdf"
)

const (
	receiptMargin     = 10.0
	receiptWidth      = 190.0 // A4 (210mm) menos as margens
	receiptLineHeight = 5.0
	receiptPageLimit  = 280.0 // a partir daqui os itens continuam na próxima página
)

// colunas da tabela de itens (somam receiptWidth)
var receiptColumns = []struct {
	title string
	width float64
	align string
}{
	{"CÓDIGO", 32, "L"},
	{"DESCRIÇÃO", 86, "L"},
	{"QTD", 20, "C"},
	{"VR. UNITÁRIO", 26, "R"},
	{"VR. TOTAL (R$)", 26, "R"},
}

var paymentMethodLabels = map[string]string{
	"cash":          "DINHEIRO",
	"credit_card":   "CARTÃO DE CRÉDITO",
	"debit_card":    "CARTÃO DE DÉBITO",
	"pix":           "PIX",
	"bank_transfer": "TRANSFERÊNCIA BANCÁRIA",
	"installments":  "A PRAZO",
	"other":         "OUTROS",
}

func GenerateSaleReceiptPDF(receipt domain.SaleReceipt) (*bytes.Buffer, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(receiptMargin, receiptMargin, receiptMargin)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	text := func(s string) string { return translate.PDFText(pdf, s) }

	// 1. CABEÇALHO DA EMPRESA
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(receiptWidth*0.6, 6, text(strings.ToUpper(receipt.Company.Name)), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(receiptWidth*0.4, 6, text(receipt.Company.Document), "", 1, "R", false, 0, "")
	pdf.CellFormat(receiptWidth*0.75, receiptLineHeight, text(strings.ToUpper(receipt.Company.Address)), "", 0, "L", false, 0, "")
	pdf.CellFormat(receiptWidth*0.25, receiptLineHeight, text(receipt.Company.Phone), "", 1, "R", false, 0, "")
	receiptDivider(pdf)

	// 2. TÍTULO E DADOS DA VENDA
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(receiptWidth, 7, text("RECIBO DE VENDA"), "", 1, "C", false, 0, "")

	if strings.EqualFold(receipt.Status, "canceled") {
		pdf.SetFont("Arial", "B", 12)
		pdf.SetTextColor(200, 0, 0)
		pdf.SetDrawColor(200, 0, 0)
		pdf.CellFormat(receiptWidth, 8, text("VENDA CANCELADA"), "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(0, 0, 0)
		pdf.Ln(1)
	}

	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(receiptWidth*0.3, receiptLineHeight, text("REALIZADA EM: "+receipt.SaleAt.Format("02/01/2006 15:04")), "", 0, "L", false, 0, "")
	pdf.CellFormat(receiptWidth*0.7, receiptLineHeight, text("VENDEDOR(A): "+strings.ToUpper(receipt.SellerName)), "", 1, "R", false, 0, "")
	pdf.CellFormat(receiptWidth, receiptLineHeight, text("CÓDIGO DA VENDA: "+strings.ToUpper(receipt.SaleID.String())), "", 1, "L", false, 0, "")
	receiptDivider(pdf)

	// 3. CLIENTE
	customerName := receipt.Customer.Name
	if customerName == "" {
		customerName = "CONSUMIDOR FINAL"
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(receiptWidth*0.6, receiptLineHeight, text("CLIENTE: "+strings.ToUpper(customerName)), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(receiptWidth*0.4, receiptLineHeight, text("CPF/CNPJ: "+receipt.Customer.Document), "", 1, "R", false, 0, "")
	pdf.CellFormat(receiptWidth*0.6, receiptLineHeight, text("ENDEREÇO: "+strings.ToUpper(receipt.Customer.Address)), "", 0, "L", false, 0, "")
	pdf.CellFormat(receiptWidth*0.4, receiptLineHeight, text("RG/IE: "+receipt.Customer.Rg), "", 1, "R", false, 0, "")
	cityState := strings.TrimSpace(receipt.Customer.City + " " + receipt.Customer.State)
	pdf.CellFormat(receiptWidth, receiptLineHeight, text("CIDADE: "+strings.ToUpper(cityState)), "", 1, "L", false, 0, "")
	receiptDivider(pdf)

	// 4. ITENS
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(receiptWidth, 6, text("** ITENS VENDIDOS **"), "", 1, "C", false, 0, "")
	receiptItemsHeader(pdf)

	var totalQuantity int32
	pdf.SetFont("Arial", "", 9)
	for _, item := range receipt.Items {
		if pdf.GetY()+receiptLineHeight > receiptPageLimit {
			pdf.AddPage()
			receiptItemsHeader(pdf)
			pdf.SetFont("Arial", "", 9)
		}

		description := translate.PrepareLabelText(item.Description, 48)
		quantity := fmt.Sprintf("%d %s", item.Quantity, item.Unit)
		values := []string{item.Code, description, quantity, formatBRL(item.UnitPrice), formatBRL(item.Total)}
		for i, col := range receiptColumns {
			pdf.CellFormat(col.width, receiptLineHeight, text(values[i]), "", 0, col.align, false, 0, "")
		}
		pdf.Ln(receiptLineHeight)
		totalQuantity += item.Quantity
	}

	// 5. TOTAIS E PAGAMENTO (precisa de ~45mm; se não couber, vai para a próxima página)
	if pdf.GetY()+45 > receiptPageLimit {
		pdf.AddPage()
	}
	receiptDivider(pdf)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(receiptWidth/3, receiptLineHeight, text(fmt.Sprintf("ITENS: %d", len(receipt.Items))), "", 0, "L", false, 0, "")
	pdf.CellFormat(receiptWidth/3, receiptLineHeight, text(fmt.Sprintf("QTD: %d", totalQuantity)), "", 0, "C", false, 0, "")
	pdf.CellFormat(receiptWidth/3, receiptLineHeight, text("SUBTOTAL: "+formatBRL(receipt.Subtotal)), "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	if receipt.DiscountAmount > 0 {
		pdf.CellFormat(receiptWidth, receiptLineHeight, text("DESCONTO: R$ "+formatBRL(receipt.DiscountAmount)), "", 1, "R", false, 0, "")
	}
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(receiptWidth, 6, text("TOTAL: R$ "+formatBRL(receipt.TotalAmount)), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	paymentLabel, ok := paymentMethodLabels[receipt.PaymentMethod]
	if !ok {
		paymentLabel = strings.ToUpper(receipt.PaymentMethod)
	}
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(receiptWidth, receiptLineHeight, text("FORMA DE PAGAMENTO: "+paymentLabel), "", 1, "L", false, 0, "")
	if receipt.DownPayment > 0 {
		pdf.CellFormat(receiptWidth, receiptLineHeight, text("ENTRADA: R$ "+formatBRL(receipt.DownPayment)), "", 1, "L", false, 0, "")
	}

	installments := max(receipt.Installments, 1)
	installmentValue := (receipt.TotalAmount - receipt.DownPayment) / float64(installments)
	pdf.CellFormat(receiptWidth, receiptLineHeight, text(fmt.Sprintf("PARCELAS: %d x R$ %s", installments, formatBRL(installmentValue))), "", 1, "L", false, 0, "")
	receiptDivider(pdf)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(receiptWidth, receiptLineHeight, text("IMPRESSÃO: "+receipt.PrintedAt.Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, fmt.Errorf("falha ao gerar buffer do PDF: %w", err)
	}

	return &buffer, nil
}

func receiptItemsHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	for _, col := range receiptColumns {
		pdf.CellFormat(col.width, 6, translate.PDFText(pdf, col.title), "B", 0, col.align, false, 0, "")
	}
	pdf.Ln(7)
}

func receiptDivider(pdf *fpdf.Fpdf) {
	pdf.Ln(1)
	y := pdf.GetY()
	pdf.SetDrawColor(120, 120, 120)
	pdf.Line(receiptMargin, y, receiptMargin+receiptWidth, y)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Ln(2)
}

// formatBRL formata no padrão brasileiro sem o símbolo (ex: 1.234,56).
func formatBRL(value float64) string {
	negative := value < 0
	cents := int64(math.Round(math.Abs(value) * 100))
	integer := fmt.Sprintf("%d", cents/100)

	var grouped strings.Builder
	for i, digit := range integer {
		if i > 0 && (len(integer)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}

	result := fmt.Sprintf("%s,%02d", grouped.String(), cents%100)
	if negative {
		result = "-" + result
	}
	return result
}
