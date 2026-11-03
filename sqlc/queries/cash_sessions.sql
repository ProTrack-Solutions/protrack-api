-- ---------------------------------------------------------------------------
-- Caixas cadastrados
-- ---------------------------------------------------------------------------

-- Caixas da empresa com o turno aberto de cada um (quando houver) e o último
-- fechamento, que serve de referência para o troco da próxima abertura
-- name: ListCashRegisters :many
SELECT cr.id,
    cr.name,
    cr.active,
    cs.id AS session_id,
    cs.opened_at,
    cs.opening_amount,
    -- LEFT JOIN: caixa fechado não tem quem abriu
    COALESCE(u.name, '')::text AS opened_by_name,
    lc.closed_at AS last_closed_at,
    lc.counted_cash AS last_counted_cash,
    COALESCE(lc.closed_by_name, '')::text AS last_closed_by_name
FROM cash_registers cr
    LEFT JOIN cash_sessions cs ON cs.register_id = cr.id AND cs.status = 'open'
    LEFT JOIN users u ON u.id = cs.user_id
    LEFT JOIN LATERAL (
        SELECT c.closed_at,
            c.counted_cash,
            cu.name AS closed_by_name
        FROM cash_sessions c
            LEFT JOIN users cu ON cu.id = c.closed_by
        WHERE c.register_id = cr.id
            AND c.status = 'closed'
        ORDER BY c.closed_at DESC
        LIMIT 1
    ) lc ON TRUE
WHERE cr.company_id = @company_id
ORDER BY cr.active DESC, cr.created_at, cr.name;

-- name: GetCashRegisterById :one
SELECT *
FROM cash_registers
WHERE id = @id
    AND company_id = @company_id;

-- name: ListActiveCashRegisterIds :many
SELECT id
FROM cash_registers
WHERE company_id = @company_id
    AND active
ORDER BY created_at;

-- Empresa nova (sem nenhum caixa) ganha o "Caixa principal"
-- name: EnsureDefaultCashRegister :exec
INSERT INTO cash_registers (company_id, name)
SELECT @company_id, 'Caixa principal'
WHERE NOT EXISTS (
        SELECT 1
        FROM cash_registers
        WHERE company_id = @company_id
    );

-- name: CreateCashRegister :one
INSERT INTO cash_registers (company_id, name)
VALUES (@company_id, @name)
RETURNING *;

-- name: UpdateCashRegister :one
UPDATE cash_registers
SET name = @name,
    active = @active,
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id
    AND company_id = @company_id
RETURNING *;

-- ---------------------------------------------------------------------------
-- Turnos
-- ---------------------------------------------------------------------------

-- name: CreateCashSession :one
INSERT INTO cash_sessions (company_id, register_id, user_id, opening_amount, opening_notes)
VALUES (@company_id, @register_id, @user_id, @opening_amount, @opening_notes)
RETURNING *;

-- name: GetOpenCashSessionByRegister :one
SELECT *
FROM cash_sessions
WHERE register_id = @register_id
    AND status = 'open';

-- Trava a linha para que movimentos e fechamento concorrentes não se cruzem
-- name: GetOpenCashSessionByRegisterForUpdate :one
SELECT *
FROM cash_sessions
WHERE register_id = @register_id
    AND status = 'open'
FOR UPDATE;

-- Venda, recebimento e devolução travam o turno só para leitura: várias rodam
-- juntas, mas o fechamento (FOR UPDATE) espera elas terminarem e vice-versa.
-- name: GetOpenCashSessionByRegisterForShare :one
SELECT id
FROM cash_sessions
WHERE register_id = @register_id
    AND status = 'open'
FOR SHARE;

-- name: ListOpenCashSessionIdsForShare :many
SELECT id
FROM cash_sessions
WHERE company_id = @company_id
    AND status = 'open'
FOR SHARE;

-- Grava o turno na variável da transação: é o valor padrão de cash_session_id
-- em sales, payment_history e sale_returns (ver migração 000066)
-- name: BindCashSession :exec
SELECT set_config('app.cash_session_id', @session_id::text, true);

-- name: GetCashSessionById :one
SELECT cs.*,
    u.name::text AS user_name,
    cr.name::text AS register_name,
    COALESCE(cu.name, '')::text AS closed_by_name
FROM cash_sessions cs
    INNER JOIN users u ON u.id = cs.user_id
    INNER JOIN cash_registers cr ON cr.id = cs.register_id
    LEFT JOIN users cu ON cu.id = cs.closed_by
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

