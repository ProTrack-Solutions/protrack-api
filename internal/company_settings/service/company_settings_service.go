package service

import (
	"context"
	"encoding/json"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
)

type Service struct {
	repo domain.RepositoryInterface
}

func NewService(repo domain.RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetCompanySetting(ctx context.Context, companyId uuid.UUID, key enums.CompanySettingsKey) (domain.CompanySettingResponse, error) {
	setting, err := s.repo.GetCompanySetting(ctx, db.GetCompanySettingParams{
		CompanyID: pgconv.OptionalUUIDToPgType(companyId),
		Key:       string(key),
	})
	if err != nil {
		return domain.CompanySettingResponse{}, err
	}

	var value any

	if err = json.Unmarshal(setting.Value, &value); err != nil {
		return domain.CompanySettingResponse{}, err
	}

	return domain.CompanySettingResponse{
		ID:        pgconv.PgUUIDToUUID(setting.ID),
		CompanyID: pgconv.PgUUIDToUUID(setting.CompanyID),
		Key:       enums.CompanySettingsKey(key),
		Value:     value,
		CreatedAt: pgconv.PgTimestamptzToTime(setting.CreatedAt),
		UpdatedAt: pgconv.PgTimestamptzToTime(setting.UpdatedAt),
	}, nil
}

func (s *Service) ListCompanySettings(ctx context.Context, companyId uuid.UUID) ([]domain.CompanySettingResponse, error) {
	settings, err := s.repo.ListCompanySettings(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return []domain.CompanySettingResponse{}, err
	}

	var response []domain.CompanySettingResponse

	for _, setting := range settings {
		var value any
		if err = json.Unmarshal(setting.Value, &value); err != nil {
			return []domain.CompanySettingResponse{}, err
		}
		response = append(response, domain.CompanySettingResponse{
			ID:        pgconv.PgUUIDToUUID(setting.ID),
			CompanyID: pgconv.PgUUIDToUUID(setting.CompanyID),
			Key:       enums.CompanySettingsKey(setting.Key),
			Value:     value,
			CreatedAt: pgconv.PgTimestamptzToTime(setting.CreatedAt),
			UpdatedAt: pgconv.PgTimestamptzToTime(setting.UpdatedAt),
		})
	}

	return response, nil
}

func (s *Service) UpsertCompanySetting(ctx context.Context, componyId uuid.UUID, req domain.UpsertCompanySettingRequest) (domain.CompanySettingResponse, error) {
	value, err := json.Marshal(req.Value)
	if err != nil {
		return domain.CompanySettingResponse{}, err
	}

	setting, err := s.repo.UpsertCompanySetting(ctx, db.UpsertCompanySettingParams{
		CompanyID: pgconv.OptionalUUIDToPgType(componyId),
		Key:       string(req.Key),
		Value:     value,
	})
	if err != nil {
		return domain.CompanySettingResponse{}, err
	}

	return domain.CompanySettingResponse{
		ID:        pgconv.PgUUIDToUUID(setting.ID),
		CreatedAt: pgconv.PgTimestamptzToTime(setting.CreatedAt),
		UpdatedAt: pgconv.PgTimestamptzToTime(setting.UpdatedAt),
	}, nil
}

func (s *Service) DeleteCompanySetting(ctx context.Context, companyId uuid.UUID, key string) error {
	return s.repo.DeleteCompanySetting(ctx, db.DeleteCompanySettingParams{
		CompanyID: pgconv.OptionalUUIDToPgType(companyId),
		Key:       key,
	})
}

func (s *Service) DeleteAllCompanySettings(ctx context.Context, companyId uuid.UUID) error {
	return s.repo.DeleteAllCompanySettings(ctx, pgconv.OptionalUUIDToPgType(companyId))
}

func (s *Service) RestorDefaultSettings(ctx context.Context, companyId uuid.UUID) error {
	for k, v := range domain.DefaultSettings {

		value, err := json.Marshal(v)
		if err != nil {
			return err
		}

		_, err = s.repo.UpsertCompanySetting(ctx, db.UpsertCompanySettingParams{
			CompanyID: pgconv.OptionalUUIDToPgType(companyId),
			Key:       string(k),
			Value:     value,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) SetDefaultSettings(ctx context.Context, tx db.DBTX, companyId uuid.UUID) error {
	repoTx := db.New(tx)

	for k, v := range domain.DefaultSettings {

		value, err := json.Marshal(v)
		if err != nil {
			return err
		}

		_, err = repoTx.UpsertCompanySetting(ctx, db.UpsertCompanySettingParams{
			CompanyID: pgconv.OptionalUUIDToPgType(companyId),
			Key:       string(k),
			Value:     value,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
