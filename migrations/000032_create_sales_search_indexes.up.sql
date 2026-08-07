CREATE INDEX IF NOT EXISTS idx_sales_company_sale_at
ON sales (company_id, sale_at DESC);

CREATE INDEX IF NOT EXISTS idx_sales_company_created_at
ON sales (company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sales_company_status
ON sales (company_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_company_payment_method
ON sales (company_id, payment_method);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id
ON sale_items (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_product_id
ON sale_items (product_id);

CREATE INDEX IF NOT EXISTS idx_accounts_receivable_sale_id_due_date
ON accounts_receivable (sale_id, due_date);