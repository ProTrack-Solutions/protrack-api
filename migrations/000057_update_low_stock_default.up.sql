-- low_stock passa a valer de verdade (antes os limites eram fixos em 5 e 10).
-- Empresas com o padrão antigo, que nunca foi usado, recebem o novo padrão 5,
-- que mantém as faixas que a tela já mostrava.
UPDATE company_settings
SET value = '5'::jsonb,
    updated_at = now()
WHERE key = 'low_stock'
  AND value = '2'::jsonb;
