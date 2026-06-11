-- name: CreatePaymentMethod :exec
INSERT INTO payment_methods(
        company_id,
        name,
        type
    )
VALUES ($1, $2, $3);
-- name: TogglePaymentMethodActive :exec
UPDATE payment_methods
SET is_active = $2
WHERE id = $1;
-- name: ListPaymentMethodIsActive :many
SELECT *
FROM payment_methods
WHERE company_id = $1
    AND is_active = TRUE;
-- name: ListPaymentMethod :many
SELECT *
FROM payment_methods
WHERE company_id = $1;
-- name: GetPaymentMethodByID :one
SELECT *
FROM payment_methods
WHERE id = $1;
-- name: GetPaymentMethodsStats :many
SELECT payment_method,
    COUNT(*) as total_sales,
    SUM(total_amount)::NUMERIC(15, 2) as total_revenue
FROM sales
WHERE company_id = $1
    AND deleted_at IS NULL
GROUP BY payment_method
ORDER BY total_sales DESC;