CREATE TABLE IF NOT EXISTS company_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    encrypted_cert_data BYTEA NOT NULL,
    encrypted_cert_nonce BYTEA NOT NULL,
    encrypted_password BYTEA NOT NULL,
    encrypted_password_nonce BYTEA NOT NULL,
    cert_subject_cn VARCHAR(150),
    expires_at TIMESTAMPTZ,
    nuvem_fiscal_status certificate_status_enum NOT NULL DEFAULT 'pending',
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_company_certificates_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_company_certificates_company UNIQUE (company_id)
);