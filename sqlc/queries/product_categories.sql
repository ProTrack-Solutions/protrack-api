-- name: CreateProductCategory :one
INSERT INTO product_categories(
        company_id,
        name,
        color,
        created_by
    )
VALUES (
        $1,
        $2,
        $3,
        $4
    )
RETURNING *;
-- name: GetProductCategoryById :one
SELECT *
FROM product_categories
WHERE id = $1
    AND deleted_at IS NULL;
-- name: ListProductCategoryByCompanyId :many
SELECT *
FROM product_categories
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: UpdateProductCategory :one
UPDATE product_categories
SET name = $2,
    color = $3,
    updated_by = $4,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;
-- name: SetProductCategoryStatus :execrows
UPDATE product_categories
SET status = $2::status_enum
WHERE id = $1;
-- name: DeleteProductCategory :exec
UPDATE product_categories
SET deleted_by = $2,
    deleted_at = NOW()
WHERE id = $1;