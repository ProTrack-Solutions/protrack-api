ALTER TABLE products
    DROP COLUMN IF EXISTS origem_mercadoria,
    DROP COLUMN IF EXISTS cfop_saida_fora_estado,
    DROP COLUMN IF EXISTS cfop_saida_dentro_estado,
    DROP COLUMN IF EXISTS csosn,
    DROP COLUMN IF EXISTS cest,
    DROP COLUMN IF EXISTS ncm;