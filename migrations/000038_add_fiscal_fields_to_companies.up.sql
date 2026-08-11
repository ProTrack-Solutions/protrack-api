ALTER TABLE companies
    ADD COLUMN inscricao_estadual VARCHAR(20),
    ADD COLUMN inscricao_estadual_isento BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN cnae VARCHAR(10),
    ADD COLUMN regime_tributario fiscal_regime_enum;