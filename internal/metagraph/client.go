package metagraph

import "net/http"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient, baseURL: "https://graph.facebook.com/v25.0"}
}
