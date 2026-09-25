package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/auth/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/service"
	discordDomain "github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	userDomain "github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	userService "github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var errLoginRateLimited = errors.New("muitas tentativas de login, aguarde 15 minutos e tente novamente")

// loginErrorResponse traduz o erro do service no status HTTP, no código
// estável (para o front tratar sem depender do texto) e na mensagem exibida.
func loginErrorResponse(err error) (int, string, string) {
	switch {
	case errors.Is(err, userService.ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error()
	case errors.Is(err, service.ErrInvalidAud):
		return http.StatusBadRequest, "INVALID_AUD", err.Error()
	case errors.Is(err, service.ErrSubscriptionNotFound):
		return http.StatusForbidden, "SUBSCRIPTION_NOT_FOUND", err.Error()
	case errors.Is(err, service.ErrSubscriptionCanceled):
		return http.StatusForbidden, "SUBSCRIPTION_CANCELED", err.Error()
	case errors.Is(err, service.ErrSubscriptionPaused):
		return http.StatusForbidden, "SUBSCRIPTION_PAUSED", err.Error()
	case errors.Is(err, service.ErrSubscriptionExpired):
		return http.StatusForbidden, "SUBSCRIPTION_EXPIRED", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "erro interno ao processar o login, tente novamente"
	}
}

// loginBindErrorMessage transforma os erros de validação do gin em uma
// mensagem legível em vez do texto cru do validator.
func loginBindErrorMessage(err error) string {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return "corpo da requisição inválido, envie um JSON com email, password e aud"
	}

	msgs := make([]string, 0, len(validationErrs))
	for _, fe := range validationErrs {
		switch {
		case fe.Field() == "Email" && fe.Tag() == "email":
			msgs = append(msgs, "informe um e-mail válido")
		case fe.Field() == "Email":
			msgs = append(msgs, "o e-mail é obrigatório")
		case fe.Field() == "Password":
			msgs = append(msgs, "a senha é obrigatória")
		case fe.Field() == "Aud":
			msgs = append(msgs, "o campo aud é obrigatório")
		default:
			msgs = append(msgs, fmt.Sprintf("campo %s inválido", fe.Field()))
		}
	}

	return strings.Join(msgs, "; ")
}

// logLoginAttempt envia para o Discord cada tentativa de login, com sucesso ou
// falha. O envio é assíncrono (DiscordLogger.Send), então não atrasa a resposta.
func (h *Handler) logLoginAttempt(c *gin.Context, req domain.LoginRequest, user *userDomain.UserResponse, loginErr error) {
	var b strings.Builder

	if user != nil {
		fmt.Fprintf(&b, "**Usuário:** %s (%s)\n", user.Name, user.Email)
		fmt.Fprintf(&b, "**User ID:** %s\n", user.ID)
		fmt.Fprintf(&b, "**Empresa ID:** %s\n", user.CompanyID)
		fmt.Fprintf(&b, "**Role:** %s\n", user.Role)
	} else {
		fmt.Fprintf(&b, "**E-mail informado:** %s\n", req.Email)
	}

	fmt.Fprintf(&b, "**Aplicação (aud):** %s\n", req.Aud)
	fmt.Fprintf(&b, "**IP:** %s\n", c.ClientIP())
	fmt.Fprintf(&b, "**User-Agent:** %s", c.Request.UserAgent())

	if loginErr == nil {
		h.discordLog.Send(discordDomain.LevelInfo, "✅ Login realizado", b.String())
		return
	}

	_, code, _ := loginErrorResponse(loginErr)
	if errors.Is(loginErr, errLoginRateLimited) {
		code = "TOO_MANY_ATTEMPTS"
	}
	fmt.Fprintf(&b, "\n**Motivo:** %s — %s", code, loginErr.Error())

	h.discordLog.Send(discordDomain.LevelWarning, "⚠️ Falha no login", b.String())
}
