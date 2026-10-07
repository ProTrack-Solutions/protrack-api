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
-- name: ListTopAnnouncementsOfDay :many
-- Avisos vigentes agora, do mais importante (manutenção) ao menos importante (sucesso)
SELECT 
    id, 
    title, 
    content,
    type, 
    starts_at, 
    expires_at,
    created_at
FROM announcements
WHERE company_id = $1
    AND is_active = TRUE
    AND deleted_at IS NULL
    AND starts_at <= NOW()
    AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY 
    CASE type
        WHEN 'maintenance' THEN 1
        WHEN 'warning' THEN 2
        WHEN 'info' THEN 3
        WHEN 'success' THEN 4
    END,
    starts_at DESC
LIMIT 4;
