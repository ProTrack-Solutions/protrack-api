-- name: CreateFiscalInvoice :one
INSERT INTO fiscal_invoices(
        company_id,
        sale_id,
        type,
        status,
        created_by
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetFiscalInvoiceByID :one
SELECT *
FROM fiscal_invoices
WHERE id = $1
    AND company_id = $2
    AND deleted_at IS NULL;

-- name: GetFiscalInvoiceBySaleAndType :one
SELECT *
FROM fiscal_invoices
WHERE sale_id = $1
    AND type = $2
    AND deleted_at IS NULL;

-- name: GetFiscalInvoiceByProviderInvoiceID :one
SELECT *
FROM fiscal_invoices
WHERE provider_invoice_id = $1
    AND deleted_at IS NULL;

-- name: ListFiscalInvoicesBySale :many
SELECT *
FROM fiscal_invoices
WHERE sale_id = $1
    AND company_id = $2
    AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ResetFiscalInvoiceForRetry :one
-- Reabre um documento fiscal 'rejected'/'cancelled' para uma nova tentativa
-- de emissão, reaproveitando o mesmo id (usado como idIntegracao no
-- provedor) em vez de criar uma linha nova — UNIQUE(sale_id, type) impediria
-- a duplicata mesmo que tentássemos.
UPDATE fiscal_invoices
SET status = 'processing',
    provider_invoice_id = NULL,
    error_message = NULL,
    cancelled_reason = NULL,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: UpdateFiscalInvoiceProcessing :one
UPDATE fiscal_invoices
SET provider_invoice_id = $2,
    status = $3,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: UpdateFiscalInvoiceAuthorized :one
-- xml_url/danfe_url usam COALESCE porque a reconciliação periódica (GET
-- .../resumo) não traz esses links, só o webhook traz — não pode apagar um
-- link já conhecido só porque essa chamada específica não o tem.
UPDATE fiscal_invoices
SET status = 'authorized',
    chave_acesso = $2,
    numero = $3,
    serie = $4,
    protocolo_autorizacao = $5,
    xml_url = COALESCE($6, xml_url),
    danfe_url = COALESCE($7, danfe_url),
    authorized_at = NOW(),
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: UpdateFiscalInvoiceRejected :one
UPDATE fiscal_invoices
SET status = 'rejected',
    error_message = $2,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: UpdateFiscalInvoiceCancelled :one
UPDATE fiscal_invoices
SET status = 'cancelled',
    cancelled_reason = $2,
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: UpdateFiscalInvoiceCancelProcessing :one
UPDATE fiscal_invoices
SET status = 'cancel_processing',
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: ListStaleProcessingFiscalInvoices :many
-- Documentos presos em 'processing'/'cancel_processing' há mais de $1 —
-- usados pela reconciliação periódica como rede de segurança para quando o
-- webhook do provedor não chega (ver internal/worker).
SELECT *
FROM fiscal_invoices
WHERE status IN ('processing', 'cancel_processing')
    AND provider_invoice_id IS NOT NULL
    AND updated_at < $1
    AND deleted_at IS NULL
ORDER BY updated_at ASC
LIMIT 200;

-- name: CreateFiscalInvoiceEvent :one
INSERT INTO fiscal_invoice_events(
        fiscal_invoice_id,
        event_type,
        payload
    )
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListFiscalInvoiceEventsByInvoiceID :many
SELECT *
FROM fiscal_invoice_events
WHERE fiscal_invoice_id = $1
ORDER BY created_at DESC;