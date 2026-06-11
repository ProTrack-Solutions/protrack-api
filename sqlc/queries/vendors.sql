-- name: CreateVendors :exec
INSERT INTO vendors (
        company_id,
        name,
        tax_id,
        email,
        phone,
        postal_code,
        address_line_1,
        address_line_2,
        number,
        neighborhood,
        city,
        state,
        country
    )
VALUES (
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
        $13
    );
-- name: ToggleVendorsActive :exec
UPDATE vendors
SET is_active = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
-- name: UpdateVendors :exec
UPDATE vendors
SET name = $2,
    tax_id = $3,
    email = $4,
    phone = $5,
    postal_code = $6,
    address_line_1 = $7,
    address_line_2 = $8,
    number = $9,
    neighborhood = $10,
    city = $11,
    state = $12,
    country = $13,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND company_id = $14;
-- name: GetVendorsById :one
SELECT *
FROM vendors
WHERE id = $1
    AND company_id = $2;
-- name: ListVendorsIsActive :many
SELECT *
FROM vendors
WHERE is_active = TRUE
    AND company_id = $1
ORDER BY name ASC;
-- name: ListVendors :many
SELECT *
FROM vendors
WHERE company_id = $1
ORDER BY created_at DESC;