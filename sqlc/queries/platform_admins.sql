-- name: GetPlatformAdminByEmail :one
SELECT *
FROM platform_admins
WHERE email = $1
    AND deleted_at IS NULL;