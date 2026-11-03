DROP TABLE IF EXISTS fiscal_webhook_events;
DROP TABLE IF EXISTS fiscal_documents;
DROP TABLE IF EXISTS company_fiscal_settings;
ALTER TABLE products DROP COLUMN IF EXISTS tax_profile_id;
ALTER TABLE product_categories DROP COLUMN IF EXISTS tax_profile_id;
DROP TABLE IF EXISTS tax_profiles;

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS csosn VARCHAR(3),
    ADD COLUMN IF NOT EXISTS cfop_saida_dentro_estado VARCHAR(4),
    ADD COLUMN IF NOT EXISTS cfop_saida_fora_estado VARCHAR(4);

-- Estrutura antiga da Nuvem Fiscal (migrations 37, 40 e 41)
-- Modelo do documento fiscal
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fiscal_invoice_type_enum') THEN
        CREATE TYPE fiscal_invoice_type_enum AS ENUM (
            'nfe',
            'nfce'
        );
    END IF;
END $$;

-- Status do documento fiscal
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fiscal_invoice_status_enum') THEN
        CREATE TYPE fiscal_invoice_status_enum AS ENUM (
            'draft',
            'processing',
            'authorized',
            'rejected',
            'cancelled',
            'cancel_processing'
        );
    END IF;
END $$;

-- Tipo de evento registrado na trilha de auditoria do documento fiscal
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fiscal_invoice_event_type_enum') THEN
        CREATE TYPE fiscal_invoice_event_type_enum AS ENUM (
            'created',
            'sent_to_provider',
            'authorized',
            'rejected',
            'cancel_requested',
            'cancelled',
            'cancel_rejected',
            'webhook_received',
            'error'
        );
    END IF;
END $$;

-- Status do certificado digital junto ao provedor
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'certificate_status_enum') THEN
        CREATE TYPE certificate_status_enum AS ENUM (
            'pending',
            'active',
            'expired',
            'invalid',
            'revoked'
        );
    END IF;
END $$;

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

CREATE TABLE IF NOT EXISTS fiscal_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    sale_id UUID NOT NULL,
    type fiscal_invoice_type_enum NOT NULL,
    status fiscal_invoice_status_enum NOT NULL DEFAULT 'draft',
    nuvem_fiscal_id VARCHAR(100),
    chave_acesso VARCHAR(44),
    numero VARCHAR(20),
    serie VARCHAR(10),
    protocolo_autorizacao VARCHAR(30),
    xml_url TEXT,
    danfe_url TEXT,
    error_message TEXT,
    cancelled_reason TEXT,
    authorized_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_fiscal_invoices_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_fiscal_invoices_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE RESTRICT,
    CONSTRAINT uq_fiscal_invoices_sale_type UNIQUE (sale_id, type)
);

CREATE TABLE IF NOT EXISTS fiscal_invoice_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fiscal_invoice_id UUID NOT NULL,
    event_type fiscal_invoice_event_type_enum NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_fiscal_invoice_events_invoice FOREIGN KEY (fiscal_invoice_id) REFERENCES fiscal_invoices(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fiscal_invoice_events_invoice_id
    ON fiscal_invoice_events (fiscal_invoice_id);
