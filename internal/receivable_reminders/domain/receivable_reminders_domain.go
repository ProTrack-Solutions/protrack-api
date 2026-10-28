package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrReceivableNotFound      = errors.New("conta a receber não encontrada")
	ErrAlreadySettled          = errors.New("esta parcela já está quitada ou cancelada")
	ErrTemplateNotConfigured   = errors.New("nenhum template de lembrete para parcela a vencer está configurado; defina um em Configurações > Sistema > WhatsApp")
	ErrCustomerWithoutWhatsApp = errors.New("o cliente não tem número de WhatsApp cadastrado")
	ErrWhatsAppUnavailable     = errors.New("envio de WhatsApp indisponível")
	ErrSendFailed              = errors.New("não foi possível enviar a mensagem pelo WhatsApp")
)

// Intervalo mínimo entre lembretes da mesma parcela
const ReminderCooldown = 24 * time.Hour

// TooSoonError lembrete repetido antes do intervalo mínimo
type TooSoonError struct {
	NextAt time.Time
}

func (e *TooSoonError) Error() string {
	return fmt.Sprintf("um lembrete desta parcela já foi enviado nas últimas 24 horas; o próximo pode ser enviado a partir de %s", e.NextAt.Format("02/01/2006 15:04"))
}

// Tipo do lembrete conforme a situação da parcela
const (
	KindOverdue = "overdue" // cobrança de parcela vencida (template de venda vencida)
	KindDue     = "due"     // aviso de parcela a vencer (template de lembrete)
)

type ReminderResponse struct {
	Kind           string    `json:"kind"`
	TemplateName   string    `json:"template_name"`
	RecipientPhone string    `json:"recipient_phone"`
	SentAt         time.Time `json:"sent_at"`
	NextReminderAt time.Time `json:"next_reminder_at"`
}
