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
SELECT b.*,
    v.name as vendor_name,
    c.name as category_name,
    pm.name as payment_method_name
FROM bills_payable b
    LEFT JOIN vendors v ON b.vendor_id = v.id
    LEFT JOIN bill_categories c ON b.category_id = c.id
    LEFT JOIN payment_methods pm ON b.payment_method_id = pm.id
WHERE b.company_id = $1
ORDER BY b.due_date ASC;
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