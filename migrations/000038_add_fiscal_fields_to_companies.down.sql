ALTER TABLE companies
    DROP COLUMN IF EXISTS regime_tributario,
    DROP COLUMN IF EXISTS cnae,
    DROP COLUMN IF EXISTS inscricao_estadual_isento,
    DROP COLUMN IF EXISTS inscricao_estadual;