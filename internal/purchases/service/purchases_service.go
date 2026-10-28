package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPerPage  = 20
	dateLayout      = "2006-01-02"
	defaultTimezone = "America/Sao_Paulo"
)

type Service struct {
	repo          *repository.Repository
	pool          *pgxpool.Pool
	subscriptions Subscriptions
	plans         Plans
	publisher     Publisher
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, subscriptions Subscriptions, plans Plans) *Service {
	return &Service{
		repo:          repo,
		pool:          pool,
		subscriptions: subscriptions,
		plans:         plans,
	}
}

// ─── Consultas ───────────────────────────────────────────────────────────────

func (s *Service) ListOrders(ctx context.Context, companyId uuid.UUID, params domain.ListOrdersParams) (domain.ListOrdersResponse, error) {
	page := max(params.Page, 1)
	perPage := params.PerPage
	if perPage == 0 {
		perPage = defaultPerPage
	}

	companyPg := pgconv.ParseUUIDToPgType(companyId)
	var status db.NullPurchaseOrderStatus
	if params.Status != "" {
		status = db.NullPurchaseOrderStatus{PurchaseOrderStatus: db.PurchaseOrderStatus(params.Status), Valid: true}
	}
	var search pgtype.Text
	if term := strings.TrimSpace(params.Search); term != "" {
		search = pgtype.Text{String: strings.TrimPrefix(term, "#"), Valid: true}
	}

	rows, err := s.repo.ListPurchaseOrders(ctx, db.ListPurchaseOrdersParams{
		CompanyID:  companyPg,
		Status:     status,
		VendorID:   pgconv.OptionalUUIDToPgType(params.VendorID),
		Search:     search,
		PageLimit:  perPage,
		PageOffset: (page - 1) * perPage,
	})
	if err != nil {
		return domain.ListOrdersResponse{}, err
	}

	loc, err := s.location(ctx, s.repo, companyPg)
	if err != nil {
		return domain.ListOrdersResponse{}, err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	resp := domain.ListOrdersResponse{Data: []domain.OrderListItem{}, Page: page, PerPage: perPage}
	for _, r := range rows {
		resp.TotalRows = r.TotalRows
		resp.Data = append(resp.Data, domain.OrderListItem{
			ID:               pgconv.PgUUIDToUUID(r.ID),
			Number:           r.Number,
			VendorName:       r.VendorName,
			Status:           r.Status,
			ExpectedDate:     datePtr(r.ExpectedDate),
			CreatedAt:        r.CreatedAt.Time,
			ItemsCount:       r.ItemsCount,
			Total:            domain.Round2(r.Total),
			Quantity:         r.Quantity,
			ReceivedQuantity: r.ReceivedQuantity,
			ReceivedTotal:    r.ReceivedTotal,
			Late:             isLate(r.Status, r.ExpectedDate, today),
		})
	}
	resp.TotalPages = int32(math.Ceil(float64(resp.TotalRows) / float64(perPage)))

	overview, err := s.repo.GetPurchasesOverview(ctx, db.GetPurchasesOverviewParams{
		CompanyID:  companyPg,
		MonthStart: pgconv.TimeToPgTimestamptz(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)),
		Today:      pgconv.ToPgDate(today),
	})
	if err != nil {
		return domain.ListOrdersResponse{}, err
	}
	counts, err := s.repo.CountPurchaseOrdersByStatus(ctx, companyPg)
	if err != nil {
		return domain.ListOrdersResponse{}, err
	}

	resp.Overview = domain.OrdersOverview{
		PendingTotal:      domain.Round2(overview.PendingTotal),
		ReceivedThisMonth: domain.Round2(overview.ReceivedThisMonth),
		LateOrders:        overview.LateOrders,
		CountByStatus:     map[string]int32{},
	}
	for _, c := range counts {
		resp.Overview.CountByStatus[c.Status] = c.Count
	}

	return resp, nil
}

