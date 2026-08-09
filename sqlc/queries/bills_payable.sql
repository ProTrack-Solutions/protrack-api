-- name: CreateBillPayable :exec
INSERT INTO bills_payable (
        company_id,
        vendor_id,
        category_id,
        payment_method_id,
        amount,
        due_date,
        status,
        description,
        notes
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
-- name: UpdateBillPayable :exec
UPDATE bills_payable
SET vendor_id = $3,
    category_id = $4,
    payment_method_id = $5,
    amount = $6,
    due_date = $7,
    status = $8,
    description = $9,
    notes = $10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND company_id = $2;
-- name: PayBill :exec
UPDATE bills_payable
SET status = 'paid',
    payment_date = $3,
    amount_paid = $4,
    payment_method_id = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND company_id = $2;
-- name: ListBillsPayable :many
SELECT
    b.*,
    v.name AS vendor_name,
    c.name AS category_name,
    pm.name AS payment_method_name,
    count(*) OVER() AS total_count
FROM bills_payable b
    LEFT JOIN vendors v ON b.vendor_id = v.id
    LEFT JOIN bill_categories c ON b.category_id = c.id
    LEFT JOIN payment_methods pm ON b.payment_method_id = pm.id
WHERE b.company_id = $1
    AND (
        sqlc.arg(search)::text = '' OR
        to_tsvector('portuguese_unaccent', coalesce(b.description, ''))
            @@ plainto_tsquery('portuguese_unaccent', sqlc.arg(search)::text)
        OR v.name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR c.name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR pm.name ILIKE '%' || sqlc.arg(search)::text || '%'
    )
    AND (sqlc.narg(status)::account_status_enum IS NULL OR b.status = sqlc.narg(status)::account_status_enum)
    AND (sqlc.narg(start_due_date)::date IS NULL OR b.due_date >= sqlc.narg(start_due_date)::date)
    AND (sqlc.narg(end_due_date)::date IS NULL OR b.due_date < (sqlc.narg(end_due_date)::date + INTERVAL '1 day'))
    AND (sqlc.narg(start_scheduled_date)::date IS NULL OR b.scheduled_date >= sqlc.narg(start_scheduled_date)::date)
    AND (sqlc.narg(end_scheduled_date)::date IS NULL OR b.scheduled_date < (sqlc.narg(end_scheduled_date)::date + INTERVAL '1 day'))
    AND (sqlc.narg(start_payment_date)::date IS NULL OR b.payment_date >= sqlc.narg(start_payment_date)::date)
    AND (sqlc.narg(end_payment_date)::date IS NULL OR b.payment_date < (sqlc.narg(end_payment_date)::date + INTERVAL '1 day'))
    AND (sqlc.narg(start_created_at)::date IS NULL OR b.created_at >= sqlc.narg(start_created_at)::date)
    AND (sqlc.narg(end_created_at)::date IS NULL OR b.created_at < (sqlc.narg(end_created_at)::date + INTERVAL '1 day'))
ORDER BY
    CASE WHEN sqlc.arg(order_field)::text = 'due_date'       AND sqlc.arg(order_by)::text = 'asc'  THEN b.due_date::timestamptz END ASC,
    CASE WHEN sqlc.arg(order_field)::text = 'due_date'       AND sqlc.arg(order_by)::text = 'desc' THEN b.due_date::timestamptz END DESC,
    CASE WHEN sqlc.arg(order_field)::text = 'scheduled_date' AND sqlc.arg(order_by)::text = 'asc'  THEN b.scheduled_date::timestamptz END ASC,
    CASE WHEN sqlc.arg(order_field)::text = 'scheduled_date' AND sqlc.arg(order_by)::text = 'desc' THEN b.scheduled_date::timestamptz END DESC,
    CASE WHEN sqlc.arg(order_field)::text = 'payment_date'   AND sqlc.arg(order_by)::text = 'asc'  THEN b.payment_date::timestamptz END ASC,
    CASE WHEN sqlc.arg(order_field)::text = 'payment_date'   AND sqlc.arg(order_by)::text = 'desc' THEN b.payment_date::timestamptz END DESC,
    CASE WHEN sqlc.arg(order_field)::text = 'created_at'     AND sqlc.arg(order_by)::text = 'asc'  THEN b.created_at END ASC,
    CASE WHEN sqlc.arg(order_field)::text = 'created_at'     AND sqlc.arg(order_by)::text = 'desc' THEN b.created_at END DESC
LIMIT $2
OFFSET $3;
-- name: GetBillsByStatus :many
SELECT *
FROM bills_payable
WHERE company_id = $1
    AND status = $2
ORDER BY due_date ASC;
-- name: GetOverdueBills :many
SELECT *
FROM bills_payable
WHERE company_id = $1
    AND due_date < CURRENT_DATE
    AND status != 'paid'
ORDER BY due_date ASC;
-- name: GetBillsById :one
SELECT *
FROM bills_payable
WHERE company_id = $1
    AND id = $2
ORDER BY due_date ASC;
-- name: ScheduleBill :exec
UPDATE bills_payable
SET status = 'scheduled',
    scheduled_date = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
    AND company_id = $2;
-- name: GetBillsPayableSummary :one
SELECT COUNT(*)::INT as total_quantity,
    COALESCE(
        SUM(amount) FILTER (
            WHERE status IN ('pending', 'overdue')
        ),
        0
    )::NUMERIC(12, 2) as total_to_pay,
    COALESCE(
        SUM(amount) FILTER (
            WHERE due_date < CURRENT_DATE
                AND status != 'paid'
        ),
        0
    )::NUMERIC(12, 2) as total_overdue,
    COALESCE(
        SUM(amount) FILTER (
            WHERE scheduled_date IS NOT NULL
                AND status != 'paid'
        ),
        0
    )::NUMERIC(12, 2) as total_scheduled
FROM bills_payable
WHERE company_id = $1;
-- name: UpdateOverdueBillsPayable :exec
UPDATE bills_payable
SET status = 'overdue'
WHERE status = 'pending'
    AND due_date::DATE < CURRENT_DATE;
-- name: CountBillsPayableByCompany :one
SELECT COUNT(*) FROM bills_payable
WHERE company_id = $1;
-- name: SumBillsPayableByCompany :one
SELECT COALESCE(SUM(amount), 0.0)::DOUBLE PRECISION AS total_amount
FROM bills_payable 
WHERE company_id = $1 AND status IN ('overdue','pending','partial');
-- name: SumBillsPayableOverdue :one
SELECT COALESCE(SUM(amount), 0.0)::DOUBLE PRECISION AS total_overdue
FROM bills_payable 
WHERE company_id = $1 AND status = 'overdue'; 
-- name: SumBillsPayableSchedule :one
SELECT COALESCE(SUM(amount), 0.0)::DOUBLE PRECISION AS total_scheduled
FROM bills_payable 
WHERE company_id = $1 AND status = 'scheduled'; 