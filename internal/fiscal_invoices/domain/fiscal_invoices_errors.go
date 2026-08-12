package domain

import "errors"

var (
	ErrCertificateNotFound       = errors.New("certificado digital não encontrado para esta empresa")
	ErrCertificateExpired        = errors.New("certificado digital expirado")
	ErrInvoiceAlreadyExists      = errors.New("já existe um documento fiscal deste tipo para esta venda")
	ErrInvoiceNotFound           = errors.New("documento fiscal não encontrado")
	ErrInvoiceNotAuthorized      = errors.New("documento fiscal ainda não está autorizado")
	ErrCancellationWindowExpired = errors.New("janela de cancelamento de 24 horas expirada")
	ErrMissingFiscalFields       = errors.New("campos fiscais obrigatórios ausentes")
	ErrInvalidJustificativa      = errors.New("justificativa deve ter no mínimo 15 caracteres")
)