func (s *Service) GetOrder(ctx context.Context, companyId, orderId uuid.UUID) (domain.OrderDetail, error) {
	return s.orderDetail(ctx, s.repo, companyId, orderId)
}

// ─── Pedido ──────────────────────────────────────────────────────────────────

func (s *Service) CreateOrder(ctx context.Context, companyId, userId uuid.UUID, req domain.SaveOrderRequest) (domain.OrderDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	defer tx.Rollback(ctx)
	repo := s.repo.WithTx(tx)

	companyPg := pgconv.ParseUUIDToPgType(companyId)
	if err := validateOrder(ctx, repo, companyPg, req); err != nil {
		return domain.OrderDetail{}, err
	}

	// Numeração sequencial por empresa, sem buracos por corrida
	if err := repo.LockCompanyPurchaseNumbers(ctx, companyId.String()); err != nil {
		return domain.OrderDetail{}, err
	}
	number, err := repo.NextPurchaseOrderNumber(ctx, companyPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}

	status := db.PurchaseOrderStatusDraft
	if req.Send {
		status = db.PurchaseOrderStatusSent
	}

	orderPg, err := repo.CreatePurchaseOrder(ctx, db.CreatePurchaseOrderParams{
		CompanyID:    companyPg,
		Number:       number,
		VendorID:     pgconv.ParseUUIDToPgType(req.VendorID),
		Status:       status,
		ExpectedDate: optionalDate(req.ExpectedDate),
		Notes:        pgconv.ParseStringToPgType(strings.TrimSpace(req.Notes)),
		CreatedBy:    pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.OrderDetail{}, err
	}

	if err := createItems(ctx, repo, orderPg, req.Items); err != nil {
		return domain.OrderDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.OrderDetail{}, err
	}
	return s.GetOrder(ctx, companyId, pgconv.PgUUIDToUUID(orderPg))
}

func (s *Service) UpdateOrder(ctx context.Context, companyId, orderId uuid.UUID, req domain.SaveOrderRequest) (domain.OrderDetail, error) {
	return s.withOrder(ctx, companyId, orderId, func(repo *repository.Repository, order db.LockPurchaseOrderRow, totals db.GetPurchaseOrderItemTotalsRow) error {
		if !domain.ActionsFor(string(order.Status), totals.ReceivedQuantity).CanEdit {
			return domain.ErrOrderNotEditable
		}
		if err := validateOrder(ctx, repo, order.CompanyID, req); err != nil {
			return err
		}
		if err := repo.UpdatePurchaseOrderHeader(ctx, db.UpdatePurchaseOrderHeaderParams{
			ID:           order.ID,
			VendorID:     pgconv.ParseUUIDToPgType(req.VendorID),
			ExpectedDate: optionalDate(req.ExpectedDate),
			Notes:        pgconv.ParseStringToPgType(strings.TrimSpace(req.Notes)),
		}); err != nil {
			return err
		}
		if err := repo.DeletePurchaseOrderItems(ctx, order.ID); err != nil {
			return err
		}
		return createItems(ctx, repo, order.ID, req.Items)
	})
}

func (s *Service) SendOrder(ctx context.Context, companyId, orderId uuid.UUID) (domain.OrderDetail, error) {
	return s.withOrder(ctx, companyId, orderId, func(repo *repository.Repository, order db.LockPurchaseOrderRow, totals db.GetPurchaseOrderItemTotalsRow) error {
		if !domain.ActionsFor(string(order.Status), totals.ReceivedQuantity).CanSend {
			return domain.ErrOrderStatus
		}
		return repo.UpdatePurchaseOrderStatus(ctx, db.UpdatePurchaseOrderStatusParams{ID: order.ID, Status: db.PurchaseOrderStatusSent})
	})
}

// CloseOrder encerra o pedido recebido em parte: o saldo pendente é descartado
func (s *Service) CloseOrder(ctx context.Context, companyId, userId, orderId uuid.UUID, req domain.CloseOrderRequest) (domain.OrderDetail, error) {
	return s.finishOrder(ctx, companyId, userId, orderId, db.PurchaseOrderStatusClosed, req.Reason)
}

// CancelOrder cancela o pedido que ainda não recebeu nada
func (s *Service) CancelOrder(ctx context.Context, companyId, userId, orderId uuid.UUID, req domain.CloseOrderRequest) (domain.OrderDetail, error) {
	return s.finishOrder(ctx, companyId, userId, orderId, db.PurchaseOrderStatusCanceled, req.Reason)
}

func (s *Service) finishOrder(ctx context.Context, companyId, userId, orderId uuid.UUID, status db.PurchaseOrderStatus, reason string) (domain.OrderDetail, error) {
	return s.withOrder(ctx, companyId, orderId, func(repo *repository.Repository, order db.LockPurchaseOrderRow, totals db.GetPurchaseOrderItemTotalsRow) error {
		actions := domain.ActionsFor(string(order.Status), totals.ReceivedQuantity)
		if (status == db.PurchaseOrderStatusClosed && !actions.CanClose) || (status == db.PurchaseOrderStatusCanceled && !actions.CanCancel) {
			return domain.ErrOrderStatus
		}
		return repo.ClosePurchaseOrder(ctx, db.ClosePurchaseOrderParams{
			ID:          order.ID,
			Status:      status,
			ClosedBy:    pgconv.ParseUUIDToPgType(userId),
			CloseReason: pgconv.ParseStringToPgType(strings.TrimSpace(reason)),
		})
	})
}

// ─── Recebimento ─────────────────────────────────────────────────────────────

// Receive dá entrada na mercadoria: soma o estoque, recalcula o custo médio
// com frete/despesas/desconto rateados, atualiza o preço de venda quando
// informado, gera as contas a pagar e avança o status do pedido.
func (s *Service) Receive(ctx context.Context, companyId, userId, orderId uuid.UUID, req domain.ReceiveRequest) (domain.OrderDetail, error) {
	return s.withOrder(ctx, companyId, orderId, func(repo *repository.Repository, order db.LockPurchaseOrderRow, totals db.GetPurchaseOrderItemTotalsRow) error {
		if !domain.ActionsFor(string(order.Status), totals.ReceivedQuantity).CanReceive {
			return domain.ErrOrderStatus
		}

		orderItems, err := repo.ListPurchaseOrderItems(ctx, order.ID)
		if err != nil {
			return err
		}
		byID := make(map[uuid.UUID]db.ListPurchaseOrderItemsRow, len(orderItems))
		for _, it := range orderItems {
			byID[pgconv.PgUUIDToUUID(it.ID)] = it
		}

		// Ordena por produto: transações concorrentes travam as linhas na mesma ordem
		items := append([]domain.ReceiptItemRequest(nil), req.Items...)
		sort.Slice(items, func(i, j int) bool {
			return byID[items[i].OrderItemID].ProductID.String() < byID[items[j].OrderItemID].ProductID.String()
		})

		lines := make([]domain.CostLine, len(items))
		seen := map[uuid.UUID]bool{}
		for i, it := range items {
			orderItem, ok := byID[it.OrderItemID]
			if !ok {
				return fmt.Errorf("%w: item não pertence a este pedido", domain.ErrInvalidPurchase)
			}
			if seen[it.OrderItemID] {
				return fmt.Errorf("%w: item repetido no recebimento", domain.ErrInvalidPurchase)
			}
			seen[it.OrderItemID] = true
			if it.Quantity > orderItem.Quantity-orderItem.ReceivedQuantity {
				return fmt.Errorf("%w (%s: pendente %d)", domain.ErrQuantityExceeded, orderItem.ProductName, orderItem.Quantity-orderItem.ReceivedQuantity)
			}
			lines[i] = domain.CostLine{Quantity: it.Quantity, UnitCost: it.UnitCost}
		}

		freight, other, discount := domain.Round2(req.Freight), domain.Round2(req.OtherExpenses), domain.Round2(req.Discount)
		landed, itemsTotal, total, err := domain.AllocateLandedCosts(lines, freight, other, discount)
		if err != nil {
			return err
		}
		if err := domain.ValidateInstallments(req.Installments, total); err != nil {
			return err
		}
		if err := validateBillRefs(ctx, repo, order.CompanyID, req); err != nil {
			return err
		}

		loc, err := s.location(ctx, repo, order.CompanyID)
		if err != nil {
			return err
		}
		receivedAt, err := receivedMoment(req.ReceivedAt, time.Now().In(loc))
		if err != nil {
			return err
		}

		receiptPg, err := repo.CreatePurchaseReceipt(ctx, db.CreatePurchaseReceiptParams{
			CompanyID:     order.CompanyID,
			OrderID:       order.ID,
			ReceivedAt:    pgconv.TimeToPgTimestamptz(receivedAt),
			InvoiceNumber: pgconv.ParseStringToPgType(strings.TrimSpace(req.InvoiceNumber)),
			InvoiceSeries: pgconv.ParseStringToPgType(strings.TrimSpace(req.InvoiceSeries)),
			InvoiceKey:    pgconv.ParseStringToPgType(strings.TrimSpace(req.InvoiceKey)),
			ItemsTotal:    pgconv.Float64ToPgNumeric(itemsTotal),
			Freight:       pgconv.Float64ToPgNumeric(freight),
			OtherExpenses: pgconv.Float64ToPgNumeric(other),
			Discount:      pgconv.Float64ToPgNumeric(discount),
			Total:         pgconv.Float64ToPgNumeric(total),
			Notes:         pgconv.ParseStringToPgType(strings.TrimSpace(req.Notes)),
			CreatedBy:     pgconv.ParseUUIDToPgType(userId),
		})
		if err != nil {
			return err
		}

		reason := fmt.Sprintf("Compra nº %d", order.Number)
		for i, it := range items {
			orderItem := byID[it.OrderItemID]

			product, err := repo.LockProductForPurchase(ctx, db.LockProductForPurchaseParams{ID: orderItem.ProductID, CompanyID: order.CompanyID})
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %s", domain.ErrProductNotFound, orderItem.ProductName)
			}
			if err != nil {
				return err
			}

			newCost := domain.WeightedAverageCost(product.Quantity, product.CostPrice, it.Quantity, landed[i], product.SellInBulk)

			var salePrice pgtype.Numeric
			if it.SalePrice != nil {
				salePrice = pgconv.Float64ToPgNumeric(domain.Round2(*it.SalePrice))
			}
			if err := repo.ApplyPurchaseToProduct(ctx, db.ApplyPurchaseToProductParams{
				ID:            product.ID,
				QuantityDelta: it.Quantity,
				CostPrice:     pgconv.Float64ToPgNumeric(newCost),
				SalePrice:     salePrice,
				UpdatedBy:     pgconv.ParseUUIDToPgType(userId),
			}); err != nil {
				return err
			}

			if err := repo.CreatePurchaseReceiptItem(ctx, db.CreatePurchaseReceiptItemParams{
				ReceiptID:        receiptPg,
				OrderItemID:      orderItem.ID,
				ProductID:        product.ID,
				Quantity:         it.Quantity,
				UnitCost:         pgconv.Float64ToPgNumeric(domain.Round4(it.UnitCost)),
				LandedUnitCost:   pgconv.Float64ToPgNumeric(landed[i]),
				PreviousCost:     pgconv.Float64ToPgNumeric(product.CostPrice),
				NewCost:          pgconv.Float64ToPgNumeric(newCost),
				PreviousQuantity: pgtype.Int4{Int32: product.Quantity, Valid: !product.SellInBulk},
			}); err != nil {
				return err
			}

			// Granel não controla saldo: só o custo muda
			if !product.SellInBulk {
				if err := repo.CreatePurchaseStockMovement(ctx, db.CreatePurchaseStockMovementParams{
					CompanyID:         order.CompanyID,
					ProductID:         product.ID,
					PurchaseReceiptID: receiptPg,
					Type:              db.StockMovementTypePURCHASE,
					Quantity:          it.Quantity,
					Reason:            pgconv.ParseStringToPgType(reason),
					CreatedBy:         pgconv.ParseUUIDToPgType(userId),
				}); err != nil {
					return err
				}
			}

			if _, err := repo.AddReceivedQuantity(ctx, db.AddReceivedQuantityParams{ID: orderItem.ID, OrderID: order.ID, Quantity: it.Quantity}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("%w (%s)", domain.ErrQuantityExceeded, orderItem.ProductName)
				}
				return err
			}
		}

		description := billDescription(order.Number, req.InvoiceNumber)
		for i, inst := range req.Installments {
			desc := description
			if len(req.Installments) > 1 {
				desc = fmt.Sprintf("%s · parcela %d/%d", description, i+1, len(req.Installments))
			}
			if err := repo.CreatePurchaseBill(ctx, db.CreatePurchaseBillParams{
				CompanyID:         order.CompanyID,
				VendorID:          order.VendorID,
				CategoryID:        pgconv.OptionalPtrUUIDToPgType(req.BillCategoryID),
				PaymentMethodID:   pgconv.OptionalPtrUUIDToPgType(req.PaymentMethodID),
				Amount:            pgconv.Float64ToPgNumeric(domain.Round2(inst.Amount)),
				DueDate:           pgconv.StringToPgDate(inst.DueDate),
				Description:       pgconv.ParseStringToPgType(desc),
				PurchaseReceiptID: receiptPg,
			}); err != nil {
				return err
			}
		}

		return refreshStatus(ctx, repo, order)
	})
}

