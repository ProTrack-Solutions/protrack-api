-- name: GetTotalInflowByPeriod :one
SELECT COALESCE(SUM(total_amount), 0)::FLOAT AS total_inflow
FROM sales
WHERE company_id = $1
    AND created_at >= $2  -- Passar '2026-06-18T00:00:00Z'
    AND created_at < $3;
-- name: GetTotalOutflowByPeriod :one
SELECT COALESCE(SUM(amount_paid), 0)::FLOAT AS total_outflow
FROM bills_payable
WHERE company_id = $1
    AND status = 'paid'
    AND payment_date BETWEEN $2 AND $3;
-- name: GetCashInFlowByCategory :many
SELECT pc.name AS category_name,
    COALESCE(SUM(si.unit_price * si.quantity), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT s.id) AS sales_count
FROM product_categories pc
    LEFT JOIN products p ON p.category_id = pc.id
    AND p.company_id = $1
    LEFT JOIN sale_items si ON si.product_id = p.id
    LEFT JOIN sales s ON s.id = si.sale_id
    AND s.company_id = $1
    AND s.status = 'paid'
WHERE pc.company_id = $1
    AND pc.status = 'ACTIVE'
GROUP BY pc.name
ORDER BY total_amount DESC;
-- name: GetCashOutFlowByCategory :many
SELECT bc.name AS category_name,
    COALESCE(SUM(COALESCE(b.amount_paid, b.amount)), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT b.id) AS bills_count
FROM bill_categories bc
    LEFT JOIN bills_payable b ON b.category_id = bc.id
    AND b.company_id = $1
    AND b.status = 'paid'
WHERE bc.company_id = $1
    AND bc.is_active = TRUE
    AND bc.deleted_at IS NULL
GROUP BY bc.name
ORDER BY total_amount DESC;
-- name: GetCashInFlowCategoryByPeriod :many
SELECT pc.name AS category_name,
    COALESCE(SUM(si.unit_price * si.quantity), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT s.id) AS sales_count
FROM product_categories pc
    LEFT JOIN products p ON p.category_id = pc.id
    AND p.company_id = $1
    LEFT JOIN sale_items si ON si.product_id = p.id
    LEFT JOIN sales s ON s.id = si.sale_id
    AND s.company_id = $1
    AND s.status = 'paid'
    AND s.created_at BETWEEN $2 AND $3
WHERE pc.company_id = $1
    AND pc.status = 'ACTIVE'
GROUP BY pc.name
ORDER BY total_amount DESC;
-- name: GetCashOutFlowCategoryByPeriod :many
SELECT bc.name AS category_name,
    COALESCE(SUM(COALESCE(b.amount_paid, b.amount)), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT b.id) AS bills_count
FROM bill_categories bc
    LEFT JOIN bills_payable b ON b.category_id = bc.id
    AND b.company_id = $1
    AND b.status = 'paid'
    AND b.created_at BETWEEN $2 AND $3
WHERE bc.company_id = $1
    AND bc.is_active = TRUE
    AND bc.deleted_at IS NULL
GROUP BY bc.name
ORDER BY total_amount DESC;
-- Consultas da visão geral (/cash-flow/overview). O intervalo chega como datas
-- locais [start_date, end_date) junto com o fuso da empresa, então o "dia" segue
-- o horário dela e não o do servidor. Entrada = venda não cancelada (data da venda)
-- e saída = conta paga (data do pagamento).
-- name: GetCompanyTimezone :one
SELECT COALESCE(NULLIF(timezone, ''), 'America/Sao_Paulo')::TEXT AS timezone
FROM companies
WHERE id = $1;

