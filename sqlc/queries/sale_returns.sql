-- name: LockCompanyReturnNumbers :exec
-- Serializa a numeração das devoluções da empresa até o fim da transação
SELECT pg_advisory_xact_lock(hashtext('sale_returns:' || sqlc.arg(company_id)::TEXT));

-- name: NextSaleReturnNumber :one
SELECT (COALESCE(MAX(number), 0) + 1)::INT AS number
FROM sale_returns
WHERE company_id = $1;

-- name: GetSaleForReturn :one
-- Trava a venda: duas devoluções simultâneas não devolvem o mesmo item duas vezes
SELECT s.id,
    s.company_id,
    s.customer_id,
    s.payment_method::text AS payment_method,
    s.status::text AS status,
    s.total_amount,
    s.created_at,
    s.deleted_at
FROM sales s
WHERE s.id = $1
    AND s.company_id = $2
FOR UPDATE;

-- name: GetSaleReturnHeader :one
-- Dados da venda mostrados na tela de devolução
SELECT s.id,
    s.customer_id,
    COALESCE(c.full_name, '')::text AS customer_name,
    COALESCE(c.store_credit, 0)::float8 AS customer_credit,
    s.payment_method::text AS payment_method,
    s.status::text AS status,
    s.subtotal::float8 AS subtotal,
    s.discount_amount::float8 AS discount_amount,
    s.total_amount::float8 AS total_amount,
    s.sale_at,
    s.created_at,
    s.deleted_at,
    EXISTS (
        SELECT 1
        FROM sale_returns sr
        WHERE sr.exchange_sale_id = s.id
    ) AS is_exchange_sale
FROM sales s
    LEFT JOIN customers c ON c.id = s.customer_id
WHERE s.id = $1
    AND s.company_id = $2;

-- name: ListReturnableItems :many
-- Itens da venda com o que já foi devolvido. net_total é o valor pago pela
-- linha (preço menos o desconto rateado da venda).
SELECT si.id,
    si.product_id,
    p.name::text AS product_name,
    p.sell_in_bulk,
    si.quantity,
    si.unit_price::float8 AS unit_price,
    COALESCE(si.discount, 0)::float8 AS discount,
    (si.unit_price * si.quantity - COALESCE(si.discount, 0))::float8 AS net_total,
    COALESCE(r.quantity, 0)::int AS returned_quantity,
    COALESCE(r.total, 0)::float8 AS returned_total
FROM sale_items si
    INNER JOIN products p ON p.id = si.product_id
    LEFT JOIN LATERAL (
        SELECT SUM(ri.quantity) AS quantity,
            SUM(ri.total) AS total
        FROM sale_return_items ri
        WHERE ri.sale_item_id = si.id
    ) r ON TRUE
WHERE si.sale_id = $1
ORDER BY p.name, si.id;

-- name: ListOpenReceivablesForReturn :many
-- Parcelas em aberto da venda, da última para a primeira: a devolução abate
-- primeiro as que vencem mais tarde
SELECT id,
    total_amount::float8 AS total_amount,
    balance::float8 AS balance
FROM accounts_receivable
WHERE sale_id = $1
    AND company_id = $2
    AND status NOT IN ('paid', 'canceled')
    AND deleted_at IS NULL
    AND balance > 0
ORDER BY installment_number DESC
FOR UPDATE;

-- name: ReduceReceivableForReturn :exec
-- Parcela zerada pela devolução: sem nada pago ela é cancelada; com pagamento
-- parcial fica quitada
UPDATE accounts_receivable
SET balance = sqlc.arg(balance),
    status = CASE
        WHEN sqlc.arg(balance)::numeric > 0 THEN status
        WHEN sqlc.arg(balance)::numeric = 0 AND total_amount > balance THEN 'paid'
        ELSE 'canceled'
    END,
    deleted_at = CASE
        WHEN sqlc.arg(balance)::numeric = 0 AND total_amount <= balance THEN CURRENT_TIMESTAMP
        ELSE deleted_at
    END,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = sqlc.arg(updated_by)
WHERE id = sqlc.arg(id);

-- name: GetOpenBalanceForReturn :one
SELECT COALESCE(SUM(balance), 0)::float8 AS open_balance
FROM accounts_receivable
WHERE sale_id = $1
    AND company_id = $2
    AND status NOT IN ('paid', 'canceled')
    AND deleted_at IS NULL;

