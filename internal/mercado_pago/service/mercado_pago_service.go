package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	invoiceHistoryService "github.com/ProTrack-Solutions/protrack-api/internal/invoice_history/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/mercado_pago/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Client struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

type Service struct {
	invoiceHistoryService *invoiceHistoryService.Service
	pool                  *pgxpool.Pool
	cfg                   *config.Config
	Client                Client
}

func NewService(invoiceHistoryService *invoiceHistoryService.Service, pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{
		invoiceHistoryService: invoiceHistoryService,
		pool:                  pool,
		cfg:                   cfg,
		Client: Client{
			accessToken: cfg.MpAccessToken,
			baseURL:     cfg.MpBaseUrl,
			httpClient: &http.Client{
				Timeout: 15 * time.Second,
			},
		},
	}
}

func (s *Service) CreateSubscription(ctx context.Context, req domain.MPPreApprovalRequest) (*domain.MPPreApprovalResponse, error) {
	url := fmt.Sprintf("%s/preapproval", s.Client.baseURL)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %w", err)
	}

	log.Debug().RawJSON("payload_mercado_pago", bodyBytes).Msg("Payload enviado ao MP")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição HTTP: %w", err)
	}

	log.Debug().Str("s.Client.accessToken", s.Client.accessToken).Msg("s.Client.accessToken")

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Client.accessToken))

	resp, err := s.Client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha na comunicação com Mercado Pago: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta do Mercado Pago: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("erro no Mercado Pago (Status %d): %s", resp.StatusCode, string(respBody))
	}

	var mpResp domain.MPPreApprovalResponse
	if err := json.Unmarshal(respBody, &mpResp); err != nil {
		return nil, fmt.Errorf("erro ao desserializar resposta do Mercado Pago: %w", err)
	}

	return &mpResp, nil
}
