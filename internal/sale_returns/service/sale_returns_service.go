package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	cashRegisterService "github.com/ProTrack-Solutions/protrack-api/internal/cash_register/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/ProTrack-Solutions/protrack-api/internal/sale_returns/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sale_returns/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *repository.Repository
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{repo: repo, pool: pool, now: time.Now}
}

const adminRole = "ADMIN"

// windowDays lê o prazo de troca da empresa (return_window_days). Sem a
// configuração vale o padrão de 30 dias.
func windowDays(ctx context.Context, q *repository.Repository, companyId uuid.UUID) (int, error) {
	setting, err := q.GetCompanySetting(ctx, db.GetCompanySettingParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Key:       string(enums.ReturnWindowDays),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultReturnWindowDays, nil
	}
	if err != nil {
		return 0, err
	}

	// A tela pode gravar número ou texto ("30")
	var raw any
	if err := json.Unmarshal([]byte(setting.Value), &raw); err != nil {
		return domain.DefaultReturnWindowDays, nil
	}
	var days int
	switch v := raw.(type) {
	case float64:
		days = int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return domain.DefaultReturnWindowDays, nil
		}
		days = n
	default:
		return domain.DefaultReturnWindowDays, nil
	}
	return min(max(days, 0), domain.MaxReturnWindowDays), nil
}

// ReturnableSale monta a tela de devolução: itens com o que ainda pode voltar,
// parcelas em aberto, prazo e devoluções anteriores.
func (s *Service) ReturnableSale(ctx context.Context, companyId uuid.UUID, role string, saleId uuid.UUID) (domain.ReturnableSale, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	sale := pgconv.ParseUUIDToPgType(saleId)

	header, err := s.repo.GetSaleReturnHeader(ctx, db.GetSaleReturnHeaderParams{ID: sale, CompanyID: company})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReturnableSale{}, domain.ErrSaleNotFound
	}
	if err != nil {
		return domain.ReturnableSale{}, err
	}

	rows, err := s.repo.ListReturnableItems(ctx, sale)
	if err != nil {
		return domain.ReturnableSale{}, err
	}

	openDebt, err := s.repo.GetOpenBalanceForReturn(ctx, db.GetOpenBalanceForReturnParams{SaleID: sale, CompanyID: company})
	if err != nil {
		return domain.ReturnableSale{}, err
	}

	previous, err := s.repo.ListSaleReturnsBySale(ctx, db.ListSaleReturnsBySaleParams{SaleID: sale, CompanyID: company})
	if err != nil {
		return domain.ReturnableSale{}, err
	}

	days, err := windowDays(ctx, s.repo, companyId)
	if err != nil {
		return domain.ReturnableSale{}, err
	}

	createdAt := pgconv.PgTimestamptzToTime(header.CreatedAt)
	within := domain.WithinWindow(createdAt, days, s.now())

	out := domain.ReturnableSale{
		SaleID:          saleId,
		CustomerID:      pgconv.PgUUIDToUUIDPtr(header.CustomerID),
		CustomerName:    header.CustomerName,
		CustomerCredit:  header.CustomerCredit,
		PaymentMethod:   header.PaymentMethod,
		Status:          header.Status,
		Subtotal:        header.Subtotal,
		DiscountAmount:  header.DiscountAmount,
		TotalAmount:     header.TotalAmount,
		CreatedAt:       createdAt,
		Canceled:        header.DeletedAt.Valid || header.Status == "canceled",
		OpenDebt:        openDebt,
		WindowDays:      days,
		Deadline:        domain.Deadline(createdAt, days),
		WithinWindow:    within,
		CanOverride:     !within && role == adminRole,
		Items:           make([]domain.ReturnableItem, 0, len(rows)),
		PreviousReturns: make([]domain.PreviousReturn, 0, len(previous)),
	}

	for _, r := range rows {
		out.Items = append(out.Items, domain.ReturnableItem{
			SaleItemID:       pgconv.PgUUIDToUUID(r.ID),
			ProductID:        pgconv.PgUUIDToUUID(r.ProductID),
			ProductName:      r.ProductName,
			SellInBulk:       r.SellInBulk,
			Quantity:         r.Quantity,
			UnitPrice:        r.UnitPrice,
			UnitAmount:       domain.UnitAmount(r.NetTotal, r.Quantity),
			ReturnedQuantity: r.ReturnedQuantity,
			Available:        max(r.Quantity-r.ReturnedQuantity, 0),
			NetTotal:         r.NetTotal,
			ReturnedTotal:    r.ReturnedTotal,
		})
	}

	for _, p := range previous {
		out.PreviousReturns = append(out.PreviousReturns, domain.PreviousReturn{
			ID:            pgconv.PgUUIDToUUID(p.ID),
			Number:        p.Number,
			Kind:          string(p.Kind),
			ReturnedTotal: p.ReturnedTotal,
			CreatedAt:     pgconv.PgTimestamptzToTime(p.CreatedAt),
		})
	}

	return out, nil
}

