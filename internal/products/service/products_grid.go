package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/repository"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/variation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	pgUniqueViolation = "23505"
	barcodeAttempts   = 5
	maxProductsKey    = "max_products"
	maxStockPerPage   = 100
)

// ListStockEntries estoque agrupado por modelo, com os mesmos totais do topo da tela de produtos
func (s *Service) ListStockEntries(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.StockEntriesResponse, error) {
	page := max(pagination.Page, 1)
	perPage := pagination.PerPage
	if perPage <= 0 {
		perPage = 10
	}
	perPage = min(perPage, maxStockPerPage)
	companyPg := pgconv.ParseUUIDToPgType(companyId)

	rows, err := s.repo.ListStockEntries(ctx, db.ListStockEntriesParams{
		CompanyID:  companyPg,
		Search:     strings.TrimSpace(pagination.Search),
		StartDate:  pgconv.StringToPgDate(pagination.StartDate),
		EndDate:    pgconv.StringToPgDate(pagination.EndDate),
		OrderBy:    pagination.OrderBy,
		PageLimit:  perPage,
		PageOffset: (page - 1) * perPage,
	})
	if err != nil {
		return domain.StockEntriesResponse{}, err
	}

	productsCount, err := s.repo.CountProductsByCompany(ctx, companyPg)
	if err != nil {
		return domain.StockEntriesResponse{}, err
	}
	totalValue, err := s.repo.GetCostTotalStock(ctx, companyPg)
	if err != nil {
		return domain.StockEntriesResponse{}, err
	}
	itens, err := s.repo.GetGlobalTotalStockQuantity(ctx, companyPg)
	if err != nil {
		return domain.StockEntriesResponse{}, err
	}
	low, err := s.repo.CountLowStockProductsByCompany(ctx, companyPg)
	if err != nil {
		return domain.StockEntriesResponse{}, err
	}

	resp := domain.StockEntriesResponse{
		Data:              make([]domain.StockEntry, 0, len(rows)),
		Page:              page,
		PerPage:           perPage,
		ProductsCount:     productsCount,
		TotalValueInStock: totalValue,
		ItensInStock:      itens,
		LowItensInStock:   int32(low),
	}
	for _, r := range rows {
		resp.TotalRows = r.TotalCount
		entry := domain.StockEntry{
			Kind:               "product",
			Key:                r.EntryKey,
			Name:               r.Name,
			ModelName:          r.ModelName,
			Description:        r.Description,
			Barcode:            r.Barcode,
			Size:               r.Size,
			Color:              r.Color,
			CategoryID:         pgconv.PgUUIDToUUID(r.CategoryID),
			Category:           r.CategoryName,
			Unit:               r.Unit,
			SellInBulk:         r.SellInBulk,
			Variations:         r.Variations,
			Colors:             r.Colors,
			Sizes:              r.Sizes,
			Quantity:           r.Quantity,
			LowStockVariations: r.LowStockVariations,
			MinCostPrice:       r.MinCostPrice,
			MaxCostPrice:       r.MaxCostPrice,
			MinSalePrice:       r.MinSalePrice,
			MaxSalePrice:       r.MaxSalePrice,
			TotalRegistered:    r.TotalRegistered,
			TotalSold:          r.TotalSold,
			CreatedAt:          pgconv.PgTimestamptzToTime(r.CreatedAt),
		}
		if r.IsModel {
			entry.Kind = "model"
			entry.Name = r.ModelName
			entry.Barcode, entry.Size, entry.Color = "", "", ""
		} else {
			id := pgconv.PgUUIDToUUID(r.ProductID)
			entry.ProductID = &id
		}
		resp.Data = append(resp.Data, entry)
	}
	resp.TotalPages = int32(math.Ceil(float64(resp.TotalRows) / float64(perPage)))
	return resp, nil
}

// ListModelVariations todas as variações de um modelo
func (s *Service) ListModelVariations(ctx context.Context, companyId uuid.UUID, modelName string) ([]domain.GridVariation, error) {
	out, err := listGridVariations(ctx, s.repo, pgconv.ParseUUIDToPgType(companyId), modelName)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, domain.ErrModelNotFound
	}
	return out, nil
}

