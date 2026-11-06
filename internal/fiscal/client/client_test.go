package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(Config{BaseURL: srv.URL + "/", KeyTest: "ptf_test_x", KeyLive: "ptf_live_y"})
}

func TestNotConfigured(t *testing.T) {
	c := New(Config{BaseURL: "http://x", KeyTest: "ptf_test_x"})
	if !c.Configured(EnvTest) || c.Configured(EnvLive) {
		t.Fatal("configured por ambiente errado")
	}
	if _, err := c.GetDocument(context.Background(), EnvLive, ModelNfce, "id"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("esperava ErrNotConfigured, veio %v", err)
	}
}

func TestEmitNfceSendsKeyAndIdempotency(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/issuers/iss-1/nfce" || r.Method != http.MethodPost {
			t.Errorf("rota errada: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ptf_live_y" {
			t.Errorf("chave errada: %q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "sale-1:1" {
			t.Errorf("idempotency errada: %q", got)
		}
		var body NfceRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Items) != 1 {
			t.Errorf("corpo errado: %v %+v", err, body)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"doc-1","model":65,"number":7,"status":"AUTHORIZED","accessKey":"3126","environment":1,"emissionType":1}`)
	})
	doc, err := c.EmitNfce(context.Background(), EnvLive, "iss-1", "sale-1:1", NfceRequest{Items: []NfceItem{{Code: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "doc-1" || doc.Status != StatusAuthorized || doc.Number != 7 {
		t.Fatalf("documento errado: %+v", doc)
	}
}

func TestEmitNfceRejectedIsNotAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"id":"doc-2","status":"REJECTED","statusCode":"778","rejectReason":"NCM inexistente"}`)
	})
	doc, err := c.EmitNfce(context.Background(), EnvTest, "iss", "k", NfceRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status != StatusRejected || doc.RejectReason == nil || *doc.RejectReason != "NCM inexistente" {
		t.Fatalf("rejeição errada: %+v", doc)
	}
}

func TestEmitNfceValidationError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"error":"perfil fiscal do emitente não configurado"}`)
	})
	_, err := c.EmitNfce(context.Background(), EnvTest, "iss", "k", NfceRequest{})
	apiErr, ok := AsAPIError(err)
	if !ok || apiErr.Status != 422 || apiErr.UserMessage() != "perfil fiscal do emitente não configurado" {
		t.Fatalf("erro errado: %v", err)
	}
}

func TestSefazUnavailable(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, `{"error":"SEFAZ não respondeu","kind":"timeout","hint":"tente de novo"}`)
	})
	_, err := c.EmitNfce(context.Background(), EnvTest, "iss", "k", NfceRequest{})
	apiErr, ok := AsAPIError(err)
	if !ok || !apiErr.SefazUnavailable() || apiErr.Hint != "tente de novo" {
		t.Fatalf("erro errado: %v", err)
	}
}

func TestUploadCertificateMultipart(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("password") != "segredo" {
			t.Errorf("senha não chegou")
		}
		f, h, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(f)
		if h.Filename != "a1.pfx" || string(data) != "PFX" {
			t.Errorf("arquivo errado: %s %q", h.Filename, data)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"subjectCN":"LOJA:12345678000199","notBefore":"2026-01-01T00:00:00Z","notAfter":"2027-01-01T00:00:00Z","serialNumber":"1","cnpj":"12345678000199"}`)
	})
	out, err := c.UploadCertificate(context.Background(), EnvTest, "iss", "a1.pfx", []byte("PFX"), "segredo")
	if err != nil {
		t.Fatal(err)
	}
	if out.NotAfter.Year() != 2027 {
		t.Fatalf("vencimento errado: %v", out.NotAfter)
	}
}

func TestCancelRejectedComesBack(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cancel") {
			t.Errorf("rota errada %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"event":{"id":"ev","type":"CANCELLATION","status":"REJECTED","rejectReason":"prazo"},"documentStatus":"AUTHORIZED","replayed":false}`)
	})
	out, err := c.CancelDocument(context.Background(), EnvTest, ModelNfce, "doc", "Venda desfeita a pedido do cliente")
	if err != nil {
		t.Fatal(err)
	}
	if out.Event.Status != "REJECTED" || out.DocumentStatus != StatusAuthorized {
		t.Fatalf("resultado errado: %+v", out)
	}
}

func sign(secret string, ts int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, body)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyWebhookSignature(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := `{"id":"1","type":"nfce.authorized"}`
	header := sign("whsec_abc", now.Unix(), body)

	if !VerifyWebhookSignature("whsec_abc", header, []byte(body), now, DefaultSignatureTolerance) {
		t.Fatal("assinatura válida recusada")
	}
	cases := map[string]struct {
		secret, header, body string
		now                  time.Time
	}{
		"segredo errado": {"whsec_x", header, body, now},
		"corpo alterado": {"whsec_abc", header, body + " ", now},
		"velho demais":   {"whsec_abc", header, body, now.Add(6 * time.Minute)},
		"sem header":     {"whsec_abc", "", body, now},
		"lixo":           {"whsec_abc", "t=abc,v1=zz", body, now},
		"sem segredo":    {"", header, body, now},
	}
	for name, tc := range cases {
		if VerifyWebhookSignature(tc.secret, tc.header, []byte(tc.body), tc.now, DefaultSignatureTolerance) {
			t.Errorf("%s: deveria recusar", name)
		}
	}
}
