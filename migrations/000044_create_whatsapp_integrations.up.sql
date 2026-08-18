-- Integração WhatsApp: modo por company, catálogo de templates, log de mensagens e uso mensal

CREATE TYPE whatsapp_mode AS ENUM ('platform_shared', 'own_waba');
CREATE TYPE whatsapp_message_category AS ENUM ('marketing', 'utility', 'authentication', 'service');
CREATE TYPE whatsapp_message_status AS ENUM ('queued', 'sent', 'delivered', 'read', 'failed');

-- Configuração de WhatsApp por company: define se ele usa o WABA compartilhado
-- da ProTrack (platform_shared) ou o próprio WABA conectado via Embedded Signup (own_waba).
CREATE TABLE company_whatsapp_configs (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id                UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    mode                     whatsapp_mode NOT NULL DEFAULT 'platform_shared',
    waba_id                  VARCHAR(64),
    phone_number_id          VARCHAR(64),
    display_phone_number     VARCHAR(32),
    -- Só populado quando mode = 'own_waba' (token do System User obtido via Embedded Signup).
    -- Nunca armazene em texto puro em produção — use pgcrypto ou KMS externo.
    access_token_encrypted   TEXT,
    monthly_message_allowance INT NOT NULL DEFAULT 0,
    is_active                BOOLEAN NOT NULL DEFAULT true,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id)
);

-- Catálogo fechado de templates. is_platform_shared_eligible controla a whitelist
-- do WABA central — só templates com essa flag podem ser usados por companies em modo platform_shared.
CREATE TABLE whatsapp_templates (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_template_name            VARCHAR(128) NOT NULL,
    category                      whatsapp_message_category NOT NULL,
    language_code                 VARCHAR(16) NOT NULL DEFAULT 'pt_BR',
    body_text                     TEXT NOT NULL,
    variables                     JSONB NOT NULL DEFAULT '[]', -- ex: ["nome", "valor", "vencimento"]
    is_platform_shared_eligible   BOOLEAN NOT NULL DEFAULT false,
    meta_approval_status          VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending | approved | rejected | paused
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (meta_template_name, language_code)
);

-- Log de cada mensagem disparada. Alimenta tanto o billing quanto o rastreio de entrega via webhook.
CREATE TABLE whatsapp_messages (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id               UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    template_id              UUID REFERENCES whatsapp_templates(id),
    category                 whatsapp_message_category NOT NULL,
    recipient_phone          VARCHAR(32) NOT NULL,
    meta_message_id          VARCHAR(128),
    status                   whatsapp_message_status NOT NULL DEFAULT 'queued',
    failure_code              INT,
    failure_reason           TEXT,
    estimated_cost_cents     INT,
    sent_at                  TIMESTAMPTZ,
    delivered_at              TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_whatsapp_messages_company_created ON whatsapp_messages (company_id, created_at);
CREATE INDEX idx_whatsapp_messages_meta_message_id ON whatsapp_messages (meta_message_id);

-- Rollup mensal por company, usado para alimentar o metered billing do Stripe (franquia + excedente).
CREATE TABLE whatsapp_usage_monthly (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id               UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    billing_period            DATE NOT NULL, -- sempre o primeiro dia do mês
    messages_sent             INT NOT NULL DEFAULT 0,
    messages_over_allowance   INT NOT NULL DEFAULT 0,
    stripe_usage_record_id    VARCHAR(128),
    synced_at                 TIMESTAMPTZ,
    UNIQUE (company_id, billing_period)
);