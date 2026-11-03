-- Corpo exato mandado à API fiscal: a nova tentativa com a mesma Idempotency-Key
-- precisa reenviar o mesmo pedido (mudar o corpo dá conflito de idempotência)
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS request_payload JSONB NOT NULL DEFAULT '{}'::jsonb;
-- Quando o worker tentou de novo pela última vez (e quantas vezes)
ALTER TABLE fiscal_documents ADD COLUMN IF NOT EXISTS retries INT NOT NULL DEFAULT 0;
