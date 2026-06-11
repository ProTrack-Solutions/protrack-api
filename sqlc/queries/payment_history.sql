-- name: CreatePaymentHistory :exec
INSERT INTO payment_history (
        company_id,
        customer_id,
        sale_id,
        payment_method_id,
        user_id,
        amount_paid,
        notes
    )
VALUES ($1, $2, $3, $4, $5, $6, $7);
-- name: ListPaymentHistory :many
SELECT ph.id,
    ph.amount_paid,
    ph.payment_date,
    ph.notes,
    c.full_name as customer_name,
    u.name as user_name,
    pm.name as payment_method_name,
    ph.sale_id
FROM payment_history ph
    INNER JOIN customers c ON ph.customer_id = c.id
    INNER JOIN users u ON ph.user_id = u.id
    LEFT JOIN payment_methods pm ON ph.payment_method_id = pm.id
WHERE ph.company_id = $1
ORDER BY ph.payment_date DESC;
-- name: GetPaymentsByCustomer :many
SELECT *
FROM payment_history
WHERE company_id = $1
    AND customer_id = $2
ORDER BY payment_date DESC;
-- name: GetPaymentsBySale :many
SELECT *
FROM payment_history
WHERE company_id = $1
    AND sale_id = $2
ORDER BY payment_date DESC;
-- name: GetTotalReceivedByPeriod :one
SELECT COALESCE(SUM(amount_paid), 0)::DECIMAL(12, 2) as total
FROM payment_history
WHERE company_id = $1
    AND payment_date BETWEEN $2 AND $3;
-- name: GetPaymentsHistoryReport :many
SELECT ph.id,
    ph.amount_paid,
    ph.payment_date,
    ph.notes,
    c.full_name as customer_name,
    u.name as user_name,
    pm.name as payment_method_name,
    ph.sale_id
FROM payment_history ph
    INNER JOIN customers c ON ph.customer_id = c.id
    INNER JOIN users u ON ph.user_id = u.id
    LEFT JOIN payment_methods pm ON ph.payment_method_id = pm.id
WHERE ph.company_id = $1
    AND ph.payment_date BETWEEN $2 AND $3
ORDER BY ph.payment_date DESC;