-- name: GetCashFlowSeries :many
-- Entradas e saídas agrupadas por dia/semana/mês numa única ida ao banco,
-- generate_series garante um ponto por período, mesmo sem movimento.
WITH buckets AS (
    SELECT gs::date AS bucket
    FROM generate_series(
            (@start_date::date)::timestamp,
            (@last_bucket::date)::timestamp,
            (@step::text)::interval
        ) AS gs
),
inflow AS (
    SELECT date_trunc(@unit::text, s.created_at AT TIME ZONE @tz::text)::date AS bucket,
        SUM(s.total_amount - s.credit_amount)::FLOAT AS total,
        COUNT(*)::INT AS movements
    FROM sales s
    WHERE s.company_id = @company_id
        AND s.deleted_at IS NULL
        AND s.status <> 'canceled'
        AND s.total_amount > s.credit_amount
        AND s.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
        AND s.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
    GROUP BY 1
),
outflow AS (
    SELECT o.bucket,
        SUM(o.total)::FLOAT AS total,
        SUM(o.movements)::INT AS movements
    FROM (
            SELECT date_trunc(@unit::text, b.payment_date::timestamp)::date AS bucket,
                SUM(COALESCE(b.amount_paid, b.amount)) AS total,
                COUNT(*) AS movements
            FROM bills_payable b
            WHERE b.company_id = @company_id
                AND b.status = 'paid'
                AND b.payment_date >= @start_date::date
                AND b.payment_date < @end_date::date
            GROUP BY 1
            UNION ALL
            -- Dinheiro devolvido a clientes em trocas/devoluções
            SELECT date_trunc(@unit::text, sr.created_at AT TIME ZONE @tz::text)::date AS bucket,
                SUM(sr.refund_amount) AS total,
                COUNT(*) AS movements
            FROM sale_returns sr
            WHERE sr.company_id = @company_id
                AND sr.refund_amount > 0
                AND sr.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
                AND sr.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
            GROUP BY 1
        ) o
    GROUP BY o.bucket
)
SELECT bk.bucket::date AS bucket,
    COALESCE(i.total, 0)::FLOAT AS inflow,
    COALESCE(o.total, 0)::FLOAT AS outflow,
    COALESCE(i.movements, 0)::INT AS inflow_count,
    COALESCE(o.movements, 0)::INT AS outflow_count
FROM buckets bk
    LEFT JOIN inflow i ON i.bucket = bk.bucket
    LEFT JOIN outflow o ON o.bucket = bk.bucket
ORDER BY bk.bucket;

-- name: GetCashFlowTotalsByRange :one
SELECT (
        SELECT COALESCE(SUM(s.total_amount - s.credit_amount), 0)
        FROM sales s
        WHERE s.company_id = @company_id
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
            AND s.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
            AND s.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
    )::FLOAT AS inflow,
    (
        (
            SELECT COALESCE(SUM(COALESCE(b.amount_paid, b.amount)), 0)
            FROM bills_payable b
            WHERE b.company_id = @company_id
                AND b.status = 'paid'
                AND b.payment_date >= @start_date::date
                AND b.payment_date < @end_date::date
        ) + (
            SELECT COALESCE(SUM(sr.refund_amount), 0)
            FROM sale_returns sr
            WHERE sr.company_id = @company_id
                AND sr.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
                AND sr.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
        )
    )::FLOAT AS outflow;

-- name: GetCashInFlowCategoryByRange :many
SELECT COALESCE(pc.name, 'Sem categoria')::TEXT AS category_name,
    SUM(si.unit_price * si.quantity)::FLOAT AS total_amount
FROM sales s
    JOIN sale_items si ON si.sale_id = s.id
    LEFT JOIN products p ON p.id = si.product_id
    LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY 1
ORDER BY total_amount DESC;

-- name: GetCashOutFlowCategoryByRange :many
SELECT o.category_name::TEXT AS category_name,
    SUM(o.total)::FLOAT AS total_amount
FROM (
        SELECT COALESCE(bc.name, 'Sem categoria') AS category_name,
            COALESCE(b.amount_paid, b.amount) AS total
        FROM bills_payable b
            LEFT JOIN bill_categories bc ON bc.id = b.category_id
        WHERE b.company_id = @company_id
            AND b.status = 'paid'
            AND b.payment_date >= @start_date::date
            AND b.payment_date < @end_date::date
        UNION ALL
        SELECT 'Devoluções a clientes' AS category_name,
            sr.refund_amount AS total
        FROM sale_returns sr
        WHERE sr.company_id = @company_id
            AND sr.refund_amount > 0
            AND sr.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
            AND sr.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
    ) o
GROUP BY 1
ORDER BY total_amount DESC;

-- name: GetCashInFlowPaymentMethodByRange :many
SELECT s.payment_method::TEXT AS payment_method,
    SUM(s.total_amount - s.credit_amount)::FLOAT AS total_amount,
    COUNT(*)::INT AS sales_count
FROM sales s
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    -- Parte paga com vale-troca não é dinheiro novo
    AND s.total_amount > s.credit_amount
    AND s.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY 1
ORDER BY total_amount DESC;

