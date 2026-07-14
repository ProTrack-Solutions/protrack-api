-- name: CreateProduct :one
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
        created_by
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
        $10
    )
RETURNING *;
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
UPDATE products
SET name = $2,
    description = $3,
    category_id = $4,
    barcode = $5,
    quantity = $6,
    size = $7,
    cost_price = $8,
    sale_price = $9,
    updated_by = $10,
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
    c.name AS category_name
FROM products p
JOIN product_categories c ON c.id = p.category_id
WHERE p.company_id = $1
    AND p.deleted_at IS NULL
ORDER BY p.created_at DESC
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