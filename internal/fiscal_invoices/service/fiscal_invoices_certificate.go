package service

import (
	"context"
	"fmt"
	"time"

	certCrypto "github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/domain"
	"github.com/google/uuid"
)

// UploadCertificate criptografa o certificado A1 e a senha antes de persistir
// localmente (defesa em profundidade), envia o certificado original (não
// criptografado) ao provedor fiscal, que precisa dele para assinar as notas,
// e então registra/atualiza a empresa emitente e o webhook de callback no
// provedor — sem isso, a emissão de notas falha no PlugNotas (empresa nunca
// cadastrada) e o status delas nunca é atualizado (webhook nunca configurado).
// UNIQUE(company_id) em company_certificates: se já existir um certificado
// para a empresa, substitui (UPDATE) em vez de duplicar.
func (s *Service) UploadCertificate(ctx context.Context, req domain.UploadCertificateRequest) error {
	// 1. Envia o certificado original ao provedor — ele precisa dos bytes
	// reais (não criptografados) para poder assinar as notas.
	providerCertID, err := s.fiscalProvider.UploadCertificado(ctx, req.CertFile, req.Password)
	if err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao enviar certificado ao provedor: %w", err)
	}

	// 2. Criptografa o certificado e a senha para armazenamento local
	// (cópia própria, defesa em profundidade — mesmo que o provedor seja
	// comprometido ou descontinuado, o ProTrack mantém uma cópia segura).
	encCert, nonceCert, err := s.certCipher.Encrypt(req.CertFile)
	if err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao criptografar certificado: %w", err)
	}
	encPassword, noncePassword, err := s.certCipher.Encrypt([]byte(req.Password))
	if err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao criptografar senha do certificado: %w", err)
	}

	// 3. Extrai CN e validade do certificado a partir do próprio arquivo,
	// para permitir alertar/bloquear emissão quando o certificado vencer
	// (ver validateFiscalFields/EmitFiscalInvoice). Falha na extração não
	// impede o upload — cert_subject_cn/expires_at só ficam vazios.
	certInfo, certParseErr := certCrypto.ParseCertificate(req.CertFile, req.Password)

	// 4. Verifica se já existe certificado para a empresa (UPDATE em vez de
	// duplicar, respeitando UNIQUE(company_id)).
	_, err = s.repo.GetCompanyCertificateByCompanyID(ctx, pgconv.ParseUUIDToPgType(req.CompanyID))
	certExists := err == nil

	certSubjectCN := pgconv.ParseStringToPgType("")
	expiresAt := pgconv.TimeToPgTimestamptz(time.Time{})
	if certParseErr == nil {
		certSubjectCN = pgconv.ParseStringToPgType(certInfo.SubjectCN)
		expiresAt = pgconv.TimeToPgTimestamptz(certInfo.NotAfter)
	}

	if certExists {
		_, err = s.repo.UpdateCompanyCertificate(ctx, db.UpdateCompanyCertificateParams{
			CompanyID:              pgconv.ParseUUIDToPgType(req.CompanyID),
			EncryptedCertData:      encCert,
			EncryptedCertNonce:     nonceCert,
			EncryptedPassword:      encPassword,
			EncryptedPasswordNonce: noncePassword,
			CertSubjectCn:          certSubjectCN,
			ExpiresAt:              expiresAt,
			ProviderStatus:         "active",
			ProviderCertID:         pgconv.ParseStringToPgType(providerCertID),
		})
	} else {
		_, err = s.repo.CreateCompanyCertificate(ctx, db.CreateCompanyCertificateParams{
			CompanyID:              pgconv.ParseUUIDToPgType(req.CompanyID),
			EncryptedCertData:      encCert,
			EncryptedCertNonce:     nonceCert,
			EncryptedPassword:      encPassword,
			EncryptedPasswordNonce: noncePassword,
			CertSubjectCn:          certSubjectCN,
			ExpiresAt:              expiresAt,
			ProviderStatus:         "active",
			ProviderCertID:         pgconv.ParseStringToPgType(providerCertID),
		})
	}
	if err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao persistir certificado: %w", err)
	}

	// 5. Registra/atualiza a empresa emitente no provedor, associando o
	// certificado recém-enviado. CadastrarEmpresa é um upsert do lado do
	// PlugNotas (POST /empresa), seguro para repetir em reuploads.
	company, err := s.companiesService.GetCompanyByID(ctx, req.CompanyID)
	if err != nil {
		return fmt.Errorf("fiscal_invoices: certificado salvo, mas falha ao carregar empresa para cadastro no provedor: %w", err)
	}

	payload, err := s.buildCadastrarEmpresaPayload(company, providerCertID)
	if err != nil {
		return fmt.Errorf("fiscal_invoices: certificado salvo, mas empresa não pôde ser cadastrada no provedor: %w", err)
	}

	if err := s.fiscalProvider.CadastrarEmpresa(ctx, payload); err != nil {
		return fmt.Errorf("fiscal_invoices: certificado salvo, mas falha ao cadastrar empresa no provedor: %w", err)
	}

	// 6. Registra o webhook de status (autorizada/rejeitada/cancelada) no
	// provedor. Exige FISCAL_WEBHOOK_CALLBACK_URL e FISCAL_WEBHOOK_SECRET
	// configurados — sem isso o ProTrack nunca fica sabendo do resultado
	// da emissão além do polling de reconciliação.
	if s.cfg.FiscalWebhookCallbackURL == "" || s.cfg.FiscalWebhookSecret == "" {
		return fmt.Errorf("fiscal_invoices: empresa cadastrada no provedor, mas webhook de status não configurado " +
			"(FISCAL_WEBHOOK_CALLBACK_URL/FISCAL_WEBHOOK_SECRET ausentes) — o status das notas só será atualizado via reconciliação periódica")
	}

	callbackURL := s.cfg.FiscalWebhookCallbackURL + "/webhooks/plugnotas"
	if err := s.fiscalProvider.ConfigurarWebhook(ctx, onlyDigits(company.Document), callbackURL, s.cfg.FiscalWebhookSecret); err != nil {
		return fmt.Errorf("fiscal_invoices: empresa cadastrada no provedor, mas falha ao configurar webhook de status: %w", err)
	}

	return nil
}

// DeleteCompanyCertificate revoga (soft delete) o certificado digital A1 da
// empresa. Não remove o cadastro da empresa no provedor nem o webhook — só
// deixa de haver um certificado válido para novas emissões (que passam a
// falhar com ErrCertificateNotFound até um novo upload).
func (s *Service) DeleteCompanyCertificate(ctx context.Context, companyID, deletedBy uuid.UUID) error {
	if err := s.repo.DeleteCompanyCertificate(ctx, db.DeleteCompanyCertificateParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyID),
		DeletedBy: pgconv.ParseUUIDToPgType(deletedBy),
	}); err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao revogar certificado: %w", err)
	}
	return nil
}