// ReverseReceipt desfaz um recebimento: devolve o estoque, recalcula o custo
// médio e cancela as parcelas. Bloqueia se alguma parcela já foi paga ou se o
// saldo atual do produto não cobre a devolução.
func (s *Service) ReverseReceipt(ctx context.Context, companyId, userId, receiptId uuid.UUID, req domain.ReverseReceiptRequest) (domain.OrderDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	defer tx.Rollback(ctx)
	repo := s.repo.WithTx(tx)

	companyPg := pgconv.ParseUUIDToPgType(companyId)
	receipt, err := repo.LockPurchaseReceipt(ctx, db.LockPurchaseReceiptParams{ID: pgconv.ParseUUIDToPgType(receiptId), CompanyID: companyPg})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrderDetail{}, domain.ErrReceiptNotFound
	}
	if err != nil {
		return domain.OrderDetail{}, err
	}
	if receipt.Status != "active" {
		return domain.OrderDetail{}, domain.ErrReceiptAlreadyReversed
	}

	order, err := repo.LockPurchaseOrder(ctx, db.LockPurchaseOrderParams{ID: receipt.OrderID, CompanyID: companyPg})
	if err != nil {
		return domain.OrderDetail{}, err
	}

	paid, err := repo.CountPaidPurchaseBills(ctx, receipt.ID)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	if paid > 0 {
		return domain.OrderDetail{}, domain.ErrReceiptHasPaidBills
	}

	items, err := repo.ListPurchaseReceiptItems(ctx, receipt.ID)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProductID.String() < items[j].ProductID.String() })

	reason := fmt.Sprintf("Estorno da compra nº %d", order.Number)
	for _, it := range items {
		product, err := repo.LockProductForPurchase(ctx, db.LockProductForPurchaseParams{ID: it.ProductID, CompanyID: companyPg})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.OrderDetail{}, domain.ErrProductNotFound
		}
		if err != nil {
			return domain.OrderDetail{}, err
		}
		if !product.SellInBulk && product.Quantity < it.Quantity {
			return domain.OrderDetail{}, fmt.Errorf("%w (%s: saldo %d, devolução %d)", domain.ErrInsufficientStock, product.Name, product.Quantity, it.Quantity)
		}

		cost := domain.ReverseAverageCost(product.Quantity, product.CostPrice, it.Quantity, it.LandedUnitCost, it.PreviousCost, product.SellInBulk)
		if err := repo.ApplyPurchaseToProduct(ctx, db.ApplyPurchaseToProductParams{
			ID:            product.ID,
			QuantityDelta: -it.Quantity,
			CostPrice:     pgconv.Float64ToPgNumeric(cost),
			UpdatedBy:     pgconv.ParseUUIDToPgType(userId),
		}); err != nil {
			return domain.OrderDetail{}, err
		}

		if !product.SellInBulk {
			if err := repo.CreatePurchaseStockMovement(ctx, db.CreatePurchaseStockMovementParams{
				CompanyID:         companyPg,
				ProductID:         product.ID,
				PurchaseReceiptID: receipt.ID,
				Type:              db.StockMovementTypePURCHASEREVERSAL,
				Quantity:          -it.Quantity,
				Reason:            pgconv.ParseStringToPgType(reason),
				CreatedBy:         pgconv.ParseUUIDToPgType(userId),
			}); err != nil {
				return domain.OrderDetail{}, err
			}
		}

		if _, err := repo.AddReceivedQuantity(ctx, db.AddReceivedQuantityParams{ID: it.OrderItemID, OrderID: order.ID, Quantity: -it.Quantity}); err != nil {
			return domain.OrderDetail{}, err
		}
	}

	if err := repo.CancelPurchaseBills(ctx, db.CancelPurchaseBillsParams{
		PurchaseReceiptID: receipt.ID,
		Note:              "Cancelada pelo " + strings.ToLower(reason[:1]) + reason[1:],
	}); err != nil {
		return domain.OrderDetail{}, err
	}

	if err := repo.MarkPurchaseReceiptReversed(ctx, db.MarkPurchaseReceiptReversedParams{
		ID:             receipt.ID,
		ReversedBy:     pgconv.ParseUUIDToPgType(userId),
		ReversalReason: pgconv.ParseStringToPgType(strings.TrimSpace(req.Reason)),
	}); err != nil {
		return domain.OrderDetail{}, err
	}

	if err := refreshStatus(ctx, repo, order); err != nil {
		return domain.OrderDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.OrderDetail{}, err
	}
	return s.GetOrder(ctx, companyId, pgconv.PgUUIDToUUID(order.ID))
}

