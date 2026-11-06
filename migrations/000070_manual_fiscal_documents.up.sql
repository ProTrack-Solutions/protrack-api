-- Emissão manual de NF-e (modelo 55) e NFC-e (65): nota avulsa, sem venda e sem
-- mexer no estoque. A nota guarda o próprio destinatário, natureza e total.

ALTER TABLE fiscal_documents ALTER COLUMN sale_id DROP NOT NULL;

-- sale = emitida a partir de uma venda do PDV; manual = tela de emissão manual
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS source VARCHAR(6) NOT NULL DEFAULT 'sale';
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS operation_nature VARCHAR(60);
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS recipient_name VARCHAR(60);
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS recipient_document VARCHAR(14);
-- Total da nota manual (a da venda vem de sales.total_amount)
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS total NUMERIC(14, 2);

ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_source;
ALTER TABLE fiscal_documents ADD CONSTRAINT chk_fiscal_documents_source CHECK (
    (source = 'sale' AND sale_id IS NOT NULL) OR (source = 'manual' AND sale_id IS NULL)
);

ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_model;
ALTER TABLE fiscal_documents ADD CONSTRAINT chk_fiscal_documents_model CHECK (model IN (55, 65));

-- A NF-e da venda ainda não existe: NF-e só nasce na emissão manual
ALTER TABLE fiscal_documents DROP CONSTRAINT IF EXISTS chk_fiscal_documents_sale_model;
ALTER TABLE fiscal_documents ADD CONSTRAINT chk_fiscal_documents_sale_model CHECK (source = 'manual' OR model = 65);
