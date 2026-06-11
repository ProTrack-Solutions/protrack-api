package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/accounts_receivable/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/accounts_receivable/repository"
	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateAccountReceivable(ctx context.Context, arg db.CreateAccountReceivableParams) error
	GetCustomerDebtSummary(ctx context.Context, customerId pgtype.UUID) (db.GetCustomerDebtSummaryRow, error)
	GetPendingReceivablesByCustomer(ctx context.Context, arg db.GetPendingReceivablesByCustomerParams) ([]db.AccountsReceivable, error)
	GetReceivablesBySale(ctx context.Context, saleId pgtype.UUID) ([]db.AccountsReceivable, error)
	ListOverdueReceivables(ctx context.Context, companyId pgtype.UUID) ([]db.ListOverdueReceivablesRow, error)
	UpdateAccountReceivableBalance(ctx context.Context, arg db.UpdateAccountReceivableBalanceParams) (pgtype.UUID, error)
	GetTotalOpenAmountByCompany(ctx context.Context, companyId pgtype.UUID) (db.GetTotalOpenAmountByCompanyRow, error)
	GetTotalOverdueAmountByCompany(ctx context.Context, companyId pgtype.UUID) (db.GetTotalOverdueAmountByCompanyRow, error)
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreateAccountReceivable(ctx context.Context, tx db.DBTX, userId uuid.UUID, companyId uuid.UUID, req domain.CreateAccountReceivableRequest) error {
	txRepo := s.repo.WithTx(tx)

	status := "pending"

	if req.Balance < req.TotalAmount {
		status = "partial"
	}

	return txRepo.CreateAccountReceivable(ctx, db.CreateAccountReceivableParams{
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		CustomerID:        pgconv.ParseUUIDToPgType(req.CustomerID),
		SaleID:            pgconv.ParseUUIDToPgType(req.SaleID),
		TotalAmount:       pgconv.Float64ToPgNumeric(req.TotalAmount),
		Balance:           pgconv.Float64ToPgNumeric(req.Balance),
		DueDate:           pgconv.StringToPgDate(req.DueDate),
		InstallmentNumber: pgconv.IntToPgInt4(int(req.InstallmentNumber)),
		TotalInstallments: pgconv.IntToPgInt4(int(req.TotalInstallments)),
		Status:            status,
		CreatedBy:         pgconv.ParseUUIDToPgType(userId),
	})
}

// CreateAccountReceivableInTx cria uma conta a receber dentro da transação tx (ex.: mesma transação da venda).
func (s *Service) CreateAccountReceivableInTx(ctx context.Context, tx db.DBTX, userId, companyId uuid.UUID, req domain.CreateAccountReceivableRequest) error {
	status := "pending"
	if req.Balance < req.TotalAmount {
		status = "partial"
	}
	repoTx := repository.NewRepository(tx)
	return repoTx.CreateAccountReceivable(ctx, db.CreateAccountReceivableParams{
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		CustomerID:        pgconv.ParseUUIDToPgType(req.CustomerID),
		SaleID:            pgconv.ParseUUIDToPgType(req.SaleID),
		TotalAmount:       pgconv.Float64ToPgNumeric(req.TotalAmount),
		Balance:           pgconv.Float64ToPgNumeric(req.Balance),
		DueDate:           pgconv.StringToPgDate(req.DueDate),
		InstallmentNumber: pgconv.IntToPgInt4(int(req.InstallmentNumber)),
		TotalInstallments: pgconv.IntToPgInt4(int(req.TotalInstallments)),
		Status:            status,
		CreatedBy:         pgconv.ParseUUIDToPgType(userId),
	})
}

func (s *Service) GetCustomerDebtSummary(ctx context.Context, customerId uuid.UUID) (domain.GetCustomerDebtSummaryRow, error) {
	account, err := s.repo.GetCustomerDebtSummary(ctx, pgconv.ParseUUIDToPgType(customerId))
	if err != nil {
		return domain.GetCustomerDebtSummaryRow{}, err
	}

	return domain.GetCustomerDebtSummaryRow{
		TotalCount:    account.TotalCount,
		TotalBalance:  pgconv.PgNumericToFloat64(account.TotalBalance),
		OldestDueDate: pgconv.PgDateToString(account.OldestDueDate),
	}, nil
}

