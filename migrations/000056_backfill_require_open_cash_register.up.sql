-- Empresas que já existiam seguem vendendo sem caixa aberto; as novas recebem a
-- exigência ligada pelos DefaultSettings. Cada uma pode mudar nas configurações.
INSERT INTO company_settings (company_id, key, value)
SELECT c.id, 'require_open_cash_register', 'false'::jsonb
FROM companies c
WHERE NOT EXISTS (
    SELECT 1
    FROM company_settings cs
    WHERE cs.company_id = c.id
      AND cs.key = 'require_open_cash_register'
);
