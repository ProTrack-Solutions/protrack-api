-- Histórico de e-mails do pedido de compra enviados ao fornecedor.
-- O envio em si é assíncrono (fila): o registro marca quando foi solicitado.
CREATE TABLE IF NOT EXISTS purchase_order_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    kind VARCHAR(10) NOT NULL CHECK (kind IN ('sent', 'updated', 'resend')),
    recipient VARCHAR(150) NOT NULL,
    cc VARCHAR(150) NULL,
    sent_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_emails_order ON purchase_order_emails (order_id, created_at DESC);