// CreateModel cadastra um modelo novo com a grade e o estoque inicial de cada
// variação. Tudo ou nada.
func (s *Service) CreateModel(ctx context.Context, userId, companyId uuid.UUID, req domain.CreateModelRequest) ([]domain.GridVariation, error) {
	model, err := domain.NormalizeModelName(req.ModelName)
	if err != nil {
		return nil, err
	}
	variations, err := domain.NormalizeGridVariations(model, req.Variations)
	if err != nil {
		return nil, err
	}

	var out []domain.GridVariation
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		companyPg := pgconv.ParseUUIDToPgType(companyId)
		if err := checkCategory(ctx, repo, companyPg, req.CategoryID); err != nil {
			return err
		}
		existing, err := listGridVariations(ctx, repo, companyPg, model)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return fmt.Errorf("%w: %q já tem %d variações. Use \"Editar modelo\" para adicionar cores ou tamanhos", domain.ErrModelAlreadyExists, existing[0].ModelName, len(existing))
		}
		if err := s.checkProductLimit(ctx, repo, companyId, len(variations)); err != nil {
			return err
		}
		if err := insertVariations(ctx, repo, gridInsert{
			companyId:   companyPg,
			userId:      pgconv.ParseUUIDToPgType(userId),
			model:       model,
			categoryId:  pgconv.ParseUUIDToPgType(req.CategoryID),
			description: strings.TrimSpace(req.Description),
			unit:        db.UnitOfMeasure("UN"),
			costPrice:   req.CostPrice,
			salePrice:   req.SalePrice,
		}, variations); err != nil {
			return err
		}
		out, err = listGridVariations(ctx, repo, companyPg, model)
		return err
	})
	return out, err
}

// AddVariations adiciona cores/tamanhos a um modelo existente, herdando
// categoria, descrição e unidade dele
func (s *Service) AddVariations(ctx context.Context, userId, companyId uuid.UUID, modelName string, req domain.AddVariationsRequest) ([]domain.GridVariation, error) {
	var out []domain.GridVariation
	err := s.inTx(ctx, func(repo *repository.Repository) error {
		companyPg := pgconv.ParseUUIDToPgType(companyId)
		existing, err := listGridVariations(ctx, repo, companyPg, modelName)
		if err != nil {
			return err
		}
		if len(existing) == 0 {
			return domain.ErrModelNotFound
		}
		base := existing[0]
		variations, err := domain.NormalizeGridVariations(base.ModelName, req.Variations)
		if err != nil {
			return err
		}
		if err := checkDuplicates(base.ModelName, existing, variations); err != nil {
			return err
		}
		if err := s.checkProductLimit(ctx, repo, companyId, len(variations)); err != nil {
			return err
		}
		if err := insertVariations(ctx, repo, gridInsert{
			companyId:   companyPg,
			userId:      pgconv.ParseUUIDToPgType(userId),
			model:       base.ModelName,
			categoryId:  pgconv.ParseUUIDToPgType(base.CategoryID),
			description: base.Description,
			unit:        db.UnitOfMeasure(base.Unit),
			costPrice:   req.CostPrice,
			salePrice:   req.SalePrice,
		}, variations); err != nil {
			return err
		}
		out, err = listGridVariations(ctx, repo, companyPg, base.ModelName)
		return err
	})
	return out, err
}

