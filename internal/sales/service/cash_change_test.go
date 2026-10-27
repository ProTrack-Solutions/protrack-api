package service

import (
	"errors"
	"testing"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
)

func ptr(v float64) *float64 { return &v }

func TestCashChange(t *testing.T) {
	tests := []struct {
		name         string
		received     *float64
		method       enums.PaymentMethod
		total        float64
		downPayment  float64
		wantReceived *float64
		wantChange   *float64
		wantErr      bool
	}{
		{"sem valor recebido", nil, enums.PaymentMethodCash, 76.5, 0, nil, nil, false},
		{"dinheiro com troco", ptr(100), enums.PaymentMethodCash, 76.5, 0, ptr(100), ptr(23.5), false},
		{"dinheiro valor exato", ptr(76.5), enums.PaymentMethodCash, 76.5, 0, ptr(76.5), ptr(0), false},
		{"dinheiro insuficiente", ptr(50), enums.PaymentMethodCash, 76.5, 0, nil, nil, true},
		{"troco sobre a entrada", ptr(50), enums.PaymentMethodInstallments, 300, 30, ptr(50), ptr(20), false},
		{"a prazo sem entrada ignora", ptr(50), enums.PaymentMethodInstallments, 300, 0, nil, nil, false},
		{"entrada insuficiente", ptr(20), enums.PaymentMethodInstallments, 300, 30, nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received, change, err := cashChange(tt.received, tt.method, tt.total, tt.downPayment)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidSale) {
					t.Fatalf("esperava ErrInvalidSale, veio %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			check := func(label string, valid bool, value float64, want *float64) {
				if want == nil {
					if valid {
						t.Errorf("%s: esperava nulo, veio %v", label, value)
					}
					return
				}
				if !valid || value != *want {
					t.Errorf("%s: esperava %v, veio %v", label, *want, value)
				}
			}
			check("recebido", received.Valid, pgconv.PgNumericToFloat64(received), tt.wantReceived)
			check("troco", change.Valid, pgconv.PgNumericToFloat64(change), tt.wantChange)
		})
	}
}
