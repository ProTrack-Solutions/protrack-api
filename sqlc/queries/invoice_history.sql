-- name: CreateInvoiceHistory :exec
INSERT INTO invoice_history (
    subscription_id,
    company_id,
    payment_method_id,
    external_payment_id,
    amount_cents,
    status,
    paid_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: UpdateInvoiceStatus :exec
UPDATE invoice_history
SET 
    status = $2,
    paid_at = $3,
    updated_at = NOW()
WHERE external_payment_id = $1;

-- name: GetInvoiceById :one
SELECT * FROM invoice_history
WHERE id = $1 LIMIT 1;

-- name: GetInvoiceByMpPaymentId :one
SELECT * FROM invoice_history
WHERE external_payment_id = $1 LIMIT 1;

-- name: ListInvoicesByCompany :many
SELECT * FROM invoice_history
WHERE company_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInvoices :one
SELECT COUNT(*) FROM invoice_history WHERE company_id = $1;