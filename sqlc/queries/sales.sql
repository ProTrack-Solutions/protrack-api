-- name: CreateSale :one
INSERT INTO sales (
        customer_id,
        company_id,
        sale_at,
        discount_amount,
        subtotal,
        total_amount,
        installments_count,
        down_payment,
        due_days,
        payment_method,
        created_by,
        status
    )
VALUES (
        $1,
        $2,
        CURRENT_DATE,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11
    )
RETURNING id;
-- name: ListSales :many
SELECT id,
    sale_at,
    total_amount,
    status,
    created_at
FROM sales
WHERE company_id = $1
    AND deleted_at IS NULL
ORDER BY created_at DESC;
-- name: GetSaleById :one
SELECT s.*,
    c.full_name as customer_name
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
WHERE s.id = $1
    AND s.company_id = $2;
-- name: DeleteSale :exec
UPDATE sales
SET deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $1
WHERE id = $2
    AND company_id = $3;
-- name: UpdateSaleStatus :exec
UPDATE sales
SET status = $1,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $2
WHERE id = $3
    AND company_id = $4;
-- name: ListSalesByCompanyAndStatus :many
SELECT s.id AS sale_id,
    s.total_amount,
    s.discount_amount,
    s.status,
    s.sale_at,
    s.created_at AS sale_date,
    si.id AS item_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount,
    p.name AS product_name,
    c.full_name AS customer_name
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
    INNER JOIN sale_items si ON s.id = si.sale_id
    INNER JOIN products p ON si.product_id = p.id
WHERE s.company_id = $1
    AND (
        (
            $2::text IS NULL
            OR $2::text = ''
        )
        OR s.status::text = $2::text
    )
ORDER BY s.created_at DESC;
-- name: CountSales :one
SELECT COUNT(*)
FROM sales
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: GetSalesPerformanceSummary :one
SELECT COUNT(*) FILTER (
        WHERE date_trunc('month', sale_at) = date_trunc('month', CURRENT_DATE)
    ) AS current_month_count,
    COUNT(*) FILTER (
        WHERE date_trunc('month', sale_at) = date_trunc('month', CURRENT_DATE - INTERVAL '1 month')
    ) AS last_month_count
FROM sales
WHERE company_id = $1
    AND deleted_at IS NULL
    AND sale_at >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month');
-- name: GetTotalAmountSummary :one
SELECT coalesce(
        SUM(total_amount) FILTER (
            WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)
        ),
        0
    )::FLOAT AS current_month_st,
    coalesce(
        SUM(total_amount) FILTER (
            WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE - INTERVAL '1 month')
        ),
        0
    )::FLOAT AS last_month_st
FROM sales
WHERE company_id = $1
    AND deleted_at IS NULL
    AND created_at >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month');
-- name: GetTotalAmountByStatus :one
SELECT COALESCE(
        SUM(total_amount) FILTER (
            WHERE status = $2
                AND company_id = $1
                AND deleted_at IS NULL
        ),
        0
    )::FLOAT AS total_pending_amount
from sales;
-- name: UpdateOverdueSalesAndAccountsGlobal :many
WITH updated_accounts AS (
    UPDATE accounts_receivable
    SET status = 'overdue'
    WHERE status = 'pending'
        AND due_date::DATE < CURRENT_DATE
    RETURNING sale_id
)
UPDATE sales
SET status = 'overdue',
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (
        SELECT sale_id
        FROM updated_accounts
    )
    AND status NOT IN ('paid', 'canceled')
RETURNING id AS sale_id,
    customer_id, company_id;
-- name: GetSaleByIdWhatsapp :one
SELECT s.*,
    c.full_name as customer_name,
    c.whatsapp as customer_whatsApp
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
WHERE s.id = $1;
-- name: GetSaleByIdJust :one
SELECT s.*,
    c.full_name as customer_name
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
WHERE s.id = $1;
-- name: ContSalesPendingAndOverdue :one
SELECT COUNT(*)
FROM sales
WHERE company_id = $1
    AND status IN ('pending', 'overdue')
    AND deleted_at IS NULL;
