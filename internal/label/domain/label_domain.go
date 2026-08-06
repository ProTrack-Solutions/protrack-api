package domain

import "github.com/google/uuid"

type GenetareTagProduct struct {
	Name    string
	Amount  float64
	Barcode string
}

// LabelLayout define as dimensões físicas e a grade de layout para a folha A4
type LabelLayout struct {
	Width      float64 // Largura da etiqueta em mm
	Height     float64 // Altura da etiqueta em mm
	Columns    int     // Quantidade de colunas por folha
	Rows       int     // Quantidade de linhas por folha
	MarginLeft float64 // Margem esquerda da folha em mm
	MarginTop  float64 // Margem superior da folha em mm
	SpacingX   float64 // Espaçamento horizontal entre etiquetas em mm
	SpacingY   float64 // Espaçamento vertical entre etiquetas em mm
}

var Layout5x13 = LabelLayout{
	Width:      38.1,
	Height:     21.2,
	Columns:    5,
	Rows:       13,
	MarginLeft: 4.75, // Centraliza perfeitamente na largura
	MarginTop:  10.7, // Centraliza perfeitamente na altura
	SpacingX:   2.5,
	SpacingY:   0.0, // Necessário para comportar as 13 linhas na A4
}

type GenetareTagProductRequest struct {
	ProductId uuid.UUID `json:"product_id"`
}
