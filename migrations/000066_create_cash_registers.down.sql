DROP INDEX IF EXISTS idx_sale_returns_cash_session;
DROP INDEX IF EXISTS idx_payment_history_cash_session;
DROP INDEX IF EXISTS idx_sales_cash_session;
ALTER TABLE sale_returns DROP COLUMN IF EXISTS cash_session_id;
ALTER TABLE payment_history DROP COLUMN IF EXISTS cash_session_id;
ALTER TABLE sales DROP COLUMN IF EXISTS cash_session_id;

DROP INDEX IF EXISTS idx_cash_sessions_register_opened_at;
DROP INDEX IF EXISTS uq_cash_sessions_open_per_register;
-- Volta a regra antiga; falha se houver dois turnos abertos do mesmo operador
CREATE UNIQUE INDEX IF NOT EXISTS uq_cash_sessions_open_per_user ON cash_sessions (company_id, user_id) WHERE status = 'open';
ALTER TABLE cash_sessions DROP COLUMN IF EXISTS register_id;

DROP TABLE IF EXISTS cash_registers;
