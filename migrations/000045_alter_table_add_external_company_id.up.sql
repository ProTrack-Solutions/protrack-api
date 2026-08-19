ALTER TABLE companies
    ADD COLUMN external_company_id VARCHAR(100) NOT NULL DEFAULT 'unknown';