package service

import (
	"context"
	"errors"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateBillsPayable(ctx context.Context, arg db.CreateBillPayableParams) error
	GetBillsByStatus(ctx context.Context, arg db.GetBillsByStatusParams) ([]db.BillsPayable, error)
	GetOverdueBills(ctx context.Context, companyId pgtype.UUID) ([]db.BillsPayable, error)
	ListBillsPayable(ctx context.Context, companyId pgtype.UUID) ([]db.ListBillsPayableRow, error)
	PayBill(ctx context.Context, arg db.PayBillParams) error
	UpdateBillPayable(ctx context.Context, arg db.UpdateBillPayableParams) error
	GetBillsById(ctx context.Context, arg db.GetBillsByIdParams) (db.BillsPayable, error)
	ScheduleBill(ctx context.Context, arg db.ScheduleBillParams) error
	GetBillsPayableSummary(ctx context.Context, companyId pgtype.UUID) (db.GetBillsPayableSummaryRow, error)
	UpdateOverdueBillsPayable(ctx context.Context) error
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

func (s *Service) CreateBillPayable(ctx context.Context, companyId uuid.UUID, req domain.CreateBillPayableRequest) error {
	return s.repo.CreateBillsPayable(ctx, db.CreateBillPayableParams{
		CompanyID:       pgconv.ParseUUIDToPgType(companyId),
		VendorID:        pgconv.ParseUUIDToPgType(req.VendorID),
		CategoryID:      pgconv.ParseUUIDToPgType(req.CategoryID),
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodID),
		Amount:          pgconv.Float64ToPgNumeric(req.Amount),
		DueDate:         pgconv.StringToPgDate(req.DueDate),
		Status:          req.Status,
		Description:     pgconv.ParseStringToPgText(req.Description),
		Notes:           pgconv.ParseStringToPgText(req.Notes),
	})
}

func (s *Service) GetBillsPayableById(ctx context.Context, req domain.GetBillsByIdRequest) (domain.BillsPayableResponse, error) {
	billPayable, err := s.repo.GetBillsById(ctx, db.GetBillsByIdParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		ID:        pgconv.ParseUUIDToPgType(req.ID),
	})
	if err != nil {
		return domain.BillsPayableResponse{}, err
	}

	statusStr := billPayable.Status.(string)

	return domain.BillsPayableResponse{
		ID:              pgconv.PgUUIDToUUID(billPayable.ID),
		CompanyID:       pgconv.PgUUIDToUUID(billPayable.CompanyID),
		VendorID:        pgconv.PgUUIDToUUID(billPayable.VendorID),
		CategoryID:      pgconv.PgUUIDToUUID(billPayable.CompanyID),
		PaymentMethodID: pgconv.PgUUIDToUUID(billPayable.PaymentMethodID),
		Amount:          pgconv.PgNumericToFloat64(billPayable.Amount),
		DueDate:         pgconv.PgDateToString(billPayable.DueDate),
		Status:          statusStr,
		Description:     pgconv.ParsePgTextToString(billPayable.Description),
		ScheduledDate:   pgconv.PgDateToString(billPayable.ScheduledDate),
		PaymentDate:     pgconv.PgDateToString(billPayable.PaymentDate),
		AmountPaid:      pgconv.PgNumericToFloat64(billPayable.AmountPaid),
		Notes:           pgconv.ParsePgTextToString(billPayable.Notes),
		CreatedAt:       pgconv.PgTimestamptzToTime(billPayable.CreatedAt),
		UpdatedAt:       pgconv.PgTimestamptzToTime(billPayable.UpdatedAt),
	}, nil
}

