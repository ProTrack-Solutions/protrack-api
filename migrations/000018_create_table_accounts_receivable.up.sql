CREATE TABLE IF NOT EXISTS accounts_receivable (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    sale_id UUID NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL,
    balance NUMERIC(10, 2) NOT NULL,
    due_date DATE NOT NULL,
    installment_number INT DEFAULT 1,
    total_installments INT DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_receivable_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_receivable_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT,
    CONSTRAINT fk_receivable_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE
);
-- Índices para performance em relatórios e buscas de cobrança
CREATE INDEX IF NOT EXISTS idx_receivable_customer ON accounts_receivable(customer_id);
CREATE INDEX IF NOT EXISTS idx_receivable_status ON accounts_receivable(status);
CREATE INDEX IF NOT EXISTS idx_receivable_due_date ON accounts_receivable(due_date);