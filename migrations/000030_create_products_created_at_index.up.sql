CREATE INDEX IF NOT EXISTS idx_products_company_created_at
ON products (company_id, created_at DESC);