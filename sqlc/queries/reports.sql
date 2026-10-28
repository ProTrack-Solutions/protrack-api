-- name: ReportSales :many
SELECT s.id,
    s.sale_at,
    COALESCE(c.full_name, 'Consumidor final')::TEXT AS customer_name,
    COALESCE(u.name, '')::TEXT AS seller_name,
    s.payment_method::TEXT AS payment_method,
    s.status::TEXT AS status,
    (s.deleted_at IS NOT NULL OR s.status = 'canceled')::BOOLEAN AS canceled,
    s.subtotal::FLOAT AS subtotal,
    s.discount_amount::FLOAT AS discount_amount,
    s.total_amount::FLOAT AS total_amount,
    COALESCE(it.quantity, 0)::INT AS items_quantity,
    COALESCE(it.cost, 0)::FLOAT AS total_cost
FROM sales s
    LEFT JOIN customers c ON c.id = s.customer_id
    LEFT JOIN users u ON u.id = s.created_by
    LEFT JOIN LATERAL (
        SELECT SUM(si.quantity) AS quantity,
            SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price)) AS cost
        FROM sale_items si
            INNER JOIN products p ON p.id = si.product_id
        WHERE si.sale_id = s.id
    ) it ON TRUE
WHERE s.company_id = @company_id
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
ORDER BY s.sale_at;

-- name: ReportSalesByDay :many
-- Um ponto por dia do período, mesmo sem venda (generate_series)
WITH days AS (
    SELECT gs::date AS day
    FROM generate_series(
            (@start_date::date)::timestamp,
            (sqlc.arg(end_date)::date - 1)::timestamp,
            interval '1 day'
        ) AS gs
),
sold AS (
    SELECT (s.sale_at AT TIME ZONE @tz::text)::date AS day,
        COUNT(*) AS sales_count,
        SUM(s.subtotal) AS subtotal,
        SUM(s.discount_amount) AS discount,
        SUM(s.total_amount) AS total,
        SUM(it.cost) AS cost
    FROM sales s
        LEFT JOIN LATERAL (
            SELECT SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price)) AS cost
            FROM sale_items si
                INNER JOIN products p ON p.id = si.product_id
            WHERE si.sale_id = s.id
        ) it ON TRUE
    WHERE s.company_id = @company_id
        AND s.deleted_at IS NULL
        AND s.status <> 'canceled'
        AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
        AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
    GROUP BY 1
)
SELECT d.day::date AS day,
    COALESCE(sd.sales_count, 0)::INT AS sales_count,
    COALESCE(sd.subtotal, 0)::FLOAT AS subtotal,
    COALESCE(sd.discount, 0)::FLOAT AS discount,
    COALESCE(sd.total, 0)::FLOAT AS total,
    COALESCE(sd.cost, 0)::FLOAT AS cost
FROM days d
    LEFT JOIN sold sd ON sd.day = d.day
ORDER BY d.day;

-- name: ReportSalesByPaymentMethod :many
SELECT s.payment_method::TEXT AS payment_method,
    COUNT(*)::INT AS sales_count,
    COALESCE(SUM(s.discount_amount), 0)::FLOAT AS discount,
    COALESCE(SUM(s.total_amount), 0)::FLOAT AS total
FROM sales s
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY s.payment_method
ORDER BY total DESC;

-- name: ReportSalesBySeller :many
SELECT COALESCE(u.name, 'Sem vendedor')::TEXT AS seller_name,
    COUNT(*)::INT AS sales_count,
    COALESCE(SUM(s.discount_amount), 0)::FLOAT AS discount,
    COALESCE(SUM(s.total_amount), 0)::FLOAT AS total,
    COALESCE(SUM(it.cost), 0)::FLOAT AS cost
FROM sales s
    LEFT JOIN users u ON u.id = s.created_by
    LEFT JOIN LATERAL (
        SELECT SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price)) AS cost
        FROM sale_items si
            INNER JOIN products p ON p.id = si.product_id
        WHERE si.sale_id = s.id
    ) it ON TRUE
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY u.id, u.name
ORDER BY total DESC;

-- name: ReportSalesByCustomer :many
SELECT COALESCE(c.full_name, 'Consumidor final')::TEXT AS customer_name,
    COALESCE(c.mobile_phone, c.home_phone, '')::TEXT AS phone,
    COUNT(*)::INT AS sales_count,
    COALESCE(SUM(s.total_amount), 0)::FLOAT AS total,
    MAX(s.sale_at)::timestamptz AS last_sale_at,
    COALESCE(MAX(c.balance_due), 0)::FLOAT AS balance_due
