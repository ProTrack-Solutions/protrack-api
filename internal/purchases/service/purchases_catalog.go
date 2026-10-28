package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/repository"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	catalogPerPage = 20
	// Feature do plano com o máximo de produtos cadastrados
	maxProductsFeature = "max_products"
	uniqueViolation    = "23505"
	barcodeAttempts    = 5
)

type Subscriptions interface {
	GetSubscriptionByCompanyID(ctx context.Context, companyID uuid.UUID) (subscriptionsDomain.SubscriptionResponse, error)
}

type Plans interface {
	GetPlanByID(ctx context.Context, planId uuid.UUID) (plansDomain.PlanResponse, error)
}

// SearchProducts busca produtos para o pedido e, junto, os modelos de grade
// que casam com o texto (para o atalho "adicionar grade").
func (s *Service) SearchProducts(ctx context.Context, companyId uuid.UUID, params domain.CatalogSearchParams) (domain.CatalogSearchResponse, error) {
	page := max(params.Page, 1)
	perPage := params.PerPage
	if perPage == 0 {
		perPage = catalogPerPage
	}
	search := strings.TrimSpace(params.Search)
	companyPg := pgconv.ParseUUIDToPgType(companyId)

	rows, err := s.repo.SearchPurchaseProducts(ctx, db.SearchPurchaseProductsParams{
		CompanyID:  companyPg,
		Search:     search,
		PageLimit:  perPage,
		PageOffset: (page - 1) * perPage,
	})
	if err != nil {
		return domain.CatalogSearchResponse{}, err
	}

	resp := domain.CatalogSearchResponse{
		Products: make([]domain.CatalogProduct, 0, len(rows)),
		Models:   []domain.CatalogModel{},
		Page:     page,
		PerPage:  perPage,
	}
	for _, r := range rows {
		resp.TotalRows = r.TotalRows
		resp.Products = append(resp.Products, domain.CatalogProduct{
			ID:         pgconv.PgUUIDToUUID(r.ID),
			Name:       r.Name,
			ModelName:  r.ModelName,
			Color:      r.Color,
			Size:       r.Size,
			Barcode:    r.Barcode,
			Unit:       r.Unit,
			SellInBulk: r.SellInBulk,
			Quantity:   r.Quantity,
			CostPrice:  r.CostPrice,
			SalePrice:  r.SalePrice,
		})
	}
	resp.TotalPages = int32(math.Ceil(float64(resp.TotalRows) / float64(perPage)))

	// Modelos só na primeira página e com algo digitado
	if search != "" && page == 1 {
		models, err := s.repo.SearchPurchaseModels(ctx, db.SearchPurchaseModelsParams{CompanyID: companyPg, Search: search})
		if err != nil {
			return domain.CatalogSearchResponse{}, err
		}
		for _, m := range models {
			resp.Models = append(resp.Models, domain.CatalogModel{ModelName: m.ModelName, Variations: m.Variations})
		}
	}

	return resp, nil
}

// ModelVariations todas as variações cadastradas de um modelo (recompra)
func (s *Service) ModelVariations(ctx context.Context, companyId uuid.UUID, modelName string) ([]domain.CatalogProduct, error) {
	return listVariations(ctx, s.repo, pgconv.ParseUUIDToPgType(companyId), modelName)
}

func (s *Service) ListCategories(ctx context.Context, companyId uuid.UUID) ([]domain.CatalogCategory, error) {
	rows, err := s.repo.ListPurchaseCategories(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return nil, err
	}
	out := make([]domain.CatalogCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.CatalogCategory{ID: pgconv.PgUUIDToUUID(r.ID), Name: r.Name})
	}
	return out, nil
}

