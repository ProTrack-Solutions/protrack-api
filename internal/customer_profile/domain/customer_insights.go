package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// PurchasedItem é um item comprado pelo cliente, já sem o que ele devolveu
type PurchasedItem struct {
	Name     string
	Model    string
	Category string
	Size     string
	Color    string
	Quantity int32
	Total    float64
	At       time.Time
}

// SaleMoment é uma compra do cliente (quando, quanto e como pagou)
type SaleMoment struct {
	At     time.Time
	Total  float64
	Method string
}

type TopProduct struct {
	Name         string    `json:"name"`
	Quantity     int32     `json:"quantity"`
	Total        float64   `json:"total"`
	LastBoughtAt time.Time `json:"last_bought_at"`
}

type CategoryShare struct {
	Name  string  `json:"name"`
	Total float64 `json:"total"`
	// Percentual do total comprado
	Share float64 `json:"share"`
}

type Favorite struct {
	Value    string  `json:"value"`
	Quantity int32   `json:"quantity"`
	Share    float64 `json:"share"`
}

type WeekdayStat struct {
	// 0 = domingo ... 6 = sábado
	Weekday int     `json:"weekday"`
	Count   int     `json:"count"`
	Total   float64 `json:"total"`
}

type PeriodStat struct {
	// morning (5h-12h), afternoon (12h-18h), evening (18h-24h), night (0h-5h)
	Period string `json:"period"`
	Count  int    `json:"count"`
}

type PaymentShare struct {
	Method string  `json:"method"`
	Count  int     `json:"count"`
	Share  float64 `json:"share"`
}

// Ritmo de compra do cliente em relação ao intervalo médio dele
const (
	RhythmNone    = "none"    // nunca comprou
	RhythmNew     = "new"     // uma compra só
	RhythmRegular = "regular" // dentro do intervalo de costume
	RhythmLate    = "late"    // passou do costume (até 3x o intervalo)
	RhythmAway    = "away"    // sumiu (mais de 3x o intervalo ou 6 meses)
)

type Insights struct {
	TopProducts     []TopProduct    `json:"top_products"`
	TopCategories   []CategoryShare `json:"top_categories"`
	FavoriteSize    *Favorite       `json:"favorite_size"`
	FavoriteColor   *Favorite       `json:"favorite_color"`
	Weekdays        []WeekdayStat   `json:"weekdays"`
	BestWeekday     *int            `json:"best_weekday"`
	Periods         []PeriodStat    `json:"periods"`
	BestPeriod      string          `json:"best_period"`
	FavoritePayment *PaymentShare   `json:"favorite_payment"`
	// Dias, em média, entre uma compra e outra
	AverageIntervalDays *float64 `json:"average_interval_days"`
	DaysSinceLast       *int     `json:"days_since_last"`
	// Data em que, pelo costume, ele deveria voltar (YYYY-MM-DD)
	NextExpectedAt *string `json:"next_expected_at"`
	Rhythm         string  `json:"rhythm"`
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func percentOf(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return round1(part / total * 100)
}

func period(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 18:
		return "afternoon"
	case hour >= 18:
		return "evening"
	default:
		return "night"
	}
}

// favorite devolve o valor mais comprado (por quantidade); vazio não conta
func favorite(counts map[string]int32, labels map[string]string) *Favorite {
	var total int32
	best, bestQty := "", int32(0)
	for k, q := range counts {
		total += q
		if q > bestQty || (q == bestQty && k < best) {
			best, bestQty = k, q
		}
	}
	if best == "" {
		return nil
	}
	return &Favorite{Value: labels[best], Quantity: bestQty, Share: percentOf(float64(bestQty), float64(total))}
}

