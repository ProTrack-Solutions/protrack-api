package email

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestBuildMessageMultipart(t *testing.T) {
	pdf := bytes.Repeat([]byte("%PDF-1.4 conteúdo "), 50)
	raw, err := buildMessage(`"Loja X via ProTrack" <noreply@protrack.com>`, Message{
		To:          []string{"fornecedor@ex.com"},
		Cc:          []string{"ana@loja.com", "  "},
		ReplyTo:     "contato@loja.com\r\nBcc: invasor@x.com",
		Subject:     "Pedido de compra nº 12 — Loja X",
		HTML:        "<p>Olá, segue o pedido.</p>",
		Attachments: []Attachment{{Filename: "pedido-12.pdf", ContentType: "application/pdf", Data: pdf}},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("mensagem inválida: %v", err)
	}
	if got := msg.Header.Get("Cc"); got != "ana@loja.com" {
		t.Errorf("Cc: %q", got)
	}
	// Quebra de linha no header não pode injetar outro cabeçalho
	if msg.Header.Get("Bcc") != "" || !strings.HasPrefix(msg.Header.Get("Reply-To"), "contato@loja.com") {
		t.Errorf("injeção de header: Reply-To=%q Bcc=%q", msg.Header.Get("Reply-To"), msg.Header.Get("Bcc"))
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "Pedido de compra nº 12 — Loja X" {
		t.Errorf("assunto: %q", subject)
	}

	_, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(msg.Body, params["boundary"])

	html, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(html.Header.Get("Content-Type"), "text/html") {
		t.Errorf("primeira parte: %q", html.Header.Get("Content-Type"))
	}

	attachment, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if attachment.FileName() != "pedido-12.pdf" {
		t.Errorf("nome do anexo: %q", attachment.FileName())
	}
	// multipart.Reader não decodifica base64: decodifica e compara com o original
	data, _ := io.ReadAll(attachment)
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(data), "\r\n", ""))
	if err != nil || !bytes.Equal(decoded, pdf) {
		t.Errorf("anexo corrompido: %v", err)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Errorf("esperava fim do multipart, veio %v", err)
	}
}
