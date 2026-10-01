package demo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	mrand "math/rand/v2"
	"strings"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// historyMonths é quantos meses completos de histórico a demo gera (além do mês atual).
const historyMonths = 6

var ErrNoActivePlan = errors.New("nenhum plano ativo encontrado para a assinatura da empresa demo")

type SeedResult struct {
	Products          int
	Customers         int
	Sales             int
	SaleItems         int
	Receivables       int
	BillsPayable      int
	PaymentsHistories int
}

// Seed apaga todos os dados da empresa demo e recria a partir do catálogo.
// É idempotente: pode rodar quantas vezes quiser (o reset diário usa ela).
// A empresa e o usuário admin mantêm os mesmos IDs, então sessões abertas
// continuam válidas depois do reset.
func Seed(ctx context.Context, pool *pgxpool.Pool) (SeedResult, error) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.Local
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return SeedResult{}, err
	}
	defer tx.Rollback(ctx)

	s := &seeder{
		tx:  tx,
		now: time.Now().In(loc),
		loc: loc,
		// Semente fixa: os dados ficam iguais a cada reset (relativos à data atual).
		rng: mrand.New(mrand.NewPCG(2026, 10)),
	}

	steps := []func(context.Context) error{
		s.clear,
		s.company,
		s.departmentsAndUsers,
		s.subscription,
		s.settings,
		s.paymentMethods,
		s.catalog,
		s.customers,
		s.sales,
		s.billsPayable,
		s.announcement,
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return SeedResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SeedResult{}, err
	}
	return s.result, nil
}

type seeder struct {
	tx     pgx.Tx
	now    time.Time
	loc    *time.Location
	rng    *mrand.Rand
	result SeedResult

	paymentMethodIDs map[string]uuid.UUID // por type (pix, cash...)
	categoryIDs      map[string]uuid.UUID // categorias de produto
	products         []seededProduct
	productWeight    int
	customerIDs      []uuid.UUID
	salesUserIDs     []uuid.UUID
}

type seededProduct struct {
	productSeed
	ID uuid.UUID
}

func (s *seeder) clear(ctx context.Context) error {
	// Ordem importa: sales/customers/users têm FKs com ON DELETE RESTRICT.
	stmts := []string{
		`DELETE FROM fiscal_invoices WHERE company_id = $1`,
		`DELETE FROM payment_history WHERE company_id = $1`,
		`DELETE FROM accounts_receivable WHERE company_id = $1`,
		`DELETE FROM sales WHERE company_id = $1`,
		`DELETE FROM bills_payable WHERE company_id = $1`,
		`DELETE FROM products WHERE company_id = $1`,
		`DELETE FROM product_categories WHERE company_id = $1`,
		`DELETE FROM customers WHERE company_id = $1`,
		`DELETE FROM vendors WHERE company_id = $1`,
		`DELETE FROM bill_categories WHERE company_id = $1`,
		`DELETE FROM payment_methods WHERE company_id = $1`,
		`DELETE FROM company_settings WHERE company_id = $1`,
		`DELETE FROM announcements WHERE company_id = $1`,
		`DELETE FROM invoice_history WHERE company_id = $1`,
		`DELETE FROM subscriptions WHERE company_id = $1`,
		`DELETE FROM subscription_payment_methods WHERE company_id = $1`,
		`DELETE FROM whatsapp_messages WHERE company_id = $1`,
		`DELETE FROM whatsapp_usage_monthly WHERE company_id = $1`,
		`DELETE FROM company_whatsapp_configs WHERE company_id = $1`,
		`DELETE FROM company_certificates WHERE company_id = $1`,
		`DELETE FROM users WHERE company_id = $1 AND id <> $2`,
		`UPDATE users SET department_id = NULL WHERE company_id = $1 AND id = $2`,
		`DELETE FROM departments WHERE company_id = $1`,
	}
	for _, stmt := range stmts {
		args := []any{CompanyID}
		if strings.Contains(stmt, "$2") {
			args = append(args, UserID)
		}
		if _, err := s.tx.Exec(ctx, stmt, args...); err != nil {
			return fmt.Errorf("limpando dados da demo (%s): %w", stmt, err)
		}
	}
	return nil
}

