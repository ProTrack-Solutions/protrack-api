UPDATE company_settings
SET value = '2'::jsonb,
    updated_at = now()
WHERE key = 'low_stock'
  AND value = '5'::jsonb;
