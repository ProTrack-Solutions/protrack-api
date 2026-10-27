-- name: CreateCashSession :one
INSERT INTO cash_sessions (company_id, user_id, opening_amount, opening_notes)
VALUES (@company_id, @user_id, @opening_amount, @opening_notes)
RETURNING *;

-- name: GetOpenCashSessionByUser :one
SELECT *
FROM cash_sessions
WHERE company_id = @company_id
    AND user_id = @user_id
    AND status = 'open';

-- Trava a linha para que movimentos e fechamento concorrentes não se cruzem
-- name: GetOpenCashSessionByUserForUpdate :one
SELECT *
FROM cash_sessions
WHERE company_id = @company_id
    AND user_id = @user_id
    AND status = 'open'
FOR UPDATE;

-- name: GetCashSessionById :one
SELECT cs.*,
    u.name::text AS user_name
FROM cash_sessions cs
    INNER JOIN users u ON u.id = cs.user_id
WHERE cs.id = @id
    AND cs.company_id = @company_id;

-- name: CloseCashSession :one
UPDATE cash_sessions
SET status = 'closed',
    closed_at = CURRENT_TIMESTAMP,
    closed_by = @closed_by,
    closing_notes = @closing_notes,
    sales_total = @sales_total,
    sales_count = @sales_count,
    received_total = @received_total,
    supplies_total = @supplies_total,
    withdrawals_total = @withdrawals_total,
    expected_cash = @expected_cash,
    counted_cash = @counted_cash,
    difference = @difference,
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id
    AND status = 'open'
RETURNING *;

-- user_id nulo lista os caixas de todos os operadores (visão do administrador)
-- name: ListCashSessions :many
SELECT cs.*,
    u.name::text AS user_name,
    COUNT(*) OVER () AS total_rows
FROM cash_sessions cs
    INNER JOIN users u ON u.id = cs.user_id
WHERE cs.company_id = @company_id
    AND (sqlc.narg(user_id)::uuid IS NULL OR cs.user_id = sqlc.narg(user_id)::uuid)
ORDER BY cs.opened_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CreateCashSessionMovement :one
INSERT INTO cash_session_movements (session_id, company_id, type, amount, reason, created_by)
VALUES (@session_id, @company_id, @type, @amount, @reason, @created_by)
RETURNING *;

-- name: ListCashSessionMovements :many
SELECT *
FROM cash_session_movements
WHERE session_id = @session_id
ORDER BY created_at DESC;

-- Vendas do operador no turno, por forma de pagamento. Na venda a prazo só a
-- entrada (down_payment) é recebida no balcão; o resto vira contas a receber.
-- O troco já está descontado: received é o que fica na gaveta, não o que o
-- cliente entregou.
-- name: GetCashSessionSalesByMethod :many
SELECT s.payment_method::text AS payment_method,
    COUNT(*)::int AS sales_count,
    COALESCE(SUM(s.total_amount), 0)::float8 AS sales_total,
    COALESCE(SUM(
        CASE
            WHEN s.payment_method = 'installments' THEN s.down_payment
            ELSE s.total_amount
        END
    ), 0)::float8 AS received,
    COUNT(*) FILTER (WHERE s.change_amount > 0)::int AS change_count,
    COALESCE(SUM(s.change_amount), 0)::float8 AS change_total
FROM sales s
WHERE s.company_id = @company_id
    AND s.created_by = @user_id
    AND s.created_at >= @start_at
    AND s.created_at < @end_at
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
GROUP BY s.payment_method;

-- Parcelas de vendas a prazo recebidas pelo operador no turno
-- name: GetCashSessionReceiptsByMethod :many
SELECT COALESCE(pm.type::text, 'other')::text AS payment_method,
    COUNT(*)::int AS receipts_count,
    COALESCE(SUM(ph.amount_paid), 0)::float8 AS received
FROM payment_history ph
    LEFT JOIN payment_methods pm ON pm.id = ph.payment_method_id
WHERE ph.company_id = @company_id
    AND ph.user_id = @user_id
    AND ph.payment_date >= @start_at
    AND ph.payment_date < @end_at
GROUP BY COALESCE(pm.type::text, 'other');
