// Package client fala com a protrack-fiscal-api (API a API). O ERP é uma conta
// integradora: cada empresa é um emitente dessa conta, e a chave usada depende
// do ambiente da empresa (ptf_test_ = homologação, ptf_live_ = produção).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNotConfigured: o .env não tem a URL ou a chave do ambiente pedido.
var ErrNotConfigured = errors.New("integração fiscal não configurada no servidor")

// APIError é uma resposta de erro da API fiscal.
type APIError struct {
	Status  int
	Code    string // campo "error" (às vezes um código como issuer_deactivated, às vezes a mensagem)
	Message string // campo "message", quando vem
	Hint    string // 502/504 da SEFAZ
	Body    []byte
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Code
	}
	if msg == "" {
		msg = strings.TrimSpace(string(e.Body))
	}
	return fmt.Sprintf("api fiscal respondeu %d: %s", e.Status, msg)
}

// UserMessage é o texto mais útil para mostrar ao lojista.
func (e *APIError) UserMessage() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// SefazUnavailable: a SEFAZ não respondeu (502/504).
func (e *APIError) SefazUnavailable() bool {
	return e.Status == http.StatusBadGateway || e.Status == http.StatusGatewayTimeout
}

// AsAPIError devolve o *APIError dentro de err, se houver.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := errors.As(err, &apiErr)
	return apiErr, ok
}

type Config struct {
	BaseURL string
	KeyTest string
	KeyLive string
	// Timeout de cada chamada. A emissão espera a SEFAZ: a API fiscal tem o próprio
	// timeout e devolve 202 (PROCESSING) quando não deu tempo.
	Timeout time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Configured diz se o ambiente tem URL e chave.
func (c *Client) Configured(env Environment) bool {
	return c.cfg.BaseURL != "" && c.key(env) != ""
}

func (c *Client) key(env Environment) string {
	if env == EnvLive {
		return c.cfg.KeyLive
	}
	return c.cfg.KeyTest
}

type request struct {
	method      string
	path        string
	body        io.Reader
	contentType string
	headers     map[string]string
}

func (c *Client) do(ctx context.Context, env Environment, r request) (*http.Response, []byte, error) {
	if !c.Configured(env) {
		return nil, nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, r.method, c.cfg.BaseURL+r.path, r.body)
	if err != nil {
		return nil, nil, fmt.Errorf("montar requisição fiscal: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key(env))
	req.Header.Set("Accept", "application/json")
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("chamar api fiscal: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("ler resposta da api fiscal: %w", err)
	}
	return resp, body, nil
}

func parseAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{Status: status, Body: body}
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	}
	if json.Unmarshal(body, &payload) == nil {
		apiErr.Code = payload.Error
		apiErr.Message = payload.Message
		apiErr.Hint = payload.Hint
	}
	return apiErr
}

func jsonBody(v any) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializar corpo fiscal: %w", err)
	}
	return bytes.NewReader(raw), nil
}

// callJSON faz a chamada e decodifica 2xx em out.
func (c *Client) callJSON(ctx context.Context, env Environment, method, path string, in, out any) error {
	body, err := jsonBody(in)
	if err != nil {
		return err
	}
	r := request{method: method, path: path, body: body}
	if in != nil {
		r.contentType = "application/json"
	}
	resp, raw, err := c.do(ctx, env, r)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decodificar resposta fiscal: %w", err)
	}
	return nil
}

func issuerPath(issuerID, suffix string) string {
	return "/issuers/" + url.PathEscape(issuerID) + suffix
}

// ---- Emitentes ----

// CreateIssuer cadastra o CNPJ nos dois ambientes (a chave só decide quem pede).
func (c *Client) CreateIssuer(ctx context.Context, env Environment, in CreateIssuerRequest) (IssuerPair, error) {
	var out struct {
		Issuers IssuerPair `json:"issuers"`
	}
	err := c.callJSON(ctx, env, http.MethodPost, "/issuers", in, &out)
	return out.Issuers, err
}

func (c *Client) ListIssuers(ctx context.Context, env Environment) ([]Issuer, error) {
	var out struct {
		Issuers []Issuer `json:"issuers"`
	}
	err := c.callJSON(ctx, env, http.MethodGet, "/issuers", nil, &out)
	return out.Issuers, err
}

// UploadCertificate repassa o A1 (.pfx) direto para a API fiscal; o ERP não guarda o arquivo.
func (c *Client) UploadCertificate(ctx context.Context, env Environment, issuerID, filename string, pfx []byte, password string) (CertificateUploaded, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("password", password); err != nil {
		return CertificateUploaded{}, err
	}
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return CertificateUploaded{}, err
	}
	if _, err := part.Write(pfx); err != nil {
		return CertificateUploaded{}, err
	}
	if err := w.Close(); err != nil {
		return CertificateUploaded{}, err
	}
	resp, raw, err := c.do(ctx, env, request{
		method:      http.MethodPost,
		path:        issuerPath(issuerID, "/certificate"),
		body:        &buf,
		contentType: w.FormDataContentType(),
	})
	if err != nil {
		return CertificateUploaded{}, err
	}
	if resp.StatusCode >= 300 {
		return CertificateUploaded{}, parseAPIError(resp.StatusCode, raw)
	}
	var out CertificateUploaded
	if err := json.Unmarshal(raw, &out); err != nil {
		return CertificateUploaded{}, fmt.Errorf("decodificar certificado: %w", err)
	}
	return out, nil
}

