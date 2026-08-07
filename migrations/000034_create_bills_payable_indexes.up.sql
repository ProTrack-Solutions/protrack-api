CREATE INDEX IF NOT EXISTS idx_bills_description_search
ON bills_payable
USING GIN (to_tsvector('portuguese_unaccent', coalesce(description, '')));

CREATE INDEX IF NOT EXISTS idx_vendors_name_trgm
ON vendors USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_bill_categories_name_trgm
ON bill_categories USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_payment_methods_name_trgm
ON payment_methods USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_bills_company_due_date
ON bills_payable (company_id, due_date);

CREATE INDEX IF NOT EXISTS idx_bills_company_created_at
ON bills_payable (company_id, created_at DESC);