package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/client"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/metagraph"
)

// redirectTransport reescreve o host/scheme de toda requisição pra apontar
// pro httptest.Server, já que metagraph.Client tem a baseURL da Graph API
// fixa no código. Preserva path/query/body — só troca o destino de rede.
type redirectTransport struct {
	target *url.URL
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

// newTestClient sobe um httptest.Server controlado pelo chamador e devolve um
// client.Client cujas requisições são redirecionadas pra ele.
func newTestClient(server *httptest.Server) *client.Client {
	target, _ := url.Parse(server.URL)
	httpClient := &http.Client{Transport: &redirectTransport{target: target}}
	graph := metagraph.NewClient(httpClient)
	return client.NewClient(graph)
}

// ---------------------------------------------------------------------------
// SendTemplateMessage
// ---------------------------------------------------------------------------

func TestSendTemplateMessage_Success(t *testing.T) {
	var capturedPath, capturedAuth string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "wamid.HBgL"}},
		})
	}))
	defer server.Close()

	c := newTestClient(server)

	messageID, err := c.SendTemplateMessage(t.Context(), "1234567890", "meu-token", domain.SendMessageRequest{
		TemplateName:    "order_confirmation",
		LanguageCode:    "pt_BR",
		RecipientPhone:  "+5511988887777",
		HeaderVariables: map[string]string{"1": "João"},
		Variables:       map[string]string{"1": "Pedido #42", "2": "R$ 99,90"},
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if messageID != "wamid.HBgL" {
		t.Errorf("messageID incorreto: %s", messageID)
	}
	if capturedPath != "/v25.0/1234567890/messages" {
		t.Errorf("path incorreto: %s", capturedPath)
	}
	if capturedAuth != "Bearer meu-token" {
		t.Errorf("Authorization incorreto: %s", capturedAuth)
	}

	if capturedBody["to"] != "+5511988887777" {
		t.Errorf("destinatário incorreto: %v", capturedBody["to"])
	}

	template, ok := capturedBody["template"].(map[string]any)
	if !ok {
		t.Fatal("payload sem campo 'template'")
	}
	if template["name"] != "order_confirmation" {
		t.Errorf("nome do template incorreto: %v", template["name"])
	}

	components, ok := template["components"].([]any)
	if !ok || len(components) != 2 {
		t.Fatalf("esperava 2 componentes (header + body), obteve: %v", template["components"])
	}

	header := components[0].(map[string]any)
	if header["type"] != "header" {
		t.Errorf("primeiro componente deveria ser 'header', obteve: %v", header["type"])
	}

	body := components[1].(map[string]any)
	if body["type"] != "body" {
		t.Errorf("segundo componente deveria ser 'body', obteve: %v", body["type"])
	}
	bodyParams := body["parameters"].([]any)
	if len(bodyParams) != 2 {
		t.Fatalf("esperava 2 parâmetros no body, obteve %d", len(bodyParams))
	}
	// orderedParameters deve preservar a ordem posicional "1", "2" mesmo vindo de um map.
	if bodyParams[0].(map[string]any)["text"] != "Pedido #42" {
		t.Errorf("parâmetro 1 do body incorreto: %v", bodyParams[0])
	}
	if bodyParams[1].(map[string]any)["text"] != "R$ 99,90" {
		t.Errorf("parâmetro 2 do body incorreto: %v", bodyParams[1])
	}
}

func TestSendTemplateMessage_NoVariables_OmitsComponents(t *testing.T) {
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "wamid.simple"}},
		})
	}))
	defer server.Close()

	c := newTestClient(server)

	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "meu-token", domain.SendMessageRequest{
		TemplateName:   "simple_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}

	template := capturedBody["template"].(map[string]any)
	if _, exists := template["components"]; exists {
		t.Errorf("esperava 'components' ausente do payload, obteve: %v", template["components"])
	}
}

func TestSendTemplateMessage_GraphAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Parâmetro inválido", "type": "OAuthException", "code": 100},
		})
	}))
	defer server.Close()

	c := newTestClient(server)

	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "token-invalido", domain.SendMessageRequest{
		TemplateName:   "meu_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	})

	if err == nil {
		t.Fatal("esperava erro da Graph API")
	}
}

func TestSendTemplateMessage_MalformedResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messages": nao-e-json`))
	}))
	defer server.Close()

	c := newTestClient(server)

	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "token", domain.SendMessageRequest{
		TemplateName:   "meu_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	})

	if err == nil {
		t.Fatal("esperava erro ao decodificar resposta malformada")
	}
}

func TestSendTemplateMessage_EmptyMessagesArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []map[string]string{}})
	}))
	defer server.Close()

	c := newTestClient(server)

	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "token", domain.SendMessageRequest{
		TemplateName:   "meu_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	})

	if err == nil {
		t.Fatal("esperava erro por resposta sem mensagens")
	}
}

func TestSendTemplateMessage_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	target, _ := url.Parse(server.URL)
	server.Close() // fecha o servidor pra forçar erro de conexão

	httpClient := &http.Client{Transport: &redirectTransport{target: target}}
	graph := metagraph.NewClient(httpClient)
	c := client.NewClient(graph)

	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "token", domain.SendMessageRequest{
		TemplateName:   "meu_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	})

	if err == nil {
		t.Fatal("esperava erro de transporte")
	}
}

// ---------------------------------------------------------------------------
// Ordenação de parâmetros posicionais (via efeito observável no payload)
// ---------------------------------------------------------------------------

func TestSendTemplateMessage_OrdersNumericVariableKeysNumerically(t *testing.T) {
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "wamid.order"}},
		})
	}))
	defer server.Close()

	c := newTestClient(server)

	// Chaves fora de ordem lexicográfica ("10" viria antes de "2" alfabeticamente).
	_, err := c.SendTemplateMessage(t.Context(), "1234567890", "token", domain.SendMessageRequest{
		TemplateName:   "meu_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
		Variables: map[string]string{
			"10": "dez",
			"2":  "dois",
			"1":  "um",
		},
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}

	template := capturedBody["template"].(map[string]any)
	components := template["components"].([]any)
	body := components[0].(map[string]any)
	params := body["parameters"].([]any)

	if len(params) != 3 {
		t.Fatalf("esperava 3 parâmetros, obteve %d", len(params))
	}
	expected := []string{"um", "dois", "dez"}
	for i, exp := range expected {
		if params[i].(map[string]any)["text"] != exp {
			t.Errorf("parâmetro %d incorreto: esperava %q, obteve %v", i, exp, params[i].(map[string]any)["text"])
		}
	}
}
