ALTER TABLE products
    ADD COLUMN ncm VARCHAR(8),
    ADD COLUMN cest VARCHAR(7),
    ADD COLUMN csosn VARCHAR(3),
    -- CFOP em dois campos opcionais (dentro/fora do estado): o correto depende
    -- da UF de destino, resolvido em tempo de emissão. Quando nulo, o service
    -- usa o padrão de mercado (5102/6102, venda de mercadoria de terceiros).
    ADD COLUMN cfop_saida_dentro_estado VARCHAR(4),
    ADD COLUMN cfop_saida_fora_estado VARCHAR(4),
    ADD COLUMN origem_mercadoria SMALLINT NOT NULL DEFAULT 0;