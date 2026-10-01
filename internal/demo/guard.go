package demo

import (
	"net/http"
	"strings"
)

// blockedPrefixes são os grupos de rotas em que a empresa demo só pode ler.
// Tudo que mexe em cobrança, credenciais, WhatsApp ou nos dados da própria
// conta fica bloqueado; o resto (vendas, clientes, contas...) é liberado e
// volta ao normal no reset diário.
var blockedPrefixes = []string{
	"/user",                         // perfil, senha e gestão de usuários
	"/companies",                    // dados e status da empresa
	"/subscription",                 // plano, forma de pagamento e cancelamento
	"/subscription-management",      // idem (tela de assinatura)
	"/subscription-payment-methods", // cartões da assinatura
	"/meta-whatsapp",                // envio de mensagens e configuração
	"/company-settings",             // impede religar o WhatsApp da demo
}

// IsBlockedAction indica se a rota (padrão do gin, ex.: "/api/v1/user/:id")
// é uma escrita proibida para a empresa demo. Leituras (GET) sempre passam.
func IsBlockedAction(method, routePath string) bool {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return false
	}

	path := strings.TrimPrefix(routePath, "/api/v1")
	for _, prefix := range blockedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
