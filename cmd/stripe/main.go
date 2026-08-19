package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/stripe/stripe-go/v86"
)

// Valor sugerido de partida: R$0,10 por mensagem excedente (8 centavos).
// Baseado num markup de ~40-60% sobre o custo estimado de mensagens utility/service
// da Meta no Brasil — ajuste livremente, é só um parâmetro do Price.
const overagePriceUnitAmountCents = 10

func main() {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		log.Fatal("defina STRIPE_SECRET_KEY antes de rodar este script")
	}

	ctx := context.Background()
	sc := stripe.NewClient(apiKey)

	// 1. BillingMeter — define como os eventos de uso são agregados no período de cobrança.
	meterParams := &stripe.BillingMeterCreateParams{
		DisplayName: stripe.String("Mensagens WhatsApp excedentes"),
		EventName:   stripe.String("whatsapp_message_overage"),
		DefaultAggregation: &stripe.BillingMeterCreateDefaultAggregationParams{
			Formula: stripe.String(string(stripe.BillingMeterDefaultAggregationFormulaSum)),
		},
	}

	newMeter, err := sc.V1BillingMeters.Create(ctx, meterParams)
	if err != nil {
		log.Fatalf("erro ao criar BillingMeter: %v", err)
	}
	fmt.Printf("Meter criado com sucesso — id: %s, event_name: %s\n", newMeter.ID, newMeter.EventName)

	// 2. Product — todo Price precisa estar associado a um Product.
	// Se você já tiver um Product genérico de add-ons, pode reaproveitar o ID dele
	// e pular esta etapa, passando o ID direto na criação do Price abaixo.
	productParams := &stripe.ProductCreateParams{
		Name:        stripe.String("WhatsApp - Mensagem excedente"),
		Description: stripe.String("Cobrança por mensagem que ultrapassar a franquia mensal do plano no WABA compartilhado da ProTrack"),
	}

	newProduct, err := sc.V1Products.Create(ctx, productParams)
	if err != nil {
		log.Fatalf("erro ao criar Product: %v", err)
	}
	fmt.Printf("Product criado com sucesso — id: %s\n", newProduct.ID)

	// 3. Price — do tipo metered, atrelado ao Meter criado no passo 1.
	priceParams := &stripe.PriceCreateParams{
		Currency:   stripe.String(string(stripe.CurrencyBRL)),
		Product:    stripe.String(newProduct.ID),
		UnitAmount: stripe.Int64(overagePriceUnitAmountCents),
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval:  stripe.String(string(stripe.PriceRecurringIntervalMonth)),
			UsageType: stripe.String(string(stripe.PriceRecurringUsageTypeMetered)),
			Meter:     stripe.String(newMeter.ID),
		},
	}

	newPrice, err := sc.V1Prices.Create(ctx, priceParams)
	if err != nil {
		log.Fatalf("erro ao criar Price: %v", err)
	}
	fmt.Printf("Price criado com sucesso — id: %s (R$%.2f por mensagem)\n", newPrice.ID, float64(overagePriceUnitAmountCents)/100)

	fmt.Println()
	fmt.Println("Guarde estes valores como env vars do protrack-api:")
	fmt.Printf("STRIPE_WHATSAPP_OVERAGE_METER_ID=%s\n", newMeter.ID)
	fmt.Printf("STRIPE_WHATSAPP_OVERAGE_PRICE_ID=%s\n", newPrice.ID)
}
