package generator

import (
	"fmt"
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
)

func generateBarcodePNG(code string, outputPath string) error {
	bc, err := code128.Encode(code)
	if err != nil {
		return fmt.Errorf("falha ao codificar o código de barras: %w", err)
	}

	scaledBC, err := barcode.Scale(bc, 300, 100)
	if err != nil {
		return fmt.Errorf("falha ao redimensionar o código de barras: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, scaledBC)
}
