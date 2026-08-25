-- name: CreateDepartment :one
INSERT INTO departments(
        company_id,
        name,
        description,
        created_by
    )
VALUES (
        $1,
        $2,
        $3,
        $4
    )
RETURNING *;
-- name: GetDepartmentById :one
SELECT *
FROM departments
WHERE id = $1
    AND deleted_at IS NULL;
-- name: ListDepartmentsByCompanyId :many
SELECT
    d.*,
    COALESCE(
        json_agg(
            json_build_object('code', m.code, 'name', m.name)
            ORDER BY m.name
        ) FILTER (WHERE m.code IS NOT NULL),
        '[]'
    ) AS modules
FROM departments d
LEFT JOIN department_modules dm ON dm.department_id = d.id
LEFT JOIN modules m ON m.code = dm.module_code
WHERE d.company_id = $1
    AND d.deleted_at IS NULL
GROUP BY d.id
ORDER BY d.created_at DESC;
-- name: SetStatusDepartment :execrows
UPDATE departments
SET status = $2::status_enum,
    updated_by = $3,
    updated_at = NOW()
WHERE id = $1;
-- name: UpdateDepartment :one
UPDATE departments
SET name = $2,
    description = $3,
    updated_by = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
-- name: DeleteDepartment :exec
UPDATE departments
SET deleted_at = NOW(),
    deleted_by = $2
WHERE id = $1;