-- name: SetSaleStatusForReturn :exec
UPDATE sales
SET status = sqlc.arg(status)::account_status_enum,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = sqlc.arg(updated_by)
WHERE id = sqlc.arg(id);

-- name: SubtractCustomerBalanceDue :exec
UPDATE customers
SET balance_due = GREATEST(balance_due - sqlc.arg(amount), 0),
    updated_at = CURRENT_TIMESTAMP,
    updated_by = sqlc.arg(updated_by)
WHERE id = sqlc.arg(id);

-- name: GetCustomerForCredit :one
SELECT id,
    company_id,
    full_name::text AS full_name,
    store_credit::float8 AS store_credit
FROM customers
WHERE id = $1
    AND company_id = $2
    AND deleted_at IS NULL
FOR UPDATE;

-- name: AddCustomerStoreCredit :one
-- amount negativo usa o crédito; o CHECK (store_credit >= 0) barra saldo negativo
UPDATE customers
SET store_credit = store_credit + sqlc.arg(amount),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING store_credit::float8 AS store_credit;

-- name: CreateCustomerCreditMovement :exec
INSERT INTO customer_credit_movements (
        company_id,
        customer_id,
        type,
        amount,
        balance_after,
        sale_return_id,
        sale_id,
        created_by
    )
VALUES (
        sqlc.arg(company_id),
        sqlc.arg(customer_id),
        sqlc.arg(type),
        sqlc.arg(amount),
        sqlc.arg(balance_after),
        sqlc.narg(sale_return_id),
        sqlc.narg(sale_id),
        sqlc.narg(created_by)
    );

-- name: ListCustomerCreditMovements :many
SELECT m.id,
    m.type,
    m.amount::float8 AS amount,
    m.balance_after::float8 AS balance_after,
    m.sale_return_id,
    sr.number AS return_number,
    m.sale_id,
    m.created_at,
    COALESCE(u.name, '')::text AS created_by_name
FROM customer_credit_movements m
    LEFT JOIN sale_returns sr ON sr.id = m.sale_return_id
    LEFT JOIN users u ON u.id = m.created_by
WHERE m.customer_id = $1
    AND m.company_id = $2
ORDER BY m.created_at DESC
LIMIT 50;

-- name: GetCustomerCredit :one
SELECT id,
    full_name::text AS full_name,
    store_credit::float8 AS store_credit
FROM customers
WHERE id = $1
    AND company_id = $2;

-- name: ListProductsForExchange :many
SELECT id,
    name::text AS name,
    sale_price::float8 AS sale_price,
    sell_in_bulk,
    COALESCE(quantity, 0)::int AS quantity
FROM products
WHERE company_id = sqlc.arg(company_id)
    AND id = ANY(sqlc.arg(ids)::UUID[])
    AND deleted_at IS NULL;

-- name: DecrementStockForExchange :execrows
UPDATE products
SET quantity = quantity - sqlc.arg(quantity)
WHERE id = sqlc.arg(id)
    AND quantity >= sqlc.arg(quantity);

-- name: IncrementStockForReturn :exec
UPDATE products
SET quantity = quantity + sqlc.arg(quantity)
WHERE id = sqlc.arg(id);

-- name: CreateReturnStockMovement :exec
INSERT INTO stock_movements (
        company_id,
        product_id,
        sale_id,
        sale_return_id,
        type,
        quantity,
        reason,
        created_by
    )
VALUES (
        sqlc.arg(company_id),
        sqlc.arg(product_id),
        sqlc.narg(sale_id),
        sqlc.narg(sale_return_id),
        sqlc.arg(type),
        sqlc.arg(quantity),
        sqlc.narg(reason),
        sqlc.arg(created_by)
    );

-- name: CreateExchangeSale :one
-- Venda com os itens levados na troca. credit_amount é a parte paga com o
-- valor dos itens devolvidos; o resto (diferença) na forma informada.
INSERT INTO sales (
        customer_id,
        company_id,
        sale_at,
        discount_amount,
        subtotal,
        total_amount,
        installments_count,
        down_payment,
        payment_method,
        created_by,
        status,
        amount_received,
        change_amount,
        credit_amount
    )
