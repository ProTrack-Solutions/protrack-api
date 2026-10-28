-- name: CustomerOverview :one
-- Indicadores da tela de clientes. Os limites do mês chegam calculados no fuso
-- da empresa.
SELECT COUNT(*) FILTER (WHERE c.deleted_at IS NULL)::int AS active_count,
    COUNT(*) FILTER (WHERE c.deleted_at IS NOT NULL)::int AS inactive_count,
    COUNT(*) FILTER (
        WHERE c.deleted_at IS NULL
            AND c.created_at >= sqlc.arg(month_start)::timestamptz
    )::int AS new_this_month,
    COUNT(*) FILTER (
        WHERE c.deleted_at IS NULL
            AND c.created_at >= sqlc.arg(last_month_start)::timestamptz
            AND c.created_at < sqlc.arg(month_start)::timestamptz
    )::int AS new_last_month,
    COUNT(*) FILTER (WHERE c.deleted_at IS NULL AND c.balance_due > 0)::int AS debtors_count,
    COALESCE(SUM(c.balance_due) FILTER (WHERE c.deleted_at IS NULL AND c.balance_due > 0), 0)::float8 AS debt_total,
    COUNT(*) FILTER (WHERE c.store_credit > 0)::int AS credit_count,
    COALESCE(SUM(c.store_credit), 0)::float8 AS credit_total,
    (
        SELECT COUNT(DISTINCT ar.customer_id)
        FROM accounts_receivable ar
        WHERE ar.company_id = sqlc.arg(company_id)
            AND ar.deleted_at IS NULL
            AND ar.status NOT IN ('paid', 'canceled')
            AND ar.balance > 0
            AND ar.due_date < sqlc.arg(today)::date
    )::int AS overdue_count,
    (
        SELECT COALESCE(SUM(ar.balance), 0)
        FROM accounts_receivable ar
        WHERE ar.company_id = sqlc.arg(company_id)
            AND ar.deleted_at IS NULL
            AND ar.status NOT IN ('paid', 'canceled')
            AND ar.balance > 0
            AND ar.due_date < sqlc.arg(today)::date
    )::float8 AS overdue_total,
    (
        SELECT COUNT(DISTINCT s.customer_id)
        FROM sales s
        WHERE s.company_id = sqlc.arg(company_id)
            AND s.customer_id IS NOT NULL
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
            AND s.created_at >= sqlc.arg(recent_since)::timestamptz
    )::int AS recent_buyers
FROM customers c
WHERE c.company_id = sqlc.arg(company_id);

-- name: BrowseCustomers :many
-- Lista de clientes com o resumo de compras e do financeiro de cada um.
-- Total comprado já desconta as devoluções.
WITH purchases AS (
    SELECT s.customer_id,
        COUNT(*) AS purchases_count,
        SUM(s.total_amount) AS purchases_total,
        MAX(s.created_at) AS last_purchase_at
    FROM sales s
    WHERE s.company_id = sqlc.arg(company_id)
        AND s.customer_id IS NOT NULL
        AND s.deleted_at IS NULL
        AND s.status <> 'canceled'
    GROUP BY s.customer_id
),
returned AS (
    SELECT s.customer_id,
        SUM(sr.returned_total) AS returned_total
    FROM sale_returns sr
        INNER JOIN sales s ON s.id = sr.sale_id
    WHERE sr.company_id = sqlc.arg(company_id)
        AND s.customer_id IS NOT NULL
    GROUP BY s.customer_id
),
overdue AS (
    SELECT ar.customer_id,
        SUM(ar.balance) AS overdue_total
    FROM accounts_receivable ar
    WHERE ar.company_id = sqlc.arg(company_id)
        AND ar.deleted_at IS NULL
        AND ar.status NOT IN ('paid', 'canceled')
        AND ar.balance > 0
        AND ar.due_date < sqlc.arg(today)::date
    GROUP BY ar.customer_id
)
SELECT c.id,
    c.full_name::text AS full_name,
    c.cpf::text AS cpf,
    COALESCE(c.email, '')::text AS email,
    COALESCE(c.whatsapp, '')::text AS whatsapp,
    COALESCE(c.mobile_phone, '')::text AS mobile_phone,
    COALESCE(c.address_city, '')::text AS address_city,
    COALESCE(c.address_state, '')::text AS address_state,
    COALESCE(c.balance_due, 0)::float8 AS balance_due,
    c.store_credit::float8 AS store_credit,
    c.created_at,
    c.deleted_at,
    COALESCE(p.purchases_count, 0)::int AS purchases_count,
    (COALESCE(p.purchases_total, 0) - COALESCE(r.returned_total, 0))::float8 AS purchases_total,
    p.last_purchase_at::timestamptz AS last_purchase_at,
    COALESCE(o.overdue_total, 0)::float8 AS overdue_total,
    COUNT(*) OVER () AS total_rows
