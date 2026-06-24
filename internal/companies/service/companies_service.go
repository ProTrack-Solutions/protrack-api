package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/validate"

	"github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	usersRepo "github.com/ProTrack-Solutions/protrack-api/internal/users/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateCompany(ctx context.Context, arg db.CreateCompanyParams) (db.Company, error)
	DeleteCompany(ctx context.Context, arg db.DeleteCompanyParams) error
	GetCompanyByDocument(ctx context.Context, document pgtype.Text) (db.Company, error)
	GetCompanyByID(ctx context.Context, id pgtype.UUID) (db.Company, error)
	ListCompanies(ctx context.Context) ([]db.Company, error)
	SetCompanyStatus(ctx context.Context, arg db.SetCompanyStatusParams) (int64, error)
	UpdateCompany(ctx context.Context, arg db.UpdateCompanyParams) (db.Company, error)
}

type Service struct {
	repo      RepositoryInterface
	usersRepo *usersRepo.Repository
	pool      *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool, repo *repository.Repository, usersRepo *usersRepo.Repository) *Service {
	return &Service{
		repo:      repo,
		usersRepo: usersRepo,
		pool:      pool,
	}
}

func (s *Service) CreateCompany(
	ctx context.Context,
	userId uuid.UUID,
	req domain.CreateCompanyParams,
) (domain.CompanyResponse, error) {
	typeDocument, err := validate.ValidateDocument(req.Document)
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CompanyResponse{}, err
	}
	defer tx.Rollback(ctx)

	companyRepo := repository.NewRepository(tx)
	userRepo := usersRepo.NewRepository(tx)

	company, err := companyRepo.CreateCompany(ctx, db.CreateCompanyParams{
		Name:         req.Name,
		TradeName:    pgconv.ParseStringToPgText(req.TradeName),
		Document:     pgconv.ParseStringToPgText(req.Document),
		DocumentType: pgconv.ParseStringToPgText(typeDocument),
		Email:        pgconv.ParseStringToPgText(req.Email),
		Phone:        pgconv.ParseStringToPgText(req.Phone),
		Website:      pgconv.ParseStringToPgText(req.Website),
		CreatedBy:    pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	err = userRepo.UpdateUserCompanyAndRole(ctx, db.UpdateUserCompanyAndRoleParams{
		ID:        pgconv.ParseUUIDToPgType(userId),
		CompanyID: company.ID,
		Role:      "ADMIN",
	})
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CompanyResponse{}, err
	}

	return domain.CompanyResponse{
		ID:        pgconv.PgUUIDToUUID(company.ID),
		Name:      company.Name,
		Document:  pgconv.ParsePgTextToString(company.Document),
		Status:    company.Status,
		CreatedBy: pgconv.PgUUIDToUUID(company.CreatedBy),
		CreatedAt: pgconv.PgTimestamptzToTime(company.CreatedAt),
	}, nil
}

func (s *Service) DeleteCompany(ctx context.Context, req domain.DeleteCompanyParams) error {
	return s.repo.DeleteCompany(ctx, db.DeleteCompanyParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		DeletedBy: pgconv.ParseUUIDToPgType(req.DeletedBy),
	})
}

