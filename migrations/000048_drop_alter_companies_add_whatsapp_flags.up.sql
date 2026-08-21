ALTER TABLE companies
    DROP COLUMN IF EXISTS is_whatsapp,
    DROP COLUMN IF EXISTS is_excess_usage;