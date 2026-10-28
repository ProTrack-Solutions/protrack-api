package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	companySettingsDomain "github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	metaDomain "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/eligibility"
	"github.com/ProTrack-Solutions/protrack-api/internal/receivable_reminders/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/receivable_reminders/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Sender interface {
	SendMessage(ctx context.Context, companyId uuid.UUID, req metaDomain.SendMessageRequest) (*metaDomain.Message, error)
}

type Settings interface {
	GetCompanySetting(ctx context.Context, companyId uuid.UUID, key enums.CompanySettingsKey) (companySettingsDomain.CompanySettingResponse, error)
}

type Eligibility interface {
	Check(ctx context.Context, companyID uuid.UUID) (eligibility.Result, error)
}

type Service struct {
	repo        *repository.Repository
	pool        *pgxpool.Pool
	sender      Sender
	settings    Settings
	eligibility Eligibility
	now         func() time.Time
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, sender Sender, settings Settings, eligibility Eligibility) *Service {
	return &Service{
		repo:        repo,
		pool:        pool,
		sender:      sender,
		settings:    settings,
		eligibility: eligibility,
		now:         time.Now,
	}
}

// Send manda o lembrete de uma parcela pelo WhatsApp. Parcela vencida usa o
// template de venda vencida; a vencer usa o template de lembrete, que a empresa
// precisa ter configurado. A parcela fica travada durante o envio e o
// lembrete só é registrado se a Meta aceitar a mensagem.
func (s *Service) Send(ctx context.Context, companyId, userId, accountId uuid.UUID) (domain.ReminderResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ReminderResponse{}, err
	}
	defer tx.Rollback(ctx)
	repo := s.repo.WithTx(tx)

	account, err := repo.LockReceivableForReminder(ctx, db.LockReceivableForReminderParams{
		ID:        pgconv.ParseUUIDToPgType(accountId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReminderResponse{}, domain.ErrReceivableNotFound
	}
	if err != nil {
		return domain.ReminderResponse{}, err
	}

	if account.Status == "paid" || account.Status == "canceled" || account.Balance <= 0 {
		return domain.ReminderResponse{}, domain.ErrAlreadySettled
	}

	loc, err := time.LoadLocation(account.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := s.now()
	kind := reminderKind(account.Status, account.DueDate, now.In(loc))

	templateKey, languageKey := enums.SaleOverdueTemplate, enums.LanguageSaleOverdueTemplate
	if kind == domain.KindDue {
		templateKey, languageKey = enums.DueReminderTemplate, enums.LanguageDueReminderTemplate
	}
	template, err := s.stringSetting(ctx, companyId, templateKey)
	if err != nil {
		return domain.ReminderResponse{}, err
	}
	if template == "" {
		return domain.ReminderResponse{}, domain.ErrTemplateNotConfigured
	}
	language, err := s.stringSetting(ctx, companyId, languageKey)
	if err != nil {
		return domain.ReminderResponse{}, err
	}
	if language == "" {
		language = "pt_BR"
	}

	if account.CustomerWhatsapp == "" {
		return domain.ReminderResponse{}, domain.ErrCustomerWithoutWhatsApp
	}

	last, err := repo.GetLastReceivableReminder(ctx, account.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.ReminderResponse{}, err
	}
	if err == nil && last.Valid {
		if next := last.Time.Add(domain.ReminderCooldown); now.Before(next) {
			return domain.ReminderResponse{}, &domain.TooSoonError{NextAt: next.In(loc)}
		}
	}

	allowed, err := s.eligibility.Check(ctx, companyId)
	if err != nil {
		return domain.ReminderResponse{}, err
	}
	if !allowed.Allowed {
		return domain.ReminderResponse{}, fmt.Errorf("%w: %s", domain.ErrWhatsAppUnavailable, allowed.Reason)
	}

	// Mesmas variáveis do envio automático de venda vencida, para os dois
	// templates poderem ter o mesmo formato
	message, err := s.sender.SendMessage(ctx, companyId, metaDomain.SendMessageRequest{
		TemplateName:   template,
		LanguageCode:   language,
		RecipientPhone: account.CustomerWhatsapp,
		Category:       "utility",
		HeaderVariables: map[string]string{
			"1": account.CompanyName,
		},
		Variables: map[string]string{
			"1": account.CustomerName,
			"2": account.SaleAt.Time.In(loc).Format("02/01/2006"),
			"3": strconv.FormatFloat(account.Balance, 'f', 2, 64),
			"4": account.DueDate.Time.Format("02/01/2006"),
			"5": account.CompanyPhone,
		},
	})
	if err != nil {
		log.Error().Err(err).Str("account_id", accountId.String()).Str("template", template).Msg("falha ao enviar lembrete de cobrança")
		return domain.ReminderResponse{}, fmt.Errorf("%w: %v", domain.ErrSendFailed, err)
	}

	var messageId pgtype.UUID
	if message != nil && message.ID != uuid.Nil {
		messageId = pgconv.ParseUUIDToPgType(message.ID)
	}

	sentAt, err := repo.CreateReceivableReminder(ctx, db.CreateReceivableReminderParams{
		CompanyID:           pgconv.ParseUUIDToPgType(companyId),
		AccountReceivableID: account.ID,
		WhatsappMessageID:   messageId,
		TemplateName:        template,
		RecipientPhone:      account.CustomerWhatsapp,
		SentBy:              pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.ReminderResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		// A mensagem já saiu: só o registro do lembrete se perdeu
		log.Error().Err(err).Str("account_id", accountId.String()).Msg("lembrete enviado mas não registrado")
		return domain.ReminderResponse{}, err
	}

	return domain.ReminderResponse{
		Kind:           kind,
		TemplateName:   template,
		RecipientPhone: maskPhone(account.CustomerWhatsapp),
		SentAt:         sentAt.Time.In(loc),
		NextReminderAt: sentAt.Time.Add(domain.ReminderCooldown).In(loc),
	}, nil
}

// reminderKind parcela vencida (pelo status ou pela data, antes do job da
// madrugada atualizar o status) usa o template de cobrança
func reminderKind(status string, dueDate pgtype.Date, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if status == "overdue" || (dueDate.Valid && dueDate.Time.Before(today)) {
		return domain.KindOverdue
	}
	return domain.KindDue
}

func (s *Service) stringSetting(ctx context.Context, companyId uuid.UUID, key enums.CompanySettingsKey) (string, error) {
	setting, err := s.settings.GetCompanySetting(ctx, companyId, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	value, _ := setting.Value.(string)
	return value, nil
}

// maskPhone "5511987654321" → "*********4321"
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	masked := make([]byte, len(phone)-4)
	for i := range masked {
		masked[i] = '*'
	}
	return string(masked) + phone[len(phone)-4:]
}
