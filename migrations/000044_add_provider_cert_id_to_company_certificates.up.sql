ALTER TABLE company_certificates
    ADD COLUMN IF NOT EXISTS provider_cert_id VARCHAR(100);
