CREATE TABLE IF NOT EXISTS fiscal_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    sale_id UUID NOT NULL,
    type fiscal_invoice_type_enum NOT NULL,
    status fiscal_invoice_status_enum NOT NULL DEFAULT 'draft',
    provider_invoice_id VARCHAR(100),
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