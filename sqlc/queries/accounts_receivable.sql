-- name: CreateAccountReceivable :exec
INSERT INTO accounts_receivable (
        company_id,
        customer_id,
        sale_id,
        total_amount,
        balance,
        due_date,
        installment_number,
        total_installments,
        status,
        created_by
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
-- name: GetPendingReceivablesByCustomer :many
SELECT *
FROM accounts_receivable
WHERE customer_id = $1
    AND company_id = $2
    AND status IN ('pending', 'partial')
    AND deleted_at IS NULL
ORDER BY due_date ASC;
-- name: UpdateAccountReceivableBalance :one
UPDATE accounts_receivable
SET balance = $1,
    status = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $4
RETURNING sale_id;
-- name: GetCustomerDebtSummary :one
SELECT COUNT(id)::int AS total_count,
    COALESCE(SUM(balance), 0)::numeric AS total_balance,
    MIN(due_date)::date AS oldest_due_date
FROM accounts_receivable
WHERE customer_id = $1
    AND status IN ('pending', 'partial')
    AND deleted_at IS NULL;
-- name: ListOverdueReceivables :many
SELECT ar.*,
    c.full_name as customer_name,
    EXTRACT(
        DAY
        FROM (CURRENT_DATE - ar.due_date)
    )::int AS days_overdue
FROM accounts_receivable ar
    JOIN customers c ON ar.customer_id = c.id
WHERE ar.company_id = $1
    AND ar.status IN ('pending', 'partial')
    AND ar.due_date < CURRENT_DATE
    AND ar.deleted_at IS NULL
ORDER BY ar.due_date ASC;
-- name: GetReceivablesBySale :many
SELECT *
FROM accounts_receivable
WHERE sale_id = $1
ORDER BY installment_number ASC;
-- name: GetTotalOpenAmountByCompany :one
SELECT company_id,
    SUM(balance)::NUMERIC(10, 2) as total_open
FROM accounts_receivable
WHERE company_id = $1
    AND status IN ('pending', 'overdue')
    AND deleted_at IS NULL
GROUP BY company_id;
-- name: GetTotalOverdueAmountByCompany :one
SELECT company_id,
    COALESCE(SUM(balance), 0)::NUMERIC(10, 2) as total_overdue
FROM accounts_receivable
WHERE company_id = $1
    AND (
        status = 'overdue'
        OR due_date < CURRENT_DATE
    )
    AND status != 'paid'
    AND deleted_at IS NULL
GROUP BY company_id;
-- name: DeleteAccountsReceivableBySaleId :exec
DELETE FROM accounts_receivable 
WHERE sale_id = $1 
    AND company_id = $2;