FROM customers c
    LEFT JOIN purchases p ON p.customer_id = c.id
    LEFT JOIN returned r ON r.customer_id = c.id
    LEFT JOIN overdue o ON o.customer_id = c.id
WHERE c.company_id = sqlc.arg(company_id)
    AND (
        sqlc.arg(status)::text = 'all'
        OR (sqlc.arg(status)::text = 'active' AND c.deleted_at IS NULL)
        OR (sqlc.arg(status)::text = 'inactive' AND c.deleted_at IS NOT NULL)
    )
    AND (
        sqlc.arg(segment)::text = 'all'
        OR (sqlc.arg(segment)::text = 'debtors' AND c.balance_due > 0)
        OR (sqlc.arg(segment)::text = 'overdue' AND o.overdue_total > 0)
        OR (sqlc.arg(segment)::text = 'credit' AND c.store_credit > 0)
        OR (sqlc.arg(segment)::text = 'no_purchases' AND p.customer_id IS NULL)
    )
    AND (
        sqlc.arg(search)::text = ''
        OR unaccent(c.full_name) ILIKE '%' || unaccent(sqlc.arg(search)::text) || '%'
        OR (sqlc.arg(search_digits)::text <> '' AND regexp_replace(c.cpf, '\D', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%')
        OR c.email ILIKE '%' || sqlc.arg(search)::text || '%'
        OR (sqlc.arg(search_digits)::text <> '' AND regexp_replace(COALESCE(c.whatsapp, '') || ' ' || COALESCE(c.mobile_phone, ''), '\D', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%')
    )
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'name' THEN unaccent(c.full_name) END ASC,
    CASE WHEN sqlc.arg(sort)::text = 'last_purchase' THEN p.last_purchase_at END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort)::text = 'total_spent' THEN COALESCE(p.purchases_total, 0) - COALESCE(r.returned_total, 0) END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'balance' THEN c.balance_due END DESC,
    c.created_at DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: GetCustomerProfile :one
SELECT c.id,
    c.full_name::text AS full_name,
    c.cpf::text AS cpf,
    COALESCE(c.rg, '')::text AS rg,
    c.birth_date,
    COALESCE(c.gender::text, '')::text AS gender,
    COALESCE(c.marital_status, '')::text AS marital_status,
    COALESCE(c.email, '')::text AS email,
    COALESCE(c.whatsapp, '')::text AS whatsapp,
    COALESCE(c.mobile_phone, '')::text AS mobile_phone,
    COALESCE(c.home_phone, '')::text AS home_phone,
    COALESCE(c.address_street, '')::text AS address_street,
    COALESCE(c.address_number, '')::text AS address_number,
    COALESCE(c.address_complement, '')::text AS address_complement,
    COALESCE(c.address_neighborhood, '')::text AS address_neighborhood,
    COALESCE(c.address_city, '')::text AS address_city,
    COALESCE(c.address_state, '')::text AS address_state,
    COALESCE(c.address_zipcode, '')::text AS address_zipcode,
    COALESCE(c.balance_due, 0)::float8 AS balance_due,
    c.store_credit::float8 AS store_credit,
    c.created_at,
    c.deleted_at,
    (
        SELECT COUNT(*)
        FROM sales s
        WHERE s.customer_id = c.id
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
    )::int AS purchases_count,
    (
        SELECT COALESCE(SUM(s.total_amount), 0)
        FROM sales s
        WHERE s.customer_id = c.id
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
    )::float8 AS gross_total,
    (
        SELECT COALESCE(SUM(sr.returned_total), 0)
        FROM sale_returns sr
            INNER JOIN sales s ON s.id = sr.sale_id
        WHERE s.customer_id = c.id
    )::float8 AS returned_total,
    (
        SELECT MIN(s.created_at)
        FROM sales s
        WHERE s.customer_id = c.id
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
    )::timestamptz AS first_purchase_at,
    (
        SELECT MAX(s.created_at)
        FROM sales s
        WHERE s.customer_id = c.id
            AND s.deleted_at IS NULL
            AND s.status <> 'canceled'
    )::timestamptz AS last_purchase_at
FROM customers c
WHERE c.id = sqlc.arg(id)
    AND c.company_id = sqlc.arg(company_id);

-- name: ListCustomerOpenInstallments :many
SELECT ar.id,
    ar.sale_id,
    ar.installment_number,
    ar.total_installments,
    ar.due_date,
    ar.total_amount::float8 AS total_amount,
    ar.balance::float8 AS balance,
    ar.status::text AS status
FROM accounts_receivable ar
WHERE ar.customer_id = sqlc.arg(customer_id)
    AND ar.company_id = sqlc.arg(company_id)
    AND ar.deleted_at IS NULL
    AND ar.status NOT IN ('paid', 'canceled')
    AND ar.balance > 0
ORDER BY ar.due_date, ar.installment_number;

-- name: ListCustomerRecentSales :many
SELECT s.id,
    s.created_at,
    s.total_amount::float8 AS total_amount,
    s.payment_method::text AS payment_method,
    s.status::text AS status,
    (SELECT COALESCE(SUM(si.quantity), 0) FROM sale_items si WHERE si.sale_id = s.id)::int AS items_count,
    COALESCE((
        SELECT string_agg(p.name, ', ' ORDER BY p.name)
        FROM sale_items si
            INNER JOIN products p ON p.id = si.product_id
        WHERE si.sale_id = s.id
    ), '')::text AS products,
    (SELECT COALESCE(SUM(sr.returned_total), 0) FROM sale_returns sr WHERE sr.sale_id = s.id)::float8 AS returned_total
FROM sales s
WHERE s.customer_id = sqlc.arg(customer_id)
    AND s.company_id = sqlc.arg(company_id)
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
ORDER BY s.created_at DESC
LIMIT 10;

-- name: SetCustomerActive :execrows
-- Inativar mantém o histórico; reativar volta o cliente para as listas
UPDATE customers
SET deleted_at = CASE WHEN sqlc.arg(active)::boolean THEN NULL ELSE CURRENT_TIMESTAMP END,
    deleted_by = CASE WHEN sqlc.arg(active)::boolean THEN NULL ELSE sqlc.arg(user_id)::uuid END,
    updated_by = sqlc.arg(user_id)::uuid,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
    AND company_id = sqlc.arg(company_id);

-- name: ListCustomerPurchasedItems :many
-- Itens comprados pelo cliente, já sem o que ele devolveu (base do perfil de compra)
SELECT p.id AS product_id,
    p.name::text AS product_name,
    COALESCE(p.model_name, '')::text AS model_name,
    COALESCE(pc.name, 'Sem categoria')::text AS category_name,
    COALESCE(p.size, '')::text AS size,
    COALESCE(p.color, '')::text AS color,
    (si.quantity - COALESCE(r.quantity, 0))::int AS quantity,
    (si.unit_price * si.quantity - COALESCE(si.discount, 0) - COALESCE(r.total, 0))::float8 AS total,
    s.created_at
FROM sale_items si
    INNER JOIN sales s ON s.id = si.sale_id
    INNER JOIN products p ON p.id = si.product_id
    LEFT JOIN product_categories pc ON pc.id = p.category_id
    LEFT JOIN LATERAL (
        SELECT SUM(ri.quantity) AS quantity,
            SUM(ri.total) AS total
        FROM sale_return_items ri
        WHERE ri.sale_item_id = si.id
    ) r ON TRUE
WHERE s.customer_id = sqlc.arg(customer_id)
    AND s.company_id = sqlc.arg(company_id)
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND si.quantity > COALESCE(r.quantity, 0);

-- name: ListCustomerSaleMoments :many
-- Quando e como o cliente comprou (dia, hora e forma de pagamento)
SELECT s.created_at,
    s.total_amount::float8 AS total_amount,
    s.payment_method::text AS payment_method
FROM sales s
WHERE s.customer_id = sqlc.arg(customer_id)
    AND s.company_id = sqlc.arg(company_id)
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
ORDER BY s.created_at
LIMIT 2000;