// CreateVariations cadastra o modelo com as combinações de cor e tamanho, cada
// uma como um produto sem estoque e com código de barras interno. Tudo ou
// nada: se uma combinação já existe ou o plano não comporta, nada é criado.
func (s *Service) CreateVariations(ctx context.Context, companyId, userId uuid.UUID, req domain.CreateVariationsRequest) ([]domain.CatalogProduct, error) {
	variations, err := domain.NormalizeVariations(req.Variations)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(req.ModelName)
	companyPg := pgconv.ParseUUIDToPgType(companyId)

	if err := s.checkProductLimit(ctx, companyId, len(variations)); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	repo := s.repo.WithTx(tx)

	if _, err := repo.GetPurchaseCategory(ctx, db.GetPurchaseCategoryParams{ID: pgconv.ParseUUIDToPgType(req.CategoryID), CompanyID: companyPg}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}

	// Combinações que já existem viram erro com a lista, em vez de duplicar
	existing, err := listVariations(ctx, repo, companyPg, model)
	if err != nil {
		return nil, err
	}
	existingKeys := map[string]bool{}
	for _, e := range existing {
		existingKeys[domain.VariationKey(e.Color, e.Size)] = true
	}
	var duplicated []string
	for _, v := range variations {
		if existingKeys[domain.VariationKey(v.Color, v.Size)] {
			duplicated = append(duplicated, domain.VariationName(model, v.Color, v.Size))
		}
	}
	if len(duplicated) > 0 {
		return nil, fmt.Errorf("%w: %s. Use \"Adicionar grade\" para pedir as variações que já existem", domain.ErrVariationsExist, strings.Join(duplicated, ", "))
	}

	for _, v := range variations {
		cost, price := req.CostPrice, req.SalePrice
		if v.CostPrice != nil {
			cost = *v.CostPrice
		}
		if v.SalePrice != nil {
			price = *v.SalePrice
		}

		barcode, err := uniqueBarcode(ctx, repo, companyPg)
		if err != nil {
			return nil, err
		}

		if _, err := repo.CreatePurchaseProduct(ctx, db.CreatePurchaseProductParams{
			CompanyID:   companyPg,
			CategoryID:  pgconv.ParseUUIDToPgType(req.CategoryID),
			Name:        domain.VariationName(model, v.Color, v.Size),
			Description: pgconv.ParseStringToPgType(strings.TrimSpace(req.Description)),
			ModelName:   pgconv.ParseStringToPgType(model),
			Color:       pgconv.ParseStringToPgType(v.Color),
			Size:        pgconv.ParseStringToPgType(v.Size),
			Barcode:     pgconv.ParseStringToPgType(barcode),
			CostPrice:   pgconv.Float64ToPgNumeric(domain.Round2(cost)),
			SalePrice:   pgconv.Float64ToPgNumeric(domain.Round2(price)),
			CreatedBy:   pgconv.ParseUUIDToPgType(userId),
		}); err != nil {
			// Cadastro simultâneo da mesma combinação: o índice único segura
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
				return nil, fmt.Errorf("%w: %s", domain.ErrVariationsExist, domain.VariationName(model, v.Color, v.Size))
			}
			return nil, err
		}
	}

	created, err := listVariations(ctx, repo, companyPg, model)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Devolve só as recém-criadas, na ordem pedida
	byKey := map[string]domain.CatalogProduct{}
	for _, c := range created {
		byKey[domain.VariationKey(c.Color, c.Size)] = c
	}
	out := make([]domain.CatalogProduct, 0, len(variations))
	for _, v := range variations {
		out = append(out, byKey[domain.VariationKey(v.Color, v.Size)])
	}
	return out, nil
}

// checkProductLimit respeita o máximo de produtos do plano para o lote inteiro
func (s *Service) checkProductLimit(ctx context.Context, companyId uuid.UUID, adding int) error {
	if s.subscriptions == nil || s.plans == nil {
		return nil
	}
	sub, err := s.subscriptions.GetSubscriptionByCompanyID(ctx, companyId)
	if err != nil {
		return err
	}
	plan, err := s.plans.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}
	for _, ft := range plan.Features {
		if ft.FeatureKey != maxProductsFeature {
			continue
		}
		count, err := s.repo.CountCompanyProducts(ctx, pgconv.ParseUUIDToPgType(companyId))
		if err != nil {
			return err
		}
		if int64(count)+int64(adding) > ft.LimitValue {
			return fmt.Errorf("%w: o plano permite %d produtos, já existem %d e o cadastro criaria mais %d", domain.ErrProductLimitReached, ft.LimitValue, count, adding)
		}
	}
	return nil
}

func listVariations(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID, model string) ([]domain.CatalogProduct, error) {
	rows, err := repo.ListModelVariations(ctx, db.ListModelVariationsParams{CompanyID: companyPg, ModelName: strings.TrimSpace(model)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.CatalogProduct, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.CatalogProduct{
			ID:         pgconv.PgUUIDToUUID(r.ID),
			Name:       r.Name,
			ModelName:  r.ModelName,
			Color:      r.Color,
			Size:       r.Size,
			Barcode:    r.Barcode,
			Unit:       r.Unit,
			SellInBulk: r.SellInBulk,
			Quantity:   r.Quantity,
			CostPrice:  r.CostPrice,
			SalePrice:  r.SalePrice,
		})
	}
	return out, nil
}

// uniqueBarcode gera códigos internos até achar um livre na empresa
func uniqueBarcode(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID) (string, error) {
	for range barcodeAttempts {
		code, err := domain.InternalEAN13()
		if err != nil {
			return "", err
		}
		exists, err := repo.BarcodeExists(ctx, db.BarcodeExistsParams{CompanyID: companyPg, Barcode: pgconv.ParseStringToPgType(code)})
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", domain.ErrBarcodeGenerationFailed
}