type returnLine struct {
	item    db.ListReturnableItemsRow
	qty     int32
	amount  float64
	restock bool
}

type newLine struct {
	product db.ListProductsForExchangeRow
	qty     int32
	total   float64
}

// Create registra a devolução ou troca numa única transação: devolve o estoque,
// abate as parcelas em aberto, gera vale-troca ou registra o reembolso e, na
// troca, cria a venda com os itens novos.
func (s *Service) Create(ctx context.Context, companyId, userId uuid.UUID, role string, req domain.CreateReturnRequest) (domain.ReturnDetail, error) {
	if req.Kind == domain.KindReturn && len(req.NewItems) > 0 {
		return domain.ReturnDetail{}, fmt.Errorf("%w: devolução não leva itens novos; use troca", domain.ErrInvalidReturn)
	}
	if req.Kind == domain.KindExchange && len(req.NewItems) == 0 {
		return domain.ReturnDetail{}, fmt.Errorf("%w: escolha os itens que o cliente vai levar na troca", domain.ErrInvalidReturn)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ReturnDetail{}, err
	}
	defer tx.Rollback(ctx)

	// Reembolso e diferença da troca passam pelo caixa, como uma venda
	if err := cashRegisterService.BindOpenSession(ctx, tx, companyId); err != nil {
		return domain.ReturnDetail{}, err
	}

	q := s.repo.WithTx(tx)
	company := pgconv.ParseUUIDToPgType(companyId)
	user := pgconv.ParseUUIDToPgType(userId)

	sale, err := q.GetSaleForReturn(ctx, db.GetSaleForReturnParams{
		ID:        pgconv.ParseUUIDToPgType(req.SaleID),
		CompanyID: company,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReturnDetail{}, domain.ErrSaleNotFound
	}
	if err != nil {
		return domain.ReturnDetail{}, err
	}
	if sale.DeletedAt.Valid || sale.Status == "canceled" {
		return domain.ReturnDetail{}, domain.ErrSaleCanceled
	}

	// 1. Prazo de troca
	days, err := windowDays(ctx, q, companyId)
	if err != nil {
		return domain.ReturnDetail{}, err
	}
	outOfWindow := !domain.WithinWindow(pgconv.PgTimestamptzToTime(sale.CreatedAt), days, s.now())
	overrideReason := ""
	if outOfWindow {
		overrideReason = strings.TrimSpace(req.OverrideReason)
		if role != adminRole {
			return domain.ReturnDetail{}, domain.ErrOutOfWindow
		}
		if overrideReason == "" {
			return domain.ReturnDetail{}, domain.ErrOverrideReasonRequired
		}
	}

	// 2. Itens devolvidos e o valor de cada um
	saleItems, err := q.ListReturnableItems(ctx, sale.ID)
	if err != nil {
		return domain.ReturnDetail{}, err
	}
	byID := make(map[uuid.UUID]db.ListReturnableItemsRow, len(saleItems))
	for _, it := range saleItems {
		byID[pgconv.PgUUIDToUUID(it.ID)] = it
	}

	lines := make([]returnLine, 0, len(req.Items))
	seen := map[uuid.UUID]bool{}
	var returnedTotal float64
	for _, r := range req.Items {
		item, ok := byID[r.SaleItemID]
		if !ok {
			return domain.ReturnDetail{}, fmt.Errorf("%w: item não pertence à venda", domain.ErrInvalidReturn)
		}
		if seen[r.SaleItemID] {
			return domain.ReturnDetail{}, fmt.Errorf("%w: item %s repetido", domain.ErrInvalidReturn, item.ProductName)
		}
		seen[r.SaleItemID] = true

		available := item.Quantity - item.ReturnedQuantity
		if r.Quantity > available {
			return domain.ReturnDetail{}, fmt.Errorf("%w: %s (disponível %d, informado %d)",
				domain.ErrQuantityExceeded, item.ProductName, max(available, 0), r.Quantity)
		}

		amount := domain.LineReturnAmount(item.NetTotal, item.ReturnedTotal, item.Quantity, item.ReturnedQuantity, r.Quantity)
		restock := r.Restock == nil || *r.Restock
		lines = append(lines, returnLine{item: item, qty: r.Quantity, amount: amount, restock: restock && !item.SellInBulk})
		returnedTotal += amount
	}
	returnedTotal = domain.Round2(returnedTotal)
	if returnedTotal <= 0 {
		return domain.ReturnDetail{}, fmt.Errorf("%w: os itens escolhidos não têm valor a devolver", domain.ErrInvalidReturn)
	}

	// 3. Itens novos da troca (preço de venda atual)
	newLines, exchangeTotal, err := loadNewItems(ctx, q, company, req.NewItems)
	if err != nil {
		return domain.ReturnDetail{}, err
	}

	// 4. Distribuição do valor devolvido
	receivables, err := q.ListOpenReceivablesForReturn(ctx, db.ListOpenReceivablesForReturnParams{SaleID: sale.ID, CompanyID: company})
	if err != nil {
		return domain.ReturnDetail{}, err
	}
	var openDebt float64
	for _, r := range receivables {
		openDebt += r.Balance
	}
	alloc := domain.Allocate(returnedTotal, exchangeTotal, openDebt)

	// 5. Diferença paga pelo cliente na troca
	var differenceMethod pgtype.Text
	var amountReceived, changeAmount pgtype.Numeric
	if alloc.Difference > 0 {
		if !domain.RefundMethods[req.DifferenceMethod] {
			return domain.ReturnDetail{}, fmt.Errorf("%w: informe como o cliente paga a diferença de R$ %.2f", domain.ErrInvalidReturn, alloc.Difference)
		}
		differenceMethod = pgconv.ParseStringToPgText(req.DifferenceMethod)
		if req.DifferenceMethod == "cash" && req.AmountReceived != nil {
			received := domain.Round2(*req.AmountReceived)
			if received < alloc.Difference {
				return domain.ReturnDetail{}, fmt.Errorf("%w: o valor recebido (R$ %.2f) é menor que a diferença (R$ %.2f)",
					domain.ErrInvalidReturn, received, alloc.Difference)
			}
			amountReceived = pgconv.Float64ToPgNumeric(received)
			changeAmount = pgconv.Float64ToPgNumeric(domain.Round2(received - alloc.Difference))
		}
	}

	// 6. Sobra: vale-troca (no cliente) ou dinheiro de volta
	customerID := sale.CustomerID
	if !customerID.Valid && req.CustomerID != nil && *req.CustomerID != uuid.Nil {
		customerID = pgconv.ParseUUIDToPgType(*req.CustomerID)
	}

	var credit, refund float64
	var refundMethod pgtype.Text
	if alloc.Rest > 0 {
		switch req.Destination {
		case domain.DestinationCredit:
			if !customerID.Valid {
				return domain.ReturnDetail{}, domain.ErrCustomerRequired
			}
			credit = alloc.Rest
		case domain.DestinationRefund:
			if !domain.RefundMethods[req.RefundMethod] {
				return domain.ReturnDetail{}, fmt.Errorf("%w: informe como o valor será devolvido", domain.ErrInvalidReturn)
			}
			refund = alloc.Rest
			refundMethod = pgconv.ParseStringToPgText(req.RefundMethod)
		default:
			return domain.ReturnDetail{}, fmt.Errorf("%w: escolha entre vale-troca e devolver o dinheiro (R$ %.2f)", domain.ErrInvalidReturn, alloc.Rest)
		}
	}

	// O cliente informado (vale-troca ou venda de troca) precisa ser da empresa
	if customerID.Valid && customerID != sale.CustomerID {
		if _, err := q.GetCustomerForCredit(ctx, db.GetCustomerForCreditParams{ID: customerID, CompanyID: company}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ReturnDetail{}, domain.ErrCustomerNotFound
			}
			return domain.ReturnDetail{}, err
		}
	}

	// 7. Venda da troca com os itens novos
	var exchangeSaleID pgtype.UUID
	if req.Kind == domain.KindExchange {
		method := domain.PaymentStoreCredit
		if alloc.Difference > 0 {
			method = req.DifferenceMethod
		}
		exchangeSaleID, err = q.CreateExchangeSale(ctx, db.CreateExchangeSaleParams{
			CustomerID:     customerID,
			CompanyID:      company,
			TotalAmount:    pgconv.Float64ToPgNumeric(exchangeTotal),
			PaymentMethod:  method,
			CreatedBy:      user,
			AmountReceived: amountReceived,
			ChangeAmount:   changeAmount,
			CreditAmount:   pgconv.Float64ToPgNumeric(alloc.ExchangeCredit),
		})
		if err != nil {
			return domain.ReturnDetail{}, err
		}

		for _, nl := range newLines {
			if !nl.product.SellInBulk {
				affected, err := q.DecrementStockForExchange(ctx, db.DecrementStockForExchangeParams{
					Quantity: pgconv.IntToPgInt4(int(nl.qty)),
					ID:       nl.product.ID,
				})
				if err != nil {
					return domain.ReturnDetail{}, err
				}
				if affected == 0 {
					return domain.ReturnDetail{}, fmt.Errorf("%w para o produto %s", domain.ErrInsufficientStock, nl.product.Name)
				}
				if err := q.CreateReturnStockMovement(ctx, db.CreateReturnStockMovementParams{
					CompanyID: company,
					ProductID: nl.product.ID,
					SaleID:    exchangeSaleID,
					Type:      db.StockMovementTypeSALE,
					Quantity:  -nl.qty,
					Reason:    pgconv.ParseStringToPgText("Troca"),
					CreatedBy: user,
				}); err != nil {
					return domain.ReturnDetail{}, err
				}
			}
			if _, err := q.CreateExchangeSaleItem(ctx, db.CreateExchangeSaleItemParams{
				SaleID:    exchangeSaleID,
				Quantity:  nl.qty,
				UnitPrice: pgconv.Float64ToPgNumeric(nl.product.SalePrice),
				ProductID: nl.product.ID,
			}); err != nil {
				return domain.ReturnDetail{}, err
			}
		}
	}

	// 8. Cabeçalho da devolução
	if err := q.LockCompanyReturnNumbers(ctx, companyId.String()); err != nil {
		return domain.ReturnDetail{}, err
	}
	number, err := q.NextSaleReturnNumber(ctx, company)
	if err != nil {
		return domain.ReturnDetail{}, err
	}

	var creditCustomer pgtype.UUID
	if credit > 0 || customerID.Valid {
		creditCustomer = customerID
	}

	returnID, err := q.CreateSaleReturn(ctx, db.CreateSaleReturnParams{
		CompanyID:        company,
		Number:           number,
		Kind:             db.SaleReturnKind(req.Kind),
		SaleID:           sale.ID,
		ExchangeSaleID:   exchangeSaleID,
		CustomerID:       creditCustomer,
		ReturnedTotal:    pgconv.Float64ToPgNumeric(returnedTotal),
		ExchangeTotal:    pgconv.Float64ToPgNumeric(exchangeTotal),
		ExchangeCredit:   pgconv.Float64ToPgNumeric(alloc.ExchangeCredit),
		DebtReduction:    pgconv.Float64ToPgNumeric(alloc.DebtReduction),
		CreditAmount:     pgconv.Float64ToPgNumeric(credit),
		RefundAmount:     pgconv.Float64ToPgNumeric(refund),
		RefundMethod:     refundMethod,
		DifferencePaid:   pgconv.Float64ToPgNumeric(alloc.Difference),
		DifferenceMethod: differenceMethod,
		Reason:           req.Reason,
		Notes:            pgconv.ParseStringToPgType(strings.TrimSpace(req.Notes)),
		OutOfWindow:      outOfWindow,
		OverrideReason:   pgconv.ParseStringToPgType(overrideReason),
		CreatedBy:        user,
	})
	if err != nil {
		return domain.ReturnDetail{}, err
	}

	// 9. Itens devolvidos e estoque
	for _, l := range lines {
		if err := q.CreateSaleReturnItem(ctx, db.CreateSaleReturnItemParams{
			ReturnID:   returnID,
			SaleItemID: l.item.ID,
			ProductID:  l.item.ProductID,
			Quantity:   l.qty,
			UnitAmount: pgconv.Float64ToPgNumeric(domain.UnitAmount(l.item.NetTotal, l.item.Quantity)),
			Total:      pgconv.Float64ToPgNumeric(l.amount),
			Restock:    l.restock,
		}); err != nil {
			return domain.ReturnDetail{}, err
		}
		if !l.restock {
			continue
		}
		if err := q.IncrementStockForReturn(ctx, db.IncrementStockForReturnParams{
			Quantity: pgconv.IntToPgInt4(int(l.qty)),
			ID:       l.item.ProductID,
		}); err != nil {
			return domain.ReturnDetail{}, err
		}
		if err := q.CreateReturnStockMovement(ctx, db.CreateReturnStockMovementParams{
			CompanyID:    company,
			ProductID:    l.item.ProductID,
			SaleID:       sale.ID,
			SaleReturnID: returnID,
			Type:         db.StockMovementTypeRETURN,
			Quantity:     l.qty,
			Reason:       pgconv.ParseStringToPgText(fmt.Sprintf("Devolução nº %d", number)),
			CreatedBy:    user,
		}); err != nil {
			return domain.ReturnDetail{}, err
		}
	}

	// 10. Abate das parcelas em aberto e do saldo devedor do cliente
	if alloc.DebtReduction > 0 {
		balances := make([]float64, len(receivables))
		for i, r := range receivables {
			balances[i] = r.Balance
		}
		for i, nb := range domain.ReduceInstallments(balances, alloc.DebtReduction) {
			if nb == balances[i] {
				continue
			}
			if err := q.ReduceReceivableForReturn(ctx, db.ReduceReceivableForReturnParams{
				Balance:   pgconv.Float64ToPgNumeric(nb),
				UpdatedBy: user,
				ID:        receivables[i].ID,
			}); err != nil {
				return domain.ReturnDetail{}, err
			}
		}

		if sale.CustomerID.Valid {
			if err := q.SubtractCustomerBalanceDue(ctx, db.SubtractCustomerBalanceDueParams{
				Amount:    pgconv.Float64ToPgNumeric(alloc.DebtReduction),
				UpdatedBy: user,
				ID:        sale.CustomerID,
			}); err != nil {
				return domain.ReturnDetail{}, err
			}
		}

		// Sem nada mais a receber, a venda fica quitada
		if domain.Round2(openDebt-alloc.DebtReduction) <= 0 {
			if err := q.SetSaleStatusForReturn(ctx, db.SetSaleStatusForReturnParams{
				Status:    "paid",
				UpdatedBy: user,
				ID:        sale.ID,
			}); err != nil {
				return domain.ReturnDetail{}, err
			}
		}
	}

	// 11. Vale-troca
	if credit > 0 {
		balance, err := q.AddCustomerStoreCredit(ctx, db.AddCustomerStoreCreditParams{
			Amount: pgconv.Float64ToPgNumeric(credit),
			ID:     customerID,
		})
		if err != nil {
			return domain.ReturnDetail{}, err
		}
		if err := q.CreateCustomerCreditMovement(ctx, db.CreateCustomerCreditMovementParams{
			CompanyID:    company,
			CustomerID:   customerID,
			Type:         "return",
			Amount:       pgconv.Float64ToPgNumeric(credit),
			BalanceAfter: pgconv.Float64ToPgNumeric(balance),
			SaleReturnID: returnID,
			CreatedBy:    user,
		}); err != nil {
			return domain.ReturnDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ReturnDetail{}, err
	}

	return s.Get(ctx, companyId, pgconv.PgUUIDToUUID(returnID))
}

// loadNewItems confere os produtos da troca e soma o valor pelo preço atual
func loadNewItems(ctx context.Context, q *repository.Repository, company pgtype.UUID, items []domain.NewItemRequest) ([]newLine, float64, error) {
	if len(items) == 0 {
		return nil, 0, nil
	}

	ids := make([]pgtype.UUID, 0, len(items))
	qty := map[uuid.UUID]int32{}
	order := []uuid.UUID{}
	for _, it := range items {
		if _, ok := qty[it.ProductID]; !ok {
			ids = append(ids, pgconv.ParseUUIDToPgType(it.ProductID))
			order = append(order, it.ProductID)
		}
		qty[it.ProductID] += it.Quantity
	}

	products, err := q.ListProductsForExchange(ctx, db.ListProductsForExchangeParams{CompanyID: company, Ids: ids})
	if err != nil {
		return nil, 0, err
	}
	byID := make(map[uuid.UUID]db.ListProductsForExchangeRow, len(products))
	for _, p := range products {
		byID[pgconv.PgUUIDToUUID(p.ID)] = p
	}

	lines := make([]newLine, 0, len(order))
	var total float64
	for _, id := range order {
		p, ok := byID[id]
		if !ok {
			return nil, 0, domain.ErrProductNotFound
		}
		if !p.SellInBulk && p.Quantity < qty[id] {
			return nil, 0, fmt.Errorf("%w para o produto %s: disponível %d, solicitado %d",
				domain.ErrInsufficientStock, p.Name, max(p.Quantity, 0), qty[id])
		}
		lineTotal := domain.Round2(p.SalePrice * float64(qty[id]))
		lines = append(lines, newLine{product: p, qty: qty[id], total: lineTotal})
		total += lineTotal
	}
	total = domain.Round2(total)
	if total <= 0 {
		return nil, 0, fmt.Errorf("%w: os itens da troca precisam ter preço de venda", domain.ErrInvalidReturn)
	}
	return lines, total, nil
}

func (s *Service) Get(ctx context.Context, companyId, id uuid.UUID) (domain.ReturnDetail, error) {
	r, err := s.repo.GetSaleReturn(ctx, db.GetSaleReturnParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReturnDetail{}, domain.ErrReturnNotFound
	}
	if err != nil {
		return domain.ReturnDetail{}, err
	}

	items, err := s.repo.ListSaleReturnItems(ctx, r.ID)
	if err != nil {
		return domain.ReturnDetail{}, err
	}

	var newItems []db.ListExchangeSaleItemsRow
	if r.ExchangeSaleID.Valid {
		newItems, err = s.repo.ListExchangeSaleItems(ctx, r.ExchangeSaleID)
		if err != nil {
			return domain.ReturnDetail{}, err
		}
	}

	var itemsCount int32
	out := domain.ReturnDetail{
		ReturnListItem: domain.ReturnListItem{
			ID:             pgconv.PgUUIDToUUID(r.ID),
			Number:         r.Number,
			Kind:           string(r.Kind),
			SaleID:         pgconv.PgUUIDToUUID(r.SaleID),
			ExchangeSaleID: pgconv.PgUUIDToUUIDPtr(r.ExchangeSaleID),
			CustomerID:     pgconv.PgUUIDToUUIDPtr(r.CustomerID),
			CustomerName:   r.CustomerName,
			ReturnedTotal:  pgconv.PgNumericToFloat64(r.ReturnedTotal),
			ExchangeTotal:  pgconv.PgNumericToFloat64(r.ExchangeTotal),
			DebtReduction:  pgconv.PgNumericToFloat64(r.DebtReduction),
			CreditAmount:   pgconv.PgNumericToFloat64(r.CreditAmount),
			RefundAmount:   pgconv.PgNumericToFloat64(r.RefundAmount),
			RefundMethod:   pgconv.ParsePgTextToStringPtr(r.RefundMethod),
			DifferencePaid: pgconv.PgNumericToFloat64(r.DifferencePaid),
			Reason:         r.Reason,
			OutOfWindow:    r.OutOfWindow,
			CreatedAt:      pgconv.PgTimestamptzToTime(r.CreatedAt),
			CreatedByName:  r.CreatedByName,
		},
		ExchangeCredit:   pgconv.PgNumericToFloat64(r.ExchangeCredit),
		DifferenceMethod: pgconv.ParsePgTextToStringPtr(r.DifferenceMethod),
		Notes:            pgconv.ParsePgTextToStringPtr(r.Notes),
		OverrideReason:   pgconv.ParsePgTextToStringPtr(r.OverrideReason),
		SaleCreatedAt:    pgconv.PgTimestamptzToTime(r.SaleCreatedAt),
		SaleTotal:        r.SaleTotal,
		Items:            make([]domain.ReturnedItem, 0, len(items)),
		NewItems:         make([]domain.ExchangeItem, 0, len(newItems)),
	}

	for _, it := range items {
		itemsCount += it.Quantity
		out.Items = append(out.Items, domain.ReturnedItem{
			ProductID:   pgconv.PgUUIDToUUID(it.ProductID),
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitAmount:  it.UnitAmount,
			Total:       it.Total,
			Restock:     it.Restock,
		})
	}
	out.ItemsCount = itemsCount

	for _, it := range newItems {
		out.NewItems = append(out.NewItems, domain.ExchangeItem{
			ProductID:   pgconv.PgUUIDToUUID(it.ProductID),
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			Total:       it.Total,
		})
	}

	return out, nil
}

func pageParams(page, perPage, defaultPerPage int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	return page, perPage
}

func (s *Service) List(ctx context.Context, companyId uuid.UUID, params domain.ListReturnsParams) (domain.ListReturnsResponse, error) {
	page, perPage := pageParams(params.Page, params.PerPage, 20)

	arg := db.ListSaleReturnsParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Search:    strings.TrimSpace(params.Search),
		RowLimit:  perPage,
		RowOffset: (page - 1) * perPage,
	}
	if params.Kind != "" {
		arg.Kind = db.NullSaleReturnKind{SaleReturnKind: db.SaleReturnKind(params.Kind), Valid: true}
	}
	if params.StartDate != "" {
		arg.StartDate = pgconv.StringToPgDate(params.StartDate)
	}
	if params.EndDate != "" {
		arg.EndDate = pgconv.StringToPgDate(params.EndDate)
	}

	rows, err := s.repo.ListSaleReturns(ctx, arg)
	if err != nil {
		return domain.ListReturnsResponse{}, err
	}

	out := domain.ListReturnsResponse{Data: make([]domain.ReturnListItem, 0, len(rows)), Page: page, PerPage: perPage}
	for _, r := range rows {
		out.Total = r.TotalRows
		out.Data = append(out.Data, domain.ReturnListItem{
			ID:             pgconv.PgUUIDToUUID(r.ID),
			Number:         r.Number,
			Kind:           string(r.Kind),
			SaleID:         pgconv.PgUUIDToUUID(r.SaleID),
			ExchangeSaleID: pgconv.PgUUIDToUUIDPtr(r.ExchangeSaleID),
			CustomerID:     pgconv.PgUUIDToUUIDPtr(r.CustomerID),
			CustomerName:   r.CustomerName,
			ReturnedTotal:  r.ReturnedTotal,
			ExchangeTotal:  r.ExchangeTotal,
			DebtReduction:  r.DebtReduction,
			CreditAmount:   r.CreditAmount,
			RefundAmount:   r.RefundAmount,
			RefundMethod:   pgconv.ParsePgTextToStringPtr(r.RefundMethod),
			DifferencePaid: r.DifferencePaid,
			Reason:         r.Reason,
			OutOfWindow:    r.OutOfWindow,
			ItemsCount:     r.ItemsCount,
			CreatedAt:      pgconv.PgTimestamptzToTime(r.CreatedAt),
			CreatedByName:  r.CreatedByName,
		})
	}
	return out, nil
}

