-- name: CreateUser :one
INSERT INTO users(
        name,
        email,
        username,
        password_hash,
        role,
        status,
        company_id,
        department_id,
        created_by,
        updated_by,
        created_at
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;
-- name: GetUserByID :one
SELECT u.*,
d.name AS department_name
FROM users u
    INNER JOIN departments d ON u.department_id  = d.id
WHERE u.id = $1
    AND u.deleted_at IS NULL;
-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1
    AND deleted_at IS NULL;
-- name: UpdateUser :one
UPDATE users
SET name = $2,
    email = $3,
    username = $4,
    role = $5,
    status = $6,
    department_id = $7,
    updated_by = $8,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;
-- name: UpdatePasswordHash :exec
UPDATE users
SET password_hash = $2
WHERE id = $1
    AND deleted_at IS NULL;
-- name: DeleteUser :exec
UPDATE users
SET deleted_at = NOW()
WHERE id = $1;
-- name: ListUsers :many
SELECT *
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC;
-- name: UpdateUserCompanyAndRole :exec
UPDATE users
SET company_id = $2,
    role = $3,
    updated_at = now()
WHERE id = $1;
-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP
WHERE id = $1;