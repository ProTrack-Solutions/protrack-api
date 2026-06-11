-- name: CreateCompany :one
INSERT INTO companies(
        name,
        trade_name,
        document,
        document_type,
        email,
        phone,
        website,
        address_street,
        address_number,
        address_complement,
        address_neighborhood,
        address_city,
        address_state,
        address_zipcode,
        address_country,
        timezone,
        created_by
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
        $13,
        $14,
        $15,
        $16,
        $17
    )
RETURNING *;
-- name: GetCompanyByID :one
SELECT *
FROM companies
WHERE id = $1
    AND deleted_at IS NULL;
-- name: GetCompanyByDocument :one
SELECT *
FROM companies
WHERE document = $1
    AND deleted_at IS NULL;
-- name: UpdateCompany :one 
UPDATE companies
SET name = $2,
    trade_name = $3,
    document = $4,
    document_type = $5,
    email = $6,
    phone = $7,
    website = $8,
    address_street = $9,
    address_number = $10,
    address_complement = $11,
    address_neighborhood = $12,
    address_city = $13,
    address_state = $14,
    address_zipcode = $15,
    address_country = $16,
    timezone = $17,
    updated_by = $18,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;
-- name: DeleteCompany :exec
UPDATE companies
SET deleted_at = NOW(),
    deleted_by = $2
WHERE id = $1
    AND deleted_at IS NULL;
-- name: ListCompanies :many
SELECT *
FROM companies
WHERE deleted_at IS NULL
ORDER BY created_at DESC;
-- name: SetCompanyStatus :execrows
UPDATE companies
SET status = $2::status_enum
WHERE id = $1;