func (s *Service) GetCompanyByDocument(ctx context.Context, document string) (domain.CompanyResponse, error) {
	_, err := validate.ValidateDocument(document)
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	company, err := s.repo.GetCompanyByDocument(ctx, pgconv.ParseStringToPgText(document))
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	return domain.CompanyResponse{
		ID:                  pgconv.PgUUIDToUUID(company.ID),
		Name:                company.Name,
		TradeName:           pgconv.ParsePgTextToString(company.TradeName),
		Document:            pgconv.ParsePgTextToString(company.Document),
		DocumentType:        pgconv.ParsePgTextToString(company.DocumentType),
		Email:               pgconv.ParsePgTextToString(company.Email),
		Phone:               pgconv.ParsePgTextToString(company.Phone),
		Website:             pgconv.ParsePgTextToString(company.Website),
		AddressStreet:       pgconv.ParsePgTextToString(company.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(company.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(company.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(company.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(company.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(company.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(company.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(company.AddressCountry),
		Status:              company.Status,
		CreatedBy:           pgconv.PgUUIDToUUID(company.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(company.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(company.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(company.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(company.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(company.DeletedAt),
	}, nil
}

func (s *Service) GetCompanyByID(ctx context.Context, id uuid.UUID) (domain.CompanyResponse, error) {
	company, err := s.repo.GetCompanyByID(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	return domain.CompanyResponse{
		ID:                  pgconv.PgUUIDToUUID(company.ID),
		Name:                company.Name,
		TradeName:           pgconv.ParsePgTextToString(company.TradeName),
		Document:            pgconv.ParsePgTextToString(company.Document),
		DocumentType:        pgconv.ParsePgTextToString(company.DocumentType),
		Email:               pgconv.ParsePgTextToString(company.Email),
		Phone:               pgconv.ParsePgTextToString(company.Phone),
		Website:             pgconv.ParsePgTextToString(company.Website),
		AddressStreet:       pgconv.ParsePgTextToString(company.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(company.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(company.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(company.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(company.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(company.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(company.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(company.AddressCountry),
		Status:              company.Status,
		CreatedBy:           pgconv.PgUUIDToUUID(company.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(company.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(company.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(company.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(company.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(company.DeletedAt),
	}, nil
}

func (s *Service) GetCompanyByIDTx(ctx context.Context, tx db.DBTX,id uuid.UUID) (domain.CompanyResponse, error) {
	repoTx := db.New(tx)
	
	company, err := repoTx.GetCompanyByID(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	return domain.CompanyResponse{
		ID:                  pgconv.PgUUIDToUUID(company.ID),
		Name:                company.Name,
		TradeName:           pgconv.ParsePgTextToString(company.TradeName),
		Document:            pgconv.ParsePgTextToString(company.Document),
		DocumentType:        pgconv.ParsePgTextToString(company.DocumentType),
		Email:               pgconv.ParsePgTextToString(company.Email),
		Phone:               pgconv.ParsePgTextToString(company.Phone),
		Website:             pgconv.ParsePgTextToString(company.Website),
		AddressStreet:       pgconv.ParsePgTextToString(company.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(company.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(company.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(company.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(company.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(company.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(company.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(company.AddressCountry),
		Status:              company.Status,
		CreatedBy:           pgconv.PgUUIDToUUID(company.CreatedBy),
		UpdatedBy:           pgconv.PgUUIDToUUID(company.UpdatedBy),
		DeletedBy:           pgconv.PgUUIDToUUID(company.DeletedBy),
		CreatedAt:           pgconv.PgTimestamptzToTime(company.CreatedAt),
		UpdatedAt:           pgconv.PgTimestamptzToTime(company.UpdatedAt),
		DeletedAt:           pgconv.PgTimestamptzToTime(company.DeletedAt),
	}, nil
}

func (s *Service) ListCompanies(ctx context.Context) ([]domain.CompanyResponse, error) {
	companies, err := s.repo.ListCompanies(ctx)
	if err != nil {
		return []domain.CompanyResponse{}, err
	}

	var response []domain.CompanyResponse

	for _, company := range companies {
		response = append(response, domain.CompanyResponse{
			ID:                  pgconv.PgUUIDToUUID(company.ID),
			Name:                company.Name,
			TradeName:           pgconv.ParsePgTextToString(company.TradeName),
			Document:            pgconv.ParsePgTextToString(company.Document),
			DocumentType:        pgconv.ParsePgTextToString(company.DocumentType),
			Email:               pgconv.ParsePgTextToString(company.Email),
			Phone:               pgconv.ParsePgTextToString(company.Phone),
			Website:             pgconv.ParsePgTextToString(company.Website),
			AddressStreet:       pgconv.ParsePgTextToString(company.AddressStreet),
			AddressNumber:       pgconv.ParsePgTextToString(company.AddressNumber),
			AddressComplement:   pgconv.ParsePgTextToString(company.AddressComplement),
			AddressNeighborhood: pgconv.ParsePgTextToString(company.AddressNeighborhood),
			AddressCity:         pgconv.ParsePgTextToString(company.AddressCity),
			AddressState:        pgconv.ParsePgTextToString(company.AddressState),
			AddressZipcode:      pgconv.ParsePgTextToString(company.AddressZipcode),
			AddressCountry:      pgconv.ParsePgTextToString(company.AddressCountry),
			Status:              company.Status,
			CreatedBy:           pgconv.PgUUIDToUUID(company.CreatedBy),
			UpdatedBy:           pgconv.PgUUIDToUUID(company.UpdatedBy),
			DeletedBy:           pgconv.PgUUIDToUUID(company.DeletedBy),
			CreatedAt:           pgconv.PgTimestamptzToTime(company.CreatedAt),
			UpdatedAt:           pgconv.PgTimestamptzToTime(company.UpdatedAt),
			DeletedAt:           pgconv.PgTimestamptzToTime(company.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) SetCompanyStatus(ctx context.Context, req domain.SetCompanyStatusParams) (int64, error) {
	count, err := s.repo.SetCompanyStatus(ctx, db.SetCompanyStatusParams{
		ID:      pgconv.ParseUUIDToPgType(req.ID),
		Column2: req.Status,
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) UpdateCompany(ctx context.Context, id uuid.UUID, req domain.UpdateCompanyRequest) (domain.CompanyResponse, error) {
	currentCompany, err := s.repo.GetCompanyByID(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	arg := db.UpdateCompanyParams{
		ID:                  pgconv.ParseUUIDToPgType(id),
		Name:                currentCompany.Name,
		TradeName:           currentCompany.TradeName,
		Document:            currentCompany.Document,
		DocumentType:        currentCompany.DocumentType,
		Email:               currentCompany.Email,
		Phone:               currentCompany.Phone,
		Website:             currentCompany.Website,
		AddressStreet:       currentCompany.AddressStreet,
		AddressNumber:       currentCompany.AddressNumber,
		AddressComplement:   currentCompany.AddressComplement,
		AddressNeighborhood: currentCompany.AddressNeighborhood,
		AddressCity:         currentCompany.AddressCity,
		AddressState:        currentCompany.AddressState,
		AddressZipcode:      currentCompany.AddressZipcode,
		AddressCountry:      currentCompany.AddressCountry,
		Timezone:            currentCompany.Timezone,
		UpdatedBy:           currentCompany.UpdatedBy,
	}

	domain.ApplyUpdateCompanyParams(req, &arg)

	company, err := s.repo.UpdateCompany(ctx, arg)
	if err != nil {
		return domain.CompanyResponse{}, err
	}

	return domain.CompanyResponse{
		ID:                  pgconv.PgUUIDToUUID(company.ID),
		Name:                company.Name,
		TradeName:           pgconv.ParsePgTextToString(company.TradeName),
		Document:            pgconv.ParsePgTextToString(company.Document),
		DocumentType:        pgconv.ParsePgTextToString(company.DocumentType),
		Email:               pgconv.ParsePgTextToString(company.Email),
		Phone:               pgconv.ParsePgTextToString(company.Phone),
		Website:             pgconv.ParsePgTextToString(company.Website),
		AddressStreet:       pgconv.ParsePgTextToString(company.AddressStreet),
		AddressNumber:       pgconv.ParsePgTextToString(company.AddressNumber),
		AddressComplement:   pgconv.ParsePgTextToString(company.AddressComplement),
		AddressNeighborhood: pgconv.ParsePgTextToString(company.AddressNeighborhood),
		AddressCity:         pgconv.ParsePgTextToString(company.AddressCity),
		AddressState:        pgconv.ParsePgTextToString(company.AddressState),
		AddressZipcode:      pgconv.ParsePgTextToString(company.AddressZipcode),
		AddressCountry:      pgconv.ParsePgTextToString(company.AddressCountry),
		UpdatedBy:           pgconv.PgUUIDToUUID(company.UpdatedBy),
		UpdatedAt:           pgconv.PgTimestamptzToTime(company.UpdatedAt),
	}, nil
}
