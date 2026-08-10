package email

import (
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/rs/zerolog/log"
)

type Sender struct {
	host     string
	port     string
	user     string
	password string
	// from é o header "From:" completo, pode conter nome de exibição
	// (ex: "ProTrack <noreply@ptsolutionss.com>").
	from string
	// envelopeFrom é o endereço puro, sem nome — é o que o comando SMTP
	// MAIL FROM exige. Extraído de "from" no NewSender.
	envelopeFrom string
}

func NewSender(cfg *config.Config) *Sender {
	rawFrom := sanitizeHeaderValue(cfg.SMTPFrom)

	envelopeFrom := rawFrom
	if addr, err := mail.ParseAddress(rawFrom); err == nil {
		envelopeFrom = addr.Address
	}
	return &Sender{
		host:         cfg.SMTPHost,
		port:         cfg.SMTPPort,
		user:         cfg.SMTPUser,
		password:     cfg.SMTPPassword,
		from:         cfg.SMTPFrom,
		envelopeFrom: envelopeFrom,
	}
}

func sanitizeHeaderValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return strings.TrimSpace(v)
}

// Send envia um e-mail HTML simples. Roda de forma síncrona — quem chama
// (o consumer) já está fora do caminho crítico da requisição HTTP.
func (s *Sender) Send(to, subject, htmlBody string) error {
	if s.host == "" || s.user == "" || s.password == "" {
		return fmt.Errorf("configuração de SMTP incompleta")
	}

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.from, to, encodedSubject, htmlBody,
	)

	log.Info().
		Str("header_from", s.from).
		Str("envelope_from", s.envelopeFrom).
		Str("to", to).
		Str("subject_encoded", encodedSubject).
		Msg("DEBUG: valores usados no envio SMTP")

	if err := smtp.SendMail(addr, auth, s.envelopeFrom, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("falha ao enviar e-mail via SMTP: %w", err)
	}

	return nil
}

// SendPasswordReset monta e envia o e-mail de recuperação de senha.
func (s *Sender) SendPasswordReset(to, name, resetURL string) error {
	subject := "Recuperação de senha — ProTrack"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 480px; margin: 0 auto;">
			<h2>Recuperação de senha</h2>
			<p>Olá, %s.</p>
			<p>Recebemos uma solicitação para redefinir sua senha no ProTrack. Clique no botão abaixo para continuar:</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="background:#111827;color:#fff;padding:12px 20px;border-radius:6px;text-decoration:none;">
					Redefinir senha
				</a>
			</p>
			<p>Este link expira em 30 minutos. Se você não solicitou essa alteração, pode ignorar este e-mail com segurança.</p>
		</div>
	`, name, resetURL)

	if err := s.Send(to, subject, body); err != nil {
		log.Error().Err(err).Str("to", to).Msg("Falha ao enviar e-mail de recuperação de senha")
		return err
	}

	return nil
}

// SendPasswordChanged notifica o usuário que a senha foi alterada
// (notificação de segurança, disparada após um reset bem-sucedido).
func (s *Sender) SendPasswordChanged(to, name string) error {
	subject := "Sua senha foi alterada — ProTrack"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 480px; margin: 0 auto;">
			<h2>Senha alterada</h2>
			<p>Olá, %s.</p>
			<p>Sua senha no ProTrack foi alterada com sucesso. Se não foi você, entre em contato com o suporte imediatamente.</p>
		</div>
	`, name)

	if err := s.Send(to, subject, body); err != nil {
		log.Error().Err(err).Str("to", to).Msg("Falha ao enviar e-mail de confirmação de troca de senha")
		return err
	}

	return nil
}
