package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	metaWhatsappService "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/service"
	planService "github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	subscriptionsService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

func StartWhatsAppUsageSyncWorker(
	subService *subscriptionsService.Service,
	metaWhatsappSvc *metaWhatsappService.Service,
	planService *planService.Service,
	discordLog *discord.DiscordLogger,
) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	c := cron.New(cron.WithLocation(loc))

	runSync := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		now := time.Now()

		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		dayAfterTomorrow := tomorrow.AddDate(0, 0, 1)

		subs, err := subService.ListSubscriptionsDueOn(ctx, subscriptionsDomain.ListSubscriptionsDueOnRequest{
			CurrentPeriodEnd:   tomorrow,
			CurrentPeriodEnd_2: dayAfterTomorrow,
		})
		if err != nil {
			discordLog.Send(domain.LevelWarning, "Erro ao ListSubscriptionsDueOn", err.Error())
			return
		}

		countSubs := len(subs)

		messageLogger := fmt.Sprintf("%d contas com uso excedente de mensagens", countSubs)

		discordLog.Send(domain.LevelInfo, "Worker WhatsApp Usage Sync rodando", messageLogger)

		for _, sub := range subs {

			plan, err := planService.GetPlanByID(ctx, sub.PlanID)
			if err != nil {
				discordLog.Send(domain.LevelWarning, "Erro ao GetPlanByID", err.Error())
				continue
			}

			var periodStart time.Time
			switch strings.ToLower(plan.BillingCycle) {
			case "monthly":
				periodStart = sub.CurrentPeriodEnd.AddDate(0, -1, 0)
			case "yearly":
				periodStart = sub.CurrentPeriodEnd.AddDate(-1, 0, 0)
			}

			if err := metaWhatsappSvc.SyncMonthlyUsage(ctx, sub.CompanyID, periodStart, sub.CurrentPeriodEnd); err != nil {
				discordLog.Send(domain.LevelWarning, "Erro ao SyncMonthlyUsage", err.Error())
				continue
			}
		}
	}

	c.AddFunc("0 0 * * *", runSync)
	c.Start()

	log.Info().Msg("Worker de sincronização de uso do WhatsApp iniciado")
	discordLog.Send(domain.LevelInfo, "Worker WhatsApp Usage Sync iniciado", "Agendado para rodar diariamente à meia-noite (America/Sao_Paulo)")
}