func (s *Service) GetBillsByStatus(ctx context.Context, req domain.GetBillsByStatusRequest) ([]domain.BillsPayableResponse, error) {
	billsPayable, err := s.repo.GetBillsByStatus(ctx, db.GetBillsByStatusParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		Status:    req.Status,
	})
	if err != nil {
		return []domain.BillsPayableResponse{}, err
	}

	var response []domain.BillsPayableResponse

	for _, billPayable := range billsPayable {
		statusStr := billPayable.Status.(string)
		response = append(response, domain.BillsPayableResponse{
			ID:              pgconv.PgUUIDToUUID(billPayable.ID),
			CompanyID:       pgconv.PgUUIDToUUID(billPayable.CompanyID),
			VendorID:        pgconv.PgUUIDToUUID(billPayable.VendorID),
			CategoryID:      pgconv.PgUUIDToUUID(billPayable.CompanyID),
			PaymentMethodID: pgconv.PgUUIDToUUID(billPayable.PaymentMethodID),
			Amount:          pgconv.PgNumericToFloat64(billPayable.Amount),
			DueDate:         pgconv.PgDateToString(billPayable.DueDate),
			Status:          statusStr,
			Description:     pgconv.ParsePgTextToString(billPayable.Description),
			ScheduledDate:   pgconv.PgDateToString(billPayable.ScheduledDate),
			PaymentDate:     pgconv.PgDateToString(billPayable.PaymentDate),
			AmountPaid:      pgconv.PgNumericToFloat64(billPayable.AmountPaid),
			Notes:           pgconv.ParsePgTextToString(billPayable.Notes),
			CreatedAt:       pgconv.PgTimestamptzToTime(billPayable.CreatedAt),
			UpdatedAt:       pgconv.PgTimestamptzToTime(billPayable.UpdatedAt),
		})
	}

	return response, nil
}

func (s *Service) GetOverdueBills(ctx context.Context, companyId uuid.UUID) ([]domain.BillsPayableResponse, error) {
	billsPayable, err := s.repo.GetOverdueBills(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.BillsPayableResponse{}, err
	}

	var response []domain.BillsPayableResponse

	for _, billPayable := range billsPayable {
		statusStr := billPayable.Status.(string)
		response = append(response, domain.BillsPayableResponse{
			ID:              pgconv.PgUUIDToUUID(billPayable.ID),
			CompanyID:       pgconv.PgUUIDToUUID(billPayable.CompanyID),
			VendorID:        pgconv.PgUUIDToUUID(billPayable.VendorID),
			CategoryID:      pgconv.PgUUIDToUUID(billPayable.CompanyID),
			PaymentMethodID: pgconv.PgUUIDToUUID(billPayable.PaymentMethodID),
			Amount:          pgconv.PgNumericToFloat64(billPayable.Amount),
			DueDate:         pgconv.PgDateToString(billPayable.DueDate),
			Status:          statusStr,
			Description:     pgconv.ParsePgTextToString(billPayable.Description),
			ScheduledDate:   pgconv.PgDateToString(billPayable.ScheduledDate),
			PaymentDate:     pgconv.PgDateToString(billPayable.PaymentDate),
			AmountPaid:      pgconv.PgNumericToFloat64(billPayable.AmountPaid),
			Notes:           pgconv.ParsePgTextToString(billPayable.Notes),
			CreatedAt:       pgconv.PgTimestamptzToTime(billPayable.CreatedAt),
			UpdatedAt:       pgconv.PgTimestamptzToTime(billPayable.UpdatedAt),
		})
	}
	return response, nil
}

func (s *Service) ListBillsPayable(ctx context.Context, companyId uuid.UUID) ([]domain.ListBillsPayableRow, error) {
	billsPayable, err := s.repo.ListBillsPayable(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListBillsPayableRow{}, err
	}

	var response []domain.ListBillsPayableRow

	for _, billPayable := range billsPayable {
		statusStr := billPayable.Status.(string)
		response = append(response, domain.ListBillsPayableRow{
			ID:                pgconv.PgUUIDToUUID(billPayable.ID),
			CompanyID:         pgconv.PgUUIDToUUID(billPayable.CompanyID),
			VendorID:          pgconv.PgUUIDToUUID(billPayable.VendorID),
			CategoryID:        pgconv.PgUUIDToUUID(billPayable.CategoryID),
			PaymentMethodID:   pgconv.PgUUIDToUUID(billPayable.PaymentMethodID),
			Amount:            pgconv.PgNumericToFloat64(billPayable.Amount),
			DueDate:           pgconv.PgDateToString(billPayable.DueDate),
			Status:            statusStr,
			Description:       pgconv.ParsePgTextToString(billPayable.Description),
			ScheduledDate:     pgconv.PgDateToString(billPayable.ScheduledDate),
			PaymentDate:       pgconv.PgDateToString(billPayable.PaymentDate),
			AmountPaid:        pgconv.PgNumericToFloat64(billPayable.AmountPaid),
			Notes:             pgconv.ParsePgTextToString(billPayable.Notes),
			CreatedAt:         pgconv.PgTimestamptzToTime(billPayable.CreatedAt),
			UpdatedAt:         pgconv.PgTimestamptzToTime(billPayable.UpdatedAt),
			VendorName:        pgconv.ParsePgTextToString(billPayable.VendorName),
			CategoryName:      pgconv.ParsePgTextToString(billPayable.CategoryName),
			PaymentMethodName: pgconv.ParsePgTextToString(billPayable.PaymentMethodName),
		})
	}

	return response, nil
}

