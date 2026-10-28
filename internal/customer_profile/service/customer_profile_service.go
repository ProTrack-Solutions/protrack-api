package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/customer_profile/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/customer_profile/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	repo *repository.Repository
	now  func() time.Time
}

func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// location é o fuso da empresa: "hoje" e "este mês" seguem o horário dela
func (s *Service) location(ctx context.Context, company pgtype.UUID) *time.Location {
	tz, err := s.repo.GetCompanyTimezone(ctx, company)
	if err != nil {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func (s *Service) Overview(ctx context.Context, companyId uuid.UUID) (domain.Overview, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	now := s.now().In(s.location(ctx, company))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	r, err := s.repo.CustomerOverview(ctx, db.CustomerOverviewParams{
		CompanyID:      company,
		MonthStart:     pgconv.TimeToPgTimestamptz(monthStart),
		LastMonthStart: pgconv.TimeToPgTimestamptz(monthStart.AddDate(0, -1, 0)),
		Today:          pgconv.ToPgDate(now),
		RecentSince:    pgconv.TimeToPgTimestamptz(now.AddDate(0, 0, -30)),
	})
	if err != nil {
		return domain.Overview{}, err
	}

	return domain.Overview{
		ActiveCount:   r.ActiveCount,
		InactiveCount: r.InactiveCount,
		NewThisMonth:  r.NewThisMonth,
		NewLastMonth:  r.NewLastMonth,
		DebtorsCount:  r.DebtorsCount,
		DebtTotal:     round2(r.DebtTotal),
		OverdueCount:  r.OverdueCount,
		OverdueTotal:  round2(r.OverdueTotal),
		CreditCount:   r.CreditCount,
		CreditTotal:   round2(r.CreditTotal),
		RecentBuyers:  r.RecentBuyers,
	}, nil
}

// digits deixa só os números da busca (CPF e telefone digitados com máscara)
func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func (s *Service) Browse(ctx context.Context, companyId uuid.UUID, params domain.BrowseParams) (domain.BrowseResponse, error) {
	page, perPage := params.Page, params.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	status, segment, sort := params.Status, params.Segment, params.Sort
	if status == "" {
		status = "active"
	}
	if segment == "" {
		segment = "all"
	}
	if sort == "" {
		sort = "recent"
	}

	company := pgconv.ParseUUIDToPgType(companyId)
	search := strings.TrimSpace(params.Search)
	// CPF e telefone são comparados só pelos dígitos (busca com ou sem máscara)
	searchDigits := digits(search)

	rows, err := s.repo.BrowseCustomers(ctx, db.BrowseCustomersParams{
		CompanyID:    company,
		Status:       status,
		Segment:      segment,
		Search:       search,
		SearchDigits: searchDigits,
		Sort:         sort,
		RowLimit:     perPage,
		RowOffset:    (page - 1) * perPage,
		Today:        pgconv.ToPgDate(s.now().In(s.location(ctx, company))),
	})
	if err != nil {
		return domain.BrowseResponse{}, err
	}

	out := domain.BrowseResponse{Data: make([]domain.CustomerRow, 0, len(rows)), Page: page, PerPage: perPage}
	for _, r := range rows {
		out.Total = r.TotalRows
		out.Data = append(out.Data, domain.CustomerRow{
			ID:             pgconv.PgUUIDToUUID(r.ID),
			FullName:       r.FullName,
			Cpf:            r.Cpf,
			Email:          r.Email,
			Whatsapp:       r.Whatsapp,
			MobilePhone:    r.MobilePhone,
			City:           r.AddressCity,
			State:          r.AddressState,
			BalanceDue:     round2(r.BalanceDue),
			OverdueTotal:   round2(r.OverdueTotal),
			StoreCredit:    round2(r.StoreCredit),
			PurchasesCount: r.PurchasesCount,
			PurchasesTotal: round2(r.PurchasesTotal),
			LastPurchaseAt: timePtr(r.LastPurchaseAt),
			CreatedAt:      pgconv.PgTimestamptzToTime(r.CreatedAt),
			Active:         !r.DeletedAt.Valid,
		})
	}
	return out, nil
}

func (s *Service) Profile(ctx context.Context, companyId, customerId uuid.UUID) (domain.Profile, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	customer := pgconv.ParseUUIDToPgType(customerId)

	c, err := s.repo.GetCustomerProfile(ctx, db.GetCustomerProfileParams{ID: customer, CompanyID: company})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrCustomerNotFound
	}
	if err != nil {
		return domain.Profile{}, err
	}

	installments, err := s.repo.ListCustomerOpenInstallments(ctx, db.ListCustomerOpenInstallmentsParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.Profile{}, err
	}
	sales, err := s.repo.ListCustomerRecentSales(ctx, db.ListCustomerRecentSalesParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.Profile{}, err
	}
	credits, err := s.repo.ListCustomerCreditMovements(ctx, db.ListCustomerCreditMovementsParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.Profile{}, err
	}

	purchased, err := s.repo.ListCustomerPurchasedItems(ctx, db.ListCustomerPurchasedItemsParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.Profile{}, err
	}
	moments, err := s.repo.ListCustomerSaleMoments(ctx, db.ListCustomerSaleMomentsParams{CustomerID: customer, CompanyID: company})
	if err != nil {
		return domain.Profile{}, err
	}

	loc := s.location(ctx, company)
	now := s.now().In(loc)
	today := now.Format("2006-01-02")
	net := round2(c.GrossTotal - c.ReturnedTotal)

	out := domain.Profile{
		ID:                  customerId,
		FullName:            c.FullName,
		Cpf:                 c.Cpf,
		Rg:                  c.Rg,
		Gender:              c.Gender,
		MaritalStatus:       c.MaritalStatus,
		Email:               c.Email,
		Whatsapp:            c.Whatsapp,
		MobilePhone:         c.MobilePhone,
		HomePhone:           c.HomePhone,
		AddressStreet:       c.AddressStreet,
		AddressNumber:       c.AddressNumber,
		AddressComplement:   c.AddressComplement,
		AddressNeighborhood: c.AddressNeighborhood,
		AddressCity:         c.AddressCity,
		AddressState:        c.AddressState,
		AddressZipcode:      c.AddressZipcode,
		CreatedAt:           pgconv.PgTimestamptzToTime(c.CreatedAt),
		Active:              !c.DeletedAt.Valid,
		BalanceDue:          round2(c.BalanceDue),
		StoreCredit:         round2(c.StoreCredit),
		PurchasesCount:      c.PurchasesCount,
		PurchasesTotal:      net,
		ReturnedTotal:       round2(c.ReturnedTotal),
		FirstPurchaseAt:     timePtr(c.FirstPurchaseAt),
		LastPurchaseAt:      timePtr(c.LastPurchaseAt),
		OpenInstallments:    make([]domain.Installment, 0, len(installments)),
		RecentSales:         make([]domain.RecentSale, 0, len(sales)),
		CreditMovements:     make([]domain.CreditMovement, 0, len(credits)),
	}
	if c.BirthDate.Valid {
		b := pgconv.PgDateToString(c.BirthDate)
		out.BirthDate = &b
	}
	if c.PurchasesCount > 0 {
		out.AverageTicket = round2(net / float64(c.PurchasesCount))
	}

	for _, i := range installments {
		due := pgconv.PgDateToString(i.DueDate)
		overdue := due < today
		if overdue {
			out.OverdueTotal += i.Balance
		}
		out.OpenInstallments = append(out.OpenInstallments, domain.Installment{
			ID:                pgconv.PgUUIDToUUID(i.ID),
			SaleID:            pgconv.PgUUIDToUUID(i.SaleID),
			InstallmentNumber: int32(pgconv.PgInt4ToInt(i.InstallmentNumber)),
			TotalInstallments: int32(pgconv.PgInt4ToInt(i.TotalInstallments)),
			DueDate:           due,
			TotalAmount:       i.TotalAmount,
			Balance:           i.Balance,
			Status:            i.Status,
			Overdue:           overdue,
		})
	}
	out.OverdueTotal = round2(out.OverdueTotal)

	items := make([]domain.PurchasedItem, 0, len(purchased))
	for _, it := range purchased {
		items = append(items, domain.PurchasedItem{
			Name:     it.ProductName,
			Model:    it.ModelName,
			Category: it.CategoryName,
			Size:     it.Size,
			Color:    it.Color,
			Quantity: it.Quantity,
			Total:    it.Total,
			At:       pgconv.PgTimestamptzToTime(it.CreatedAt),
		})
	}
	saleMoments := make([]domain.SaleMoment, 0, len(moments))
	for _, m := range moments {
		saleMoments = append(saleMoments, domain.SaleMoment{
			At:     pgconv.PgTimestamptzToTime(m.CreatedAt),
			Total:  m.TotalAmount,
			Method: m.PaymentMethod,
		})
	}
	out.Insights = domain.BuildInsights(items, saleMoments, now, loc)

	for _, sl := range sales {
		out.RecentSales = append(out.RecentSales, domain.RecentSale{
			ID:            pgconv.PgUUIDToUUID(sl.ID),
			CreatedAt:     pgconv.PgTimestamptzToTime(sl.CreatedAt),
			TotalAmount:   sl.TotalAmount,
			PaymentMethod: sl.PaymentMethod,
			Status:        sl.Status,
			ItemsCount:    sl.ItemsCount,
			Products:      sl.Products,
			ReturnedTotal: sl.ReturnedTotal,
		})
	}

	for i, m := range credits {
		if i == 10 {
			break
		}
		out.CreditMovements = append(out.CreditMovements, domain.CreditMovement{
			ID:           pgconv.PgUUIDToUUID(m.ID),
			Type:         m.Type,
			Amount:       m.Amount,
			BalanceAfter: m.BalanceAfter,
			SaleReturnID: pgconv.PgUUIDToUUIDPtr(m.SaleReturnID),
			ReturnNumber: pgconv.PgInt4ToIntPtr(m.ReturnNumber),
			SaleID:       pgconv.PgUUIDToUUIDPtr(m.SaleID),
			CreatedAt:    pgconv.PgTimestamptzToTime(m.CreatedAt),
		})
	}

	return out, nil
}

// SetActive inativa (mantendo o histórico) ou reativa o cliente
func (s *Service) SetActive(ctx context.Context, companyId, userId, customerId uuid.UUID, active bool) (domain.Profile, error) {
	affected, err := s.repo.SetCustomerActive(ctx, db.SetCustomerActiveParams{
		Active:    active,
		UserID:    pgconv.ParseUUIDToPgType(userId),
		ID:        pgconv.ParseUUIDToPgType(customerId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return domain.Profile{}, err
	}
	if affected == 0 {
		return domain.Profile{}, domain.ErrCustomerNotFound
	}
	return s.Profile(ctx, companyId, customerId)
}
