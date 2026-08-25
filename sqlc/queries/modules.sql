-- name: ListModules :many
SELECT code, name
FROM modules
ORDER BY name;

-- name: GetModule :one
SELECT code, name
FROM modules
WHERE code = $1;