func (c *Client) GetCertificate(ctx context.Context, env Environment, issuerID string) (CertificateStatus, error) {
	var out CertificateStatus
	err := c.callJSON(ctx, env, http.MethodGet, issuerPath(issuerID, "/certificate"), nil, &out)
	return out, err
}

func (c *Client) PutFiscalProfile(ctx context.Context, env Environment, issuerID string, in FiscalProfileRequest) (FiscalProfile, error) {
	var out FiscalProfile
	err := c.callJSON(ctx, env, http.MethodPut, issuerPath(issuerID, "/fiscal-profile"), in, &out)
	return out, err
}

func (c *Client) GetFiscalProfile(ctx context.Context, env Environment, issuerID string) (FiscalProfile, error) {
	var out FiscalProfile
	err := c.callJSON(ctx, env, http.MethodGet, issuerPath(issuerID, "/fiscal-profile"), nil, &out)
	return out, err
}

// SefazStatus consulta o serviço da NFC-e (modelo 65) com o A1 do emitente.
func (c *Client) SefazStatus(ctx context.Context, env Environment, issuerID string) (SefazStatus, error) {
	var out SefazStatus
	err := c.callJSON(ctx, env, http.MethodGet, issuerPath(issuerID, "/sefaz-status?model=65"), nil, &out)
	return out, err
}

// CopyFromHomologation copia A1 e perfil fiscal para o emitente de produção (chave live).
func (c *Client) CopyFromHomologation(ctx context.Context, liveIssuerID string, csc CSC) (CopyFromHomologationResult, error) {
	var out CopyFromHomologationResult
	err := c.callJSON(ctx, EnvLive, http.MethodPost, issuerPath(liveIssuerID, "/copy-from-homologation"),
		map[string]any{"csc": csc}, &out)
	return out, err
}

// ---- NFC-e ----

// EmitNfce emite a nota. Rejeição da SEFAZ (422 com a nota) volta como Document com
// status REJECTED, sem erro; 202 volta PROCESSING. Erro só quando não há nota.
func (c *Client) EmitNfce(ctx context.Context, env Environment, issuerID, idempotencyKey string, in NfceRequest) (Document, error) {
	body, err := jsonBody(in)
	if err != nil {
		return Document{}, err
	}
	resp, raw, err := c.do(ctx, env, request{
		method:      http.MethodPost,
		path:        issuerPath(issuerID, "/nfce"),
		body:        body,
		contentType: "application/json",
		headers:     map[string]string{"Idempotency-Key": idempotencyKey},
	})
	if err != nil {
		return Document{}, err
	}
	if resp.StatusCode < 300 || resp.StatusCode == http.StatusUnprocessableEntity {
		var doc Document
		if json.Unmarshal(raw, &doc) == nil && doc.ID != "" && doc.Status != "" {
			return doc, nil
		}
	}
	return Document{}, parseAPIError(resp.StatusCode, raw)
}

func (c *Client) GetNfce(ctx context.Context, env Environment, documentID string) (Document, error) {
	var out Document
	err := c.callJSON(ctx, env, http.MethodGet, "/nfce/"+url.PathEscape(documentID), nil, &out)
	return out, err
}

// Danfce devolve o PDF de 80 mm (só nota autorizada ou em contingência).
func (c *Client) Danfce(ctx context.Context, env Environment, documentID string) ([]byte, error) {
	return c.raw(ctx, env, "/nfce/"+url.PathEscape(documentID)+"/danfce")
}

// Xml devolve o XML autorizado.
func (c *Client) Xml(ctx context.Context, env Environment, documentID string) ([]byte, error) {
	return c.raw(ctx, env, "/nfce/"+url.PathEscape(documentID)+"/xml")
}

func (c *Client) raw(ctx context.Context, env Environment, path string) ([]byte, error) {
	resp, raw, err := c.do(ctx, env, request{method: http.MethodGet, path: path})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, parseAPIError(resp.StatusCode, raw)
	}
	return raw, nil
}

// CancelNfce pede o cancelamento. Rejeição do evento (422) volta no resultado.
func (c *Client) CancelNfce(ctx context.Context, env Environment, documentID, reason string) (CancellationResult, error) {
	body, err := jsonBody(map[string]string{"reason": reason})
	if err != nil {
		return CancellationResult{}, err
	}
	resp, raw, err := c.do(ctx, env, request{
		method:      http.MethodPost,
		path:        "/nfce/" + url.PathEscape(documentID) + "/cancel",
		body:        body,
		contentType: "application/json",
	})
	if err != nil {
		return CancellationResult{}, err
	}
	if resp.StatusCode < 300 || resp.StatusCode == http.StatusUnprocessableEntity {
		var out CancellationResult
		if json.Unmarshal(raw, &out) == nil && out.Event.ID != "" {
			return out, nil
		}
	}
	return CancellationResult{}, parseAPIError(resp.StatusCode, raw)
}
