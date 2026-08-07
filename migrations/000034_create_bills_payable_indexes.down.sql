DROP INDEX IF NOT EXISTS idx_bills_description_search;
DROP INDEX IF NOT EXISTS idx_vendors_name_trgm;
DROP INDEX IF NOT EXISTS idx_bill_categories_name_trgm;
DROP INDEX IF NOT EXISTS idx_payment_methods_name_trgm;
DROP INDEX IF NOT EXISTS idx_bills_company_due_date;
DROP INDEX IF NOT EXISTS idx_bills_company_created_at;