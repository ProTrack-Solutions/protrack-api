INSERT INTO company_settings (company_id, key, value)
SELECT c.id, 'language_sale_overdue_template', '"pt_BR"'::jsonb
FROM companies c
WHERE NOT EXISTS (
    SELECT 1
    FROM company_settings cs
    WHERE cs.company_id = c.id
      AND cs.key = 'language_sale_overdue_template'
);
