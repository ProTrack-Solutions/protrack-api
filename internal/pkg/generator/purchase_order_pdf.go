package generator

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// PurchaseOrderPDFData dados do pedido de compra impresso para o fornecedor
type PurchaseOrderPDFData struct {
	Number          int32
	Updated         bool
	IssuedAt        time.Time
	ExpectedDate    string // DD/MM/AAAA ou vazio
	Notes           string
	CompanyName     string
	CompanyDocument string
	CompanyEmail    string
	CompanyPhone    string
	VendorName      string
	BuyerName       string
	Items           []PurchaseOrderPDFItem
	Total           float64
}

type PurchaseOrderPDFItem struct {
	Product  string
	Barcode  string
	Quantity int32
	UnitCost float64
	Total    float64
}

// GeneratePurchaseOrderPDF pedido de compra em A4 retrato: cabeçalho com a
// empresa compradora e o fornecedor, tabela de itens e total.
func GeneratePurchaseOrderPDF(d PurchaseOrderPDFData) (*bytes.Buffer, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AliasNbPages("{nb}")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pageWidth, _ := pdf.GetPageSize()
	width := pageWidth - 24
	cols := []float64{width * 0.46, width * 0.18, width * 0.09, width * 0.13, width * 0.14}

	pdf.SetFooterFunc(func() {
		pdf.SetY(-11)
		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(width/2, 4, tr(fmt.Sprintf("Pedido de compra nº %d · emitido em %s", d.Number, d.IssuedAt.Format("02/01/2006 15:04"))), "", 0, "L", false, 0, "")
		pdf.CellFormat(width/2, 4, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	header := func() {
		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(67, 56, 202)
		pdf.SetTextColor(255, 255, 255)
		for i, label := range []string{"Produto", "Código", "Qtd.", "Custo unit.", "Total"} {
			align := "L"
			if i >= 2 {
				align = "R"
			}
			pdf.CellFormat(cols[i], 7, tr(label), "", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.AddPage()

	title := fmt.Sprintf("Pedido de compra nº %d", d.Number)
	if d.Updated {
		title += " (atualizado)"
	}
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(width, 9, tr(title), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Comprador e fornecedor lado a lado
	half := width/2 - 3
	y := pdf.GetY()
	box := func(x float64, label string, lines []string) {
		pdf.SetXY(x, y)
		pdf.SetFillColor(238, 242, 255)
		pdf.RoundedRect(x, y, half, 30, 2, "1234", "F")
		pdf.SetXY(x+3, y+3)
		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(half-6, 4, tr(label), "", 2, "L", false, 0, "")
		for i, line := range lines {
			if line == "" {
				continue
			}
			if i == 0 {
				pdf.SetFont("Arial", "B", 10)
				pdf.SetTextColor(17, 24, 39)
			} else {
				pdf.SetFont("Arial", "", 8)
				pdf.SetTextColor(55, 65, 81)
			}
			pdf.CellFormat(half-6, 5, tr(fitText(pdf, tr, line, half-6)), "", 2, "L", false, 0, "")
		}
	}
	box(12, "COMPRADOR", []string{d.CompanyName, docLine(d.CompanyDocument), d.CompanyEmail, d.CompanyPhone})
	box(12+half+6, "FORNECEDOR", []string{d.VendorName, "Responsável pela compra: " + d.BuyerName, deliveryLine(d.ExpectedDate)})
	pdf.SetXY(12, y+34)

	header()
	pdf.SetHeaderFunc(func() {
		if pdf.PageNo() > 1 {
			header()
		}
	})

	pdf.SetFont("Arial", "", 8)
	for i, it := range d.Items {
		pdf.SetTextColor(31, 41, 55)
		if i%2 == 1 {
			pdf.SetFillColor(249, 250, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(cols[0], 6.5, tr(fitText(pdf, tr, it.Product, cols[0])), "", 0, "L", true, 0, "")
		pdf.CellFormat(cols[1], 6.5, it.Barcode, "", 0, "L", true, 0, "")
		pdf.CellFormat(cols[2], 6.5, formatNumberBR(float64(it.Quantity), 0), "", 0, "R", true, 0, "")
		pdf.CellFormat(cols[3], 6.5, "R$ "+formatNumberBR(it.UnitCost, 2), "", 0, "R", true, 0, "")
		pdf.CellFormat(cols[4], 6.5, "R$ "+formatNumberBR(it.Total, 2), "", 0, "R", true, 0, "")
		pdf.Ln(-1)
	}

	var quantity int32
	for _, it := range d.Items {
		quantity += it.Quantity
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(224, 231, 255)
	pdf.SetTextColor(17, 24, 39)
	itemsLabel := "itens"
	if len(d.Items) == 1 {
		itemsLabel = "item"
	}
	pdf.CellFormat(cols[0]+cols[1], 8, tr(fmt.Sprintf("Total · %d %s", len(d.Items), itemsLabel)), "", 0, "L", true, 0, "")
	pdf.CellFormat(cols[2], 8, formatNumberBR(float64(quantity), 0), "", 0, "R", true, 0, "")
	pdf.CellFormat(cols[3], 8, "", "", 0, "R", true, 0, "")
	pdf.CellFormat(cols[4], 8, "R$ "+formatNumberBR(d.Total, 2), "", 1, "R", true, 0, "")

	if d.Notes != "" {
		pdf.Ln(4)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(width, 5, tr("Observações"), "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(55, 65, 81)
		pdf.MultiCell(width, 4.5, tr(d.Notes), "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

func docLine(document string) string {
	if document == "" {
		return ""
	}
	return "CNPJ/CPF: " + document
}

func deliveryLine(expected string) string {
	if expected == "" {
		return ""
	}
	return "Entrega prevista: " + expected
}
