-- name: CreateProduct :one
-- Registra a entrada inicial em stock_movements na mesma instrução (produtos a granel não controlam estoque)
WITH new_product AS (
    INSERT INTO products(
            company_id,
            name,
            description,
            category_id,
            barcode,
            quantity,
            size,
            cost_price,
            sale_price,
            created_by,
            sell_in_bulk,
            unit
        )
    VALUES(
            $1,
            $2,
            $3,
            $4,
            $5,
            $6,
            $7,
            $8,
            $9,
            $10,
            $11,
            $12
        )
    RETURNING *
),
initial_stock AS (
    INSERT INTO stock_movements (company_id, product_id, type, quantity, reason, created_by)
    SELECT np.company_id, np.id, 'ENTRY', np.quantity, 'Cadastro do produto', np.created_by
    FROM new_product np
    WHERE np.sell_in_bulk = FALSE
        AND np.quantity <> 0
)
SELECT *
FROM new_product;
-- name: GetProductById :one
SELECT *
FROM products
WHERE id = $1
    AND deleted_at IS NULL;
-- name: GetProductByBarcode :one
SELECT *
FROM products
WHERE barcode = $1
    AND deleted_at IS NULL;
-- name: ListProductsByCompany :many
SELECT p.*,
    c.name AS category_name
FROM products p
    INNER JOIN product_categories c ON p.category_id = c.id
WHERE p.company_id = $1
    AND p.deleted_at IS NULL;
-- name: ListProductsByCategoryId :many
SELECT *
FROM products
WHERE category_id = $1
    AND company_id = $2
    AND deleted_at IS NULL;
-- name: UpdateProduct :one
-- A quantidade não é editada aqui: entradas e ajustes de estoque passam pelo AdjustStock, que registra a movimentação
UPDATE products
SET name = $2,
    description = $3,
    category_id = $4,
    barcode = $5,
    size = $6,
    cost_price = $7,
    sale_price = $8,
    updated_by = $9,
    unit = $10,
    updated_at = NOW()
WHERE id = $1
    AND deleted_at IS NULL
RETURNING *;
-- name: DeleteProduct :exec
UPDATE products
SET deleted_by = $2,
    deleted_at = NOW()
WHERE id = $1;
-- name: DecrementStock :exec
UPDATE products
SET quantity = quantity - $1
WHERE id = $2
    AND quantity >= $1;
-- name: IncrementStock :exec
UPDATE products
SET quantity = quantity + $1
WHERE id = $2;
-- name: AdjustStock :one
-- Soma (ou subtrai) a quantidade ao saldo atual e registra a movimentação na mesma instrução.
-- Incremento relativo + trava da linha pelo UPDATE evitam perder vendas concorrentes.
-- Nenhuma linha retornada = produto inexistente, a granel ou sem saldo suficiente.
WITH updated_product AS (
    UPDATE products
    SET quantity = quantity + sqlc.arg(quantity)::INT,
        updated_by = sqlc.arg(created_by)::UUID,
        updated_at = NOW()
    WHERE id = sqlc.arg(product_id)::UUID
        AND company_id = sqlc.arg(company_id)::UUID
        AND deleted_at IS NULL
        AND sell_in_bulk = FALSE
        AND quantity + sqlc.arg(quantity)::INT >= 0
    RETURNING id,
        company_id,
        quantity
),
movement AS (
    INSERT INTO stock_movements (company_id, product_id, type, quantity, reason, created_by)
    SELECT up.company_id,
        up.id,
        sqlc.arg(type)::stock_movement_type,
        sqlc.arg(quantity)::INT,
        sqlc.narg(reason)::TEXT,
        sqlc.arg(created_by)::UUID
    FROM updated_product up
    RETURNING id,
        product_id,
        type,
        quantity,
        reason,
        created_at
)
SELECT m.id,
    m.product_id,
    m.type,
    m.quantity,
    m.reason,
    m.created_at,
    up.quantity AS current_quantity
FROM movement m
    INNER JOIN updated_product up ON up.id = m.product_id;
-- name: CountProducts :one
SELECT SUM(quantity)
FROM products
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: GetProductsPerformanceSummary :one
SELECT COALESCE(
        SUM(quantity) FILTER (
            WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)
        ),
        0
    )::FLOAT AS current_month_qty,
    COALESCE(
        SUM(quantity) FILTER (
            WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE - INTERVAL '1 month')
        ),
        0
    )::FLOAT AS last_month_qty
FROM products
WHERE company_id = $1
    AND deleted_at IS NULL
    AND created_at >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month');
-- name: GetCostTotalStock :one
SELECT COALESCE(SUM(cost_price * quantity), 0)::FLOAT AS total_stock_value
FROM products
WHERE company_id = $1
    AND deleted_at IS NULL
    AND quantity > 0;
