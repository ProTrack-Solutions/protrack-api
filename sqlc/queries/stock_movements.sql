-- name: CreateStockMovement :exec
INSERT INTO stock_movements (
        company_id,
        product_id,
        sale_id,
        type,
        quantity,
        reason,
        created_by
    )
VALUES ($1, $2, $3, $4, $5, $6, $7);