// UpdateModel renomeia o modelo, troca categoria/descrição de todas as
// variações e aplica os preços enviados por variação
func (s *Service) UpdateModel(ctx context.Context, userId, companyId uuid.UUID, currentName string, req domain.UpdateModelRequest) ([]domain.GridVariation, error) {
	newName, err := domain.NormalizeModelName(req.ModelName)
	if err != nil {
		return nil, err
	}

	var out []domain.GridVariation
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		companyPg := pgconv.ParseUUIDToPgType(companyId)
		existing, err := lockModel(ctx, repo, companyPg, currentName)
		if err != nil {
			return err
		}

		if !strings.EqualFold(newName, existing[0].ModelName) {
			other, err := listGridVariations(ctx, repo, companyPg, newName)
			if err != nil {
				return err
			}
			if len(other) > 0 {
				return fmt.Errorf("%w: %q. Para juntar os dois, use \"Agrupar em modelo\"", domain.ErrModelAlreadyExists, other[0].ModelName)
			}
		}
		if err := checkCategory(ctx, repo, companyPg, req.CategoryID); err != nil {
			return err
		}

		byID := make(map[uuid.UUID]domain.GridVariation, len(existing))
		for _, v := range existing {
			byID[v.ID] = v
		}
		prices := make(map[uuid.UUID]domain.VariationPriceInput, len(req.Variations))
		barcodes := make(map[uuid.UUID]pgtype.Text, len(req.Variations))
		seenBarcode := map[string]bool{}
		for _, p := range req.Variations {
			current, ok := byID[p.ID]
			if !ok {
				return fmt.Errorf("%w: a variação %s não pertence ao modelo", domain.ErrInvalidGrid, p.ID)
			}
			prices[p.ID] = p
			if p.Barcode == nil {
				continue
			}
			code := strings.TrimSpace(*p.Barcode)
			if code == "" || code == current.Barcode {
				continue
			}
			if seenBarcode[code] {
				return fmt.Errorf("%w: o código %s aparece em mais de uma variação", domain.ErrInvalidGrid, code)
			}
			seenBarcode[code] = true
			exists, err := repo.GridBarcodeExists(ctx, db.GridBarcodeExistsParams{CompanyID: companyPg, Barcode: code})
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("%w: %s (%s)", domain.ErrBarcodeAlreadyExists, code, current.Name)
			}
			barcodes[p.ID] = pgconv.ParseStringToPgText(code)
		}

		description := strings.TrimSpace(req.Description)
		for _, v := range existing {
			name := variation.Name(newName, v.Color, v.Size)
			if len([]rune(name)) > variation.MaxProductName {
				return fmt.Errorf("%w: o nome %q passa de %d caracteres", domain.ErrInvalidGrid, name, variation.MaxProductName)
			}
			cost, price := v.CostPrice, v.SalePrice
			if p, ok := prices[v.ID]; ok {
				cost, price = p.CostPrice, p.SalePrice
			}
			if err := repo.UpdateGridVariation(ctx, db.UpdateGridVariationParams{
				ID:          pgconv.ParseUUIDToPgType(v.ID),
				CompanyID:   companyPg,
				ModelName:   newName,
				Name:        name,
				CategoryID:  pgconv.ParseUUIDToPgType(req.CategoryID),
				Description: pgconv.ParseStringToPgText(description),
				CostPrice:   pgconv.Float64ToPgNumeric(domain.Round2(cost)),
				SalePrice:   pgconv.Float64ToPgNumeric(domain.Round2(price)),
				Barcode:     barcodes[v.ID],
				UpdatedBy:   pgconv.ParseUUIDToPgType(userId),
			}); err != nil {
				return translateGridError(err)
			}
		}
		out, err = listGridVariations(ctx, repo, companyPg, newName)
		return err
	})
	return out, err
}