-- name: GetTop5BestSellingProducts :many
SELECT p.id,
    p.name,
    COALESCE(SUM(si.quantity), 0)::INTEGER AS total_quantity_sold
FROM products p
    INNER JOIN sale_items si ON si.product_id = p.id
    INNER JOIN sales s ON s.id = si.sale_id
WHERE s.company_id = $1
    AND s.deleted_at IS NULL
GROUP BY p.id,
    p.name
ORDER BY total_quantity_sold DESC
LIMIT 5;
-- name: GetInventoryReport :many
SELECT p.name,
    c.name AS category_name,
    p.quantity,
    p.sale_price,
    (p.quantity * p.sale_price)::NUMERIC(10, 2) AS total_value,
    p.cost_price,
    p.barcode,
    p.created_at
FROM products p
    LEFT JOIN product_categories c ON p.category_id = c.id
WHERE p.company_id = $1
    AND p.created_at BETWEEN $2 AND $3
    AND p.deleted_at IS NULL
ORDER BY p.name ASC;
-- name: ListProductsByDate :many
SELECT p.id,
    p.name,
    p.cost_price,
    p.quantity,
    p.category_id,
    p.created_at,
    c.name AS category_name
FROM products p
    INNER JOIN product_categories c ON p.category_id = c.id
WHERE p.company_id = $1
    AND p.deleted_at IS NULL
    AND p.created_at >= $2
    AND p.created_at <= $3;
-- name: ListProductsByCategoryAndDate :many
SELECT p.id,
    p.name,
    p.cost_price,
    p.quantity,
    p.category_id,
    p.created_at,
    c.name AS category_name
FROM products p
    INNER JOIN product_categories c ON p.category_id = c.id
WHERE p.category_id = $1
    AND p.deleted_at IS NULL
    AND p.created_at >= $2
    AND p.created_at <= $3;
-- name: CountProductsByCompany :one
SELECT COUNT(*) FROM products
WHERE company_id = $1
    AND deleted_at IS NULL;
-- name: ListProductsByCompanyPaginated :many
SELECT
    p.id, p.company_id, p.category_id, p.name, p.description, p.barcode,
    p.quantity, p.size, p.cost_price, p.sale_price,
    p.created_by, p.updated_by, p.deleted_by,
    p.created_at, p.updated_at, p.deleted_at,
    p.sell_in_bulk, p.unit,
    c.name AS category_name,
    COALESCE(sm.total_registered, 0)::INT AS total_registered,
    COALESCE(sm.total_sold, 0)::INT AS total_sold,
    count(*) OVER() AS total_count
FROM products p
JOIN product_categories c ON c.id = p.category_id
LEFT JOIN LATERAL (
    SELECT SUM(m.quantity) FILTER (WHERE m.type = 'ENTRY') AS total_registered,
        -SUM(m.quantity) FILTER (WHERE m.type IN ('SALE', 'SALE_CANCEL')) AS total_sold
    FROM stock_movements m
    WHERE m.product_id = p.id
) sm ON TRUE
WHERE p.company_id = $1
    AND p.deleted_at IS NULL
    AND (
        sqlc.arg(search)::text = '' OR
        to_tsvector(
            'portuguese_unaccent',
            coalesce(p.name, '') || ' ' || coalesce(p.description, '')
        ) @@ plainto_tsquery('portuguese_unaccent', sqlc.arg(search)::text)
        OR p.barcode ILIKE '%' || sqlc.arg(search)::text || '%'
        OR c.name ILIKE '%' || sqlc.arg(search)::text || '%'
    )
    AND (sqlc.narg(start_date)::date IS NULL OR p.created_at >= sqlc.narg(start_date)::date)
    AND (sqlc.narg(end_date)::date IS NULL OR p.created_at < (sqlc.narg(end_date)::date + INTERVAL '1 day'))
ORDER BY
    CASE WHEN sqlc.arg(order_by)::text = 'asc'  THEN p.created_at END ASC,
    CASE WHEN sqlc.arg(order_by)::text = 'desc' THEN p.created_at END DESC
LIMIT $2
OFFSET $3;
-- name: CountLowStockProductsByCompany :one
SELECT COUNT(*) AS low_stock_count
FROM products
WHERE company_id = $1
  AND quantity < 5
  AND deleted_at IS NULL;
-- name: GetGeneralTotalStockValue :one
SELECT 
    COALESCE(SUM(quantity * cost_price), 0.0)::DOUBLE PRECISION AS total_cost_value
FROM products
WHERE company_id = $1 
    AND deleted_at IS NULL;
-- name: GetGlobalTotalStockQuantity :one
SELECT COALESCE(SUM(quantity), 0)::INT AS total_items
FROM products
WHERE company_id = $1 
    AND deleted_at IS NULL;