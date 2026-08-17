package client

import (
	"context"
	"errors"

	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/metagraph"
)

type Client struct {
	graph *metagraph.Client
}

func NewClient(graph *metagraph.Client) *Client {
	return &Client{graph: graph}
}

func (c *Client) SendTemplateMessage(ctx context.Context, phoneNumberID, accessToken string, req domain.SendMessageRequest) (metaMessageID string, err error) {
	// TODO: implementar a chamada real à Graph API (POST /{phoneNumberID}/messages) usando c.graph.
	return "", errors.New("SendTemplateMessage not implemented")
}
