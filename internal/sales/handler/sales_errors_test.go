package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/gin-gonic/gin"
)

func TestRespondSaleError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"dados inválidos", fmt.Errorf("%w: adicione pelo menos um produto", domain.ErrInvalidSale), http.StatusBadRequest, "dados inválidos para a venda: adicione pelo menos um produto"},
		{"atualização inválida", fmt.Errorf("%w: informe a quantidade de parcelas", domain.ErrInvalidSaleUpdate), http.StatusBadRequest, "dados inválidos para atualizar a venda: informe a quantidade de parcelas"},
		{"venda não encontrada", domain.ErrSaleNotFound, http.StatusNotFound, "venda não encontrada"},
		{"cliente não encontrado", domain.ErrSaleCustomerNotFound, http.StatusNotFound, "cliente não encontrado"},
		{"produto não encontrado", fmt.Errorf("%w (item 1)", domain.ErrSaleProductNotFound), http.StatusNotFound, "produto não encontrado (item 1)"},
		{"estoque insuficiente", fmt.Errorf("%w para o produto X", domain.ErrInsufficientStock), http.StatusConflict, "estoque insuficiente para o produto X"},
		{"venda já cancelada", domain.ErrSaleAlreadyCanceled, http.StatusConflict, "venda já foi cancelada"},
		{"erro inesperado não vaza detalhes", errors.New("pq: connection refused"), http.StatusInternalServerError, "erro interno ao processar a venda, tente novamente"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			respondSaleError(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Fatalf("status esperado %d, obteve %d", tt.wantStatus, w.Code)
			}

			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("resposta inválida: %v", err)
			}
			if body["error"] != tt.wantBody {
				t.Fatalf("mensagem esperada %q, obteve %q", tt.wantBody, body["error"])
			}
		})
	}
}
