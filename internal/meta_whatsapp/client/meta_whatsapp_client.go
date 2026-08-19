package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/metagraph"
)

type Client struct {
	graph *metagraph.Client
}

func NewClient(graph *metagraph.Client) *Client {
	return &Client{graph: graph}
}

type templatePayload struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Template         templateBody `json:"template"`
}

type templateBody struct {
	Name       string              `json:"name"`
	Language   templateLanguage    `json:"language"`
	Components []templateComponent `json:"components,omitempty"`
}

type templateLanguage struct {
	Code string `json:"code"`
}

type templateComponent struct {
	Type       string              `json:"type"`
	Parameters []templateParameter `json:"parameters"`
}

type templateParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type sendMessageResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// orderedParameters converte um map de variáveis posicionais (chaves "1", "2", "3"...)
// numa slice ordenada de templateParameter, respeitando a posição numérica —
// resolve o problema de map[string]string não ter ordem de iteração garantida.
func orderedParameters(variables map[string]string) []templateParameter {
	if len(variables) == 0 {
		return nil
	}

	keys := make([]string, 0, len(variables))
	for k := range variables {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		numI, errI := strconv.Atoi(keys[i])
		numJ, errJ := strconv.Atoi(keys[j])
		if errI != nil || errJ != nil {
			return keys[i] < keys[j] // fallback: ordem alfabética se a chave não for numérica
		}
		return numI < numJ
	})

	parameters := make([]templateParameter, 0, len(keys))
	for _, k := range keys {
		parameters = append(parameters, templateParameter{Type: "text", Text: variables[k]})
	}

	return parameters
}

func (c *Client) SendTemplateMessage(ctx context.Context, phoneNumberID, accessToken string, req domain.SendMessageRequest) (string, error) {
	payload := templatePayload{
		MessagingProduct: "whatsapp",
		To:               req.RecipientPhone,
		Type:             "template",
		Template: templateBody{
			Name:     req.TemplateName,
			Language: templateLanguage{Code: req.LanguageCode},
		},
	}

	var components []templateComponent

	if headerParams := orderedParameters(req.HeaderVariables); len(headerParams) > 0 {
		components = append(components, templateComponent{Type: "header", Parameters: headerParams})
	}

	if bodyParams := orderedParameters(req.Variables); len(bodyParams) > 0 {
		components = append(components, templateComponent{Type: "body", Parameters: bodyParams})
	}

	if len(components) > 0 {
		payload.Template.Components = components
	}

	respBody, err := c.graph.Do(ctx, "POST", phoneNumberID+"/messages", accessToken, payload)
	if err != nil {
		return "", fmt.Errorf("erro ao enviar mensagem de template: %w", err)
	}

	var response sendMessageResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta da Graph API: %w", err)
	}

	if len(response.Messages) == 0 {
		return "", fmt.Errorf("resposta da Graph API sem mensagens: %s", string(respBody))
	}

	return response.Messages[0].ID, nil
}
