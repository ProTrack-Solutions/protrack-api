package generator

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/label/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/translate"
	"github.com/go-pdf/fpdf"
)

func GenerateLabelSheetPDF(products []domain.GenetareTagProduct, layout domain.LabelLayout) (*bytes.Buffer, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	maxPerPage := layout.Columns * layout.Rows
	tempImgPrefix := "temp_barcode_"

	for index, product := range products {
		if index > 0 && index%maxPerPage == 0 {
			pdf.AddPage()
		}

		pageIndex := index % maxPerPage
		col := pageIndex % layout.Columns
		row := pageIndex / layout.Columns

		posX := layout.MarginLeft + float64(col)*(layout.Width+layout.SpacingX)
		posY := layout.MarginTop + float64(row)*(layout.Height+layout.SpacingY)

		// Borda externa da etiqueta
		pdf.SetDrawColor(180, 180, 180)
		pdf.Rect(posX, posY, layout.Width, layout.Height, "D")

		// 1. NOME DO PRODUTO (Máx 18 chars para caber na largura de 38.1mm)
		pdf.SetFont("Arial", "B", 6.5)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(posX+1.5, posY+1.2)

		productName := translate.PrepareLabelText(product.Name, 18)
		pdf.CellFormat(layout.Width-3, 2.5, translate.PDFText(pdf, productName), "", 0, "L", false, 0, "")

		// 2. LABEL "PREÇO"
		pdf.SetFont("Arial", "B", 5)
		pdf.SetTextColor(50, 50, 50)
		pdf.SetXY(posX+1.5, posY+4.0)
		pdf.CellFormat(12, 2, translate.PDFText(pdf, "PREÇO"), "", 0, "L", false, 0, "")

		// 3. VALOR (Ex: R$ 3,50)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(posX+1.5, posY+6.0)

		formattedPrice := fmt.Sprintf("R$ %.2f", product.Amount)
		formattedPrice = strings.Replace(formattedPrice, ".", ",", 1)
		pdf.CellFormat(layout.Width-3, 3, formattedPrice, "", 0, "L", false, 0, "")

		// 4. IMAGEM DO CÓDIGO DE BARRAS (Proporção ajustada para a altura de 21.2mm)
		tempImgPath := fmt.Sprintf("%s%s.png", tempImgPrefix, product.Barcode)
		err := generateBarcodePNG(product.Barcode, tempImgPath)
		if err == nil {
			bcWidth := layout.Width - 4
			bcHeight := float64(6.5)
			bcPosX := posX + 2.0
			bcPosY := posY + 9.8

			pdf.ImageOptions(tempImgPath, bcPosX, bcPosY, bcWidth, bcHeight, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
			os.Remove(tempImgPath)
		}

		// 5. NÚMERO DO CÓDIGO DE BARRAS (Fonte 5.5pt para alinhamento na base)
		pdf.SetFont("Arial", "", 5.5)
		pdf.SetTextColor(50, 50, 50)
		pdf.SetXY(posX+2.0, posY+17.0)
		pdf.CellFormat(layout.Width-4, 2, product.Barcode, "", 0, "L", false, 0, "")
	}

	var buffer bytes.Buffer
	err := pdf.Output(&buffer)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar buffer do PDF: %w", err)
	}

	return &buffer, nil
}