func (s *Service) GetPendingReceivablesByCustomer(ctx context.Context, companyId, customerId uuid.UUID) ([]domain.AccountsReceivableResponse, error) {
	accounts, err := s.repo.GetPendingReceivablesByCustomer(ctx, db.GetPendingReceivablesByCustomerParams{
		CustomerID: pgconv.ParseUUIDToPgType(customerId),
		CompanyID:  pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return []domain.AccountsReceivableResponse{}, err
	}

	var response []domain.AccountsReceivableResponse

	for _, account := range accounts {
		response = append(response, domain.AccountsReceivableResponse{
			ID:                pgconv.PgUUIDToUUID(account.ID),
			CompanyID:         pgconv.PgUUIDToUUID(account.CompanyID),
			CustomerID:        pgconv.PgUUIDToUUID(account.CustomerID),
			SaleID:            pgconv.PgUUIDToUUID(account.SaleID),
			TotalAmount:       pgconv.PgNumericToFloat64(account.TotalAmount),
			Balance:           pgconv.PgNumericToFloat64(account.Balance),
			DueDate:           pgconv.PgDateToString(account.DueDate),
			InstallmentNumber: int64(pgconv.PgInt4ToInt(account.InstallmentNumber)),
			TotalInstallments: int64(pgconv.PgInt4ToInt(account.TotalInstallments)),
			Status:            account.Status,
			CreatedAt:         pgconv.PgTimestamptzToTime(account.CreatedAt),
			CreatedBy:         pgconv.PgUUIDToUUID(account.CreatedBy),
			UpdatedAt:         pgconv.PgTimestamptzToTime(account.UpdatedAt),
			UpdatedBy:         pgconv.PgUUIDToUUID(account.UpdatedBy),
			DeletedAt:         pgconv.PgTimestamptzToTime(account.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) GetReceivablesBySale(ctx context.Context, saleId uuid.UUID) ([]domain.AccountsReceivableResponse, error) {
	accounts, err := s.repo.GetReceivablesBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if err != nil {
		return []domain.AccountsReceivableResponse{}, err
	}

	var response []domain.AccountsReceivableResponse

	for _, account := range accounts {
		response = append(response, domain.AccountsReceivableResponse{
			ID:                pgconv.PgUUIDToUUID(account.ID),
			CompanyID:         pgconv.PgUUIDToUUID(account.CompanyID),
			CustomerID:        pgconv.PgUUIDToUUID(account.CustomerID),
			SaleID:            pgconv.PgUUIDToUUID(account.SaleID),
			TotalAmount:       pgconv.PgNumericToFloat64(account.TotalAmount),
			Balance:           pgconv.PgNumericToFloat64(account.Balance),
			DueDate:           pgconv.PgDateToString(account.DueDate),
			InstallmentNumber: int64(pgconv.PgInt4ToInt(account.InstallmentNumber)),
			TotalInstallments: int64(pgconv.PgInt4ToInt(account.TotalInstallments)),
			Status:            account.Status,
			CreatedAt:         pgconv.PgTimestamptzToTime(account.CreatedAt),
			CreatedBy:         pgconv.PgUUIDToUUID(account.CreatedBy),
			UpdatedAt:         pgconv.PgTimestamptzToTime(account.UpdatedAt),
			UpdatedBy:         pgconv.PgUUIDToUUID(account.UpdatedBy),
			DeletedAt:         pgconv.PgTimestamptzToTime(account.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) GetReceivablesBySaleTx(ctx context.Context, tx db.DBTX, saleId uuid.UUID) ([]domain.AccountsReceivableResponse, error) {
	repoTx := db.New(tx)

	accounts, err := repoTx.GetReceivablesBySale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if err != nil {
		return []domain.AccountsReceivableResponse{}, err
	}

	var response []domain.AccountsReceivableResponse

	for _, account := range accounts {
		response = append(response, domain.AccountsReceivableResponse{
			ID:                pgconv.PgUUIDToUUID(account.ID),
			CompanyID:         pgconv.PgUUIDToUUID(account.CompanyID),
			CustomerID:        pgconv.PgUUIDToUUID(account.CustomerID),
			SaleID:            pgconv.PgUUIDToUUID(account.SaleID),
			TotalAmount:       pgconv.PgNumericToFloat64(account.TotalAmount),
			Balance:           pgconv.PgNumericToFloat64(account.Balance),
			DueDate:           pgconv.PgDateToString(account.DueDate),
			InstallmentNumber: int64(pgconv.PgInt4ToInt(account.InstallmentNumber)),
			TotalInstallments: int64(pgconv.PgInt4ToInt(account.TotalInstallments)),
			Status:            account.Status,
			CreatedAt:         pgconv.PgTimestamptzToTime(account.CreatedAt),
			CreatedBy:         pgconv.PgUUIDToUUID(account.CreatedBy),
			UpdatedAt:         pgconv.PgTimestamptzToTime(account.UpdatedAt),
			UpdatedBy:         pgconv.PgUUIDToUUID(account.UpdatedBy),
			DeletedAt:         pgconv.PgTimestamptzToTime(account.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) ListOverdueReceivablesTx(ctx context.Context, tx db.DBTX, companyId uuid.UUID) ([]domain.ListOverdueReceivablesRow, error) {
	repoTx := s.repo.WithTx(tx)

	accounts, err := repoTx.ListOverdueReceivables(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListOverdueReceivablesRow{}, err
	}

	var response []domain.ListOverdueReceivablesRow

	for _, account := range accounts {
		response = append(response, domain.ListOverdueReceivablesRow{
			ID:                pgconv.PgUUIDToUUID(account.CompanyID),
			CompanyID:         pgconv.PgUUIDToUUID(account.CompanyID),
			CustomerID:        pgconv.PgUUIDToUUID(account.CustomerID),
			SaleID:            pgconv.PgUUIDToUUID(account.SaleID),
			TotalAmount:       pgconv.PgNumericToFloat64(account.TotalAmount),
			Balance:           pgconv.PgNumericToFloat64(account.Balance),
			DueDate:           pgconv.PgDateToString(account.DueDate),
			InstallmentNumber: int64(pgconv.PgInt4ToInt(account.InstallmentNumber)),
			TotalInstallments: int64(pgconv.PgInt4ToInt(account.TotalInstallments)),
			Status:            account.Status,
			CreatedAt:         pgconv.PgTimestamptzToTime(account.CreatedAt),
			CreatedBy:         pgconv.PgUUIDToUUID(account.CreatedBy),
			UpdatedAt:         pgconv.PgTimestamptzToTime(account.UpdatedAt),
			UpdatedBy:         pgconv.PgUUIDToUUID(account.UpdatedBy),
			DeletedAt:         pgconv.PgTimestamptzToTime(account.DeletedAt),
			CustomerName:      account.CustomerName,
			DaysOverdue:       account.DaysOverdue,
		})
	}

	return response, nil
}

func (s *Service) ListOverdueReceivables(ctx context.Context, companyId uuid.UUID) ([]domain.ListOverdueReceivablesRow, error) {
	accounts, err := s.repo.ListOverdueReceivables(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListOverdueReceivablesRow{}, err
	}

	var response []domain.ListOverdueReceivablesRow

	for _, account := range accounts {
		response = append(response, domain.ListOverdueReceivablesRow{
			ID:                pgconv.PgUUIDToUUID(account.CompanyID),
			CompanyID:         pgconv.PgUUIDToUUID(account.CompanyID),
			CustomerID:        pgconv.PgUUIDToUUID(account.CustomerID),
			SaleID:            pgconv.PgUUIDToUUID(account.SaleID),
			TotalAmount:       pgconv.PgNumericToFloat64(account.TotalAmount),
			Balance:           pgconv.PgNumericToFloat64(account.Balance),
			DueDate:           pgconv.PgDateToString(account.DueDate),
			InstallmentNumber: int64(pgconv.PgInt4ToInt(account.InstallmentNumber)),
			TotalInstallments: int64(pgconv.PgInt4ToInt(account.TotalInstallments)),
			Status:            account.Status,
			CreatedAt:         pgconv.PgTimestamptzToTime(account.CreatedAt),
			CreatedBy:         pgconv.PgUUIDToUUID(account.CreatedBy),
			UpdatedAt:         pgconv.PgTimestamptzToTime(account.UpdatedAt),
			UpdatedBy:         pgconv.PgUUIDToUUID(account.UpdatedBy),
			DeletedAt:         pgconv.PgTimestamptzToTime(account.DeletedAt),
			CustomerName:      account.CustomerName,
			DaysOverdue:       account.DaysOverdue,
		})
	}

	return response, nil
}

func (s *Service) UpdateAccountReceivableBalance(ctx context.Context, companyId, customerId, userId uuid.UUID, req domain.UpdateAccountReceivableBalanceRequest) (uuid.UUID, error) {
	accounts, err := s.repo.GetPendingReceivablesByCustomer(ctx, db.GetPendingReceivablesByCustomerParams{
		CustomerID: pgconv.ParseUUIDToPgType(customerId),
		CompanyID:  pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return uuid.Nil, err
	}

	remaining := req.Balance

	var saleID uuid.UUID

	for _, account := range accounts {
		if remaining <= 0 {
			break
		}

		currentAccountBalance := pgconv.PgNumericToFloat64(account.Balance)

		var amountToApply float64
		var newBalance float64
		var newStatus string

		if remaining >= currentAccountBalance {
			amountToApply = currentAccountBalance
			newBalance = 0
			newStatus = "paid"
		} else {
			amountToApply = remaining
			newBalance = currentAccountBalance - remaining
			newStatus = "partial"
		}

		saleIdPg, err := s.repo.UpdateAccountReceivableBalance(ctx, db.UpdateAccountReceivableBalanceParams{
			Balance:   pgconv.Float64ToPgNumeric(newBalance),
			Status:    newStatus,
			UpdatedBy: pgconv.ParseUUIDToPgType(userId),
			ID:        account.ID,
		})
		if err != nil {
			return uuid.Nil, err
		}

		saleID = pgconv.PgUUIDToUUID(saleIdPg)

		remaining -= amountToApply
	}
	return saleID, nil
}

func (s *Service) UpdateAccountReceivableBalanceTx(ctx context.Context, tx db.DBTX, companyId, customerId, userId uuid.UUID, req domain.UpdateAccountReceivableBalanceRequest) (uuid.UUID, error) {
	repoTx := db.New(tx)

	accounts, err := repoTx.GetPendingReceivablesByCustomer(ctx, db.GetPendingReceivablesByCustomerParams{
		CustomerID: pgconv.ParseUUIDToPgType(customerId),
		CompanyID:  pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return uuid.Nil, err
	}

	var saleID uuid.UUID

	remaining := req.Balance

	for _, account := range accounts {
		if remaining <= 0 {
			break
		}

		currentAccountBalance := pgconv.PgNumericToFloat64(account.Balance)

		var amountToApply float64
		var newBalance float64
		var newStatus string

		if remaining >= currentAccountBalance {
			amountToApply = currentAccountBalance
			newBalance = 0
			newStatus = "paid"
		} else {
			amountToApply = remaining
			newBalance = currentAccountBalance - remaining
			newStatus = "partial"
		}

		saleIdPg, err := repoTx.UpdateAccountReceivableBalance(ctx, db.UpdateAccountReceivableBalanceParams{
			Balance:   pgconv.Float64ToPgNumeric(newBalance),
			Status:    newStatus,
			UpdatedBy: pgconv.ParseUUIDToPgType(userId),
			ID:        account.ID,
		})
		if err != nil {
			return uuid.Nil, err
		}
		saleID = pgconv.PgUUIDToUUID(saleIdPg)

		remaining -= amountToApply
	}
	return saleID, nil
}

func (s *Service) GetTotalOpenAmountByCompany(ctx context.Context, companyId uuid.UUID) (float64, error) {
	total, err := s.repo.GetTotalOpenAmountByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return pgconv.PgNumericToFloat64(total.TotalOpen), nil
}

func (s *Service) GetTotalOverdueAmountByCompany(ctx context.Context, companyId uuid.UUID) (float64, error) {
	total, err := s.repo.GetTotalOverdueAmountByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return pgconv.PgNumericToFloat64(total.TotalOverdue), nil
}