// BuildInsights traça o perfil de compra: o que, quando e como o cliente
// compra, e se ele está dentro do ritmo de costume. Datas no fuso da empresa.
func BuildInsights(items []PurchasedItem, moments []SaleMoment, now time.Time, loc *time.Location) Insights {
	out := Insights{
		TopProducts:   []TopProduct{},
		TopCategories: []CategoryShare{},
		Weekdays:      make([]WeekdayStat, 7),
		Periods: []PeriodStat{
			{Period: "morning"}, {Period: "afternoon"}, {Period: "evening"}, {Period: "night"},
		},
		Rhythm: RhythmNone,
	}
	for i := range out.Weekdays {
		out.Weekdays[i].Weekday = i
	}

	// O que compra: produto (agrupado pelo modelo da grade), categoria, tamanho e cor
	products := map[string]*TopProduct{}
	categories := map[string]float64{}
	sizes, colors := map[string]int32{}, map[string]int32{}
	sizeLabels, colorLabels := map[string]string{}, map[string]string{}
	var itemsTotal float64
	for _, it := range items {
		name := it.Name
		if it.Model != "" {
			name = it.Model
		}
		key := strings.ToLower(name)
		p, ok := products[key]
		if !ok {
			p = &TopProduct{Name: name}
			products[key] = p
		}
		p.Quantity += it.Quantity
		p.Total += it.Total
		if it.At.After(p.LastBoughtAt) {
			p.LastBoughtAt = it.At
		}

		categories[it.Category] += it.Total
		itemsTotal += it.Total

		if s := strings.TrimSpace(it.Size); s != "" {
			k := strings.ToUpper(s)
			sizes[k] += it.Quantity
			sizeLabels[k] = k
		}
		if c := strings.TrimSpace(it.Color); c != "" {
			k := strings.ToLower(c)
			colors[k] += it.Quantity
			colorLabels[k] = c
		}
	}

	for _, p := range products {
		p.Total = round2(p.Total)
		out.TopProducts = append(out.TopProducts, *p)
	}
	sort.Slice(out.TopProducts, func(i, j int) bool {
		a, b := out.TopProducts[i], out.TopProducts[j]
		if a.Quantity != b.Quantity {
			return a.Quantity > b.Quantity
		}
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		return a.Name < b.Name
	})
	if len(out.TopProducts) > 5 {
		out.TopProducts = out.TopProducts[:5]
	}

	for name, total := range categories {
		out.TopCategories = append(out.TopCategories, CategoryShare{Name: name, Total: round2(total), Share: percentOf(total, itemsTotal)})
	}
	sort.Slice(out.TopCategories, func(i, j int) bool {
		a, b := out.TopCategories[i], out.TopCategories[j]
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		return a.Name < b.Name
	})
	if len(out.TopCategories) > 4 {
		out.TopCategories = out.TopCategories[:4]
	}

	out.FavoriteSize = favorite(sizes, sizeLabels)
	out.FavoriteColor = favorite(colors, colorLabels)

	// Quando e como compra
	methods := map[string]int{}
	for _, m := range moments {
		local := m.At.In(loc)
		wd := &out.Weekdays[int(local.Weekday())]
		wd.Count++
		wd.Total += m.Total
		for i := range out.Periods {
			if out.Periods[i].Period == period(local.Hour()) {
				out.Periods[i].Count++
			}
		}
		methods[m.Method]++
	}
	for i := range out.Weekdays {
		out.Weekdays[i].Total = round2(out.Weekdays[i].Total)
	}

	// Dia e período preferidos só com pelo menos 2 compras nele
	best := -1
	for i, wd := range out.Weekdays {
		if wd.Count >= 2 && (best < 0 || wd.Count > out.Weekdays[best].Count) {
			best = i
		}
	}
	if best >= 0 {
		out.BestWeekday = &best
	}
	bestPeriod := -1
	for i, p := range out.Periods {
		if p.Count >= 2 && (bestPeriod < 0 || p.Count > out.Periods[bestPeriod].Count) {
			bestPeriod = i
		}
	}
	if bestPeriod >= 0 {
		out.BestPeriod = out.Periods[bestPeriod].Period
	}

	if len(methods) > 0 {
		var method string
		count := 0
		for m, c := range methods {
			if c > count || (c == count && m < method) {
				method, count = m, c
			}
		}
		out.FavoritePayment = &PaymentShare{Method: method, Count: count, Share: percentOf(float64(count), float64(len(moments)))}
	}

	// Ritmo: dias de compra distintos, intervalo médio e quanto tempo faz da última
	if len(moments) == 0 {
		return out
	}
	day := func(t time.Time) time.Time {
		l := t.In(loc)
		return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	}
	days := []time.Time{}
	for _, m := range moments {
		d := day(m.At)
		if len(days) == 0 || !days[len(days)-1].Equal(d) {
			days = append(days, d)
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	today := day(now)
	last := days[len(days)-1]
	since := int(math.Round(today.Sub(last).Hours() / 24))
	out.DaysSinceLast = &since

	if len(days) < 2 {
		out.Rhythm = RhythmNew
		return out
	}

	avg := last.Sub(days[0]).Hours() / 24 / float64(len(days)-1)
	avg = math.Max(avg, 1)
	rounded := round1(avg)
	out.AverageIntervalDays = &rounded
	next := last.AddDate(0, 0, int(math.Round(avg))).Format("2006-01-02")
	out.NextExpectedAt = &next

	switch s := float64(since); {
	case s > 180 || s > avg*3:
		out.Rhythm = RhythmAway
	case s > avg*1.5:
		out.Rhythm = RhythmLate
	default:
		out.Rhythm = RhythmRegular
	}
	return out
}