-- user_id nulo lista os turnos de todos (visão do administrador). Com user_id,
-- só os turnos de que o operador participou: abriu, fechou ou vendeu/recebeu.
-- name: ListCashSessions :many
SELECT cs.*,
    u.name::text AS user_name,
    cr.name::text AS register_name,
    COALESCE(cu.name, '')::text AS closed_by_name,
    COUNT(*) OVER () AS total_rows
FROM cash_sessions cs
    INNER JOIN users u ON u.id = cs.user_id
    INNER JOIN cash_registers cr ON cr.id = cs.register_id
    LEFT JOIN users cu ON cu.id = cs.closed_by
WHERE cs.company_id = @company_id
    AND (sqlc.narg(register_id)::uuid IS NULL OR cs.register_id = sqlc.narg(register_id)::uuid)
    AND (
        sqlc.narg(user_id)::uuid IS NULL
        OR cs.user_id = sqlc.narg(user_id)::uuid
        OR cs.closed_by = sqlc.narg(user_id)::uuid
        OR EXISTS (
            SELECT 1 FROM sales s
            WHERE s.cash_session_id = cs.id AND s.created_by = sqlc.narg(user_id)::uuid
        )
        OR EXISTS (
            SELECT 1 FROM payment_history ph
            WHERE ph.cash_session_id = cs.id AND ph.user_id = sqlc.narg(user_id)::uuid
        )
    )
ORDER BY cs.opened_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: UserTookPartInCashSession :one
SELECT (
        cs.user_id = @user_id
        OR cs.closed_by = @user_id
        OR EXISTS (SELECT 1 FROM sales s WHERE s.cash_session_id = cs.id AND s.created_by = @user_id)
        OR EXISTS (SELECT 1 FROM payment_history ph WHERE ph.cash_session_id = cs.id AND ph.user_id = @user_id)
    )::bool AS took_part
FROM cash_sessions cs
WHERE cs.id = @session_id;

-- name: CreateCashSessionMovement :one
INSERT INTO cash_session_movements (session_id, company_id, type, amount, reason, created_by, transfer_id)
VALUES (@session_id, @company_id, @type, @amount, @reason, @created_by, sqlc.narg(transfer_id))
RETURNING *;

-- transfer_label descreve o movimento gerado por uma transferência entre caixas
-- name: ListCashSessionMovements :many
SELECT m.*,
    u.name::text AS created_by_name,
    (CASE
        WHEN m.transfer_id IS NULL THEN ''
        WHEN m.session_id = t.from_session_id AND m.type = 'withdrawal' THEN 'Transferência para ' || rto.name
        WHEN m.session_id = t.to_session_id THEN 'Transferência do ' || rfrom.name
        WHEN t.status = 'rejected' THEN 'Transferência devolvida (recusada pelo ' || rto.name || ')'
        ELSE 'Transferência devolvida (cancelada)'
    END)::text AS transfer_label
FROM cash_session_movements m
    INNER JOIN users u ON u.id = m.created_by
    LEFT JOIN cash_transfers t ON t.id = m.transfer_id
    LEFT JOIN cash_sessions sf ON sf.id = t.from_session_id
    LEFT JOIN cash_registers rfrom ON rfrom.id = sf.register_id
    LEFT JOIN cash_sessions st ON st.id = t.to_session_id
    LEFT JOIN cash_registers rto ON rto.id = st.register_id
WHERE m.session_id = @session_id
ORDER BY m.created_at DESC;

-- ---------------------------------------------------------------------------
-- Transferência de troco entre caixas
-- ---------------------------------------------------------------------------

-- Trava os turnos sempre na mesma ordem (por id): duas transferências cruzadas
-- ao mesmo tempo não se travam uma à outra
-- name: LockCashSessions :many
SELECT *
FROM cash_sessions
WHERE id = ANY(@ids::uuid[])
ORDER BY id
FOR UPDATE;

-- name: CreateCashTransfer :one
INSERT INTO cash_transfers (company_id, from_session_id, to_session_id, amount, reason, created_by)
VALUES (@company_id, @from_session_id, @to_session_id, @amount, @reason, @created_by)
RETURNING *;

-- name: GetCashTransferForUpdate :one
SELECT *
FROM cash_transfers
WHERE id = @id
    AND company_id = @company_id
FOR UPDATE;

-- name: ResolveCashTransfer :exec
UPDATE cash_transfers
SET status = @status,
    resolved_by = @resolved_by,
    resolved_at = CURRENT_TIMESTAMP
