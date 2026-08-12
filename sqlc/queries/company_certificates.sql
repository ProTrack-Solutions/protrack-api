-- name: CreateCompanyCertificate :one
INSERT INTO company_certificates(
        company_id,
        encrypted_cert_data,
        encrypted_cert_nonce,
        encrypted_password,
        encrypted_password_nonce,
        cert_subject_cn,
        expires_at,
        provider_status,
        provider_cert_id,
        created_by
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetCompanyCertificateByCompanyID :one
SELECT *
FROM company_certificates
WHERE company_id = $1
    AND deleted_at IS NULL;

-- name: UpdateCompanyCertificate :one
UPDATE company_certificates
SET encrypted_cert_data = $2,
    encrypted_cert_nonce = $3,
    encrypted_password = $4,
    encrypted_password_nonce = $5,
    cert_subject_cn = $6,
    expires_at = $7,
    provider_status = $8,
    provider_cert_id = $9,
    updated_by = $10,
    updated_at = NOW()
WHERE company_id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCompanyCertificate :exec
UPDATE company_certificates
SET deleted_at = NOW(),
    deleted_by = $2
WHERE company_id = $1
    AND deleted_at IS NULL;