// RemoveVariation desativa uma variação sem saldo
func (s *Service) RemoveVariation(ctx context.Context, userId, companyId, productId uuid.UUID) error {
	affected, err := s.repo.WithTx(s.pool).RemoveGridVariation(ctx, db.RemoveGridVariationParams{
		ID:        pgconv.ParseUUIDToPgType(productId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		DeletedBy: pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	product, err := s.repo.GetProductById(ctx, pgconv.ParseUUIDToPgType(productId))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && pgconv.PgUUIDToUUID(product.CompanyID) != companyId) {
		return domain.ErrProductNotFound
	}
	if err != nil {
		return err
	}
	if !product.ModelName.Valid {
		return fmt.Errorf("%w: o produto não faz parte de uma grade", domain.ErrInvalidGrid)
	}
	return domain.ErrVariationHasStock
}

// GridStockMovement lança entrada ou ajuste em várias variações do modelo de uma vez
func (s *Service) GridStockMovement(ctx context.Context, userId, companyId uuid.UUID, modelName string, req domain.GridStockMovementRequest) ([]domain.StockMovementResponse, error) {
	if err := domain.ValidateGridStockMovement(req); err != nil {
		return nil, err
	}

	var out []domain.StockMovementResponse
	err := s.inTx(ctx, func(repo *repository.Repository) error {
		companyPg := pgconv.ParseUUIDToPgType(companyId)
		existing, err := listGridVariations(ctx, repo, companyPg, modelName)
		if err != nil {
			return err
		}
		if len(existing) == 0 {
			return domain.ErrModelNotFound
		}
		byID := make(map[uuid.UUID]domain.GridVariation, len(existing))
		for _, v := range existing {
			byID[v.ID] = v
		}

		reason := strings.TrimSpace(req.Reason)
		for _, it := range req.Items {
			v, ok := byID[it.ProductID]
			if !ok {
				return fmt.Errorf("%w: a variação %s não pertence ao modelo", domain.ErrInvalidGrid, it.ProductID)
			}
			if it.Quantity == 0 {
				continue
			}
			mv, err := repo.AdjustStock(ctx, db.AdjustStockParams{
				Quantity:  it.Quantity,
				CreatedBy: pgconv.ParseUUIDToPgType(userId),
				ProductID: pgconv.ParseUUIDToPgType(it.ProductID),
				CompanyID: companyPg,
				Type:      db.StockMovementType(req.Type),
				Reason:    pgconv.ParseStringToPgText(reason),
			})
			if errors.Is(err, pgx.ErrNoRows) {
				if v.SellInBulk {
					return fmt.Errorf("%w: %s", domain.ErrProductSoldInBulk, v.Name)
				}
				return fmt.Errorf("%w: %s tem %d em estoque", domain.ErrInsufficientStock, v.Name, v.Quantity)
			}
			if err != nil {
				return err
			}
			out = append(out, domain.StockMovementResponse{
				ID:              pgconv.PgUUIDToUUID(mv.ID),
				ProductID:       pgconv.PgUUIDToUUID(mv.ProductID),
				Type:            string(mv.Type),
				Quantity:        mv.Quantity,
				Reason:          pgconv.ParsePgTextToString(mv.Reason),
				CurrentQuantity: int32(pgconv.PgInt4ToInt(mv.CurrentQuantity)),
				CreatedAt:       pgconv.PgTimestamptzToTime(mv.CreatedAt),
			})
		}
		return nil
	})
	return out, err
}

// GroupProducts transforma produtos soltos em variações de um modelo (novo ou
// existente). O produto continua o mesmo, então saldo, histórico e vendas ficam.
func (s *Service) GroupProducts(ctx context.Context, userId, companyId uuid.UUID, req domain.GroupProductsRequest) ([]domain.GridVariation, error) {
	model, err := domain.NormalizeModelName(req.ModelName)
	if err != nil {
		return nil, err
	}
	items, err := domain.NormalizeGroupItems(model, req.Items)
	if err != nil {
		return nil, err
	}

	var out []domain.GridVariation
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		companyPg := pgconv.ParseUUIDToPgType(companyId)
		existing, err := listGridVariations(ctx, repo, companyPg, model)
		if err != nil {
			return err
		}

		categoryID := req.CategoryID
		if len(existing) > 0 {
			model = existing[0].ModelName
			categoryID = existing[0].CategoryID
		} else {
			if categoryID == uuid.Nil {
				return fmt.Errorf("%w: escolha a categoria do modelo", domain.ErrInvalidGrid)
			}
			if err := checkCategory(ctx, repo, companyPg, categoryID); err != nil {
				return err
			}
		}

		ids := make([]pgtype.UUID, 0, len(items))
		for _, it := range items {
			ids = append(ids, pgconv.ParseUUIDToPgType(it.ProductID))
		}
		locked, err := repo.LockGridProducts(ctx, db.LockGridProductsParams{CompanyID: companyPg, Ids: ids})
		if err != nil {
			return err
		}
		if len(locked) != len(items) {
			return domain.ErrProductNotFound
		}
		for _, p := range locked {
			if p.ModelName.Valid {
				return fmt.Errorf("%w: %q já faz parte do modelo %q", domain.ErrInvalidGrid, p.Name, p.ModelName.String)
			}
			if p.SellInBulk {
				return fmt.Errorf("%w: %q é vendido a granel e não entra em grade", domain.ErrInvalidGrid, p.Name)
			}
		}

		asInputs := make([]domain.GridVariationInput, 0, len(items))
		for _, it := range items {
			asInputs = append(asInputs, domain.GridVariationInput{Color: it.Color, Size: it.Size})
		}
		if err := checkDuplicates(model, existing, asInputs); err != nil {
			return err
		}

		for _, it := range items {
			if err := repo.AssignProductToModel(ctx, db.AssignProductToModelParams{
				ID:         pgconv.ParseUUIDToPgType(it.ProductID),
				CompanyID:  companyPg,
				ModelName:  model,
				Color:      pgconv.ParseStringToPgText(it.Color),
				Size:       pgconv.ParseStringToPgText(it.Size),
				Name:       variation.Name(model, it.Color, it.Size),
				CategoryID: pgconv.ParseUUIDToPgType(categoryID),
				UpdatedBy:  pgconv.ParseUUIDToPgType(userId),
			}); err != nil {
				return translateGridError(err)
			}
		}
		out, err = listGridVariations(ctx, repo, companyPg, model)
		return err
	})
	return out, err
}

