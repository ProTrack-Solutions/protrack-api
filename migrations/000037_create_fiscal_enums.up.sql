-- Regime tributário (MVP: só Simples Nacional; enum pronto pra estender depois)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fiscal_regime_enum') THEN
        CREATE TYPE fiscal_regime_enum AS ENUM (
            'simples_nacional'
        );
    END IF;
END $$;

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