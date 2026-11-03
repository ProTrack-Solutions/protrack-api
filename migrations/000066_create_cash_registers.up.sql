-- Caixas físicos da empresa ("Caixa 1", "Balcão"). O turno (cash_sessions)
-- passa a ser do caixa, não do operador: vários funcionários vendem no mesmo
-- caixa aberto, e qualquer um deles pode fazer sangria, suprimento e fechar.
CREATE TABLE IF NOT EXISTS cash_registers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(60) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    -- Só para a migração: de qual operador vieram os turnos antigos
    legacy_user_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cash_registers_company ON cash_registers (company_id, active);

-- Cada operador que já tinha turnos vira um caixa, para os turnos abertos hoje
-- não colidirem no índice "um aberto por caixa". O primeiro operador da empresa
-- fica com o "Caixa principal"; os outros podem ser renomeados ou desativados.
INSERT INTO cash_registers (company_id, name, legacy_user_id)
SELECT r.company_id,
    CASE WHEN r.rn = 1 THEN 'Caixa principal' ELSE LEFT('Caixa de ' || u.name, 60) END,
    r.user_id
FROM (
        SELECT company_id,
            user_id,
            ROW_NUMBER() OVER (PARTITION BY company_id ORDER BY MIN(opened_at)) AS rn
        FROM cash_sessions
        GROUP BY company_id, user_id
    ) r
    INNER JOIN users u ON u.id = r.user_id;

-- Empresas sem nenhum turno ganham o caixa padrão
INSERT INTO cash_registers (company_id, name)
SELECT c.id, 'Caixa principal'
FROM companies c
WHERE NOT EXISTS (SELECT 1 FROM cash_registers cr WHERE cr.company_id = c.id);

ALTER TABLE cash_sessions ADD COLUMN IF NOT EXISTS register_id UUID NULL REFERENCES cash_registers(id) ON DELETE RESTRICT;

UPDATE cash_sessions cs
SET register_id = cr.id
FROM cash_registers cr
WHERE cr.company_id = cs.company_id
    AND cr.legacy_user_id = cs.user_id;

ALTER TABLE cash_sessions ALTER COLUMN register_id SET NOT NULL;
ALTER TABLE cash_registers DROP COLUMN legacy_user_id;

-- user_id continua sendo quem abriu o turno
DROP INDEX IF EXISTS uq_cash_sessions_open_per_user;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cash_sessions_open_per_register ON cash_sessions (register_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_cash_sessions_register_opened_at ON cash_sessions (register_id, opened_at DESC);

-- Em qual turno cada venda, recebimento e devolução aconteceu. O valor padrão
-- vem da variável da transação app.cash_session_id, que a API grava ao achar
-- o caixa aberto no começo da operação: tudo que é inserido naquela transação
-- (inclusive a venda nova de uma troca) fica ligado ao turno sem precisar
-- passar o id por cada serviço. Fora dessa transação o valor fica nulo.
ALTER TABLE sales ADD COLUMN IF NOT EXISTS cash_session_id UUID NULL
    DEFAULT NULLIF(current_setting('app.cash_session_id', true), '')::uuid
    REFERENCES cash_sessions(id) ON DELETE SET NULL;
ALTER TABLE payment_history ADD COLUMN IF NOT EXISTS cash_session_id UUID NULL
    DEFAULT NULLIF(current_setting('app.cash_session_id', true), '')::uuid
    REFERENCES cash_sessions(id) ON DELETE SET NULL;
ALTER TABLE sale_returns ADD COLUMN IF NOT EXISTS cash_session_id UUID NULL
    DEFAULT NULLIF(current_setting('app.cash_session_id', true), '')::uuid
    REFERENCES cash_sessions(id) ON DELETE SET NULL;

-- Histórico: liga pela regra antiga (feito pelo operador dentro do turno dele)
UPDATE sales s
SET cash_session_id = cs.id
FROM cash_sessions cs
WHERE s.company_id = cs.company_id
    AND s.created_by = cs.user_id
    AND s.created_at >= cs.opened_at
    AND s.created_at < COALESCE(cs.closed_at, 'infinity');

UPDATE payment_history ph
SET cash_session_id = cs.id
FROM cash_sessions cs
WHERE ph.company_id = cs.company_id
    AND ph.user_id = cs.user_id
    AND ph.payment_date >= cs.opened_at
    AND ph.payment_date < COALESCE(cs.closed_at, 'infinity');

UPDATE sale_returns sr
SET cash_session_id = cs.id
FROM cash_sessions cs
WHERE sr.company_id = cs.company_id
    AND sr.created_by = cs.user_id
    AND sr.created_at >= cs.opened_at
    AND sr.created_at < COALESCE(cs.closed_at, 'infinity');

CREATE INDEX IF NOT EXISTS idx_sales_cash_session ON sales (cash_session_id) WHERE cash_session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_history_cash_session ON payment_history (cash_session_id) WHERE cash_session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sale_returns_cash_session ON sale_returns (cash_session_id) WHERE cash_session_id IS NOT NULL;
