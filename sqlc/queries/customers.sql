-- name: CreateCustomers :one
INSERT INTO customers(
        company_id,
        full_name,
        birth_date,
        cpf,
        rg,
        marital_status,
        gender,
        whatsapp,
        mobile_phone,
        home_phone,
        email,
        address_street,
        address_number,
        address_complement,
        address_neighborhood,
        address_city,
        address_state,
        address_zipcode,
        address_country,
        balance_due,
        created_by
    )
VALUES(
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11,
        $12,
        $13,
        $14,
        $15,
        $16,
        $17,
        $18,
        $19,
        $20,
        $21
    )
RETURNING id;
-- name: GetCustomerById :one
SELECT *
FROM customers
WHERE id = $1
    AND deleted_at IS NULL;
-- name: GetCustomerByCPF :one
SELECT *
FROM customers
WHERE cpf = $1
    AND deleted_at IS NULL;
-- name: ListCustomers :many
SELECT *
FROM customers
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: UpdateCustomer :exec
UPDATE customers
SET full_name = $2,
    birth_date = $3,
    cpf = $4,
    rg = $5,
    marital_status = $6,
    gender = $7,
    whatsapp = $8,
    mobile_phone = $9,
    home_phone = $10,
    email = $11,
    address_street = $12,
    address_number = $13,
    address_complement = $14,
    address_neighborhood = $15,
    address_city = $16,
    address_state = $17,
    address_zipcode = $18,
    address_country = $19,
    balance_due = $20,
    updated_by = $21,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND deleted_at IS NULL;
-- name: UpdateBalanceDueCustomer :exec
UPDATE customers
SET balance_due = balance_due + $2,
    updated_by = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND deleted_at IS NULL;
-- name: DeleteCustomer :exec
UPDATE customers
SET deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1
    AND deleted_at IS NULL;
-- name: CountCustomers :one
SELECT COUNT(*)
FROM customers
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: GetCustomersPerformanceSummary :one
SELECT COUNT(*) FILTER (
        WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)
    ) AS current_month_count,
    COUNT(*) FILTER (
        WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE - INTERVAL '1 month')
    ) AS last_month_count
FROM customers
WHERE company_id = $1
    AND deleted_at IS NULL
    AND created_at >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month');
-- name: UpdateCustomerBalance :exec
UPDATE customers
SET balance_due = $2,
    -- Atribui o valor final calculado no Go
    updated_by = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND deleted_at IS NULL;