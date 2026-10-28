-- Variações de grade (vestuário): cada combinação cor × tamanho é um produto
-- normal, agrupado pelo modelo. size já existia; entram o modelo e a cor.
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS model_name VARCHAR(150) NULL,
    ADD COLUMN IF NOT EXISTS color VARCHAR(50) NULL;

-- Recompra: busca das variações de um modelo
CREATE INDEX IF NOT EXISTS idx_products_company_model ON products (company_id, lower(model_name))
WHERE model_name IS NOT NULL AND deleted_at IS NULL;

-- A mesma combinação modelo/cor/tamanho não pode ser cadastrada duas vezes
CREATE UNIQUE INDEX IF NOT EXISTS uq_products_model_variation ON products (
    company_id,
    lower(model_name),
    lower(COALESCE(color, '')),
    lower(COALESCE(size, ''))
)
WHERE model_name IS NOT NULL AND deleted_at IS NULL;