WHERE id = @id
    AND status = 'pending';

-- Transferências pendentes que saem ou chegam no turno
-- name: ListPendingCashTransfers :many
SELECT t.id,
    t.amount,
    t.reason,
    t.created_at,
    (t.to_session_id = @session_id)::bool AS incoming,
    rfrom.name::text AS from_register_name,
    rto.name::text AS to_register_name,
    u.name::text AS created_by_name
FROM cash_transfers t
    INNER JOIN cash_sessions sf ON sf.id = t.from_session_id
    INNER JOIN cash_registers rfrom ON rfrom.id = sf.register_id
    INNER JOIN cash_sessions st ON st.id = t.to_session_id
    INNER JOIN cash_registers rto ON rto.id = st.register_id
    INNER JOIN users u ON u.id = t.created_by
WHERE t.status = 'pending'
    AND (t.from_session_id = @session_id OR t.to_session_id = @session_id)
ORDER BY t.created_at;

-- name: CountPendingCashTransfers :one
SELECT COUNT(*)::int
FROM cash_transfers
WHERE status = 'pending'
    AND (from_session_id = @session_id OR to_session_id = @session_id);

-- ---------------------------------------------------------------------------
-- Resumo do turno: tudo que foi gravado com o cash_session_id dele
-- ---------------------------------------------------------------------------

-- Vendas do turno, por forma de pagamento. Na venda a prazo só a entrada
-- (down_payment) é recebida no balcão; o resto vira contas a receber.
-- O troco já está descontado: received é o que fica na gaveta, não o que o
-- cliente entregou. A parte paga com vale-troca/troca (credit_amount) não entra.
-- name: GetCashSessionSalesByMethod :many
SELECT s.payment_method::text AS payment_method,
    COUNT(*)::int AS sales_count,
    COALESCE(SUM(s.total_amount), 0)::float8 AS sales_total,
    COALESCE(SUM(
        CASE
            WHEN s.payment_method = 'installments' THEN s.down_payment
            ELSE s.total_amount - s.credit_amount
        END
    ), 0)::float8 AS received,
    COUNT(*) FILTER (WHERE s.change_amount > 0)::int AS change_count,
    COALESCE(SUM(s.change_amount), 0)::float8 AS change_total
FROM sales s
WHERE s.cash_session_id = @session_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
GROUP BY s.payment_method;

-- Parcelas de vendas a prazo recebidas no turno
-- name: GetCashSessionReceiptsByMethod :many
SELECT COALESCE(pm.type::text, 'other')::text AS payment_method,
    COUNT(*)::int AS receipts_count,
    COALESCE(SUM(ph.amount_paid), 0)::float8 AS received
FROM payment_history ph
    LEFT JOIN payment_methods pm ON pm.id = ph.payment_method_id
WHERE ph.cash_session_id = @session_id
GROUP BY COALESCE(pm.type::text, 'other');

-- Dinheiro devolvido a clientes em trocas/devoluções do turno
-- name: GetCashSessionRefundsByMethod :many
SELECT COALESCE(sr.refund_method, 'other')::text AS payment_method,
    COUNT(*)::int AS refunds_count,
    COALESCE(SUM(sr.refund_amount), 0)::float8 AS refunded
FROM sale_returns sr
WHERE sr.cash_session_id = @session_id
    AND sr.refund_amount > 0
GROUP BY sr.refund_method;

-- Quanto cada operador vendeu e recebeu no turno
-- name: GetCashSessionByOperator :many
WITH ops AS (
    SELECT s.created_by AS user_id,
        1 AS sales_count,
        s.total_amount AS sales_total,
        0::numeric AS receipts_total
    FROM sales s
    WHERE s.cash_session_id = @session_id
        AND s.deleted_at IS NULL
        AND s.status <> 'canceled'
    UNION ALL
    SELECT ph.user_id,
        0,
        0,
        ph.amount_paid
    FROM payment_history ph
    WHERE ph.cash_session_id = @session_id
)
SELECT o.user_id,
    u.name::text AS user_name,
    SUM(o.sales_count)::int AS sales_count,
    COALESCE(SUM(o.sales_total), 0)::float8 AS sales_total,
    COALESCE(SUM(o.receipts_total), 0)::float8 AS receipts_total
FROM ops o
    INNER JOIN users u ON u.id = o.user_id
GROUP BY o.user_id, u.name
ORDER BY sales_total DESC;
