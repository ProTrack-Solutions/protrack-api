-- name: GetCompanyWhatsAppConfig :one
SELECT * FROM company_whatsapp_configs
WHERE company_id = $1;

-- name: UpsertCompanyWhatsAppConfig :one
INSERT INTO company_whatsapp_configs (
    company_id, mode, waba_id, phone_number_id, display_phone_number,
    access_token_encrypted, monthly_message_allowance, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (company_id) DO UPDATE SET
    mode = EXCLUDED.mode,
    waba_id = EXCLUDED.waba_id,
    phone_number_id = EXCLUDED.phone_number_id,
    display_phone_number = EXCLUDED.display_phone_number,
    access_token_encrypted = EXCLUDED.access_token_encrypted,
    monthly_message_allowance = EXCLUDED.monthly_message_allowance,
    is_active = EXCLUDED.is_active,
    updated_at = now()
RETURNING *;

-- name: GetEligibleTemplateByName :one
-- Usado pelo serviço de disparo pra validar se um template pode ser usado
-- no WABA compartilhado antes de chamar a Graph API.
SELECT * FROM whatsapp_templates
WHERE meta_template_name = $1
  AND language_code = $2
  AND is_platform_shared_eligible = true
  AND meta_approval_status = 'approved';

-- name: ListApprovedTemplates :many
SELECT * FROM whatsapp_templates
WHERE meta_approval_status = 'approved'
ORDER BY category, meta_template_name;

-- name: CreateWhatsAppMessage :one
INSERT INTO whatsapp_messages (
    company_id, template_id, category, recipient_phone,
    meta_message_id, status, estimated_cost_cents
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateWhatsAppMessageStatus :one
-- Chamado pelo handler de webhook quando chega um status novo (sent/delivered/failed).
UPDATE whatsapp_messages SET
    status = $2,
    failure_code = $3,
    failure_reason = $4,
    delivered_at = CASE WHEN $2 = 'delivered' THEN now() ELSE delivered_at END,
    sent_at = CASE WHEN $2 = 'sent' AND sent_at IS NULL THEN now() ELSE sent_at END
WHERE meta_message_id = $1
RETURNING *;

-- name: GetMessageByMetaID :one
SELECT * FROM whatsapp_messages
WHERE meta_message_id = $1;

-- name: CountMessagesInPeriod :one
-- Alimenta o job de sincronização de uso mensal com o Stripe.
SELECT COUNT(*) AS total
FROM whatsapp_messages
WHERE company_id = $1
  AND created_at >= $2
  AND created_at < $3
  AND status IN ('sent', 'delivered', 'read');

-- name: UpsertMonthlyUsage :one
INSERT INTO whatsapp_usage_monthly (
    company_id, billing_period, messages_sent, messages_over_allowance
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (company_id, billing_period) DO UPDATE SET
    messages_sent = EXCLUDED.messages_sent,
    messages_over_allowance = EXCLUDED.messages_over_allowance
RETURNING *;

-- name: MarkUsageSyncedWithStripe :exec
UPDATE whatsapp_usage_monthly SET
    stripe_usage_record_id = $2,
    synced_at = now()
WHERE id = $1;