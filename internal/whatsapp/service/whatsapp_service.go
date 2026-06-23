package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Service struct {
	cfg              *config.Config
	companiesService *service.Service
}

func NewService(cfg *config.Config, companiesService *service.Service) *Service {
	return &Service{
		cfg:              cfg,
		companiesService: companiesService,
	}
}

func (s *Service) CreateInstance(ctx context.Context, companyID uuid.UUID) (string, error) {
	url := fmt.Sprintf("%s/instance/create", s.cfg.EvolutionApiUrl)

	var Integration string

	company, err := s.companiesService.GetCompanyByID(ctx, companyID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve company: %w", err)
	}

	if company.DocumentType == "CPF" {
		Integration = "WHATSAPP-BAILEYS"
	} else {
		Integration = "WHATSAPP-BUSINESS"
	}

	instanceName := fmt.Sprintf("%s-%s", company.Name, companyID.String())

	payload := map[string]any{
		"instanceName": instanceName,
		"integration":  Integration,
		"token":        companyID.String(),
		"qrcode":       true,
	}

	log.Info().Str("url", url).Interface("payload", payload).Msg("Enviando solicitação para criar instância no Evolution API")

	jsonPayload, _ := json.Marshal(payload)

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apikey", s.cfg.EvolutionKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create instance: %s", body)
	}

	var result domain.EvolutionCreateResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	connectUrl := fmt.Sprintf("%s/instance/connect/%s", s.cfg.EvolutionApiUrl, instanceName)
	var qrCode string

	for i := 0; i < 5; i++ {
		time.Sleep(5 * time.Second)

		reqConnect, err := http.NewRequestWithContext(ctx, "GET", connectUrl, nil)
		if err != nil {
			return "", err
		}
		reqConnect.Header.Set("apikey", s.cfg.EvolutionKey)

		resConnect, err := client.Do(reqConnect)
		if err != nil {
			log.Warn().Err(err).Msg("Erro ao conectar à Evolution API, tentando novamente...")
			continue
		}
		defer resConnect.Body.Close()

		bodyConnect, _ := io.ReadAll(resConnect.Body)
		if resConnect.StatusCode != http.StatusOK {
			log.Warn().Str("response", string(bodyConnect)).Msg("Resposta inesperada da Evolution API, tentando novamente...")
			continue
		}

		var resultConnect domain.EvolutionConnectResponse
		if err := json.Unmarshal(bodyConnect, &resultConnect); err != nil {
			log.Warn().Err(err).Msg("Erro ao decodificar resposta da Evolution API, tentando novamente...")
			continue
		}

		qrCode = resultConnect.Code
		break
	}

	return qrCode, nil
}
