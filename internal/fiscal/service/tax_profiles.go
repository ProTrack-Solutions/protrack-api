package service

import (
	"context"
	"errors"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func toTaxProfile(p db.TaxProfile) domain.TaxProfile {
	return domain.TaxProfile{
		ID: pgconv.PgUUIDToUUID(p.ID),
		TaxProfileInput: domain.TaxProfileInput{
			Name:              p.Name,
			IcmsCSOSN:         pgconv.ParsePgTextToString(p.IcmsCsosn),
			IcmsCST:           pgconv.ParsePgTextToString(p.IcmsCst),
			IcmsRate:          numericToRate(p.IcmsRate),
			IcmsBaseReduction: numericToRate(p.IcmsBaseReduction),
			IcmsFcpRate:       numericToRate(p.IcmsFcpRate),
			CFOP:              p.Cfop,
			BenefitCode:       pgconv.ParsePgTextToString(p.BenefitCode),
			PisCST:            p.PisCst,
			PisRate:           numericToRate(p.PisRate),
			CofinsCST:         p.CofinsCst,
			CofinsRate:        numericToRate(p.CofinsRate),
			IbsCbsCST:         p.IbsCbsCst,
			IbsCbsClassCode:   p.IbsCbsClassCode,
			IbsStateRate:      numericToRate(p.IbsStateRate),
			IbsCityRate:       numericToRate(p.IbsCityRate),
			CbsRate:           numericToRate(p.CbsRate),
		},
		UpdatedAt: pgconv.PgTimestamptzToTime(p.UpdatedAt),
	}
}

// ToTaxProfile é usado pela emissão para resolver o perfil de cada item.
func ToTaxProfile(p db.TaxProfile) domain.TaxProfile { return toTaxProfile(p) }

func (s *Service) taxRegime(ctx context.Context, companyId uuid.UUID) (string, *uuid.UUID, error) {
	row, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return "", nil, err
	}
	return pgconv.ParsePgTextToString(row.TaxRegime), pgconv.PgUUIDToUUIDPtr(row.DefaultTaxProfileID), nil
}

func (s *Service) ListTaxProfiles(ctx context.Context, companyId uuid.UUID) ([]domain.TaxProfile, error) {
	rows, err := s.repo.ListTaxProfiles(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return nil, err
	}
	_, defaultId, err := s.taxRegime(ctx, companyId)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaxProfile, 0, len(rows))
	for _, r := range rows {
		p := toTaxProfile(db.TaxProfile{
			ID: r.ID, CompanyID: r.CompanyID, Name: r.Name, IcmsCsosn: r.IcmsCsosn, IcmsCst: r.IcmsCst,
			IcmsRate: r.IcmsRate, IcmsBaseReduction: r.IcmsBaseReduction, IcmsFcpRate: r.IcmsFcpRate,
			Cfop: r.Cfop, BenefitCode: r.BenefitCode, PisCst: r.PisCst, PisRate: r.PisRate,
			CofinsCst: r.CofinsCst, CofinsRate: r.CofinsRate, IbsCbsCst: r.IbsCbsCst,
			IbsCbsClassCode: r.IbsCbsClassCode, IbsStateRate: r.IbsStateRate, IbsCityRate: r.IbsCityRate,
			CbsRate: r.CbsRate, UpdatedAt: r.UpdatedAt,
		})
		p.ProductsCount = r.ProductsCount
		p.CategoriesCount = r.CategoriesCount
		p.IsDefault = defaultId != nil && *defaultId == p.ID
		out = append(out, p)
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Service) checkName(ctx context.Context, repo *repository.Repository, companyId uuid.UUID, name string, except *uuid.UUID) error {
	exists, err := repo.TaxProfileNameExists(ctx, db.TaxProfileNameExistsParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Name:      name,
		ExceptID:  pgconv.OptionalPtrUUIDToPgType(except),
	})
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrTaxProfileNameExists
	}
	return nil
}

