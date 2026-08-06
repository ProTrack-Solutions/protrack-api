CREATE INDEX IF NOT EXISTS idx_customers_search
ON customers
USING GIN (
    to_tsvector('portuguese_unaccent', coalesce(full_name, ''))
);


CREATE INDEX IF NOT EXISTS idx_customers_cpf_trgm
ON customers USING GIN (cpf gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_rg_trgm
ON customers USING GIN (rg gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_email_trgm
ON customers USING GIN (email gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_whatsapp_trgm
ON customers USING GIN (whatsapp gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_mobile_phone_trgm
ON customers USING GIN (mobile_phone gin_trgm_ops);


CREATE INDEX IF NOT EXISTS idx_customers_company_created_at
ON customers (company_id, created_at DESC);