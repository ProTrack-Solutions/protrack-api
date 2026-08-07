CREATE INDEX IF NOT EXISTS idx_customers_full_name_trgm
ON customers
USING GIN (full_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_receivable_company_created_at
ON accounts_receivable (company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_receivable_company_due_date
ON accounts_receivable (company_id, due_date);