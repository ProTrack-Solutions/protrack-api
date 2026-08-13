package handler

import (
	"fmt"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/domain"
)

// ResolveIncomingMessage decide a resposta do bot para uma mensagem recebida.
// Se o texto corresponder à chave de alguma opção do menu, retorna a mensagem
// configurada para essa opção. Caso contrário (primeiro contato ou opção
// inválida), retorna a mensagem de boas-vindas junto com o menu de opções.
func ResolveIncomingMessage(config domain.WhatsappBotConfig, incomingText string) string {
	trimmed := strings.TrimSpace(incomingText)

	for _, option := range config.WhatsappBotOption {
		if strings.EqualFold(strings.TrimSpace(option.OptionKey), trimmed) {
			return option.ResponseMessage
		}
	}

	return buildMenuMessage(config)
}

func buildMenuMessage(config domain.WhatsappBotConfig) string {
	var sb strings.Builder

	sb.WriteString(config.WelcomeMessage)

	if len(config.WhatsappBotOption) > 0 {
		sb.WriteString("\n\n")
		for _, option := range config.WhatsappBotOption {
			sb.WriteString(fmt.Sprintf("%s - %s\n", option.OptionKey, option.Label))
		}
		sb.WriteString("\nDigite o número da opção desejada.")
	}

	return sb.String()
}