// ─── Auxiliares ──────────────────────────────────────────────────────────────

// withOrder abre a transação, trava o pedido e roda a operação; devolve o
// pedido atualizado depois do commit.
func (s *Service) withOrder(ctx context.Context, companyId, orderId uuid.UUID, fn func(repo *repository.Repository, order db.LockPurchaseOrderRow, totals db.GetPurchaseOrderItemTotalsRow) error) (domain.OrderDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	defer tx.Rollback(ctx)
	repo := s.repo.WithTx(tx)

	order, err := repo.LockPurchaseOrder(ctx, db.LockPurchaseOrderParams{
		ID:        pgconv.ParseUUIDToPgType(orderId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrderDetail{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.OrderDetail{}, err
	}

	totals, err := repo.GetPurchaseOrderItemTotals(ctx, order.ID)
	if err != nil {
		return domain.OrderDetail{}, err
	}

	if err := fn(repo, order, totals); err != nil {
		return domain.OrderDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.OrderDetail{}, err
	}
	return s.GetOrder(ctx, companyId, orderId)
}

func refreshStatus(ctx context.Context, repo *repository.Repository, order db.LockPurchaseOrderRow) error {
	totals, err := repo.GetPurchaseOrderItemTotals(ctx, order.ID)
	if err != nil {
		return err
	}
	status := domain.StatusAfterReceiving(string(order.Status), totals.Quantity, totals.ReceivedQuantity)
	if status == string(order.Status) {
		return nil
	}
	return repo.UpdatePurchaseOrderStatus(ctx, db.UpdatePurchaseOrderStatusParams{ID: order.ID, Status: db.PurchaseOrderStatus(status)})
}

func validateOrder(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID, req domain.SaveOrderRequest) error {
	vendor, err := repo.GetPurchaseVendor(ctx, db.GetPurchaseVendorParams{ID: pgconv.ParseUUIDToPgType(req.VendorID), CompanyID: companyPg})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrVendorNotFound
	}
	if err != nil {
		return err
	}
	if !vendor.IsActive {
		return domain.ErrVendorInactive
	}

	ids := make([]pgtype.UUID, 0, len(req.Items))
	seen := map[uuid.UUID]bool{}
	for _, it := range req.Items {
		if seen[it.ProductID] {
			return fmt.Errorf("%w: o mesmo produto aparece mais de uma vez no pedido", domain.ErrInvalidPurchase)
		}
		seen[it.ProductID] = true
		ids = append(ids, pgconv.ParseUUIDToPgType(it.ProductID))
	}

	products, err := repo.ListProductsForPurchase(ctx, db.ListProductsForPurchaseParams{CompanyID: companyPg, Ids: ids})
	if err != nil {
		return err
	}
	if len(products) != len(ids) {
		return domain.ErrProductNotFound
	}
	return nil
}

func createItems(ctx context.Context, repo *repository.Repository, orderPg pgtype.UUID, items []domain.OrderItemRequest) error {
	for _, it := range items {
		if err := repo.CreatePurchaseOrderItem(ctx, db.CreatePurchaseOrderItemParams{
			OrderID:   orderPg,
			ProductID: pgconv.ParseUUIDToPgType(it.ProductID),
			Quantity:  it.Quantity,
			UnitCost:  pgconv.Float64ToPgNumeric(domain.Round4(it.UnitCost)),
		}); err != nil {
			return err
		}
	}
	return nil
}

// validateBillRefs garante que categoria e forma de pagamento são da empresa
func validateBillRefs(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID, req domain.ReceiveRequest) error {
	if req.BillCategoryID != nil {
		ok, err := repo.BillCategoryBelongsToCompany(ctx, db.BillCategoryBelongsToCompanyParams{ID: pgconv.ParseUUIDToPgType(*req.BillCategoryID), CompanyID: companyPg})
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: categoria de despesa não encontrada", domain.ErrInvalidPurchase)
		}
	}
	if req.PaymentMethodID != nil {
		ok, err := repo.PaymentMethodBelongsToCompany(ctx, db.PaymentMethodBelongsToCompanyParams{ID: pgconv.ParseUUIDToPgType(*req.PaymentMethodID), CompanyID: companyPg})
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: forma de pagamento não encontrada", domain.ErrInvalidPurchase)
		}
	}
	return nil
}

