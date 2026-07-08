-- name: CreateAnnouncements :exec 
INSERT INTO announcements (
    company_id, 
    title, 
    content, 
    type, 
    starts_at, 
    expires_at, 
    is_active, 
    created_by
)
VALUES ($1, $2, $3, $4, COALESCE($5, NOW()), $6, $7, $8);
-- name: ListAnnoucements :many
SELECT 
    id, 
    title, 
    type, 
    is_active, 
    starts_at, 
    expires_at,
    created_at
FROM announcements
WHERE company_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
-- name: CountAnnoucementsByCompany :one
SELECT COUNT(*) FROM announcements
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: DeleteAnnoucements :exec
UPDATE announcements
SET 
    is_active = FALSE, 
    updated_at = NOW(), 
    deleted_by = $1,
    deleted_at = NOW()
WHERE id = $2 AND company_id = $3;