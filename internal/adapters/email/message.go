package email

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
)

// Attachment arquivo anexado ao e-mail
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

// Message e-mail completo: cópia, resposta para outro endereço, nome de
// exibição do remetente e anexos. O endereço de envio é sempre o do SMTP
// configurado; FromName só troca o nome que o destinatário vê.
type Message struct {
	To          []string
	Cc          []string
	ReplyTo     string
	FromName    string
	Subject     string
	HTML        string
	Attachments []Attachment
}

// SendMessage envia a mensagem como multipart/mixed (HTML + anexos em base64)
func (s *Sender) SendMessage(m Message) error {
	if s.host == "" || s.user == "" || s.password == "" {
		return fmt.Errorf("configuração de SMTP incompleta")
	}
	if len(m.To) == 0 {
		return fmt.Errorf("e-mail sem destinatário")
	}

	from := s.from
	if name := sanitizeHeaderValue(m.FromName); name != "" {
		from = (&mail.Address{Name: name, Address: s.envelopeFrom}).String()
	}

	body, err := buildMessage(from, m)
	if err != nil {
		return err
	}

	recipients := append(append([]string{}, m.To...), m.Cc...)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err := smtp.SendMail(addr, auth, s.envelopeFrom, recipients, body); err != nil {
		return fmt.Errorf("falha ao enviar e-mail via SMTP: %w", err)
	}
	return nil
}

func buildMessage(from string, m Message) ([]byte, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	header := func(key, value string) {
		if value = sanitizeHeaderValue(value); value != "" {
			fmt.Fprintf(&b, "%s: %s\r\n", key, value)
		}
	}

	header("From", from)
	header("To", strings.Join(sanitizeAll(m.To), ", "))
	header("Cc", strings.Join(sanitizeAll(m.Cc), ", "))
	header("Reply-To", m.ReplyTo)
	header("Subject", mime.QEncoding.Encode("UTF-8", sanitizeHeaderValue(m.Subject)))
	header("MIME-Version", "1.0")
	header("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", boundary))
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	writeBase64(&b, []byte(m.HTML))

	for _, a := range m.Attachments {
		contentType := a.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		filename := mime.QEncoding.Encode("UTF-8", sanitizeHeaderValue(a.Filename))
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprintf(&b, "Content-Type: %s; name=\"%s\"\r\n", contentType, filename)
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&b, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename)
		writeBase64(&b, a.Data)
	}

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes(), nil
}

// writeBase64 quebra em linhas de 76 caracteres, como pede o RFC 2045
func writeBase64(b *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString("\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded)
	b.WriteString("\r\n")
}

func sanitizeAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = sanitizeHeaderValue(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func randomBoundary() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "protrack-" + hex.EncodeToString(buf), nil
}