// --- auxiliares ---

func (s *Service) inTx(ctx context.Context, fn func(repo *repository.Repository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(s.repo.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type gridInsert struct {
	companyId   pgtype.UUID
	userId      pgtype.UUID
	model       string
	categoryId  pgtype.UUID
	description string
	unit        db.UnitOfMeasure
	costPrice   float64
	salePrice   float64
}

func insertVariations(ctx context.Context, repo *repository.Repository, base gridInsert, variations []domain.GridVariationInput) error {
	if base.unit == "" {
		base.unit = db.UnitOfMeasure("UN")
	}
	for _, v := range variations {
		barcode := v.Barcode
		if barcode != "" {
			exists, err := repo.GridBarcodeExists(ctx, db.GridBarcodeExistsParams{CompanyID: base.companyId, Barcode: barcode})
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("%w: %s (%s)", domain.ErrBarcodeAlreadyExists, barcode, variation.Name(base.model, v.Color, v.Size))
			}
		} else {
			code, err := uniqueBarcode(ctx, repo, base.companyId)
			if err != nil {
				return err
			}
			barcode = code
		}

		if _, err := repo.CreateGridProduct(ctx, db.CreateGridProductParams{
			CompanyID:   base.companyId,
			CategoryID:  base.categoryId,
			Name:        variation.Name(base.model, v.Color, v.Size),
			Description: pgconv.ParseStringToPgText(base.description),
			ModelName:   base.model,
			Color:       pgconv.ParseStringToPgText(v.Color),
			Size:        pgconv.ParseStringToPgText(v.Size),
			Barcode:     barcode,
			Quantity:    v.Quantity,
			CostPrice:   pgconv.Float64ToPgNumeric(domain.PriceOrDefault(v.CostPrice, base.costPrice)),
			SalePrice:   pgconv.Float64ToPgNumeric(domain.PriceOrDefault(v.SalePrice, base.salePrice)),
			CreatedBy:   base.userId,
			Unit:        base.unit,
		}); err != nil {
			return translateGridError(err)
		}
	}
	return nil
}

// checkDuplicates recusa combinações que o modelo já tem, listando todas
func checkDuplicates(model string, existing []domain.GridVariation, variations []domain.GridVariationInput) error {
	have := make(map[string]string, len(existing))
	for _, e := range existing {
		have[variation.Key(e.Color, e.Size)] = e.Name
	}
	var dup []string
	for _, v := range variations {
		if name, ok := have[variation.Key(v.Color, v.Size)]; ok {
			dup = append(dup, name)
		}
	}
	if len(dup) > 0 {
		return fmt.Errorf("%w: %s", domain.ErrVariationsExist, strings.Join(dup, ", "))
	}
	return nil
}

func checkCategory(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID, categoryID uuid.UUID) error {
	ok, err := repo.GridCategoryExists(ctx, db.GridCategoryExistsParams{ID: pgconv.ParseUUIDToPgType(categoryID), CompanyID: companyPg})
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrProductCategoryNotFound
	}
	return nil
}

// checkProductLimit respeita o máximo de produtos do plano para o lote inteiro
func (s *Service) checkProductLimit(ctx context.Context, repo *repository.Repository, companyId uuid.UUID, adding int) error {
	if s.subscriptionsService == nil || s.plansService == nil {
		return nil
	}
	sub, err := s.subscriptionsService.GetSubscriptionByCompanyID(ctx, companyId)
	if err != nil {
		return err
	}
	plan, err := s.plansService.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}
	for _, ft := range plan.Features {
		if ft.FeatureKey != maxProductsKey {
			continue
		}
		count, err := repo.CountActiveProducts(ctx, pgconv.ParseUUIDToPgType(companyId))
		if err != nil {
			return err
		}
		if int64(count)+int64(adding) > ft.LimitValue {
			return fmt.Errorf("%w: o plano permite %d produtos, já existem %d e a grade criaria mais %d", domain.ErrProductLimitReached, ft.LimitValue, count, adding)
		}
	}
	return nil
}