func (s *seeder) company(ctx context.Context) error {
	_, err := s.tx.Exec(ctx, `
		INSERT INTO companies (
			id, name, trade_name, document, document_type, email, phone, website,
			address_street, address_number, address_neighborhood, address_city,
			address_state, address_zipcode, address_country, status, timezone,
			external_company_id
		) VALUES ($1, $2, $3, $4, 'CNPJ', $5, $6, $7, $8, $9, $10, $11, $12, $13, 'BR', 'ACTIVE', 'America/Sao_Paulo', 'demo')
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			trade_name = EXCLUDED.trade_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			status = 'ACTIVE',
			deleted_at = NULL,
			deleted_by = NULL,
			updated_at = NOW()`,
		CompanyID,
		"Casa & Obra Materiais de Construção Ltda",
		"Casa & Obra",
		cnpj("45123987"),
		"contato@casaeobra.example.com",
		"1932415500",
		"casaeobra.example.com",
		"Avenida das Amoreiras", "1520", "Parque Itália", "Campinas", "SP", "13036225",
	)
	if err != nil {
		return fmt.Errorf("criando empresa demo: %w", err)
	}
	return nil
}

func (s *seeder) departmentsAndUsers(ctx context.Context) error {
	departments := []struct {
		Name, Description string
		Modules           []string
	}{
		{"Administração", "Gestão geral da loja", nil},
		{"Vendas", "Balcão e atendimento", []string{"sales", "customers"}},
		{"Estoque", "Recebimento e conferência de mercadorias", []string{"inventory"}},
		{"Financeiro", "Contas a pagar, receber e fluxo de caixa", []string{"financial", "reports"}},
	}

	deptIDs := make(map[string]uuid.UUID, len(departments))
	for _, d := range departments {
		id := uuid.New()
		deptIDs[d.Name] = id
		if _, err := s.tx.Exec(ctx,
			`INSERT INTO departments (id, company_id, name, description, created_by) VALUES ($1, $2, $3, $4, $5)`,
			id, CompanyID, d.Name, d.Description, UserID,
		); err != nil {
			return fmt.Errorf("criando departamento %s: %w", d.Name, err)
		}
		if len(d.Modules) > 0 {
			// Só vincula módulos que existem no banco (a tabela modules é populada à parte).
			if _, err := s.tx.Exec(ctx,
				`INSERT INTO department_modules (department_id, module_code)
				 SELECT $1, code FROM modules WHERE code = ANY($2)`,
				id, d.Modules,
			); err != nil {
				return fmt.Errorf("vinculando módulos ao departamento %s: %w", d.Name, err)
			}
		}
	}

	// Ninguém faz login com senha nesses usuários (a demo entra por /auth/demo),
	// então a senha é aleatória e descartada.
	hash, err := randomPasswordHash()
	if err != nil {
		return err
	}

	_, err = s.tx.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, role, status, company_id, department_id)
		VALUES ($1, 'Visitante Demo', $2, $3, 'ADMIN', 'ACTIVE', $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			role = 'ADMIN',
			status = 'ACTIVE',
			company_id = EXCLUDED.company_id,
			department_id = EXCLUDED.department_id,
			deleted_at = NULL,
			deleted_by = NULL,
			updated_at = NOW()`,
		UserID, UserEmail, hash, CompanyID, deptIDs["Administração"],
	)
	if err != nil {
		return fmt.Errorf("criando usuário demo: %w", err)
	}

	staff := []struct{ Name, Email, Dept string }{
		{"Luciana Prado", "luciana@demo.protrack.com.br", "Vendas"},
		{"Marcos Tavares", "marcos@demo.protrack.com.br", "Vendas"},
		{"Sérgio Antunes", "sergio@demo.protrack.com.br", "Estoque"},
		{"Fernanda Lacerda", "fernanda@demo.protrack.com.br", "Financeiro"},
	}
	s.salesUserIDs = []uuid.UUID{UserID}
	for _, u := range staff {
		id := uuid.New()
		if _, err := s.tx.Exec(ctx, `
			INSERT INTO users (id, name, email, password_hash, role, status, company_id, department_id, created_by)
			VALUES ($1, $2, $3, $4, 'USER', 'ACTIVE', $5, $6, $7)`,
			id, u.Name, u.Email, hash, CompanyID, deptIDs[u.Dept], UserID,
		); err != nil {
			return fmt.Errorf("criando usuário %s: %w", u.Email, err)
		}
		if u.Dept == "Vendas" {
			s.salesUserIDs = append(s.salesUserIDs, id)
		}
	}
	return nil
}

func (s *seeder) subscription(ctx context.Context) error {
	var planID uuid.UUID
	err := s.tx.QueryRow(ctx,
		`SELECT id FROM plans WHERE active ORDER BY price_cents DESC LIMIT 1`,
	).Scan(&planID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoActivePlan
	}
	if err != nil {
		return fmt.Errorf("buscando plano da demo: %w", err)
	}

	// Sem external_subscription_id/payment_method: a assinatura nunca passa
	// pelo Stripe. O fim do período fica longe para o worker de cobrança
	// (ListSubscriptionsDueOn) nunca pegar essa empresa.
	_, err = s.tx.Exec(ctx, `
		INSERT INTO subscriptions (company_id, plan_id, status, current_period_start, current_period_end)
		VALUES ($1, $2, 'active', $3, $4)`,
		CompanyID, planID, s.now, s.now.AddDate(10, 0, 0),
	)
	if err != nil {
		return fmt.Errorf("criando assinatura demo: %w", err)
	}
	return nil
}

func (s *seeder) settings(ctx context.Context) error {
	for key, value := range domain.DefaultSettings {
		// WhatsApp desligado: os clientes da demo são fictícios e o worker de
		// vendas vencidas não pode mandar mensagem para esses telefones.
		if key == enums.IsWhatsappActive {
			value = false
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if _, err := s.tx.Exec(ctx,
			`INSERT INTO company_settings (company_id, key, value) VALUES ($1, $2, $3)`,
			CompanyID, string(key), string(raw),
		); err != nil {
			return fmt.Errorf("criando configuração %s: %w", key, err)
		}
	}
	return nil
}

func (s *seeder) paymentMethods(ctx context.Context) error {
	methods := []struct{ Name, Type string }{
		{"Dinheiro", "cash"},
		{"Pix", "pix"},
		{"Cartão de Crédito", "credit_card"},
		{"Cartão de Débito", "debit_card"},
		{"Transferência Bancária", "bank_transfer"},
		{"Crediário", "installments"},
	}
	s.paymentMethodIDs = make(map[string]uuid.UUID, len(methods))
	for _, m := range methods {
		id := uuid.New()
		s.paymentMethodIDs[m.Type] = id
		if _, err := s.tx.Exec(ctx,
			`INSERT INTO payment_methods (id, company_id, name, type) VALUES ($1, $2, $3, $4)`,
			id, CompanyID, m.Name, m.Type,
		); err != nil {
			return fmt.Errorf("criando forma de pagamento %s: %w", m.Name, err)
		}
	}
	return nil
}

func (s *seeder) catalog(ctx context.Context) error {
	s.categoryIDs = make(map[string]uuid.UUID, len(productCategories))
	for _, c := range productCategories {
		id := uuid.New()
		s.categoryIDs[c.Name] = id
		if _, err := s.tx.Exec(ctx,
			`INSERT INTO product_categories (id, company_id, name, color, created_by) VALUES ($1, $2, $3, $4, $5)`,
			id, CompanyID, c.Name, c.Color, UserID,
		); err != nil {
			return fmt.Errorf("criando categoria %s: %w", c.Name, err)
		}
	}

	created := s.historyStart().AddDate(0, 0, -10)
	rows := make([][]any, 0, len(products))
	for i, p := range products {
		id := uuid.New()
		s.products = append(s.products, seededProduct{productSeed: p, ID: id})
		s.productWeight += p.Weight

		var quantity any = p.Quantity
		if p.SellInBulk {
			quantity = nil
		}
		rows = append(rows, []any{
			id, CompanyID, s.categoryIDs[p.Category], p.Name, ean13(fmt.Sprintf("789%09d", 100000+i)),
			quantity, p.Cost, p.Price, p.Unit, p.SellInBulk, UserID, created, created,
		})
	}
	if err := insertRows(ctx, s.tx, "products", []string{
		"id", "company_id", "category_id", "name", "barcode",
		"quantity", "cost_price", "sale_price", "unit", "sell_in_bulk", "created_by", "created_at", "updated_at",
	}, rows); err != nil {
		return fmt.Errorf("criando produtos: %w", err)
	}
	s.result.Products = len(rows)

	vendorRows := make([][]any, 0, len(vendors))
	for i, v := range vendors {
		vendorRows = append(vendorRows, []any{
			uuid.New(), CompanyID, v.Name, cnpj(fmt.Sprintf("3014%04d", 1201+i)), v.Email, v.Phone,
			"13040000", "Rodovia Anhanguera, km 98", fmt.Sprintf("%d", 100+i*35), "Distrito Industrial", v.City, "SP",
		})
	}
	if err := insertRows(ctx, s.tx, "vendors", []string{
		"id", "company_id", "name", "tax_id", "email", "phone",
		"postal_code", "address_line_1", "number", "neighborhood", "city", "state",
	}, vendorRows); err != nil {
		return fmt.Errorf("criando fornecedores: %w", err)
	}

	for _, c := range billCategories {
		if _, err := s.tx.Exec(ctx,
			`INSERT INTO bill_categories (company_id, name, description) VALUES ($1, $2, $3)`,
			CompanyID, c.Name, c.Description,
		); err != nil {
			return fmt.Errorf("criando categoria de conta %s: %w", c.Name, err)
		}
	}
	return nil
}

func (s *seeder) customers(ctx context.Context) error {
	rows := make([][]any, 0, len(customerNames))
	for i, c := range customerNames {
		id := uuid.New()
		s.customerIDs = append(s.customerIDs, id)

		birth := time.Date(1965+s.rng.IntN(38), time.Month(1+s.rng.IntN(12)), 1+s.rng.IntN(28), 0, 0, 0, 0, time.UTC)
		email := strings.ToLower(strings.Join(strings.Fields(removeAccents(c.Name))[:2], ".")) + "@example.com"
		phone := fmt.Sprintf("1998%07d", 1000000+i*7919)
		created := s.historyStart().AddDate(0, 0, -5+i*4)

		rows = append(rows, []any{
			id, CompanyID, c.Name, birth, cpf(s.rng), c.Gender, phone, phone, email,
			streets[i%len(streets)], fmt.Sprintf("%d", 40+s.rng.IntN(1900)), neighborhoods[i%len(neighborhoods)],
			"Campinas", "SP", fmt.Sprintf("130%05d", 10000+s.rng.IntN(89999)),
			UserID, created, created,
		})
	}
	if err := insertRows(ctx, s.tx, "customers", []string{
		"id", "company_id", "full_name", "birth_date", "cpf", "gender", "whatsapp", "mobile_phone", "email",
		"address_street", "address_number", "address_neighborhood", "address_city", "address_state", "address_zipcode",
		"created_by", "created_at", "updated_at",
	}, rows); err != nil {
		return fmt.Errorf("criando clientes: %w", err)
	}
	s.result.Customers = len(rows)
	return nil
}

// sales replica a mesma regra do sales_service.CreateSale (subtotal pelo
// preço de venda, desconto rateado entre itens, crediário gerando parcelas em
// accounts_receivable) para que os dados se comportem como vendas reais.
func (s *seeder) sales(ctx context.Context) error {
	today := dateOf(s.now)
	start := s.historyStart()
	totalDays := today.Sub(start).Hours() / 24

	var saleRows, itemRows, arRows, historyRows [][]any
	balanceDue := make(map[uuid.UUID]float64)

	for day := start; !day.After(today); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Sunday {
			continue
		}

		// Movimento cresce ao longo do período e cai no sábado (meio período).
		progress := day.Sub(start).Hours() / 24 / totalDays
		expected := 7.0 * (0.75 + 0.55*progress)
		closeHour := 18
		if day.Weekday() == time.Saturday {
			expected *= 0.55
			closeHour = 13
		}
		count := int(expected) + s.rng.IntN(3) - 1

		for n := 0; n < count; n++ {
			saleAt := day.Add(time.Duration(8+s.rng.IntN(closeHour-8))*time.Hour + time.Duration(s.rng.IntN(60))*time.Minute)
			if saleAt.After(s.now) {
				continue
			}

			saleID := uuid.New()
			createdBy := s.salesUserIDs[s.rng.IntN(len(s.salesUserIDs))]

			// Itens
			type line struct {
				product seededProduct
				qty     int
			}
			var lines []line
			seen := map[uuid.UUID]bool{}
			for range 1 + s.rng.IntN(4) {
				p := s.pickProduct()
				if seen[p.ID] {
					continue
				}
				seen[p.ID] = true
				lines = append(lines, line{p, 1 + s.rng.IntN(p.MaxQty)})
			}

			subtotal := 0.0
			for _, l := range lines {
				subtotal += float64(l.qty) * l.product.Price
			}
			discount := 0.0
			if subtotal > 300 && s.rng.IntN(4) == 0 {
				discount = round2(subtotal * 0.05)
			}
			total := round2(subtotal - discount)

			method := s.pickPaymentMethod(total)
			var customerID any
			if method == "installments" || s.rng.IntN(10) < 6 {
				customerID = s.customerIDs[s.rng.IntN(len(s.customerIDs))]
			}

			status := "paid"
			downPayment := 0.0
			var installments int32
			var dueDays any
			if method == "installments" {
				installments = int32(2 + s.rng.IntN(4))
				dueDays = 10
				if s.rng.IntN(3) == 0 {
					downPayment = round2(total * 0.2)
				}

				custID := customerID.(uuid.UUID)
				installmentValue := round2((total - downPayment) / float64(installments))
				anyOverdue, allPaid := false, true

				for i := 0; i < int(installments); i++ {
					due := installmentDueDate(saleAt, i, 10)
					arStatus, balance := "pending", installmentValue

					if due.Before(today) {
						// 88% das parcelas vencidas foram pagas; o resto fica em atraso.
						if s.rng.IntN(100) < 88 {
							arStatus, balance = "paid", 0
							paidAt := due.AddDate(0, 0, -s.rng.IntN(4)).Add(time.Duration(9+s.rng.IntN(8)) * time.Hour)
							if paidAt.Before(saleAt) {
								paidAt = saleAt
							}
							historyRows = append(historyRows, []any{
								uuid.New(), CompanyID, custID, saleID, s.paymentMethodIDs["pix"], createdBy,
								installmentValue, paidAt, fmt.Sprintf("Parcela %d/%d", i+1, installments),
							})
						} else {
							arStatus = "overdue"
							anyOverdue = true
						}
					}
					if arStatus != "paid" {
						allPaid = false
						balanceDue[custID] += balance
					}

					arRows = append(arRows, []any{
						uuid.New(), CompanyID, custID, saleID, installmentValue, balance, due,
						i + 1, int(installments), arStatus, createdBy, saleAt, saleAt,
					})
				}

				switch {
				case anyOverdue:
					status = "overdue"
				case allPaid:
					status = "paid"
				default:
					status = "pending"
				}
			}

			saleRows = append(saleRows, []any{
				saleID, customerID, CompanyID, saleAt, discount, round2(subtotal), total, downPayment,
				installments, dueDays, method, status, saleAt, createdBy, saleAt,
			})

			for _, l := range lines {
				itemTotal := float64(l.qty) * l.product.Price
				itemRows = append(itemRows, []any{
					uuid.New(), saleID, l.product.ID, l.qty, l.product.Price,
					round2(discount * itemTotal / subtotal), saleAt,
				})
			}
		}
	}

	if err := insertRows(ctx, s.tx, "sales", []string{
		"id", "customer_id", "company_id", "sale_at", "discount_amount", "subtotal", "total_amount", "down_payment",
		"installments_count", "due_days", "payment_method", "status", "created_at", "created_by", "updated_at",
	}, saleRows); err != nil {
		return fmt.Errorf("criando vendas: %w", err)
	}
	if err := insertRows(ctx, s.tx, "sale_items", []string{
		"id", "sale_id", "product_id", "quantity", "unit_price", "discount", "created_at",
	}, itemRows); err != nil {
		return fmt.Errorf("criando itens de venda: %w", err)
	}
	if err := insertRows(ctx, s.tx, "accounts_receivable", []string{
		"id", "company_id", "customer_id", "sale_id", "total_amount", "balance", "due_date",
		"installment_number", "total_installments", "status", "created_by", "created_at", "updated_at",
	}, arRows); err != nil {
		return fmt.Errorf("criando contas a receber: %w", err)
	}
	if err := insertRows(ctx, s.tx, "payment_history", []string{
		"id", "company_id", "customer_id", "sale_id", "payment_method_id", "user_id",
		"amount_paid", "payment_date", "notes",
	}, historyRows); err != nil {
		return fmt.Errorf("criando histórico de pagamentos: %w", err)
	}

	for customerID, balance := range balanceDue {
		if _, err := s.tx.Exec(ctx,
			`UPDATE customers SET balance_due = $1 WHERE id = $2`,
			round2(balance), customerID,
		); err != nil {
			return fmt.Errorf("atualizando saldo devedor: %w", err)
		}
	}

	s.result.Sales = len(saleRows)
	s.result.SaleItems = len(itemRows)
	s.result.Receivables = len(arRows)
	s.result.PaymentsHistories = len(historyRows)
	return nil
}

func (s *seeder) billsPayable(ctx context.Context) error {
	categoryIDs := map[string]uuid.UUID{}
	rows, err := s.tx.Query(ctx, `SELECT id, name FROM bill_categories WHERE company_id = $1`, CompanyID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		categoryIDs[name] = id
	}
	rows.Close()

	type vendorRef struct {
		ID   uuid.UUID
		Name string
	}
	var vendorRefs []vendorRef
	rows, err = s.tx.Query(ctx, `SELECT id, name FROM vendors WHERE company_id = $1 ORDER BY name`, CompanyID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v vendorRef
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			rows.Close()
			return err
		}
		vendorRefs = append(vendorRefs, v)
	}
	rows.Close()

	today := dateOf(s.now)
	firstMonth := s.historyStart()
	lastMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, s.loc).AddDate(0, 1, 0)

	var billRows [][]any
	overdueLeft := 2 // algumas contas recentes em atraso para os alertas do dashboard
	scheduledLeft := 1

	addBill := func(vendorID any, category, description string, due time.Time, amount float64) {
		amount = round2(amount)
		created := due.AddDate(0, 0, -20)
		if created.After(s.now) {
			created = s.now
		}

		status := "pending"
		var paymentDate, amountPaid, paymentMethodID, scheduledDate any
		switch {
		case due.Before(today) && overdueLeft > 0 && today.Sub(due) < 20*24*time.Hour && category != "Salários":
			status = "overdue"
			overdueLeft--
		case due.Before(today):
			status = "paid"
			paymentDate = due.AddDate(0, 0, -s.rng.IntN(3))
			amountPaid = amount
			paymentMethodID = s.paymentMethodIDs["bank_transfer"]
			if s.rng.IntN(2) == 0 {
				paymentMethodID = s.paymentMethodIDs["pix"]
			}
		case scheduledLeft > 0 && due.Sub(today) > 3*24*time.Hour:
			status = "scheduled"
			scheduledDate = due.AddDate(0, 0, -1)
			paymentMethodID = s.paymentMethodIDs["bank_transfer"]
			scheduledLeft--
		}

		billRows = append(billRows, []any{
			uuid.New(), CompanyID, vendorID, categoryIDs[category], paymentMethodID, amount, due, status,
			description, scheduledDate, paymentDate, amountPaid, created, created,
		})
	}

	for month := firstMonth; month.Before(lastMonth.AddDate(0, 1, 0)); month = month.AddDate(0, 1, 0) {
		label := month.Format("01/2006")

		for _, b := range recurringBills {
			amount := b.Amount * (1 + (s.rng.Float64()*2-1)*b.Variation)
			addBill(nil, b.Category, fmt.Sprintf("%s - %s", b.Description, label), month.AddDate(0, 0, b.Day-1), amount)
		}

		// Reposição de estoque: 3 a 4 pedidos de fornecedores por mês.
		for i := range 3 + s.rng.IntN(2) {
			v := vendorRefs[(int(month.Month())+i)%len(vendorRefs)]
			due := month.AddDate(0, 0, 3+i*8+s.rng.IntN(5))
			addBill(v.ID, "Fornecedores", fmt.Sprintf("Pedido de mercadorias - %s", v.Name), due, 2800+s.rng.Float64()*3200)
		}

		if month.Month()%2 == 0 {
			addBill(nil, "Marketing", fmt.Sprintf("Anúncios online - %s", label), month.AddDate(0, 0, 24), 450+s.rng.Float64()*250)
		}
	}

	if err := insertRows(ctx, s.tx, "bills_payable", []string{
		"id", "company_id", "vendor_id", "category_id", "payment_method_id", "amount", "due_date", "status",
		"description", "scheduled_date", "payment_date", "amount_paid", "created_at", "updated_at",
	}, billRows); err != nil {
		return fmt.Errorf("criando contas a pagar: %w", err)
	}
	s.result.BillsPayable = len(billRows)
	return nil
}

func (s *seeder) announcement(ctx context.Context) error {
	_, err := s.tx.Exec(ctx, `
		INSERT INTO announcements (company_id, title, content, type, starts_at)
		VALUES ($1, $2, $3, 'info', $4)`,
		CompanyID,
		"Bem-vindo à demonstração do ProTrack",
		"Os dados desta conta são fictícios e voltam ao estado original todos os dias. Fique à vontade para testar!",
		s.now.AddDate(0, 0, -1),
	)
	if err != nil {
		return fmt.Errorf("criando aviso de boas-vindas: %w", err)
	}
	return nil
}

// historyStart é o primeiro dia do histórico: início do mês, historyMonths atrás.
func (s *seeder) historyStart() time.Time {
	return time.Date(s.now.Year(), s.now.Month(), 1, 0, 0, 0, 0, s.loc).AddDate(0, -historyMonths, 0)
}

func (s *seeder) pickProduct() seededProduct {
	r := s.rng.IntN(s.productWeight)
	for _, p := range s.products {
		if r < p.Weight {
			return p
		}
		r -= p.Weight
	}
	return s.products[len(s.products)-1]
}

func (s *seeder) pickPaymentMethod(total float64) string {
	r := s.rng.IntN(100)
	if total >= 400 && r < 30 {
		return "installments"
	}
	switch {
	case r < 38:
		return "pix"
	case r < 63:
		return "credit_card"
	case r < 80:
		return "debit_card"
	case r < 92:
		return "cash"
	default:
		return "bank_transfer"
	}
}

// installmentDueDate segue a mesma regra de vencimento do CreateSale.
func installmentDueDate(base time.Time, i, dueDay int) time.Time {
	month := base.Month() + time.Month(i)
	if base.Day() >= dueDay {
		month++
	}
	return time.Date(base.Year(), month, dueDay, 0, 0, 0, 0, base.Location())
}

// insertRows faz INSERT multi-linha em lotes, respeitando o limite de
// parâmetros do Postgres.
func insertRows(ctx context.Context, tx pgx.Tx, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	batchSize := 60000 / len(cols)
	if batchSize > 500 {
		batchSize = 500
	}

	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))

		var sb strings.Builder
		fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", table, strings.Join(cols, ", "))
		args := make([]any, 0, (end-start)*len(cols))
		for i, row := range rows[start:end] {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteByte('(')
			for j, v := range row {
				if j > 0 {
					sb.WriteString(", ")
				}
				args = append(args, v)
				fmt.Fprintf(&sb, "$%d", len(args))
			}
			sb.WriteByte(')')
		}

		if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
			return err
		}
	}
	return nil
}

func randomPasswordHash() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return argon2id.CreateHash(hex.EncodeToString(buf), argon2id.DefaultParams)
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// cpf gera um CPF fictício com dígitos verificadores válidos.
func cpf(rng *mrand.Rand) string {
	d := make([]int, 11)
	for i := range 9 {
		d[i] = rng.IntN(10)
	}
	for pos := 9; pos <= 10; pos++ {
		sum := 0
		for i := 0; i < pos; i++ {
			sum += d[i] * (pos + 1 - i)
		}
		r := (sum * 10) % 11
		if r == 10 {
			r = 0
		}
		d[pos] = r
	}
	var sb strings.Builder
	for _, v := range d {
		sb.WriteByte(byte('0' + v))
	}
	return sb.String()
}

// cnpj gera um CNPJ fictício (matriz 0001) com dígitos verificadores válidos
// a partir da raiz de 8 dígitos.
func cnpj(root string) string {
	digits := []byte(root[:8] + "0001")
	for _, weights := range [][]int{
		{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2},
		{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2},
	} {
		sum := 0
		for i, w := range weights {
			sum += int(digits[i]-'0') * w
		}
		r := sum % 11
		dv := 0
		if r >= 2 {
			dv = 11 - r
		}
		digits = append(digits, byte('0'+dv))
	}
	return string(digits)
}

// ean13 completa 12 dígitos com o dígito verificador do EAN-13.
func ean13(base string) string {
	sum := 0
	for i, c := range base[:12] {
		n := int(c - '0')
		if i%2 == 1 {
			n *= 3
		}
		sum += n
	}
	return base[:12] + fmt.Sprintf("%d", (10-sum%10)%10)
}

func removeAccents(s string) string {
	r := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i",
		"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
	)
	return r.Replace(s)
}
