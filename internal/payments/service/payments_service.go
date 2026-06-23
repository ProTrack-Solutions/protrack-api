package service

import (
	"context"
	"errors"

	accReceivableDomain "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/domain"
	accReceivableService "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/service"
	customerDomain "github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	customerService "github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	paymentHistoryDomain "github.com/ProTrack-Solutions/protrack-api/internal/payment_history/domain"
	paymentHistoryService "github.com/ProTrack-Solutions/protrack-api/internal/payment_history/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/payments/domain"
	saleService "github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool                  *pgxpool.Pool
	paymentHistoryService *paymentHistoryService.Service
	accReceivableService  *accReceivableService.Service
	customersService      *customerService.Service
	saleService           *saleService.Service
}

func NewService(
	pool *pgxpool.Pool,
	paymentHistoryService *paymentHistoryService.Service,
	accReceivableService *accReceivableService.Service,
	customersService *customerService.Service,
	saleService *saleService.Service,
) *Service {
	return &Service{
		pool:                  pool,
		paymentHistoryService: paymentHistoryService,
		accReceivableService:  accReceivableService,
		customersService:      customersService,
		saleService:           saleService,
	}
}

func (s *Service) NewPayment(ctx context.Context, companyId, userId uuid.UUID, req domain.CreatePaymentRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	customer, err := s.customersService.GetCustomerByIdTx(ctx, tx, req.CustomerID)
	if err != nil {
		return err
	}

	if customer.BalanceDue < req.AmountPaid {
		return errors.New("The amount entered is greater than the outstanding balance.")
	}

	reqAcc := accReceivableDomain.UpdateAccountReceivableBalanceRequest{
		Balance: req.AmountPaid,
	}

	saleId, err := s.accReceivableService.UpdateAccountReceivableBalanceTx(ctx, tx, companyId, req.CustomerID, userId, reqAcc)
	if err != nil {
		return err
	}

	accounts, err := s.accReceivableService.GetReceivablesBySaleTx(ctx, tx, saleId)

	if err == nil && len(accounts) > 0 {
		var totalRemaining float64 = 0
		for _, acc := range accounts {
			totalRemaining += acc.Balance
		}

		var status string

		if totalRemaining <= 0 {
			status = "paid"
		} else {
			status = "partial"
		}

		if err := s.saleService.UpdateSaleStatusTx(ctx, tx, saleId, companyId, userId, status); err != nil {
			return err
		}
	}
	/* if err == nil && len(accounts) > 0 {
		paidCount := 0
		for _, acc := range accounts {
			if acc.Balance <= 0 {
				paidCount++
			}
		}

		var status string

		if paidCount == len(accounts) {
			status = "paid"
		} else if paidCount > 0 {
			status = "partial"
		} else {
			status = "pending"
		}

		if err := s.saleService.UpdateSaleStatusTx(ctx, tx, saleId, companyId, userId, status); err != nil {
			return err
		}
	} */

	// paidCount := 0
	// totalCount := len(accounts)

	// for _, account := range accounts {
	// 	if account.Balance <= 0 {
	// 		paidCount++
	// 	}
	// }

	// var status string
	// if paidCount == totalCount && totalCount > 0 {
	// 	status = "paid"
	// } else if paidCount > 0 {
	// 	status = "partial"
	// } else {
	// 	status = "pending"
	// }

	// if err := s.saleService.UpdateSaleStatusTx(ctx, tx, saleId, companyId, saleId, status); err != nil {
	// 	return err
	// }

	var reqCustomer customerDomain.UpdateBalanceDueCustomerRequest

	reqCustomer.BalanceDue = req.AmountPaid
	reqCustomer.UpdatedBy = userId

	if err := s.customersService.UpdateCustomerBalanceSubTx(ctx, tx, req.CustomerID, reqCustomer); err != nil {
		return err
	}

	var reqPH paymentHistoryDomain.CreatePaymentHistoryRequest

	reqPH.AmountPaid = req.AmountPaid
	reqPH.CompanyID = companyId
	reqPH.CustomerID = req.CustomerID
	reqPH.Notes = req.Notes
	reqPH.PaymentMethodID = req.PaymentMethodID
	reqPH.SaleID = saleId
	reqPH.UserID = userId

	if err := s.paymentHistoryService.CreatePaymentHistoryTx(ctx, tx, reqPH); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