func (s *Service) PayBill(ctx context.Context, req domain.PayBillRequest) error {
	currentBillPayable, err := s.repo.GetBillsById(ctx, db.GetBillsByIdParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		ID:        pgconv.ParseUUIDToPgType(req.ID),
	})
	if err != nil {
		return err
	}

	amountToPay := req.AmountPaid
	if amountToPay == 0 {
		amountToPay = pgconv.PgNumericToFloat64(currentBillPayable.Amount)
	}

	return s.repo.PayBill(ctx, db.PayBillParams{
		ID:              pgconv.ParseUUIDToPgType(req.ID),
		CompanyID:       pgconv.ParseUUIDToPgType(req.CompanyID),
		PaymentDate:     pgconv.StringToPgDate(req.PaymentDate),
		AmountPaid:      pgconv.Float64ToPgNumeric(amountToPay),
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodID),
	})
}

func (s *Service) UpdateBillPayable(ctx context.Context, req domain.UpdateBillPayableRequest) error {
	currentBillPayable, err := s.repo.GetBillsById(ctx, db.GetBillsByIdParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		ID:        pgconv.ParseUUIDToPgType(req.ID),
	})
	if err != nil {
		return err
	}

	arg := db.UpdateBillPayableParams{
		ID:              currentBillPayable.ID,
		CompanyID:       currentBillPayable.CompanyID,
		VendorID:        currentBillPayable.VendorID,
		CategoryID:      currentBillPayable.CategoryID,
		PaymentMethodID: currentBillPayable.PaymentMethodID,
		Amount:          currentBillPayable.Amount,
		DueDate:         currentBillPayable.DueDate,
		Status:          currentBillPayable.Status,
		Description:     currentBillPayable.Description,
		Notes:           currentBillPayable.Notes,
	}

	domain.ApplyUpdateBillPayableParams(req, &arg)

	return s.repo.UpdateBillPayable(ctx, arg)
}

func (s *Service) ScheduleBill(ctx context.Context, req domain.ScheduleBillRequest) error {
	current, err := s.repo.GetBillsById(ctx, db.GetBillsByIdParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		ID:        pgconv.ParseUUIDToPgType(req.ID),
	})
	if err != nil {
		return err
	}

	statusStr := current.Status.(string)

	if statusStr == "paid" {
		return errors.New("It is not allowed to schedule an account with the following status:" + statusStr)
	}

	return s.repo.ScheduleBill(ctx, db.ScheduleBillParams{
		ID:            pgconv.ParseUUIDToPgType(req.ID),
		CompanyID:     pgconv.ParseUUIDToPgType(req.CompanyID),
		ScheduledDate: pgconv.StringToPgDate(req.ScheduledDate),
	})
}

func (s *Service) GetBillsPayableSummary(ctx context.Context, companyId uuid.UUID) (domain.GetBillsPayableSummaryResponse, error) {
	billsSummary, err := s.repo.GetBillsPayableSummary(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.GetBillsPayableSummaryResponse{}, err
	}

	totalOverdue := pgconv.PgNumericToFloat64(billsSummary.TotalOverdue)

	var status string

	if totalOverdue <= 0 {
		status = "ok"
	} else if totalOverdue > 0 && totalOverdue < float64(billsSummary.TotalQuantity) {
		status = "pending"
	} else {
		status = "all pending"
	}

	return domain.GetBillsPayableSummaryResponse{
		TotalQuantity:  billsSummary.TotalQuantity,
		TotalToPay:     pgconv.PgNumericToFloat64(billsSummary.TotalToPay),
		TotalOverdue:   pgconv.PgNumericToFloat64(billsSummary.TotalOverdue),
		TotalScheduled: pgconv.PgNumericToFloat64(billsSummary.TotalScheduled),
		GeneralStatus:  status,
	}, nil
}

func (s *Service) UpdateOverdueBillsPayable(ctx context.Context) error {
	return s.repo.UpdateOverdueBillsPayable(ctx)
}