-- name: ListSalesWithDetails :many
SELECT -- Dados da venda
    s.id AS sale_id,
    s.sale_at,
    s.subtotal,
    s.discount_amount,
    s.total_amount,
    s.installments_count,
    s.payment_method,
    s.status AS sale_status,
    s.down_payment,
    -- Dados do cliente
    c.id AS customer_id,
    c.full_name AS customer_name,
    -- Dados dos produtos (itens da venda)
    si.id AS sale_item_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount AS item_discount,
    p.name AS product_name,
    -- Parcelas (accounts_receivable)
    ar.id AS installment_id,
    ar.total_amount AS installment_total_amount,
    ar.balance AS installment_balance,
    ar.due_date,
    ar.installment_number,
    ar.status AS installment_status
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
    INNER JOIN sale_items si ON s.id = si.sale_id
    INNER JOIN products p ON si.product_id = p.id
    LEFT JOIN accounts_receivable ar ON s.id = ar.sale_id
WHERE s.company_id = $1 -- id da empresa
    AND s.deleted_at IS NULL
ORDER BY s.sale_at DESC,
    si.id,
    ar.installment_number;
-- name: ListSalesWithDetailsPendingOverdue :many
SELECT -- Dados da venda
    s.id AS sale_id,
    s.sale_at,
    s.subtotal,
    s.discount_amount,
    s.total_amount,
    s.installments_count,
    s.payment_method,
    s.status AS sale_status,
    s.down_payment,
    -- Dados do cliente
    c.id AS customer_id,
    c.full_name AS customer_name,
    -- Dados dos produtos (itens da venda)
    si.id AS sale_item_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount AS item_discount,
    p.name AS product_name,
    -- Parcelas (accounts_receivable)
    ar.id AS installment_id,
    ar.total_amount AS installment_total_amount,
    ar.balance AS installment_balance,
    ar.due_date,
    ar.installment_number,
    ar.status AS installment_status
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
    INNER JOIN sale_items si ON s.id = si.sale_id
    INNER JOIN products p ON si.product_id = p.id
    LEFT JOIN accounts_receivable ar ON s.id = ar.sale_id
WHERE s.company_id = $1 -- id da empresa
    AND s.deleted_at IS NULL
    AND s.status IN ('pending', 'overdue')
ORDER BY s.sale_at DESC,
    si.id,
    ar.installment_number;
-- name: GetPendingSalesDetailedReport :many
SELECT -- Dados da venda
    s.id AS sale_id,
    s.sale_at,
    s.subtotal,
    s.discount_amount,
    s.total_amount,
    s.installments_count,
    s.payment_method,
    s.status AS sale_status,
    -- Dados do cliente
    c.id AS customer_id,
    c.full_name AS customer_name,
    -- Dados dos produtos (itens da venda)
    si.id AS sale_item_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount AS item_discount,
    p.name AS product_name,
    -- Parcelas (accounts_receivable)
    ar.id AS installment_id,
    ar.total_amount AS installment_total_amount,
    ar.balance AS installment_balance,
    ar.due_date,
    ar.installment_number,
    ar.status AS installment_status
FROM sales s
    INNER JOIN customers c ON s.customer_id = c.id
    INNER JOIN sale_items si ON s.id = si.sale_id
    INNER JOIN products p ON si.product_id = p.id
    LEFT JOIN accounts_receivable ar ON s.id = ar.sale_id
WHERE s.company_id = $1
    AND s.deleted_at IS NULL
    AND s.status IN ('pending', 'overdue') -- Filtro de intervalo de datas (Inclusivo)
    AND s.sale_at BETWEEN $2 AND $3
ORDER BY s.sale_at DESC,
    si.id,
    ar.installment_number;