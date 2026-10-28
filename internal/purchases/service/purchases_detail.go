package service

import (
	"context"
	"errors"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// orderDetail monta o pedido completo: itens com pendências, recebimentos com
// seus itens e parcelas, e as ações liberadas no status atual.
func (s *Service) orderDetail(ctx context.Context, repo *repository.Repository, companyId, orderId uuid.UUID) (domain.OrderDetail, error) {
	orderPg := pgconv.ParseUUIDToPgType(orderId)

	order, err := repo.GetPurchaseOrder(ctx, db.GetPurchaseOrderParams{ID: orderPg, CompanyID: pgconv.ParseUUIDToPgType(companyId)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrderDetail{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.OrderDetail{}, err
	}

	items, err := repo.ListPurchaseOrderItems(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	receipts, err := repo.ListPurchaseReceipts(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	receiptItems, err := repo.ListPurchaseReceiptItemsByOrder(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	bills, err := repo.ListPurchaseBillsByOrder(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	emails, err := repo.ListPurchaseOrderEmails(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}
	vendorEmail, err := repo.GetPurchaseOrderVendorEmail(ctx, orderPg)
	if err != nil {
		return domain.OrderDetail{}, err
	}

	detail := domain.OrderDetail{
		ID:            pgconv.PgUUIDToUUID(order.ID),
		Number:        order.Number,
		VendorID:      pgconv.PgUUIDToUUID(order.VendorID),
		VendorName:    order.VendorName,
		Status:        order.Status,
		ExpectedDate:  datePtr(order.ExpectedDate),
		Notes:         textPtr(order.Notes),
		CreatedByName: order.CreatedByName,
		CreatedAt:     order.CreatedAt.Time,
		UpdatedAt:     order.UpdatedAt.Time,
		SentAt:        timePtr(order.SentAt),
		ClosedAt:      timePtr(order.ClosedAt),
		CloseReason:   textPtr(order.CloseReason),
		VendorEmail:   vendorEmail,
		Emails:        make([]domain.OrderEmail, 0, len(emails)),
		Items:         make([]domain.OrderItem, 0, len(items)),
		Receipts:      make([]domain.Receipt, 0, len(receipts)),
	}

	for _, it := range items {
		total := domain.Round2(float64(it.Quantity) * it.UnitCost)
		detail.Total += total
		detail.Quantity += it.Quantity
		detail.ReceivedQuantity += it.ReceivedQuantity
		detail.Items = append(detail.Items, domain.OrderItem{
			ID:               pgconv.PgUUIDToUUID(it.ID),
			ProductID:        pgconv.PgUUIDToUUID(it.ProductID),
			ProductName:      it.ProductName,
			ModelName:        it.ModelName,
			Color:            it.Color,
			Size:             it.Size,
			Barcode:          it.Barcode,
			Unit:             it.Unit,
			SellInBulk:       it.SellInBulk,
			StockQuantity:    it.StockQuantity,
			CurrentCost:      it.CurrentCost,
			SalePrice:        it.SalePrice,
			Quantity:         it.Quantity,
			UnitCost:         it.UnitCost,
			Total:            total,
			ReceivedQuantity: it.ReceivedQuantity,
			PendingQuantity:  it.Quantity - it.ReceivedQuantity,
		})
	}
	detail.Total = domain.Round2(detail.Total)

	itemsByReceipt := map[uuid.UUID][]domain.ReceiptItem{}
	for _, it := range receiptItems {
		id := pgconv.PgUUIDToUUID(it.ReceiptID)
		itemsByReceipt[id] = append(itemsByReceipt[id], domain.ReceiptItem{
			OrderItemID:    pgconv.PgUUIDToUUID(it.OrderItemID),
			ProductID:      pgconv.PgUUIDToUUID(it.ProductID),
			ProductName:    it.ProductName,
			Quantity:       it.Quantity,
			UnitCost:       it.UnitCost,
			LandedUnitCost: it.LandedUnitCost,
			PreviousCost:   it.PreviousCost,
			NewCost:        it.NewCost,
		})
	}

	billsByReceipt := map[uuid.UUID][]domain.ReceiptBill{}
	for _, b := range bills {
		id := pgconv.PgUUIDToUUID(b.PurchaseReceiptID)
		billsByReceipt[id] = append(billsByReceipt[id], domain.ReceiptBill{
			ID:          pgconv.PgUUIDToUUID(b.ID),
			Description: b.Description,
			Amount:      b.Amount,
			DueDate:     pgconv.PgDateToString(b.DueDate),
			Status:      b.Status,
			AmountPaid:  b.AmountPaid,
			PaymentDate: datePtr(b.PaymentDate),
		})
	}

	for _, r := range receipts {
		id := pgconv.PgUUIDToUUID(r.ID)
		if r.Status == "active" {
			detail.ReceivedTotal += r.Total
		}
		receipt := domain.Receipt{
			ID:             id,
			Status:         r.Status,
			ReceivedAt:     r.ReceivedAt.Time,
			InvoiceNumber:  r.InvoiceNumber,
			InvoiceSeries:  r.InvoiceSeries,
			InvoiceKey:     r.InvoiceKey,
			ItemsTotal:     r.ItemsTotal,
			Freight:        r.Freight,
			OtherExpenses:  r.OtherExpenses,
			Discount:       r.Discount,
			Total:          r.Total,
			Notes:          textPtr(r.Notes),
			CreatedByName:  r.CreatedByName,
			ReversedAt:     timePtr(r.ReversedAt),
			ReversedByName: r.ReversedByName,
			ReversalReason: textPtr(r.ReversalReason),
			Items:          itemsByReceipt[id],
			Bills:          billsByReceipt[id],
		}
		if receipt.Items == nil {
			receipt.Items = []domain.ReceiptItem{}
		}
		if receipt.Bills == nil {
			receipt.Bills = []domain.ReceiptBill{}
		}
		detail.Receipts = append(detail.Receipts, receipt)
	}
	detail.ReceivedTotal = domain.Round2(detail.ReceivedTotal)

	for _, e := range emails {
		detail.Emails = append(detail.Emails, domain.OrderEmail{
			Kind:       e.Kind,
			Recipient:  e.Recipient,
			SentByName: e.SentByName,
			CreatedAt:  e.CreatedAt.Time,
		})
	}

	actions := domain.ActionsFor(order.Status, detail.ReceivedQuantity)
	detail.CanEdit = actions.CanEdit
	detail.CanSend = actions.CanSend
	detail.CanReceive = actions.CanReceive
	detail.CanClose = actions.CanClose
	detail.CanCancel = actions.CanCancel
	detail.CanEmail = actions.CanEmail

	return detail, nil
}