FROM sales s
    LEFT JOIN customers c ON c.id = s.customer_id
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY c.id, c.full_name, c.mobile_phone, c.home_phone
ORDER BY total DESC;

-- name: ReportSalesByProduct :many
-- Receita líquida do item já considera o desconto da venda rateado (sale_items.discount)
SELECT p.name::TEXT AS product_name,
    COALESCE(pc.name, 'Sem categoria')::TEXT AS category_name,
    SUM(si.quantity)::INT AS quantity,
    SUM(si.quantity * si.unit_price)::FLOAT AS gross,
    SUM(COALESCE(si.discount, 0))::FLOAT AS discount,
    SUM(si.quantity * si.unit_price - COALESCE(si.discount, 0))::FLOAT AS net,
    SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price))::FLOAT AS cost
FROM sale_items si
    INNER JOIN sales s ON s.id = si.sale_id
    INNER JOIN products p ON p.id = si.product_id
    LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY p.id, p.name, pc.name
ORDER BY net DESC;

-- name: ReportSalesByCategory :many
SELECT COALESCE(pc.name, 'Sem categoria')::TEXT AS category_name,
    COUNT(DISTINCT p.id)::INT AS products_count,
    SUM(si.quantity)::INT AS quantity,
    SUM(si.quantity * si.unit_price - COALESCE(si.discount, 0))::FLOAT AS net,
    SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price))::FLOAT AS cost
FROM sale_items si
    INNER JOIN sales s ON s.id = si.sale_id
    INNER JOIN products p ON p.id = si.product_id
    LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
GROUP BY pc.id, pc.name
ORDER BY net DESC;

-- name: ReportInventory :many
-- Posição atual do estoque: não depende do período
SELECT p.name::TEXT AS product_name,
    COALESCE(pc.name, 'Sem categoria')::TEXT AS category_name,
    COALESCE(p.barcode, '')::TEXT AS barcode,
    p.unit::TEXT AS unit,
    p.sell_in_bulk,
    COALESCE(p.quantity, 0)::INT AS quantity,
    COALESCE(p.cost_price, 0)::FLOAT AS cost_price,
    COALESCE(p.sale_price, 0)::FLOAT AS sale_price
FROM products p
    LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE p.company_id = @company_id
    AND p.deleted_at IS NULL
ORDER BY p.name;

-- name: ReportStockMovements :many
SELECT sm.created_at,
    p.name::TEXT AS product_name,
    sm.type::TEXT AS type,
    sm.quantity,
    COALESCE(sm.reason, '')::TEXT AS reason,
    COALESCE(u.name, '')::TEXT AS user_name
FROM stock_movements sm
    INNER JOIN products p ON p.id = sm.product_id
    LEFT JOIN users u ON u.id = sm.created_by
WHERE sm.company_id = @company_id
    AND sm.created_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND sm.created_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
ORDER BY sm.created_at;

-- name: ReportPaymentsReceived :many
SELECT ph.payment_date,
    COALESCE(c.full_name, 'Consumidor final')::TEXT AS customer_name,
    COALESCE(pm.name, '')::TEXT AS payment_method_name,
    COALESCE(u.name, '')::TEXT AS user_name,
    ph.amount_paid::FLOAT AS amount_paid,
    COALESCE(ph.notes, '')::TEXT AS notes
FROM payment_history ph
    LEFT JOIN customers c ON c.id = ph.customer_id
    LEFT JOIN users u ON u.id = ph.user_id
    LEFT JOIN payment_methods pm ON pm.id = ph.payment_method_id
WHERE ph.company_id = @company_id
    AND ph.payment_date >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND ph.payment_date < (@end_date::date)::timestamp AT TIME ZONE @tz::text
ORDER BY ph.payment_date;

-- name: ReportOpenInstallments :many
-- Parcelas com saldo em aberto que vencem no período; days_overdue é calculado
-- contra today (data local da empresa) e fica 0 para as que ainda vão vencer
SELECT ar.due_date,
    COALESCE(c.full_name, '')::TEXT AS customer_name,
    COALESCE(c.mobile_phone, c.home_phone, '')::TEXT AS phone,
    COALESCE(ar.installment_number, 0)::INT AS installment_number,
    COALESCE(ar.total_installments, 0)::INT AS total_installments,
    ar.total_amount::FLOAT AS total_amount,
    ar.balance::FLOAT AS balance,
    GREATEST(sqlc.arg(today)::date - ar.due_date, 0)::INT AS days_overdue
