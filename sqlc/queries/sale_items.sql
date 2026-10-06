-- name: CreateSaleItem :exec
INSERT INTO sale_items (
        sale_id,
        product_id,
        quantity,
        unit_price,
        discount
    )
VALUES ($1, $2, $3, $4, $5);
-- name: DeleteSaleItem :exec
DELETE FROM sale_items
WHERE id = $1;
-- name: DeleteItemsBySale :exec
DELETE FROM sale_items
WHERE sale_id = $1;
-- name: UpdateSaleItemsDiscount :exec
-- Rateia o desconto da venda (em R$) entre os itens, proporcional ao valor de cada um
UPDATE sale_items
SET discount = ROUND(
        sqlc.arg(discount_amount)::NUMERIC * quantity * unit_price / NULLIF(sqlc.arg(subtotal)::NUMERIC, 0),
        2
    )
WHERE sale_id = sqlc.arg(sale_id);
-- name: ListItemsBySaleForRestock :many
SELECT si.product_id,
    si.quantity,
    p.sell_in_bulk
FROM sale_items si
    INNER JOIN products p ON si.product_id = p.id
WHERE si.sale_id = $1;
;
-- name: ListItemsFromPendingSale :many
SELECT si.id,
    si.sale_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount,
    p.name as product_name
FROM sale_items si
    INNER JOIN sales s ON si.sale_id = s.id
    INNER JOIN products p ON si.product_id = p.id
WHERE si.sale_id = $1
    AND s.status = 'PENDING';
-- name: ListItemsByCompany :many
SELECT si.id,
    si.sale_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount,
    p.name as product_name
FROM sale_items si
    INNER JOIN sales s ON si.sale_id = s.id
    INNER JOIN products p ON si.product_id = p.id
WHERE s.company_id = $1;
-- name: ListItemsByDate :many
SELECT si.id,
    si.sale_id,
    si.product_id,
    si.quantity,
    si.unit_price,
    si.discount,
    p.name as product_name
FROM sale_items si
    INNER JOIN sales s ON si.sale_id = s.id
    INNER JOIN products p ON si.product_id = p.id
WHERE s.company_id = $1 -- Compara apenas o Mês e o Ano, ignorando o dia e a hora
    AND DATE_TRUNC('month', s.created_at) = DATE_TRUNC('month', $2::timestamptz);