-- name: GetCashFlowForecast :many
-- O que está previsto entrar (parcelas em aberto) e sair (contas em aberto)
-- em cada dia de [start_date, end_date)
WITH days AS (
    SELECT gs::date AS day
    FROM generate_series(
            (@start_date::date)::timestamp,
            (@end_date::date)::timestamp - INTERVAL '1 day',
            INTERVAL '1 day'
        ) AS gs
),
receivable AS (
    SELECT ar.due_date AS day,
        SUM(ar.balance)::FLOAT AS total
    FROM accounts_receivable ar
    WHERE ar.company_id = @company_id
        AND ar.deleted_at IS NULL
        AND ar.status IN ('pending', 'partial', 'overdue')
        AND ar.due_date >= @start_date::date
        AND ar.due_date < @end_date::date
    GROUP BY 1
),
payable AS (
    SELECT b.due_date AS day,
        SUM(b.amount - COALESCE(b.amount_paid, 0))::FLOAT AS total
    FROM bills_payable b
    WHERE b.company_id = @company_id
        AND b.status NOT IN ('paid', 'canceled')
        AND b.due_date >= @start_date::date
        AND b.due_date < @end_date::date
    GROUP BY 1
)
SELECT d.day::date AS day,
    COALESCE(r.total, 0)::FLOAT AS receivable,
    COALESCE(p.total, 0)::FLOAT AS payable
FROM days d
    LEFT JOIN receivable r ON r.day = d.day
    LEFT JOIN payable p ON p.day = d.day
ORDER BY d.day;

-- name: GetCashFlowOverdue :one
-- Valores em aberto com vencimento antes de hoje
SELECT (
        SELECT COALESCE(SUM(ar.balance), 0)
        FROM accounts_receivable ar
        WHERE ar.company_id = @company_id
            AND ar.deleted_at IS NULL
            AND ar.status IN ('pending', 'partial', 'overdue')
            AND ar.due_date < @today::date
    )::FLOAT AS receivable,
    (
        SELECT COALESCE(SUM(b.amount - COALESCE(b.amount_paid, 0)), 0)
        FROM bills_payable b
        WHERE b.company_id = @company_id
            AND b.status NOT IN ('paid', 'canceled')
            AND b.due_date < @today::date
    )::FLOAT AS payable;

-- name: ListRecentCashMovements :many
-- Últimas entradas e saídas misturadas, para o extrato da tela
SELECT m.id,
    m.kind,
    m.description,
    m.detail,
    m.amount,
    m.occurred_at
FROM (
        (
            SELECT s.id,
                'inflow'::TEXT AS kind,
                COALESCE(c.full_name, 'Consumidor final')::TEXT AS description,
                s.payment_method::TEXT AS detail,
                (s.total_amount - s.credit_amount)::FLOAT AS amount,
                s.created_at AS occurred_at
            FROM sales s
                LEFT JOIN customers c ON c.id = s.customer_id
            WHERE s.company_id = @company_id
                AND s.deleted_at IS NULL
                AND s.status <> 'canceled'
                AND s.total_amount > s.credit_amount
            ORDER BY s.created_at DESC
            LIMIT @row_limit::int
        )
        UNION ALL
        (
            SELECT sr.id,
                'outflow'::TEXT AS kind,
                ('Devolução nº ' || sr.number)::TEXT AS description,
                COALESCE(sr.refund_method, 'other')::TEXT AS detail,
                sr.refund_amount::FLOAT AS amount,
                sr.created_at AS occurred_at
            FROM sale_returns sr
            WHERE sr.company_id = @company_id
                AND sr.refund_amount > 0
            ORDER BY sr.created_at DESC
            LIMIT @row_limit::int
        )
        UNION ALL
        (
            SELECT b.id,
                'outflow'::TEXT AS kind,
                COALESCE(NULLIF(b.description, ''), v.name, 'Conta paga')::TEXT AS description,
                COALESCE(bc.name, 'Sem categoria')::TEXT AS detail,
                COALESCE(b.amount_paid, b.amount)::FLOAT AS amount,
                (b.payment_date::timestamp AT TIME ZONE @tz::text) AS occurred_at
            FROM bills_payable b
                LEFT JOIN vendors v ON v.id = b.vendor_id
                LEFT JOIN bill_categories bc ON bc.id = b.category_id
            WHERE b.company_id = @company_id
                AND b.status = 'paid'
                AND b.payment_date IS NOT NULL
            ORDER BY b.payment_date DESC, b.updated_at DESC
            LIMIT @row_limit::int
        )
    ) AS m
ORDER BY m.occurred_at DESC
LIMIT @row_limit::int;
