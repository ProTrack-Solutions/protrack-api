-- name: LockReceivableForReminder :one
-- Trava a parcela: dois cliques simultâneos não mandam dois lembretes
SELECT ar.id,
    ar.status,
    ar.balance::FLOAT AS balance,
    ar.due_date,
    COALESCE(ar.installment_number, 0)::INT AS installment_number,
    COALESCE(ar.total_installments, 0)::INT AS total_installments,
    c.full_name::TEXT AS customer_name,
    COALESCE(c.whatsapp, '')::TEXT AS customer_whatsapp,
    s.sale_at,
    COALESCE(NULLIF(co.trade_name, ''), co.name)::TEXT AS company_name,
    COALESCE(co.phone, '')::TEXT AS company_phone,
    COALESCE(NULLIF(co.timezone, ''), 'America/Sao_Paulo')::TEXT AS timezone
FROM accounts_receivable ar
    INNER JOIN customers c ON c.id = ar.customer_id
    INNER JOIN sales s ON s.id = ar.sale_id
    INNER JOIN companies co ON co.id = ar.company_id
WHERE ar.id = $1
    AND ar.company_id = $2
    AND ar.deleted_at IS NULL
FOR UPDATE OF ar;

-- name: GetLastReceivableReminder :one
SELECT sent_at
FROM receivable_reminders
WHERE account_receivable_id = $1
ORDER BY sent_at DESC
LIMIT 1;

-- name: CreateReceivableReminder :one
INSERT INTO receivable_reminders (
        company_id,
        account_receivable_id,
        whatsapp_message_id,
        template_name,
        recipient_phone,
        sent_by
    )
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING sent_at;
