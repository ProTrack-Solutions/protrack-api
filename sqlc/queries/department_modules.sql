-- name: DepartmentHasModule :one
SELECT EXISTS (
    SELECT 1
    FROM department_modules
    WHERE department_id = $1
      AND module_code = $2
) AS has_access;

-- name: ListModulesByDepartment :many
SELECT m.code, m.name
FROM department_modules dm
JOIN modules m ON m.code = dm.module_code
WHERE dm.department_id = $1
ORDER BY m.name;

-- name: AddModuleToDepartment :exec
INSERT INTO department_modules (department_id, module_code)
VALUES ($1, $2)
ON CONFLICT (department_id, module_code) DO NOTHING;

-- name: RemoveModuleFromDepartment :exec
DELETE FROM department_modules
WHERE department_id = $1
  AND module_code = $2;

-- name: ReplaceDepartmentModules :exec
DELETE FROM department_modules
WHERE department_id = $1;