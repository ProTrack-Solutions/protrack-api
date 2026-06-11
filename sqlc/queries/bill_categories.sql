-- name: CreateBillCategories :exec
INSERT INTO bill_categories (company_id, name, description)
VALUES ($1, $2, $3);
-- name: GetBillCategoriesById :one
SELECT *
FROM bill_categories
WHERE id = $1;
-- name: ListBillCategories :many
SELECT *
FROM bill_categories
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: ListBillCategoriesActive :many
SELECT *
FROM bill_categories
WHERE id = $1
    AND is_active = TRUE
    AND deleted_at IS NULL;
-- name: ToggleBillCategoriesActive :exec
UPDATE bill_categories
SET is_active = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
-- name: DeleteBillCategories :exec
UPDATE bill_categories
SET deleted_at = CURRENT_TIMESTAMP,
    is_active = FALSE
WHERE id = $1;