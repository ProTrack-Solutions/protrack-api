ALTER TABLE companies
    ADD COLUMN is_whatsapp     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_excess_usage BOOLEAN NOT NULL DEFAULT false;