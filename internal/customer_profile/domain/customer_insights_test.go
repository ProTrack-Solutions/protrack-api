package domain

import (
	"testing"
	"time"
)

func TestBuildInsights(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, loc) }

	// Sextas (2, 9, 16, 23/10/2026) à tarde, a cada 7 dias, quase sempre no Pix
	moments := []SaleMoment{
		{At: at(2, 15), Total: 100, Method: "pix"},
		{At: at(9, 16), Total: 80, Method: "pix"},
		{At: at(16, 14), Total: 120, Method: "cash"},
		{At: at(23, 17), Total: 90, Method: "pix"},
	}
	items := []PurchasedItem{
		{Name: "Camiseta Básica – Preta – M", Model: "Camiseta Básica", Category: "Camisetas", Size: "m", Color: "Preta", Quantity: 2, Total: 100, At: at(2, 15)},
		{Name: "Camiseta Básica – Branca – M", Model: "Camiseta Básica", Category: "Camisetas", Size: "M", Color: "Branca", Quantity: 1, Total: 50, At: at(16, 14)},
		{Name: "Calça Jeans", Category: "Calças", Size: "40", Color: "preta", Quantity: 1, Total: 150, At: at(9, 16)},
	}

	ins := BuildInsights(items, moments, at(28, 10), loc)

	if len(ins.TopProducts) != 2 || ins.TopProducts[0].Name != "Camiseta Básica" || ins.TopProducts[0].Quantity != 3 {
		t.Fatalf("produtos: %+v", ins.TopProducts)
	}
	if ins.TopCategories[0].Name != "Calças" || ins.TopCategories[0].Share != 50 {
		t.Fatalf("categorias: %+v", ins.TopCategories)
	}
	if ins.FavoriteSize == nil || ins.FavoriteSize.Value != "M" || ins.FavoriteSize.Quantity != 3 {
		t.Fatalf("tamanho: %+v", ins.FavoriteSize)
	}
	if ins.FavoriteColor == nil || ins.FavoriteColor.Quantity != 3 {
		t.Fatalf("cor: %+v", ins.FavoriteColor)
	}
	if ins.BestWeekday == nil || *ins.BestWeekday != int(time.Friday) || ins.BestPeriod != "afternoon" {
		t.Fatalf("quando: %v %s", ins.BestWeekday, ins.BestPeriod)
	}
	if ins.FavoritePayment == nil || ins.FavoritePayment.Method != "pix" || ins.FavoritePayment.Share != 75 {
		t.Fatalf("pagamento: %+v", ins.FavoritePayment)
	}
	if ins.AverageIntervalDays == nil || *ins.AverageIntervalDays != 7 || *ins.DaysSinceLast != 5 || *ins.NextExpectedAt != "2026-10-30" {
		t.Fatalf("ritmo: %v %v %v", *ins.AverageIntervalDays, *ins.DaysSinceLast, *ins.NextExpectedAt)
	}
	if ins.Rhythm != RhythmRegular {
		t.Fatalf("ritmo: %s", ins.Rhythm)
	}

	if r := BuildInsights(items, moments, at(23, 10).AddDate(0, 0, 14), loc).Rhythm; r != RhythmLate {
		t.Errorf("14 dias sem comprar (média 7): esperava late, veio %s", r)
	}
	if r := BuildInsights(items, moments, at(23, 10).AddDate(0, 0, 30), loc).Rhythm; r != RhythmAway {
		t.Errorf("30 dias sem comprar: esperava away, veio %s", r)
	}
}

func TestBuildInsightsSemCompras(t *testing.T) {
	ins := BuildInsights(nil, nil, time.Now(), time.UTC)
	if ins.Rhythm != RhythmNone || ins.BestWeekday != nil || ins.FavoritePayment != nil || len(ins.Weekdays) != 7 {
		t.Fatalf("sem compras: %+v", ins)
	}
	one := BuildInsights(nil, []SaleMoment{{At: time.Now(), Total: 10, Method: "pix"}}, time.Now(), time.UTC)
	if one.Rhythm != RhythmNew || one.AverageIntervalDays != nil || one.BestWeekday != nil {
		t.Fatalf("uma compra: %+v", one)
	}
}
