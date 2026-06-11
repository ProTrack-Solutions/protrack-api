package service

import (
	"context"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/customers/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/customers/repository"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/domain/enums"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateCustomer(ctx context.Context, arg db.CreateCustomersParams) (pgtype.UUID, error)
	DeleteCustomer(ctx context.Context, arg db.DeleteCustomerParams) error
	GetCustomerByCPF(ctx context.Context, cpf string) (db.Customer, error)
	GetCustomerById(ctx context.Context, id pgtype.UUID) (db.Customer, error)
	ListCustomers(ctx context.Context, companyID pgtype.UUID) ([]db.Customer, error)
	UpdateBalanceDueCustomer(ctx context.Context, arg db.UpdateBalanceDueCustomerParams) error
	UpdateCustomer(ctx context.Context, arg db.UpdateCustomerParams) error
	CountCustomers(ctx context.Context, companyId pgtype.UUID) (int64, error)
	GetCustomersPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetCustomersPerformanceSummaryRow, error)
	UpdateCustomerBalance(ctx context.Context, arg db.UpdateCustomerBalanceParams) error
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

func (s *Service) CreateCustomer(ctx context.Context, req domain.CreateCustomersRequest) (uuid.UUID, error) {
	id, err := s.repo.CreateCustomer(ctx, db.CreateCustomersParams{
		CompanyID:           pgconv.ParseUUIDToPgType(req.CompanyID),
		FullName:            req.FullName,
		BirthDate:           pgconv.StringToPgDate(req.BirthDate),
		Cpf:                 req.Cpf,
		Rg:                  pgconv.ParseStringToPgText(req.Rg),
		MaritalStatus:       pgconv.ParseStringToPgText(req.MaritalStatus),
		Gender:              req.Gender,
		Whatsapp:            pgconv.ParseStringToPgText(req.Whatsapp),
		MobilePhone:         pgconv.ParseStringToPgText(req.MobilePhone),
		HomePhone:           pgconv.ParseStringToPgText(req.HomePhone),
		Email:               req.Email,
		AddressStreet:       pgconv.ParseStringToPgText(req.AddressStreet),
		AddressNumber:       pgconv.ParseStringToPgText(req.AddressNumber),
		AddressComplement:   pgconv.ParseStringToPgText(req.AddressComplement),
		AddressNeighborhood: pgconv.ParseStringToPgText(req.AddressNeighborhood),
		AddressCity:         pgconv.ParseStringToPgText(req.AddressCity),
		AddressState:        pgconv.ParseStringToPgText(req.AddressState),
		AddressZipcode:      pgconv.ParseStringToPgText(req.AddressZipcode),
		AddressCountry:      pgconv.ParseStringToPgText(req.AddressCountry),
		BalanceDue:          pgconv.Float64ToPgNumeric(req.BalanceDue),
		CreatedBy:           pgconv.ParseUUIDToPgType(req.CreatedBy),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return pgconv.PgUUIDToUUID(id), nil
}

func (s *Service) DeleteCustomer(ctx context.Context, req domain.DeleteCustomerRequest) error {
	return s.repo.DeleteCustomer(ctx, db.DeleteCustomerParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		DeletedBy: pgconv.ParseUUIDToPgType(req.DeletedBy),
	})
}

func (s *Service) GetCustomerByCPF(ctx context.Context, cpf string) (domain.CustomerResponse, error) {
	customer, err := s.repo.GetCustomerByCPF(ctx, cpf)
	if err != nil {
		return domain.CustomerResponse{}, err
	}

	return domain.CustomerResponse{
		ID:                  pgconv.PgUUIDToUUID(customer.ID),
		CompanyID:           pgconv.PgUUIDToUUID(customer.CompanyID),
		FullName:            customer.FullName,
		BirthDate:           pgconv.PgDateToString(customer.BirthDate),
		Cpf:                 customer.Cpf,
		Rg:                  pgconv.ParsePgTextToString(customer.Rg),
		MaritalStatus:       pgconv.ParsePgTextToString(customer.MaritalStatus),
		Gender:              enums.Gender(customer.Gender.(string)),
		Whatsapp:            pgconv.ParsePgTextToString(customer.Whatsapp),
		MobilePhone:         pgconv.ParsePgTextToString(customer.MobilePhone),
		HomePhone:           pgconv.ParsePgTextToString(customer.HomePhone),
		Email:               customer.Email,
		AddressStreet:       pgconv.ParsePgTextToString(customer.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(customer.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(customer.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(customer.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(customer.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(customer.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(customer.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(customer.AddressCountry),
		BalanceDue:          pgconv.PgNumericToFloat64(customer.BalanceDue),
		CreatedBy:           pgconv.PgUUIDToUUID(customer.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(customer.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(customer.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(customer.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(customer.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(customer.DeletedAt),
	}, nil
}

func (s *Service) GetCustomerById(ctx context.Context, id uuid.UUID) (domain.CustomerResponse, error) {
	customer, err := s.repo.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.CustomerResponse{}, err
	}

	return domain.CustomerResponse{
		ID:                  pgconv.PgUUIDToUUID(customer.ID),
		CompanyID:           pgconv.PgUUIDToUUID(customer.CompanyID),
		FullName:            customer.FullName,
		BirthDate:           pgconv.PgDateToString(customer.BirthDate),
		Cpf:                 customer.Cpf,
		Rg:                  pgconv.ParsePgTextToString(customer.Rg),
		MaritalStatus:       pgconv.ParsePgTextToString(customer.MaritalStatus),
		Gender:              enums.Gender(customer.Gender.(string)),
		Whatsapp:            pgconv.ParsePgTextToString(customer.Whatsapp),
		MobilePhone:         pgconv.ParsePgTextToString(customer.MobilePhone),
		HomePhone:           pgconv.ParsePgTextToString(customer.HomePhone),
		Email:               customer.Email,
		AddressStreet:       pgconv.ParsePgTextToString(customer.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(customer.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(customer.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(customer.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(customer.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(customer.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(customer.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(customer.AddressCountry),
		BalanceDue:          pgconv.PgNumericToFloat64(customer.BalanceDue),
		CreatedBy:           pgconv.PgUUIDToUUID(customer.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(customer.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(customer.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(customer.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(customer.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(customer.DeletedAt),
	}, nil
}

func (s *Service) GetCustomerByIdTx(ctx context.Context, tx db.DBTX, id uuid.UUID) (domain.CustomerResponse, error) {
	repoTx := s.repo.WithTx(tx)

	customer, err := repoTx.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.CustomerResponse{}, err
	}

	return domain.CustomerResponse{
		ID:                  pgconv.PgUUIDToUUID(customer.ID),
		CompanyID:           pgconv.PgUUIDToUUID(customer.CompanyID),
		FullName:            customer.FullName,
		BirthDate:           pgconv.PgDateToString(customer.BirthDate),
		Cpf:                 customer.Cpf,
		Rg:                  pgconv.ParsePgTextToString(customer.Rg),
		MaritalStatus:       pgconv.ParsePgTextToString(customer.MaritalStatus),
		Gender:              enums.Gender(customer.Gender.(string)),
		Whatsapp:            pgconv.ParsePgTextToString(customer.Whatsapp),
		MobilePhone:         pgconv.ParsePgTextToString(customer.MobilePhone),
		HomePhone:           pgconv.ParsePgTextToString(customer.HomePhone),
		Email:               customer.Email,
		AddressStreet:       pgconv.ParsePgTextToString(customer.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(customer.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(customer.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(customer.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(customer.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(customer.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(customer.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(customer.AddressCountry),
		BalanceDue:          pgconv.PgNumericToFloat64(customer.BalanceDue),
		CreatedBy:           pgconv.PgUUIDToUUID(customer.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(customer.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(customer.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(customer.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(customer.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(customer.DeletedAt),
	}, nil
}

func (s *Service) ListCustomers(ctx context.Context, companyId uuid.UUID) ([]domain.CustomerResponse, error) {
	customers, err := s.repo.ListCustomers(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.CustomerResponse{}, err
	}

	var response []domain.CustomerResponse

	for _, customer := range customers {
		response = append(response, domain.CustomerResponse{
			ID:                  pgconv.PgUUIDToUUID(customer.ID),
			CompanyID:           pgconv.PgUUIDToUUID(customer.CompanyID),
			FullName:            customer.FullName,
			BirthDate:           pgconv.PgDateToString(customer.BirthDate),
			Cpf:                 customer.Cpf,
			Rg:                  pgconv.ParsePgTextToString(customer.Rg),
			MaritalStatus:       pgconv.ParsePgTextToString(customer.MaritalStatus),
			Gender:              enums.Gender(customer.Gender.(string)),
			Whatsapp:            pgconv.ParsePgTextToString(customer.Whatsapp),
			MobilePhone:         pgconv.ParsePgTextToString(customer.MobilePhone),
			HomePhone:           pgconv.ParsePgTextToString(customer.HomePhone),
			Email:               customer.Email,
			AddressStreet:       pgconv.ParsePgTextToString(customer.AddressStreet),
			AddressNumber:       pgconv.ParsePgTextToString(customer.AddressNumber),
			AddressComplement:   pgconv.ParsePgTextToString(customer.AddressComplement),
			AddressNeighborhood: pgconv.ParsePgTextToString(customer.AddressNeighborhood),
			AddressCity:         pgconv.ParsePgTextToString(customer.AddressCity),
			AddressState:        pgconv.ParsePgTextToString(customer.AddressState),
			AddressZipcode:      pgconv.ParsePgTextToString(customer.AddressZipcode),
			AddressCountry:      pgconv.ParsePgTextToString(customer.AddressCountry),
			BalanceDue:          pgconv.PgNumericToFloat64(customer.BalanceDue),
			CreatedBy:           pgconv.PgUUIDToUUID(customer.CreatedBy),
			UpdatedBy:           pgconv.PgUUIDToUUID(customer.UpdatedBy),
			DeletedBy:           pgconv.PgUUIDToUUID(customer.DeletedBy),
			CreatedAt:           pgconv.PgTimestamptzToTime(customer.CreatedAt),
			UpdatedAt:           pgconv.PgTimestamptzToTime(customer.UpdatedAt),
			DeletedAt:           pgconv.PgTimestamptzToTime(customer.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) UpdateBalanceDueCustomer(ctx context.Context, id uuid.UUID, req domain.UpdateBalanceDueCustomerRequest) error {
	customer, err := s.repo.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return nil
	}

	arg := db.UpdateBalanceDueCustomerParams{
		ID:         pgconv.ParseUUIDToPgType(id),
		BalanceDue: customer.BalanceDue,
		UpdatedBy:  customer.UpdatedBy,
	}

	if req.BalanceDue != 0 {
		arg.BalanceDue = pgconv.Float64ToPgNumeric(req.BalanceDue)
	}

	if req.UpdatedBy != uuid.Nil {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}

	if err := s.repo.UpdateBalanceDueCustomer(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateCustomerBalanceSubTx(ctx context.Context, tx db.DBTX, id uuid.UUID, req domain.UpdateBalanceDueCustomerRequest) error {
	repoTx := db.New(tx)

	customer, err := repoTx.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return err
	}

	newBalance := pgconv.PgNumericToFloat64(customer.BalanceDue) - (req.BalanceDue - req.Prohibited)

	arg := db.UpdateCustomerBalanceParams{
		ID:         pgconv.ParseUUIDToPgType(id),
		BalanceDue: pgconv.Float64ToPgNumeric(newBalance),
		UpdatedBy:  customer.UpdatedBy,
	}

	if req.UpdatedBy != uuid.Nil {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}

	if err := repoTx.UpdateCustomerBalance(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateCustomerBalanceAddTx(ctx context.Context, tx db.DBTX, id uuid.UUID, req domain.UpdateBalanceDueCustomerRequest) error {
	repoTx := db.New(tx)

	customer, err := repoTx.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return err
	}

	newBalance := pgconv.PgNumericToFloat64(customer.BalanceDue) + (req.BalanceDue - req.Prohibited)

	arg := db.UpdateCustomerBalanceParams{
		ID:         pgconv.ParseUUIDToPgType(id),
		BalanceDue: pgconv.Float64ToPgNumeric(newBalance),
		UpdatedBy:  customer.UpdatedBy,
	}

	if req.UpdatedBy != uuid.Nil {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}

	if err := repoTx.UpdateCustomerBalance(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateCustomer(ctx context.Context, id uuid.UUID, req domain.UpdateCustomerRequest) error {
	currentCustomer, err := s.repo.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return nil
	}

	arg := db.UpdateCustomerParams{
		ID:                  currentCustomer.ID,
		FullName:            currentCustomer.FullName,
		BirthDate:           currentCustomer.BirthDate,
		Cpf:                 currentCustomer.Cpf,
		Rg:                  currentCustomer.Rg,
		MaritalStatus:       currentCustomer.MaritalStatus,
		Gender:              currentCustomer.Gender,
		Whatsapp:            currentCustomer.Whatsapp,
		MobilePhone:         currentCustomer.MobilePhone,
		HomePhone:           currentCustomer.HomePhone,
		Email:               currentCustomer.Email,
		AddressStreet:       currentCustomer.AddressStreet,
		AddressNumber:       currentCustomer.AddressNumber,
		AddressComplement:   currentCustomer.AddressComplement,
		AddressNeighborhood: currentCustomer.AddressNeighborhood,
		AddressCity:         currentCustomer.AddressCity,
		AddressState:        currentCustomer.AddressState,
		AddressZipcode:      currentCustomer.AddressZipcode,
		AddressCountry:      currentCustomer.AddressCountry,
		BalanceDue:          currentCustomer.BalanceDue,
		UpdatedBy:           currentCustomer.UpdatedBy,
	}

	domain.ApplyUpdateCustomerParams(req, &arg)

	if err := s.repo.UpdateCustomer(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) CountCustomers(ctx context.Context, companyId uuid.UUID) (int64, error) {
	count, err := s.repo.CountCustomers(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) GetCustomersPerformanceSummary(ctx context.Context, companyId uuid.UUID) (float64, error) {
	res, err := s.repo.GetCustomersPerformanceSummary(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	var percentage float64

	if res.LastMonthCount > 0 {
		percentage = ((float64(res.CurrentMonthCount) - float64(res.LastMonthCount)) / float64(res.LastMonthCount)) * 100
	} else {
		if res.CurrentMonthCount > 0 {
			percentage = 100.0
		} else {
			percentage = 0
		}
	}

	return percentage, nil
}
