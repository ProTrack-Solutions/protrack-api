-- name: GetCompanySetting :one
-- Busca o valor de uma config específica de uma empresa.
SELECT * FROM company_settings
WHERE company_id = $1 AND key = $2;

-- name: ListCompanySettings :many
-- Lista todas as configs de uma empresa de uma vez — útil pra telas de
-- "configurações" que mostram tudo junto, evitando N chamadas separadas.
SELECT * FROM company_settings
WHERE company_id = $1
ORDER BY key;

-- name: UpsertCompanySetting :one
-- Cria a config se não existir, ou atualiza o valor se já existir
-- (aproveita o UNIQUE (company_id, key) da tabela).
INSERT INTO company_settings (
    company_id, key, value
) VALUES (
    $1, $2, $3
)
ON CONFLICT (company_id, key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = now()
RETURNING *;

-- name: DeleteCompanySetting :exec
-- Remove uma config específica, voltando pro comportamento padrão do sistema
-- (útil quando "ausência da linha" significa "usar o valor default").
DELETE FROM company_settings
WHERE company_id = $1 AND key = $2;

-- name: DeleteAllCompanySettings :exec
-- Remove todas as configs de uma empresa — normalmente não precisa ser chamado
-- manualmente (o ON DELETE CASCADE da FK já cuida disso quando a empresa é
-- deletada), mas é útil pra um "reset de configurações" administrativo.
DELETE FROM company_settings
WHERE company_id = $1;