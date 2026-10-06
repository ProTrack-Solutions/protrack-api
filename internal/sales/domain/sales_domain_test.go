package domain

import (
	"errors"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
)

func validCashSale() CreateSaleRequest {
	return CreateSaleRequest{
		PaymentMethod: enums.PaymentMethodCash,
		Items:         []CreateSaleItemRequest{{ProductID: uuid.New(), Quantity: 1}},
	}
}

func validInstallmentSale() CreateSaleRequest {
	req := validCashSale()
	req.PaymentMethod = enums.PaymentMethodInstallments
	req.CustomerID = uuid.New()
	req.InstallmentsCount = 3
	req.DueDays = 10
	return req
}

func TestValidateCreateSaleRequest(t *testing.T) {
	productID := uuid.New()

	tests := []struct {
		name    string
		mutate  func(*CreateSaleRequest)
		base    func() CreateSaleRequest
		wantErr bool
	}{
		{name: "venda à vista válida", base: validCashSale},
		{name: "venda a prazo válida", base: validInstallmentSale},
		{name: "forma de pagamento vazia", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.PaymentMethod = "" }, wantErr: true},
		{name: "forma de pagamento desconhecida", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.PaymentMethod = "boleto" }, wantErr: true},
		{name: "a prazo sem cliente", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.CustomerID = uuid.Nil }, wantErr: true},
		{name: "a prazo sem parcelas", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.InstallmentsCount = 0 }, wantErr: true},
		{name: "a prazo com parcelas acima do limite", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.InstallmentsCount = MaxInstallments + 1 }, wantErr: true},
		{name: "a prazo sem dia de vencimento", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.DueDays = 0 }, wantErr: true},
		{name: "a prazo com dia de vencimento 32", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.DueDays = 32 }, wantErr: true},
		{name: "entrada negativa", base: validInstallmentSale, mutate: func(r *CreateSaleRequest) { r.Prohibited = -1 }, wantErr: true},
		{name: "desconto negativo", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.DiscountAmount = -5 }, wantErr: true},
		{name: "desconto em reais acima de 100", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.DiscountAmount = 150.50 }, wantErr: false},
		{name: "sem itens", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.Items = nil }, wantErr: true},
		{name: "item sem produto", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.Items[0].ProductID = uuid.Nil }, wantErr: true},
		{name: "item com quantidade zero", base: validCashSale, mutate: func(r *CreateSaleRequest) { r.Items[0].Quantity = 0 }, wantErr: true},
		{
			name: "produto repetido",
			base: validCashSale,
			mutate: func(r *CreateSaleRequest) {
				r.Items = []CreateSaleItemRequest{{ProductID: productID, Quantity: 1}, {ProductID: productID, Quantity: 2}}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.base()
			if tt.mutate != nil {
				tt.mutate(&req)
			}

			err := ValidateCreateSaleRequest(req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("esperava nil, obteve: %v", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidSale) {
				t.Fatalf("esperava ErrInvalidSale, obteve: %v", err)
			}
		})
	}
}
