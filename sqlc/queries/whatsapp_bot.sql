-- queries/whatsapp_bot.sql

-- ===== whatsapp_bot_configs =====

-- name: CreateWhatsAppBotConfig :one
INSERT INTO whatsapp_bot_configs (
    company_id, instance_name, welcome_message, is_active
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetWhatsAppBotConfigByCompanyID :one
SELECT * FROM whatsapp_bot_configs
WHERE company_id = $1;

-- name: GetWhatsAppBotConfigByInstanceName :one
SELECT * FROM whatsapp_bot_configs
WHERE instance_name = $1 AND is_active = true;

-- name: UpdateWhatsAppBotConfig :one
UPDATE whatsapp_bot_configs
SET
    welcome_message = $2,
    is_active = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateWhatsAppBotInstanceName :one
UPDATE whatsapp_bot_configs
SET
    instance_name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetWhatsAppBotActive :exec
UPDATE whatsapp_bot_configs
SET is_active = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteWhatsAppBotConfig :exec
DELETE FROM whatsapp_bot_configs
WHERE id = $1;


-- ===== whatsapp_bot_menu_options =====

-- name: CreateWhatsAppBotMenuOption :one
INSERT INTO whatsapp_bot_menu_options (
    bot_config_id, option_key, label, response_message, order_index
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListWhatsAppBotMenuOptions :many
SELECT * FROM whatsapp_bot_menu_options
WHERE bot_config_id = $1
ORDER BY order_index ASC, option_key ASC;

-- name: GetWhatsAppBotMenuOptionByKey :one
SELECT * FROM whatsapp_bot_menu_options
WHERE bot_config_id = $1 AND option_key = $2;

-- name: UpdateWhatsAppBotMenuOption :one
UPDATE whatsapp_bot_menu_options
SET
    label = $2,
    response_message = $3,
    order_index = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteWhatsAppBotMenuOption :exec
DELETE FROM whatsapp_bot_menu_options
WHERE id = $1;

-- name: DeleteAllMenuOptionsForBotConfig :exec
DELETE FROM whatsapp_bot_menu_options
WHERE bot_config_id = $1;


-- ===== consulta composta (config + opções de uma vez) =====

-- name: GetWhatsAppBotConfigWithOptionsByInstance :many
SELECT
    c.id AS bot_config_id,
    c.company_id,
    c.instance_name,
    c.welcome_message,
    c.is_active,
    o.id AS option_id,
    o.option_key,
    o.label,
    o.response_message,
    o.order_index
FROM whatsapp_bot_configs c
LEFT JOIN whatsapp_bot_menu_options o ON o.bot_config_id = c.id
WHERE c.instance_name = $1 AND c.is_active = true
ORDER BY o.order_index ASC, o.option_key ASC;