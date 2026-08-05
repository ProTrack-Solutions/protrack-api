package translate

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

// PDFText converte strings em UTF-8 para ISO-8859-1 (Latin-1)
// para evitar quebras em caracteres acentuados no gofpdf.
func PDFText(pdf *fpdf.Fpdf, text string) string {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	return tr(text)
}

func PrepareLabelText(name string, maxLen int) string {
	upper := strings.ToUpper(name)
	runes := []rune(upper)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return upper
}
