-- NFC-e pela protrack-fiscal-api (API a API). Substitui a tentativa antiga com a
-- Nuvem Fiscal (migrations 37-41), que nunca teve código usando.

-- 1) Tabelas antigas: só saem se estiverem vazias (nunca apagar nota fiscal)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM fiscal_invoices) THEN
        RAISE EXCEPTION 'fiscal_invoices tem registros; migre-os antes de remover a tabela';
    END IF;
    IF EXISTS (SELECT 1 FROM company_certificates) THEN
        RAISE EXCEPTION 'company_certificates tem registros; remova-os antes (o A1 agora fica na API fiscal)';
    END IF;
END $$;

DROP TABLE IF EXISTS fiscal_invoice_events;
DROP TABLE IF EXISTS fiscal_invoices;
DROP TABLE IF EXISTS company_certificates;
DROP TYPE IF EXISTS fiscal_invoice_event_type_enum;
DROP TYPE IF EXISTS fiscal_invoice_status_enum;
DROP TYPE IF EXISTS fiscal_invoice_type_enum;
DROP TYPE IF EXISTS certificate_status_enum;

-- CSOSN e CFOP passam para o perfil tributário; NCM, CEST e origem continuam no produto
ALTER TABLE products
    DROP COLUMN IF EXISTS csosn,
    DROP COLUMN IF EXISTS cfop_saida_dentro_estado,
    DROP COLUMN IF EXISTS cfop_saida_fora_estado;

-- 2) Perfis tributários: o que vai em cada item da NFC-e
CREATE TABLE IF NOT EXISTS tax_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(80) NOT NULL,
    -- ICMS: CSOSN (CRT 1/4) ou CST (CRT 2/3)
    icms_csosn VARCHAR(3),
    icms_cst VARCHAR(2),
    icms_rate NUMERIC(7, 4),
    icms_base_reduction NUMERIC(7, 4),
    icms_fcp_rate NUMERIC(7, 4),
    cfop VARCHAR(4) NOT NULL DEFAULT '5102',
    benefit_code VARCHAR(10),
    -- PIS/COFINS: 04-09 sem valor; 49/99 com alíquota (base = valor do item)
    pis_cst VARCHAR(2) NOT NULL DEFAULT '49',
    pis_rate NUMERIC(7, 4),
    cofins_cst VARCHAR(2) NOT NULL DEFAULT '49',
    cofins_rate NUMERIC(7, 4),
    -- IBS/CBS (reforma tributária): obrigatório em todo item
    ibs_cbs_cst VARCHAR(3) NOT NULL,
    ibs_cbs_class_code VARCHAR(6) NOT NULL,
    ibs_state_rate NUMERIC(7, 4),
    ibs_city_rate NUMERIC(7, 4),
    cbs_rate NUMERIC(7, 4),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_tax_profiles_icms CHECK ((icms_csosn IS NULL) <> (icms_cst IS NULL)),
    CONSTRAINT chk_tax_profiles_cfop CHECK (cfop ~ '^5\d{3}$'),
    CONSTRAINT chk_tax_profiles_ibs_cbs CHECK (ibs_cbs_cst ~ '^\d{3}$' AND ibs_cbs_class_code ~ '^\d{6}$'),
    CONSTRAINT chk_tax_profiles_ibs_rates CHECK (
        (ibs_state_rate IS NULL AND ibs_city_rate IS NULL AND cbs_rate IS NULL)
        OR (ibs_state_rate IS NOT NULL AND ibs_city_rate IS NOT NULL AND cbs_rate IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tax_profiles_company_name
    ON tax_profiles (company_id, lower(name)) WHERE deleted_at IS NULL;

ALTER TABLE product_categories
    ADD COLUMN IF NOT EXISTS tax_profile_id UUID REFERENCES tax_profiles(id) ON DELETE SET NULL;

-- A unidade (uCom) já existe em products.unit (unit_of_measure, venda a granel)
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS tax_profile_id UUID REFERENCES tax_profiles(id) ON DELETE SET NULL;

-- 3) Configuração fiscal da empresa (o emitente vive na API fiscal)
CREATE TABLE IF NOT EXISTS company_fiscal_settings (
    company_id UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    -- Emite NFC-e ao finalizar a venda
    enabled BOOLEAN NOT NULL DEFAULT false,
    -- test = homologação, live = produção
    environment VARCHAR(4) NOT NULL DEFAULT 'test',
    issuer_id_test UUID,
    issuer_id_live UUID,
    -- CRT: 1 Simples, 2 Simples excesso de sublimite, 3 Normal, 4 MEI
    tax_regime CHAR(1),
    default_tax_profile_id UUID REFERENCES tax_profiles(id) ON DELETE SET NULL,
    certificate_expires_at TIMESTAMPTZ,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_company_fiscal_environment CHECK (environment IN ('test', 'live')),
    CONSTRAINT chk_company_fiscal_regime CHECK (tax_regime IS NULL OR tax_regime IN ('1', '2', '3', '4'))
);

-- 4) Notas emitidas: uma linha por tentativa de emissão da venda
CREATE TABLE IF NOT EXISTS fiscal_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE RESTRICT,
    model SMALLINT NOT NULL DEFAULT 65,
    environment VARCHAR(4) NOT NULL,
    attempt INT NOT NULL,
    idempotency_key VARCHAR(100) NOT NULL UNIQUE,
    -- id da nota na API fiscal (nulo enquanto não chegou lá)
    external_id UUID UNIQUE,
    -- Situação da API fiscal; FAILED = não virou nota (API fora, dado inválido)
    status VARCHAR(12) NOT NULL DEFAULT 'PENDING',
    number INT,
    access_key VARCHAR(44),
    protocol VARCHAR(30),
    status_code VARCHAR(5),
    reject_reason TEXT,
    error_message TEXT,
    emission_type SMALLINT,
    authorized_at TIMESTAMPTZ,
    cancel_reason TEXT,
    cancelled_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_fiscal_documents_environment CHECK (environment IN ('test', 'live')),
    CONSTRAINT chk_fiscal_documents_status CHECK (status IN (
        'PENDING', 'PROCESSING', 'AUTHORIZED', 'REJECTED', 'ERROR', 'CANCELLED',
        'CONTINGENCY', 'SUPERSEDED', 'INUTILIZED', 'FAILED'
    )),
    CONSTRAINT uq_fiscal_documents_sale_attempt UNIQUE (sale_id, model, attempt)
);

-- Só uma nota "viva" por venda: rejeitada/falha permite nova tentativa
CREATE UNIQUE INDEX IF NOT EXISTS uq_fiscal_documents_live_per_sale
    ON fiscal_documents (sale_id, model)
    WHERE status NOT IN ('REJECTED', 'FAILED', 'ERROR');

CREATE INDEX IF NOT EXISTS idx_fiscal_documents_company_created
    ON fiscal_documents (company_id, created_at DESC);

-- Varredura do worker de conferência
CREATE INDEX IF NOT EXISTS idx_fiscal_documents_pending
    ON fiscal_documents (updated_at)
    WHERE status IN ('PENDING', 'PROCESSING', 'CONTINGENCY');

-- 5) Webhooks recebidos (descartar entregas repetidas pelo id do evento)
CREATE TABLE IF NOT EXISTS fiscal_webhook_events (
    id UUID PRIMARY KEY,
    type VARCHAR(40) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