func (s *Service) SearchSales(ctx context.Context, companyId uuid.UUID, params domain.SearchSalesParams) (domain.SaleSearchResponse, error) {
	page, perPage := pageParams(params.Page, params.PerPage, 10)

	rows, err := s.repo.SearchSalesForReturn(ctx, db.SearchSalesForReturnParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Search:    strings.TrimSpace(params.Search),
		RowLimit:  perPage,
		RowOffset: (page - 1) * perPage,
	})
	if err != nil {
		return domain.SaleSearchResponse{}, err
	}

	out := domain.SaleSearchResponse{Data: make([]domain.SaleSearchItem, 0, len(rows)), Page: page, PerPage: perPage}
	for _, r := range rows {
		out.Total = r.TotalRows
		out.Data = append(out.Data, domain.SaleSearchItem{
			ID:            pgconv.PgUUIDToUUID(r.ID),
			CreatedAt:     pgconv.PgTimestamptzToTime(r.CreatedAt),
			TotalAmount:   r.TotalAmount,
			PaymentMethod: r.PaymentMethod,
			CustomerName:  r.CustomerName,
			ItemsCount:    r.ItemsCount,
			ReturnedCount: r.ReturnedCount,
			Products:      r.Products,
		})
	}
	return out, nil
}

func (s *Service) CustomerCredit(ctx context.Context, companyId, customerId uuid.UUID) (domain.CustomerCredit, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	customer := pgconv.ParseUUIDToPgType(customerId)

	c, err := s.repo.GetCustomerCredit(ctx, db.GetCustomerCreditParams{ID: customer, CompanyID: company})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CustomerCredit{}, domain.ErrCustomerNotFound
	}
	if err != nil {
		return domain.CustomerCredit{}, err
	}

	rows, err := s.repo.ListCustomerCreditMovements(ctx, db.ListCustomerCreditMovementsParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.CustomerCredit{}, err
	}

	out := domain.CustomerCredit{
		CustomerID:   customerId,
		CustomerName: c.FullName,
		StoreCredit:  c.StoreCredit,
		Movements:    make([]domain.CreditMovement, 0, len(rows)),
	}
	for _, m := range rows {
		out.Movements = append(out.Movements, domain.CreditMovement{
			ID:            pgconv.PgUUIDToUUID(m.ID),
			Type:          m.Type,
			Amount:        m.Amount,
			BalanceAfter:  m.BalanceAfter,
			SaleReturnID:  pgconv.PgUUIDToUUIDPtr(m.SaleReturnID),
			ReturnNumber:  pgconv.PgInt4ToIntPtr(m.ReturnNumber),
			SaleID:        pgconv.PgUUIDToUUIDPtr(m.SaleID),
			CreatedAt:     pgconv.PgTimestamptzToTime(m.CreatedAt),
			CreatedByName: m.CreatedByName,
		})
	}
	return out, nil
}