// CreateTaxProfile cria o perfil; o primeiro da empresa vira o padrão.
func (s *Service) CreateTaxProfile(ctx context.Context, companyId, userId uuid.UUID, in domain.TaxProfileInput) (domain.TaxProfile, error) {
	regime, _, err := s.taxRegime(ctx, companyId)
	if err != nil {
		return domain.TaxProfile{}, err
	}
	in, err = in.Normalize(regime)
	if err != nil {
		return domain.TaxProfile{}, err
	}
	var created db.TaxProfile
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		if err := s.checkName(ctx, repo, companyId, in.Name, nil); err != nil {
			return err
		}
		created, err = repo.CreateTaxProfile(ctx, db.CreateTaxProfileParams{
			CompanyID:         pgconv.ParseUUIDToPgType(companyId),
			Name:              in.Name,
			IcmsCsosn:         optText(in.IcmsCSOSN),
			IcmsCst:           optText(in.IcmsCST),
			IcmsRate:          rateToNumeric(in.IcmsRate),
			IcmsBaseReduction: rateToNumeric(in.IcmsBaseReduction),
			IcmsFcpRate:       rateToNumeric(in.IcmsFcpRate),
			Cfop:              in.CFOP,
			BenefitCode:       optText(in.BenefitCode),
			PisCst:            in.PisCST,
			PisRate:           rateToNumeric(in.PisRate),
			CofinsCst:         in.CofinsCST,
			CofinsRate:        rateToNumeric(in.CofinsRate),
			IbsCbsCst:         in.IbsCbsCST,
			IbsCbsClassCode:   in.IbsCbsClassCode,
			IbsStateRate:      rateToNumeric(in.IbsStateRate),
			IbsCityRate:       rateToNumeric(in.IbsCityRate),
			CbsRate:           rateToNumeric(in.CbsRate),
			CreatedBy:         pgconv.ParseUUIDToPgType(userId),
		})
		if isUniqueViolation(err) {
			return domain.ErrTaxProfileNameExists
		}
		if err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		if !row.DefaultTaxProfileID.Valid {
			row.DefaultTaxProfileID = created.ID
			_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		}
		return err
	})
	if err != nil {
		return domain.TaxProfile{}, err
	}
	out := toTaxProfile(created)
	_, defaultId, _ := s.taxRegime(ctx, companyId)
	out.IsDefault = defaultId != nil && *defaultId == out.ID
	return out, nil
}

func (s *Service) UpdateTaxProfile(ctx context.Context, companyId, userId, id uuid.UUID, in domain.TaxProfileInput) (domain.TaxProfile, error) {
	regime, defaultId, err := s.taxRegime(ctx, companyId)
	if err != nil {
		return domain.TaxProfile{}, err
	}
	in, err = in.Normalize(regime)
	if err != nil {
		return domain.TaxProfile{}, err
	}
	var updated db.TaxProfile
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		if _, err := repo.GetTaxProfile(ctx, db.GetTaxProfileParams{
			ID: pgconv.ParseUUIDToPgType(id), CompanyID: pgconv.ParseUUIDToPgType(companyId),
		}); errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaxProfileNotFound
		} else if err != nil {
			return err
		}
		if err := s.checkName(ctx, repo, companyId, in.Name, &id); err != nil {
			return err
		}
		updated, err = repo.UpdateTaxProfile(ctx, db.UpdateTaxProfileParams{
			ID:                pgconv.ParseUUIDToPgType(id),
			CompanyID:         pgconv.ParseUUIDToPgType(companyId),
			Name:              in.Name,
			IcmsCsosn:         optText(in.IcmsCSOSN),
			IcmsCst:           optText(in.IcmsCST),
			IcmsRate:          rateToNumeric(in.IcmsRate),
			IcmsBaseReduction: rateToNumeric(in.IcmsBaseReduction),
			IcmsFcpRate:       rateToNumeric(in.IcmsFcpRate),
			Cfop:              in.CFOP,
			BenefitCode:       optText(in.BenefitCode),
			PisCst:            in.PisCST,
			PisRate:           rateToNumeric(in.PisRate),
			CofinsCst:         in.CofinsCST,
			CofinsRate:        rateToNumeric(in.CofinsRate),
			IbsCbsCst:         in.IbsCbsCST,
			IbsCbsClassCode:   in.IbsCbsClassCode,
			IbsStateRate:      rateToNumeric(in.IbsStateRate),
			IbsCityRate:       rateToNumeric(in.IbsCityRate),
			CbsRate:           rateToNumeric(in.CbsRate),
			UpdatedBy:         pgconv.ParseUUIDToPgType(userId),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaxProfileNotFound
		}
		if isUniqueViolation(err) {
			return domain.ErrTaxProfileNameExists
		}
		return err
	})
	if err != nil {
		return domain.TaxProfile{}, err
	}
	out := toTaxProfile(updated)
	out.IsDefault = defaultId != nil && *defaultId == out.ID
	return out, nil
}

// DeleteTaxProfile exclui o perfil; categorias e produtos que o usavam voltam a herdar.
func (s *Service) DeleteTaxProfile(ctx context.Context, companyId, userId, id uuid.UUID) error {
	return s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		n, err := repo.DeleteTaxProfile(ctx, db.DeleteTaxProfileParams{
			ID: pgconv.ParseUUIDToPgType(id), CompanyID: pgconv.ParseUUIDToPgType(companyId),
			UpdatedBy: pgconv.ParseUUIDToPgType(userId),
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return domain.ErrTaxProfileNotFound
		}
		return repo.DetachTaxProfile(ctx, db.DetachTaxProfileParams{
			ID: pgconv.ParseUUIDToPgType(id), CompanyID: pgconv.ParseUUIDToPgType(companyId),
		})
	})
}

