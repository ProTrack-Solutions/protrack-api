DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM fiscal_documents WHERE source = 'manual') THEN
        RAISE EXCEPTION 'fiscal_documents tem notas manuais; não dá para voltar sale_id a obrigatório';
    END IF;
END $$;

ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_sale_model;
ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_model;
ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_source;
ALTER TABLE fiscal_documents DROP COLUMN IF EXISTS total;
ALTER TABLE fiscal_documents DROP COLUMN IF EXISTS recipient_document;
ALTER TABLE fiscal_documents DROP COLUMN IF EXISTS recipient_name;
ALTER TABLE fiscal_documents DROP COLUMN IF EXISTS operation_nature;
ALTER TABLE fiscal_documents DROP COLUMN IF EXISTS source;
ALTER TABLE fiscal_documents ALTER COLUMN sale_id SET NOT NULL;