// uniqueBarcode gera códigos internos até achar um livre na empresa
func uniqueBarcode(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID) (string, error) {
	for range barcodeAttempts {
		code, err := variation.InternalEAN13()
		if err != nil {
			return "", err
		}
		exists, err := repo.GridBarcodeExists(ctx, db.GridBarcodeExistsParams{CompanyID: companyPg, Barcode: code})
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", domain.ErrBarcodeGeneration
}

type gridLister interface {
	ListGridVariations(ctx context.Context, arg db.ListGridVariationsParams) ([]db.ListGridVariationsRow, error)
}

func listGridVariations(ctx context.Context, repo gridLister, companyPg pgtype.UUID, model string) ([]domain.GridVariation, error) {
	rows, err := repo.ListGridVariations(ctx, db.ListGridVariationsParams{
		CompanyID: companyPg,
		ModelName: strings.Join(strings.Fields(model), " "),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.GridVariation, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.GridVariation{
			ID:              pgconv.PgUUIDToUUID(r.ID),
			Name:            r.Name,
			ModelName:       r.ModelName,
			Color:           r.Color,
			Size:            r.Size,
			Barcode:         r.Barcode,
			Description:     r.Description,
			CategoryID:      pgconv.PgUUIDToUUID(r.CategoryID),
			CategoryName:    r.CategoryName,
			Unit:            r.Unit,
			SellInBulk:      r.SellInBulk,
			Quantity:        int32(pgconv.PgInt4ToInt(r.Quantity)),
			CostPrice:       r.CostPrice,
			SalePrice:       r.SalePrice,
			TotalRegistered: r.TotalRegistered,
			TotalSold:       r.TotalSold,
			CreatedAt:       pgconv.PgTimestamptzToTime(r.CreatedAt),
		})
	}
	return out, nil
}

// lockModel lista e trava as variações do modelo até o fim da transação
func lockModel(ctx context.Context, repo *repository.Repository, companyPg pgtype.UUID, model string) ([]domain.GridVariation, error) {
	existing, err := listGridVariations(ctx, repo, companyPg, model)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return nil, domain.ErrModelNotFound
	}
	ids := make([]pgtype.UUID, 0, len(existing))
	for _, v := range existing {
		ids = append(ids, pgconv.ParseUUIDToPgType(v.ID))
	}
	if _, err := repo.LockGridProducts(ctx, db.LockGridProductsParams{CompanyID: companyPg, Ids: ids}); err != nil {
		return nil, err
	}
	return existing, nil
}

// translateGridError: índice único da grade (cadastro simultâneo) e categoria inexistente
func translateGridError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrVariationsExist
	}
	return domain.TranslateProductDBError(err)
}
