CREATE TYPE cash_transfer_status AS ENUM ('pending', 'accepted', 'rejected', 'canceled');

-- Troco levado de um caixa para outro. O dinheiro sai da gaveta de origem no
-- envio (sangria ligada à transferência) e só entra no destino quando alguém
-- daquele caixa aceita (suprimento ligado). Recusada ou cancelada, volta para a
-- origem com um suprimento de devolução. Os dois turnos não fecham com
-- transferência pendente.
CREATE TABLE IF NOT EXISTS cash_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    from_session_id UUID NOT NULL REFERENCES cash_sessions(id) ON DELETE CASCADE,
    to_session_id UUID NOT NULL REFERENCES cash_sessions(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    reason TEXT NULL,
    status cash_transfer_status NOT NULL DEFAULT 'pending',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_cash_transfers_sessions CHECK (from_session_id <> to_session_id),
    CONSTRAINT chk_cash_transfers_resolved CHECK (
        (status = 'pending' AND resolved_at IS NULL)
        OR (status <> 'pending' AND resolved_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_cash_transfers_from_pending ON cash_transfers (from_session_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_cash_transfers_to_pending ON cash_transfers (to_session_id) WHERE status = 'pending';

-- Sangrias e suprimentos gerados por uma transferência apontam para ela
ALTER TABLE cash_session_movements ADD COLUMN IF NOT EXISTS transfer_id UUID NULL REFERENCES cash_transfers(id) ON DELETE SET NULL;