VALUES (
        sqlc.narg(customer_id),
        sqlc.arg(company_id),
        CURRENT_DATE,
        0,
        sqlc.arg(total_amount),
        sqlc.arg(total_amount),
        0,
        0,
        sqlc.arg(payment_method)::payment_method_enum,
        sqlc.arg(created_by),
        'paid',
        sqlc.narg(amount_received),
        sqlc.narg(change_amount),
        sqlc.arg(credit_amount)
    )
RETURNING id;

-- name: CreateExchangeSaleItem :one
INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, discount, unit_cost)
SELECT sqlc.arg(sale_id),
    p.id,
    sqlc.arg(quantity),
    sqlc.arg(unit_price),
    0,
    p.cost_price
FROM products p
WHERE p.id = sqlc.arg(product_id)
RETURNING id;

-- name: CreateSaleReturn :one
INSERT INTO sale_returns (
        company_id,
        number,
        kind,
        sale_id,
        exchange_sale_id,
        customer_id,
        returned_total,
        exchange_total,
        exchange_credit,
        debt_reduction,
        credit_amount,
        refund_amount,
        refund_method,
        difference_paid,
        difference_method,
        reason,
        notes,
        out_of_window,
        override_reason,
        created_by
    )
VALUES (
        sqlc.arg(company_id),
        sqlc.arg(number),
        sqlc.arg(kind),
        sqlc.arg(sale_id),
        sqlc.narg(exchange_sale_id),
        sqlc.narg(customer_id),
        sqlc.arg(returned_total),
        sqlc.arg(exchange_total),
        sqlc.arg(exchange_credit),
        sqlc.arg(debt_reduction),
        sqlc.arg(credit_amount),
        sqlc.arg(refund_amount),
        sqlc.narg(refund_method),
        sqlc.arg(difference_paid),
        sqlc.narg(difference_method),
        sqlc.arg(reason),
        sqlc.narg(notes),
        sqlc.arg(out_of_window),
        sqlc.narg(override_reason),
        sqlc.arg(created_by)
    )
RETURNING id;

-- name: CreateSaleReturnItem :exec
INSERT INTO sale_return_items (return_id, sale_item_id, product_id, quantity, unit_amount, total, restock)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListSaleReturns :many
SELECT sr.id,
    sr.number,
    sr.kind,
    sr.sale_id,
    sr.exchange_sale_id,
    sr.customer_id,
    COALESCE(c.full_name, sc.full_name, '')::text AS customer_name,
    sr.returned_total::float8 AS returned_total,
    sr.exchange_total::float8 AS exchange_total,
    sr.debt_reduction::float8 AS debt_reduction,
    sr.credit_amount::float8 AS credit_amount,
    sr.refund_amount::float8 AS refund_amount,
    sr.refund_method,
    sr.difference_paid::float8 AS difference_paid,
    sr.reason,
    sr.out_of_window,
    sr.created_at,
    COALESCE(u.name, '')::text AS created_by_name,
    (SELECT COALESCE(SUM(ri.quantity), 0) FROM sale_return_items ri WHERE ri.return_id = sr.id)::int AS items_count,
    COUNT(*) OVER () AS total_rows
FROM sale_returns sr
    INNER JOIN sales s ON s.id = sr.sale_id
    LEFT JOIN customers c ON c.id = sr.customer_id
    LEFT JOIN customers sc ON sc.id = s.customer_id
    LEFT JOIN users u ON u.id = sr.created_by
WHERE sr.company_id = sqlc.arg(company_id)
    AND (sqlc.narg(kind)::sale_return_kind IS NULL OR sr.kind = sqlc.narg(kind)::sale_return_kind)
    AND (sqlc.narg(start_date)::date IS NULL OR sr.created_at >= sqlc.narg(start_date)::date)
    AND (sqlc.narg(end_date)::date IS NULL OR sr.created_at < sqlc.narg(end_date)::date + 1)
    AND (
        sqlc.arg(search)::text = ''
        OR sr.number::text = sqlc.arg(search)::text
        OR COALESCE(c.full_name, sc.full_name, '') ILIKE '%' || sqlc.arg(search)::text || '%'
        OR EXISTS (
            SELECT 1
            FROM sale_return_items ri
                INNER JOIN products p ON p.id = ri.product_id
            WHERE ri.return_id = sr.id
                AND p.name ILIKE '%' || sqlc.arg(search)::text || '%'
        )
        -- Itens levados na troca
        OR EXISTS (
            SELECT 1
            FROM sale_items si
                INNER JOIN products p ON p.id = si.product_id
            WHERE si.sale_id = sr.exchange_sale_id
                AND p.name ILIKE '%' || sqlc.arg(search)::text || '%'
        )
    )
