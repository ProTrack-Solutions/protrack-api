package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
)

type Handler struct {
	service domain.ServiceInterface
}

func NewHandler(service domain.ServiceInterface) *Handler {
	return &Handler{service: service}
}