func (s *Service) location(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID) (*time.Location, error) {
	tz, err := repo.GetCompanyTimezone(ctx, companyPg)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(defaultTimezone)
	}
	return loc, nil
}

// receivedMoment data do recebimento: hoje (ou vazio) usa o horário atual;
// um dia anterior fica ao meio-dia local, para não cair no dia vizinho por fuso.
func receivedMoment(date string, now time.Time) (time.Time, error) {
	if date == "" {
		return now, nil
	}
	day, err := time.ParseInLocation(dateLayout, date, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: data de recebimento inválida", domain.ErrInvalidPurchase)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case day.After(today):
		return time.Time{}, fmt.Errorf("%w: a data de recebimento não pode ser futura", domain.ErrInvalidPurchase)
	case day.Equal(today):
		return now, nil
	default:
		return day.Add(12 * time.Hour), nil
	}
}

func billDescription(orderNumber int32, invoice string) string {
	desc := fmt.Sprintf("Compra nº %d", orderNumber)
	if invoice = strings.TrimSpace(invoice); invoice != "" {
		desc += " · NF " + invoice
	}
	return desc
}

func optionalDate(date string) pgtype.Date {
	if date == "" {
		return pgtype.Date{}
	}
	return pgconv.StringToPgDate(date)
}

func datePtr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format(dateLayout)
	return &s
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func isLate(status string, expected pgtype.Date, today time.Time) bool {
	return (status == domain.StatusSent || status == domain.StatusPartial) && expected.Valid && expected.Time.Before(today)
}
