package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/metagraph"
)

type Client struct {
	graph *metagraph.Client
}

func NewClient(graph *metagraph.Client) *Client {
	return &Client{graph: graph}
}

// templatePayload espelha o formato que a Graph API espera pro envio de template.
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

	if len(req.Variables) > 0 {
		var parameters []templateParameter
		for _, value := range req.Variables {
			parameters = append(parameters, templateParameter{Type: "text", Text: value})
		}
		payload.Template.Components = []templateComponent{
			{Type: "body", Parameters: parameters},
		}
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
