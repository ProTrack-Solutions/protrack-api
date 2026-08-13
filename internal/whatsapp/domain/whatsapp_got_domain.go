package domain

import (
	"time"

	"github.com/google/uuid"
)

type EvolutionWebhookPayload struct {
	Event    string `json:"event"`
	Instance string `json:"instance"`
	// Sender é o JID do remetente baseado em número de telefone
	// ("<numero>@s.whatsapp.net"), enviado pela Evolution API na raiz do payload
	// (irmão de "data", não dentro dele). Quando o contato tem a privacidade de
	// número ativada (migração da Meta para LID), data.key.remoteJid vem como um
	// identificador interno "<id>@lid" em vez do número real — nesse caso, Sender
	// é a fonte confiável do número de telefone para responder a mensagem.
	// Confirmado em produção em 2026-08-13 (ver log com raw_payload).
	Sender string `json:"sender"`
	Data   struct {
		Key struct {
			RemoteJid string `json:"remoteJid"`
			FromMe    bool   `json:"fromMe"`
		} `json:"key"`
		Message struct {
			Conversation string `json:"conversation"`
		} `json:"message"`
		MessageType string `json:"messageType"`
	} `json:"data"`
}

type CreateWhatsAppBotMenuOptionRequest struct {
	BotConfigID     uuid.UUID `json:"bot_config_id"`
	OptionKey       string    `json:"option_key"`
	Label           string    `json:"label"`
	ResponseMessage string    `json:"response_message"`
	OrderIndex      int32     `json:"order_index"`
}

type CreateWhatsAppBotConfigRequest struct {
	WelcomeMessage string                               `json:"welcome_message"`
	MenuOption     []CreateWhatsAppBotMenuOptionRequest `json:"menu_option"`
}

type WhatsappBotMenuOption struct {
	ID              uuid.UUID `json:"id"`
	OptionKey       string    `json:"option_key"`
	Label           string    `json:"label"`
	ResponseMessage string    `json:"response_message"`
	OrderIndex      int32     `json:"order_index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WhatsappBotConfig struct {
	ID                uuid.UUID               `json:"id"`
	InstanceName      string                  `json:"instance_name"`
	WelcomeMessage    string                  `json:"welcome_message"`
	IsActive          bool                    `json:"is_active"`
	CreatedAt         time.Time               `json:"created_at"`
	WhatsappBotOption []WhatsappBotMenuOption `json:"whatsapp_not_option"`
}

type UpdateWhatsAppBotMenuOptionRequest struct {
	OptionKey       string `json:"option_key"`
	Label           string `json:"label"`
	ResponseMessage string `json:"response_message"`
	OrderIndex      int32  `json:"order_index"`
}

type UpdateWhatsAppBotConfigRequest struct {
	WelcomeMessage string                               `json:"welcome_message"`
	IsActive       bool                                 `json:"is_active"`
	MenuOption     []CreateWhatsAppBotMenuOptionRequest `json:"menu_option"`
}
