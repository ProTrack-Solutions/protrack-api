ALTER TABLE companies
    DROP COLUMN IF EXISTS external_company_id VARCHAR(100) NOT NULL DEFAULT 'unknown';