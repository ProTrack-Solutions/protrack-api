CREATE TYPE cash_session_status AS ENUM ('open', 'closed');
CREATE TYPE cash_movement_type AS ENUM ('withdrawal', 'supply');

-- Turno de caixa de um operador. As vendas e recebimentos do turno não ficam
-- ligados aqui: são os feitos pelo operador (created_by / user_id) entre
-- opened_at e closed_at. Os totais do fechamento ficam gravados como retrato,
-- para o relatório não mudar se uma venda for cancelada depois.
CREATE TABLE IF NOT EXISTS cash_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status cash_session_status NOT NULL DEFAULT 'open',
    opening_amount NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (opening_amount >= 0),
    opening_notes TEXT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMPTZ NULL,
    closed_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    closing_notes TEXT NULL,
    sales_total NUMERIC(12, 2) NULL,
    sales_count INT NULL,
    received_total NUMERIC(12, 2) NULL,
    supplies_total NUMERIC(12, 2) NULL,
    withdrawals_total NUMERIC(12, 2) NULL,
    expected_cash NUMERIC(12, 2) NULL,
    counted_cash NUMERIC(12, 2) NULL CHECK (counted_cash >= 0),
    difference NUMERIC(12, 2) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_cash_sessions_closed CHECK (
        (status = 'open' AND closed_at IS NULL)
        OR (status = 'closed' AND closed_at IS NOT NULL AND counted_cash IS NOT NULL)
    )
);

-- Um operador só pode ter um caixa aberto por vez
CREATE UNIQUE INDEX IF NOT EXISTS uq_cash_sessions_open_per_user ON cash_sessions (company_id, user_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_cash_sessions_company_opened_at ON cash_sessions (company_id, opened_at DESC);

-- Sangrias (retirada de dinheiro da gaveta) e suprimentos (reforço de troco)
CREATE TABLE IF NOT EXISTS cash_session_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES cash_sessions(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    type cash_movement_type NOT NULL,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    reason TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cash_session_movements_session ON cash_session_movements (session_id, created_at);

-- Consultas do resumo do turno: vendas e recebimentos do operador num intervalo
CREATE INDEX IF NOT EXISTS idx_sales_company_created_by_created_at ON sales (company_id, created_by, created_at);
CREATE INDEX IF NOT EXISTS idx_payment_history_company_user_date ON payment_history (company_id, user_id, payment_date);
