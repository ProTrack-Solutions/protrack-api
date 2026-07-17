-- name: GetTotalInflowByPeriod :one
SELECT COALESCE(SUM(total_amount), 0)::FLOAT AS total_inflow
FROM sales
WHERE company_id = $1
    AND created_at >= $2  -- Passar '2026-06-18T00:00:00Z'
    AND created_at < $3;
-- name: GetTotalOutflowByPeriod :one
SELECT COALESCE(SUM(amount_paid), 0)::FLOAT AS total_outflow
FROM bills_payable
WHERE company_id = $1
    AND status = 'paid'
    AND payment_date BETWEEN $2 AND $3;
-- name: GetCashInFlowByCategory :many
SELECT pc.name AS category_name,
    COALESCE(SUM(si.unit_price * si.quantity), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT s.id) AS sales_count
FROM product_categories pc
    LEFT JOIN products p ON p.category_id = pc.id
    AND p.company_id = $1
    LEFT JOIN sale_items si ON si.product_id = p.id
    LEFT JOIN sales s ON s.id = si.sale_id
    AND s.company_id = $1
    AND s.status = 'paid'
WHERE pc.company_id = $1
    AND pc.status = 'ACTIVE'
GROUP BY pc.name
ORDER BY total_amount DESC;
-- name: GetCashOutFlowByCategory :many
SELECT bc.name AS category_name,
    COALESCE(SUM(COALESCE(b.amount_paid, b.amount)), 0)::FLOAT AS total_amount,
    COUNT(DISTINCT b.id) AS bills_count
FROM bill_categories bc
    LEFT JOIN bills_payable b ON b.category_id = bc.id
    AND b.company_id = $1
    AND b.status = 'paid'
WHERE bc.company_id = $1
    AND bc.is_active = TRUE
    AND bc.deleted_at IS NULL
GROUP BY bc.name
ORDER BY total_amount DESC;