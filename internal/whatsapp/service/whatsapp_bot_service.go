package service

import (
	"context"
	"fmt"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/domain"
	"github.com/google/uuid"
)

func (s *Service) CreateWhatsAppBot(ctx context.Context, companyId uuid.UUID, req domain.CreateWhatsAppBotConfigRequest) error {
	company, err := s.companiesService.GetCompanyByID(ctx, companyId)
	if err != nil {
		return fmt.Errorf("failed to retrieve company: %w", err)
	}

	instanceName := fmt.Sprintf("%s-%s", company.Name, companyId.String())

	webhookURL := fmt.Sprintf("%s/api/v1/whatsapp/webhook/%s", s.cfg.ApiBaseURL, companyId.String())
	if err := s.whatsApp.SetWebhook(ctx, instanceName, webhookURL); err != nil {
		return fmt.Errorf("failed to configure webhook: %w", err)
	}

	wpBot, err := s.repo.CreateWhatsAppBotConfig(ctx, db.CreateWhatsAppBotConfigParams{
		CompanyID:      pgconv.OptionalUUIDToPgType(companyId),
		InstanceName:   instanceName,
		WelcomeMessage: req.WelcomeMessage,
		IsActive:       true,
	})
	if err != nil {
		return err
	}

	for _, mp := range req.MenuOption {
		_, err := s.repo.CreateWhatsAppBotMenuOption(ctx, db.CreateWhatsAppBotMenuOptionParams{
			BotConfigID:     wpBot.ID,
			OptionKey:       mp.OptionKey,
			Label:           mp.Label,
			ResponseMessage: mp.ResponseMessage,
			OrderIndex:      mp.OrderIndex,
		})
		if err != nil {
			return err
		}

	}

	return nil
}

func (s *Service) GetWhatsAppBotConfigWithOptionsByCompanyID(ctx context.Context, companyId uuid.UUID) (domain.WhatsappBotConfig, error) {
	company, err := s.companiesService.GetCompanyByID(ctx, companyId)
	if err != nil {
		return domain.WhatsappBotConfig{}, fmt.Errorf("failed to retrieve company: %w", err)
	}

	instanceName := fmt.Sprintf("%s-%s", company.Name, companyId.String())

	rows, err := s.repo.GetWhatsAppBotConfigWithOptionsByInstance(ctx, instanceName)
	if err != nil {
		return domain.WhatsappBotConfig{}, fmt.Errorf("failed to retrieve whatsapp bot config: %w", err)
	}

	if len(rows) == 0 {
		return domain.WhatsappBotConfig{}, domain.ErrWhatsAppBotConfigNotFound
	}

	first := rows[0]
	config := domain.WhatsappBotConfig{
		ID:                pgconv.PgUUIDToUUID(first.BotConfigID),
		InstanceName:      first.InstanceName,
		WelcomeMessage:    first.WelcomeMessage,
		IsActive:          first.IsActive,
		WhatsappBotOption: make([]domain.WhatsappBotMenuOption, 0, len(rows)),
	}

	for _, row := range rows {
		// LEFT JOIN: se não houver opções cadastradas, option_id vem NULL.
		if !row.OptionID.Valid {
			continue
		}

		config.WhatsappBotOption = append(config.WhatsappBotOption, domain.WhatsappBotMenuOption{
			ID:              pgconv.PgUUIDToUUID(row.OptionID),
			OptionKey:       row.OptionKey.String,
			Label:           row.Label.String,
			ResponseMessage: row.ResponseMessage.String,
			OrderIndex:      row.OrderIndex.Int32,
		})
	}

	return config, nil
}

func (s *Service) UpdateWhatsAppBotConfig(ctx context.Context, id, companyId uuid.UUID, req domain.UpdateWhatsAppBotConfigRequest) error {
	existing, err := s.repo.GetWhatsAppBotConfigByCompanyID(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return domain.ErrWhatsAppBotConfigNotFound
	}

	if pgconv.PgUUIDToUUID(existing.ID) != id {
		return domain.ErrWhatsAppBotConfigNotFound
	}

	if _, err := s.repo.UpdateWhatsAppBotConfig(ctx, db.UpdateWhatsAppBotConfigParams{
		ID:             pgconv.ParseUUIDToPgType(id),
		WelcomeMessage: req.WelcomeMessage,
		IsActive:       req.IsActive,
	}); err != nil {
		return err
	}

	if err := s.repo.DeleteAllMenuOptionsForBotConfig(ctx, pgconv.ParseUUIDToPgType(id)); err != nil {
		return err
	}

	for _, mp := range req.MenuOption {
		if _, err := s.repo.CreateWhatsAppBotMenuOption(ctx, db.CreateWhatsAppBotMenuOptionParams{
			BotConfigID:     pgconv.ParseUUIDToPgType(id),
			OptionKey:       mp.OptionKey,
			Label:           mp.Label,
			ResponseMessage: mp.ResponseMessage,
			OrderIndex:      mp.OrderIndex,
		}); err != nil {
			return err
		}
	}

	return nil
}

// SendBotReply envia a resposta do bot para o número que escreveu, através da
// instância da Evolution API associada à empresa.
func (s *Service) SendBotReply(ctx context.Context, instanceName, remoteJid, message string) error {
	number := strings.SplitN(remoteJid, "@", 2)[0]
	return s.whatsApp.SendWhatsAppMessage(number, message, instanceName)
}