ORDER BY sr.created_at DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: GetSaleReturn :one
SELECT sr.*,
    COALESCE(c.full_name, sc.full_name, '')::text AS customer_name,
    COALESCE(u.name, '')::text AS created_by_name,
    s.created_at AS sale_created_at,
    s.total_amount::float8 AS sale_total
FROM sale_returns sr
    INNER JOIN sales s ON s.id = sr.sale_id
    LEFT JOIN customers c ON c.id = sr.customer_id
    LEFT JOIN customers sc ON sc.id = s.customer_id
    LEFT JOIN users u ON u.id = sr.created_by
WHERE sr.id = $1
    AND sr.company_id = $2;

-- name: ListSaleReturnItems :many
SELECT ri.id,
    ri.product_id,
    p.name::text AS product_name,
    ri.quantity,
    ri.unit_amount::float8 AS unit_amount,
    ri.total::float8 AS total,
    ri.restock
FROM sale_return_items ri
    INNER JOIN products p ON p.id = ri.product_id
WHERE ri.return_id = $1
ORDER BY p.name;

-- name: ListExchangeSaleItems :many
SELECT si.product_id,
    p.name::text AS product_name,
    si.quantity,
    si.unit_price::float8 AS unit_price,
    (si.unit_price * si.quantity)::float8 AS total
FROM sale_items si
    INNER JOIN products p ON p.id = si.product_id
WHERE si.sale_id = $1
ORDER BY p.name;

-- name: ListSaleReturnsBySale :many
SELECT id,
    number,
    kind,
    returned_total::float8 AS returned_total,
    created_at
FROM sale_returns
WHERE sale_id = $1
    AND company_id = $2
ORDER BY created_at;

-- name: SearchSalesForReturn :many
-- Vendas que ainda podem ter itens devolvidos (não canceladas e com saldo de itens)
SELECT s.id,
    s.created_at,
    s.total_amount::float8 AS total_amount,
    s.payment_method::text AS payment_method,
    COALESCE(c.full_name, '')::text AS customer_name,
    (SELECT COALESCE(SUM(si.quantity), 0) FROM sale_items si WHERE si.sale_id = s.id)::int AS items_count,
    (SELECT COALESCE(SUM(ri.quantity), 0)
     FROM sale_return_items ri
        INNER JOIN sale_items si ON si.id = ri.sale_item_id
     WHERE si.sale_id = s.id)::int AS returned_count,
    (SELECT string_agg(p.name, ', ' ORDER BY p.name)
     FROM sale_items si
        INNER JOIN products p ON p.id = si.product_id
     WHERE si.sale_id = s.id)::text AS products,
    COUNT(*) OVER () AS total_rows
FROM sales s
    LEFT JOIN customers c ON c.id = s.customer_id
WHERE s.company_id = sqlc.arg(company_id)
    AND s.deleted_at IS NULL
    AND s.status <> 'canceled'
    AND (
        sqlc.arg(search)::text = ''
        OR s.id::text ILIKE sqlc.arg(search)::text || '%'
        OR c.full_name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR EXISTS (
            SELECT 1
            FROM sale_items si
                INNER JOIN products p ON p.id = si.product_id
            WHERE si.sale_id = s.id
                AND p.name ILIKE '%' || sqlc.arg(search)::text || '%'
        )
    )
ORDER BY s.created_at DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountSaleReturnsBySale :one
-- Vendas com devolução ou geradas por troca não podem ser canceladas nem editadas
SELECT (
        SELECT COUNT(*)
        FROM sale_returns sr
        WHERE sr.sale_id = sqlc.arg(sale_id)
    )::int AS returns_count,
    EXISTS (
        SELECT 1
        FROM sale_returns sr
        WHERE sr.exchange_sale_id = sqlc.arg(sale_id)
    ) AS is_exchange_sale;