// ensureProfile confere que o perfil (se informado) é da empresa.
func (s *Service) ensureProfile(ctx context.Context, companyId uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	_, err := s.repo.GetTaxProfile(ctx, db.GetTaxProfileParams{
		ID: pgconv.ParseUUIDToPgType(*id), CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrTaxProfileNotFound
	}
	return err
}

// ---------- Categorias ----------

func (s *Service) ListCategories(ctx context.Context, companyId uuid.UUID) ([]domain.CategoryFiscal, error) {
	rows, err := s.repo.ListCategoriesFiscal(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return nil, err
	}
	out := make([]domain.CategoryFiscal, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.CategoryFiscal{
			ID:           pgconv.PgUUIDToUUID(r.ID),
			Name:         r.Name,
			Color:        pgconv.ParsePgTextToStringPtr(r.Color),
			TaxProfileID: pgconv.PgUUIDToUUIDPtr(r.TaxProfileID),
		})
	}
	return out, nil
}

func (s *Service) SetCategoryTaxProfile(ctx context.Context, companyId, categoryId uuid.UUID, profileId *uuid.UUID) error {
	if err := s.ensureProfile(ctx, companyId, profileId); err != nil {
		return err
	}
	n, err := s.repo.SetCategoryTaxProfile(ctx, db.SetCategoryTaxProfileParams{
		TaxProfileID: pgconv.OptionalPtrUUIDToPgType(profileId),
		ID:           pgconv.ParseUUIDToPgType(categoryId),
		CompanyID:    pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// ---------- Produtos ----------

func (s *Service) GetProductFiscal(ctx context.Context, companyId, productId uuid.UUID) (domain.ProductFiscal, error) {
	r, err := s.repo.GetProductFiscal(ctx, db.GetProductFiscalParams{
		ID: pgconv.ParseUUIDToPgType(productId), CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProductFiscal{}, domain.ErrProductNotFound
	}
	if err != nil {
		return domain.ProductFiscal{}, err
	}
	return domain.ProductFiscal{
		ProductID:            pgconv.PgUUIDToUUID(r.ID),
		Name:                 r.Name,
		ModelName:            pgconv.ParsePgTextToStringPtr(r.ModelName),
		NCM:                  pgconv.ParsePgTextToStringPtr(r.Ncm),
		CEST:                 pgconv.ParsePgTextToStringPtr(r.Cest),
		Unit:                 string(r.Unit),
		Origin:               r.OrigemMercadoria,
		TaxProfileID:         pgconv.PgUUIDToUUIDPtr(r.TaxProfileID),
		CategoryTaxProfileID: pgconv.PgUUIDToUUIDPtr(r.CategoryTaxProfileID),
	}, nil
}

// UpdateProductFiscal grava NCM, CEST, unidade, origem e perfil de um produto ou,
// com modelName, de todas as variações do modelo de grade. Devolve quantos mudaram.
func (s *Service) UpdateProductFiscal(ctx context.Context, companyId, userId uuid.UUID, productId *uuid.UUID, modelName string, in domain.ProductFiscalInput) (int64, error) {
	in, err := in.Normalize()
	if err != nil {
		return 0, err
	}
	if err := s.ensureProfile(ctx, companyId, in.TaxProfileID); err != nil {
		return 0, err
	}
	params := db.UpdateProductFiscalParams{
		Ncm:          optText(in.NCM),
		Cest:         optText(in.CEST),
		Origin:       in.Origin,
		TaxProfileID: pgconv.OptionalPtrUUIDToPgType(in.TaxProfileID),
		UpdatedBy:    pgconv.ParseUUIDToPgType(userId),
		CompanyID:    pgconv.ParseUUIDToPgType(companyId),
		ProductID:    pgconv.OptionalPtrUUIDToPgType(productId),
		ModelName:    optText(strings.TrimSpace(modelName)),
	}
	if in.Unit != "" {
		params.Unit = db.NullUnitOfMeasure{UnitOfMeasure: db.UnitOfMeasure(in.Unit), Valid: true}
	}
	if !params.ProductID.Valid && !params.ModelName.Valid {
		return 0, domain.ErrProductNotFound
	}
	n, err := s.repo.UpdateProductFiscal(ctx, params)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, domain.ErrProductNotFound
	}
	return n, nil
}
