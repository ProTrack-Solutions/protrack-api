-- Grade de produtos: cada combinação cor × tamanho é um produto normal,
-- agrupado por model_name (ver migração 000062).

-- name: ListStockEntries :many
-- Estoque agrupado: cada modelo vira uma linha (com os totais das variações) e
-- cada produto sem modelo continua sendo uma linha própria. A busca casa com
-- qualquer variação, mas os totais do modelo sempre consideram todas elas.
WITH base AS (
    SELECT p.*,
        CASE
            WHEN p.model_name IS NULL THEN 'p:' || p.id::TEXT
            ELSE 'm:' || lower(p.model_name)
        END AS entry_key
    FROM products p
    WHERE p.company_id = sqlc.arg(company_id)::UUID
        AND p.deleted_at IS NULL
),
matched AS (
    SELECT DISTINCT b.entry_key
    FROM base b
        LEFT JOIN product_categories c ON c.id = b.category_id
    WHERE (
            sqlc.arg(search)::TEXT = ''
            OR to_tsvector(
                'portuguese_unaccent',
                coalesce(b.name, '') || ' ' || coalesce(b.description, '')
            ) @@ plainto_tsquery('portuguese_unaccent', sqlc.arg(search)::TEXT)
            OR b.name ILIKE '%' || sqlc.arg(search)::TEXT || '%'
            OR b.model_name ILIKE '%' || sqlc.arg(search)::TEXT || '%'
            OR b.barcode ILIKE '%' || sqlc.arg(search)::TEXT || '%'
            OR c.name ILIKE '%' || sqlc.arg(search)::TEXT || '%'
        )
        AND (sqlc.narg(start_date)::DATE IS NULL OR b.created_at >= sqlc.narg(start_date)::DATE)
        AND (sqlc.narg(end_date)::DATE IS NULL OR b.created_at < (sqlc.narg(end_date)::DATE + INTERVAL '1 day'))
),
low_stock AS (
    SELECT COALESCE((
            SELECT (cs.value #>> '{}')::NUMERIC
            FROM company_settings cs
            WHERE cs.company_id = sqlc.arg(company_id)::UUID
                AND cs.key = 'low_stock'
        ), 5) AS threshold
),
entries AS (
    SELECT b.entry_key::TEXT AS entry_key,
        BOOL_OR(b.model_name IS NOT NULL) AS is_model,
        COALESCE(MIN(b.model_name), '')::TEXT AS model_name,
        (array_agg(b.id ORDER BY b.created_at))[1]::UUID AS product_id,
        (array_agg(b.name ORDER BY b.created_at))[1]::TEXT AS name,
        COALESCE((array_agg(b.description ORDER BY b.created_at))[1], '')::TEXT AS description,
        COALESCE((array_agg(b.barcode ORDER BY b.created_at))[1], '')::TEXT AS barcode,
        COALESCE((array_agg(b.size ORDER BY b.created_at))[1], '')::TEXT AS size,
        COALESCE((array_agg(b.color ORDER BY b.created_at))[1], '')::TEXT AS color,
        (array_agg(b.category_id ORDER BY b.created_at))[1]::UUID AS category_id,
        COALESCE(MIN(c.name), '')::TEXT AS category_name,
        MIN(b.unit::TEXT)::TEXT AS unit,
        BOOL_OR(b.sell_in_bulk) AS sell_in_bulk,
        COUNT(*)::INT AS variations,
        COUNT(DISTINCT lower(b.color))::INT AS colors,
        COUNT(DISTINCT lower(b.size))::INT AS sizes,
        COALESCE(SUM(b.quantity), 0)::INT AS quantity,
        COUNT(*) FILTER (
            WHERE NOT b.sell_in_bulk
                AND b.quantity < (SELECT threshold FROM low_stock)
        )::INT AS low_stock_variations,
        COALESCE(MIN(b.cost_price), 0)::FLOAT AS min_cost_price,
        COALESCE(MAX(b.cost_price), 0)::FLOAT AS max_cost_price,
        COALESCE(MIN(b.sale_price), 0)::FLOAT AS min_sale_price,
        COALESCE(MAX(b.sale_price), 0)::FLOAT AS max_sale_price,
        COALESCE(SUM(sm.total_registered), 0)::INT AS total_registered,
        COALESCE(SUM(sm.total_sold), 0)::INT AS total_sold,
        MIN(b.created_at)::TIMESTAMPTZ AS created_at
    FROM base b
        INNER JOIN matched m ON m.entry_key = b.entry_key
        LEFT JOIN product_categories c ON c.id = b.category_id
        LEFT JOIN LATERAL (
            SELECT SUM(mv.quantity) FILTER (WHERE mv.type = 'ENTRY') AS total_registered,
                - SUM(mv.quantity) FILTER (WHERE mv.type IN ('SALE', 'SALE_CANCEL')) AS total_sold
            FROM stock_movements mv
            WHERE mv.product_id = b.id
        ) sm ON TRUE
    GROUP BY b.entry_key
)
SELECT e.*,
    COUNT(*) OVER() AS total_count
FROM entries e
ORDER BY CASE WHEN sqlc.arg(order_by)::TEXT = 'desc' THEN e.created_at END DESC,
    CASE WHEN sqlc.arg(order_by)::TEXT <> 'desc' THEN e.created_at END ASC,
    e.entry_key
LIMIT sqlc.arg(page_limit)::INT OFFSET sqlc.arg(page_offset)::INT;

-- name: ListGridVariations :many
-- Todas as variações de um modelo, na ordem em que foram cadastradas
SELECT p.id,
    p.name,
    p.model_name::TEXT AS model_name,
    COALESCE(p.color, '')::TEXT AS color,
    COALESCE(p.size, '')::TEXT AS size,
    COALESCE(p.barcode, '')::TEXT AS barcode,
    COALESCE(p.description, '')::TEXT AS description,
    p.category_id,
    COALESCE(c.name, '')::TEXT AS category_name,
    p.unit::TEXT AS unit,
    p.sell_in_bulk,
    p.quantity,
    COALESCE(p.cost_price, 0)::FLOAT AS cost_price,
    COALESCE(p.sale_price, 0)::FLOAT AS sale_price,
    COALESCE(sm.total_registered, 0)::INT AS total_registered,
    COALESCE(sm.total_sold, 0)::INT AS total_sold,
    p.created_at
FROM products p
    LEFT JOIN product_categories c ON c.id = p.category_id
    LEFT JOIN LATERAL (
        SELECT SUM(mv.quantity) FILTER (WHERE mv.type = 'ENTRY') AS total_registered,
            - SUM(mv.quantity) FILTER (WHERE mv.type IN ('SALE', 'SALE_CANCEL')) AS total_sold
        FROM stock_movements mv
        WHERE mv.product_id = p.id
    ) sm ON TRUE
WHERE p.company_id = sqlc.arg(company_id)::UUID
    AND p.deleted_at IS NULL
    AND p.model_name IS NOT NULL
    AND lower(p.model_name) = lower(sqlc.arg(model_name)::TEXT)
ORDER BY p.created_at,
    p.id;

-- name: LockGridProducts :many
-- Trava as linhas dos produtos (variações ou produtos a agrupar) até o fim da transação
SELECT p.id,
    p.name,
    p.model_name,
    COALESCE(p.color, '')::TEXT AS color,
    COALESCE(p.size, '')::TEXT AS size,
    p.category_id,
    p.sell_in_bulk,
    p.quantity
FROM products p
WHERE p.company_id = sqlc.arg(company_id)::UUID
    AND p.deleted_at IS NULL
    AND p.id = ANY(sqlc.arg(ids)::UUID [])
FOR UPDATE;

-- name: CreateGridProduct :one
-- Variação nova com a entrada inicial de estoque na mesma instrução
WITH new_product AS (
    INSERT INTO products (
            company_id,
            category_id,
            name,
            description,
            model_name,
            color,
            size,
            barcode,
            quantity,
            cost_price,
            sale_price,
            created_by,
            sell_in_bulk,
            unit,
            created_at
        )
    VALUES (
            sqlc.arg(company_id)::UUID,
            sqlc.arg(category_id)::UUID,
            sqlc.arg(name)::TEXT,
            sqlc.narg(description)::TEXT,
            sqlc.arg(model_name)::TEXT,
            sqlc.narg(color)::TEXT,
            sqlc.narg(size)::TEXT,
            sqlc.arg(barcode)::TEXT,
            sqlc.arg(quantity)::INT,
            sqlc.arg(cost_price)::NUMERIC,
            sqlc.arg(sale_price)::NUMERIC,
            sqlc.arg(created_by)::UUID,
            FALSE,
            sqlc.arg(unit)::unit_of_measure,
            -- clock_timestamp: dentro da transação NOW() é igual para todas e a ordem da grade se perderia
            clock_timestamp()
        )
    RETURNING id,
        company_id,
        quantity,
        created_by
),
initial_stock AS (
    INSERT INTO stock_movements (company_id, product_id, type, quantity, reason, created_by)
    SELECT np.company_id,
        np.id,
        'ENTRY',
        np.quantity,
        'Cadastro do produto',
        np.created_by
    FROM new_product np
    WHERE np.quantity <> 0
)
SELECT id
FROM new_product;

-- name: UpdateGridVariation :exec
-- Edição do modelo: nome, categoria e descrição valem para todas as variações; preços por variação
UPDATE products
SET model_name = sqlc.arg(model_name)::TEXT,
    name = sqlc.arg(name)::TEXT,
    category_id = sqlc.arg(category_id)::UUID,
    description = sqlc.narg(description)::TEXT,
    cost_price = sqlc.arg(cost_price)::NUMERIC,
    sale_price = sqlc.arg(sale_price)::NUMERIC,
    barcode = COALESCE(sqlc.narg(barcode)::TEXT, barcode),
    updated_by = sqlc.arg(updated_by)::UUID,
    updated_at = NOW()
WHERE id = sqlc.arg(id)::UUID
    AND company_id = sqlc.arg(company_id)::UUID
    AND deleted_at IS NULL;

-- name: AssignProductToModel :exec
-- Agrupa um produto já existente num modelo; saldo, histórico e vendas continuam no mesmo produto
UPDATE products
SET model_name = sqlc.arg(model_name)::TEXT,
    color = sqlc.narg(color)::TEXT,
    size = sqlc.narg(size)::TEXT,
    name = sqlc.arg(name)::TEXT,
    category_id = sqlc.arg(category_id)::UUID,
    updated_by = sqlc.arg(updated_by)::UUID,
    updated_at = NOW()
WHERE id = sqlc.arg(id)::UUID
    AND company_id = sqlc.arg(company_id)::UUID
    AND deleted_at IS NULL;

-- name: RemoveGridVariation :execrows
-- Desativa uma variação sem saldo (exclusão lógica, o histórico continua)
UPDATE products
SET deleted_at = NOW(),
    deleted_by = sqlc.arg(deleted_by)::UUID
WHERE id = sqlc.arg(id)::UUID
    AND company_id = sqlc.arg(company_id)::UUID
    AND model_name IS NOT NULL
    AND deleted_at IS NULL
    AND quantity = 0;

-- name: GridCategoryExists :one
SELECT EXISTS (
        SELECT 1
        FROM product_categories
        WHERE id = sqlc.arg(id)::UUID
            AND company_id = sqlc.arg(company_id)::UUID
            AND deleted_at IS NULL
    );

-- name: GridBarcodeExists :one
SELECT EXISTS (
        SELECT 1
        FROM products
        WHERE company_id = sqlc.arg(company_id)::UUID
            AND barcode = sqlc.arg(barcode)::TEXT
            AND deleted_at IS NULL
    );

-- name: CountActiveProducts :one
SELECT COUNT(*)::INT
FROM products
WHERE company_id = sqlc.arg(company_id)::UUID
    AND deleted_at IS NULL;