FROM accounts_receivable ar
    INNER JOIN sales s ON s.id = ar.sale_id
    LEFT JOIN customers c ON c.id = ar.customer_id
WHERE ar.company_id = @company_id
    AND ar.deleted_at IS NULL
    AND s.deleted_at IS NULL
    AND ar.status NOT IN ('paid', 'canceled')
    AND ar.balance > 0
    AND ar.due_date >= @start_date::date
    AND ar.due_date < @end_date::date
ORDER BY ar.due_date, customer_name;

-- name: ReportBillsPayable :many
SELECT b.due_date,
    COALESCE(v.name, '')::TEXT AS vendor_name,
    COALESCE(bc.name, 'Sem categoria')::TEXT AS category_name,
    COALESCE(b.description, '')::TEXT AS description,
    b.status::TEXT AS status,
    b.amount::FLOAT AS amount,
    COALESCE(b.amount_paid, 0)::FLOAT AS amount_paid,
    b.payment_date
FROM bills_payable b
    LEFT JOIN vendors v ON v.id = b.vendor_id
    LEFT JOIN bill_categories bc ON bc.id = b.category_id
WHERE b.company_id = @company_id
    AND b.due_date >= @start_date::date
    AND b.due_date < @end_date::date
ORDER BY b.due_date;

-- name: ReportCashSessions :many
SELECT cs.opened_at,
    cs.closed_at,
    cs.status::TEXT AS status,
    COALESCE(u.name, '')::TEXT AS user_name,
    cs.opening_amount::FLOAT AS opening_amount,
    COALESCE(cs.sales_count, 0)::INT AS sales_count,
    COALESCE(cs.sales_total, 0)::FLOAT AS sales_total,
    COALESCE(cs.received_total, 0)::FLOAT AS received_total,
    COALESCE(cs.supplies_total, 0)::FLOAT AS supplies_total,
    COALESCE(cs.withdrawals_total, 0)::FLOAT AS withdrawals_total,
    COALESCE(cs.expected_cash, 0)::FLOAT AS expected_cash,
    COALESCE(cs.counted_cash, 0)::FLOAT AS counted_cash,
    COALESCE(cs.difference, 0)::FLOAT AS difference
FROM cash_sessions cs
    LEFT JOIN users u ON u.id = cs.user_id
WHERE cs.company_id = @company_id
    AND cs.opened_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND cs.opened_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text
ORDER BY cs.opened_at;

-- name: ReportDreRevenue :one
-- DRE por competência: vendas pela data da venda
SELECT COUNT(*)::INT AS sales_count,
    COALESCE(SUM(s.subtotal), 0)::FLOAT AS gross,
    COALESCE(SUM(s.discount_amount), 0)::FLOAT AS discount,
    COALESCE(SUM(s.total_amount), 0)::FLOAT AS net,
    COALESCE(SUM(it.cost), 0)::FLOAT AS cost
FROM sales s
    LEFT JOIN LATERAL (
        SELECT SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price)) AS cost
        FROM sale_items si
            INNER JOIN products p ON p.id = si.product_id
        WHERE si.sale_id = s.id
    ) it ON TRUE
WHERE s.company_id = @company_id
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND s.sale_at >= (@start_date::date)::timestamp AT TIME ZONE @tz::text
    AND s.sale_at < (@end_date::date)::timestamp AT TIME ZONE @tz::text;

-- name: ReportDreExpenses :many
-- Despesas por competência: contas a pagar pelo vencimento, exceto canceladas
SELECT COALESCE(bc.name, 'Sem categoria')::TEXT AS category_name,
    COUNT(*)::INT AS bills_count,
    COALESCE(SUM(b.amount), 0)::FLOAT AS total
FROM bills_payable b
    LEFT JOIN bill_categories bc ON bc.id = b.category_id
WHERE b.company_id = @company_id
    AND b.status <> 'canceled'
    AND b.due_date >= @start_date::date
    AND b.due_date < @end_date::date
GROUP BY bc.id, bc.name
ORDER BY total DESC;

-- name: ReportCompanyHeader :one
SELECT COALESCE(NULLIF(trade_name, ''), name)::TEXT AS company_name,
    COALESCE(NULLIF(timezone, ''), 'America/Sao_Paulo')::TEXT AS timezone
FROM companies
WHERE id = $1;
