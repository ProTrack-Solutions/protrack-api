-- name: LockCompanyPurchaseNumbers :exec
-- Serializa a numeração dos pedidos da empresa até o fim da transação
SELECT pg_advisory_xact_lock(hashtext('purchase_orders:' || sqlc.arg(company_id)::TEXT));

-- name: NextPurchaseOrderNumber :one
SELECT (COALESCE(MAX(number), 0) + 1)::INT AS number
FROM purchase_orders
WHERE company_id = $1;

-- name: GetPurchaseVendor :one
SELECT id, name, is_active
FROM vendors
WHERE id = $1
    AND company_id = $2;

-- name: ListProductsForPurchase :many
SELECT id, name, sell_in_bulk
FROM products
WHERE company_id = sqlc.arg(company_id)
    AND id = ANY(sqlc.arg(ids)::UUID[])
    AND deleted_at IS NULL;

-- name: CreatePurchaseOrder :one
INSERT INTO purchase_orders (
        company_id,
        number,
        vendor_id,
        status,
        expected_date,
        notes,
        created_by,
        sent_at
    )
VALUES (
        sqlc.arg(company_id),
        sqlc.arg(number),
        sqlc.arg(vendor_id),
        sqlc.arg(status),
        sqlc.narg(expected_date),
        sqlc.narg(notes),
        sqlc.arg(created_by),
        CASE WHEN sqlc.arg(status)::purchase_order_status = 'sent' THEN NOW() END
    )
RETURNING id;

-- name: UpdatePurchaseOrderHeader :exec
UPDATE purchase_orders
SET vendor_id = sqlc.arg(vendor_id),
    expected_date = sqlc.narg(expected_date),
    notes = sqlc.narg(notes),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: DeletePurchaseOrderItems :exec
DELETE FROM purchase_order_items
WHERE order_id = $1;

-- name: CreatePurchaseOrderItem :exec
INSERT INTO purchase_order_items (order_id, product_id, quantity, unit_cost)
VALUES ($1, $2, $3, $4);

-- name: LockPurchaseOrder :one
SELECT id, company_id, number, vendor_id, status
FROM purchase_orders
WHERE id = $1
    AND company_id = $2
FOR UPDATE;

-- name: UpdatePurchaseOrderStatus :exec
-- sent_at é gravado só na primeira vez que o pedido sai do rascunho
UPDATE purchase_orders
SET status = sqlc.arg(status),
    sent_at = CASE
        WHEN sent_at IS NULL AND sqlc.arg(status)::purchase_order_status <> 'draft' THEN NOW()
        ELSE sent_at
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ClosePurchaseOrder :exec
UPDATE purchase_orders
SET status = sqlc.arg(status),
    closed_at = NOW(),
    closed_by = sqlc.arg(closed_by),
    close_reason = sqlc.narg(close_reason),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: GetPurchaseOrder :one
SELECT po.id,
    po.number,
    po.vendor_id,
    v.name::TEXT AS vendor_name,
    po.status::TEXT AS status,
    po.expected_date,
    po.notes,
    COALESCE(u.name, '')::TEXT AS created_by_name,
    po.created_at,
    po.updated_at,
    po.sent_at,
    po.closed_at,
    po.close_reason
FROM purchase_orders po
    INNER JOIN vendors v ON v.id = po.vendor_id
    LEFT JOIN users u ON u.id = po.created_by
WHERE po.id = $1
    AND po.company_id = $2;

-- name: ListPurchaseOrderItems :many
SELECT poi.id,
    poi.product_id,
    p.name::TEXT AS product_name,
    COALESCE(p.model_name, '')::TEXT AS model_name,
    COALESCE(p.color, '')::TEXT AS color,
    COALESCE(p.size, '')::TEXT AS size,
    COALESCE(p.barcode, '')::TEXT AS barcode,
    p.unit::TEXT AS unit,
    p.sell_in_bulk,
    COALESCE(p.quantity, 0)::INT AS stock_quantity,
    COALESCE(p.cost_price, 0)::FLOAT AS current_cost,
    COALESCE(p.sale_price, 0)::FLOAT AS sale_price,
    poi.quantity,
    poi.unit_cost::FLOAT AS unit_cost,
    poi.received_quantity
FROM purchase_order_items poi
    INNER JOIN products p ON p.id = poi.product_id
WHERE poi.order_id = $1
ORDER BY p.name;

-- name: GetPurchaseOrderItemTotals :one
SELECT COALESCE(SUM(quantity), 0)::INT AS quantity,
    COALESCE(SUM(received_quantity), 0)::INT AS received_quantity
FROM purchase_order_items
WHERE order_id = $1;

-- name: ListPurchaseOrders :many
-- Filtros opcionais: status, fornecedor e busca por número ou nome do fornecedor
SELECT po.id,
    po.number,
    v.name::TEXT AS vendor_name,
    po.status::TEXT AS status,
    po.expected_date,
    po.created_at,
    COALESCE(it.items_count, 0)::INT AS items_count,
    COALESCE(it.total, 0)::FLOAT AS total,
    COALESCE(it.quantity, 0)::INT AS quantity,
    COALESCE(it.received_quantity, 0)::INT AS received_quantity,
    COALESCE(rc.received_total, 0)::FLOAT AS received_total,
    COUNT(*) OVER()::INT AS total_rows
FROM purchase_orders po
    INNER JOIN vendors v ON v.id = po.vendor_id
    LEFT JOIN LATERAL (
        SELECT COUNT(*) AS items_count,
            SUM(poi.quantity * poi.unit_cost) AS total,
            SUM(poi.quantity) AS quantity,
            SUM(poi.received_quantity) AS received_quantity
        FROM purchase_order_items poi
        WHERE poi.order_id = po.id
    ) it ON TRUE
    LEFT JOIN LATERAL (
        SELECT SUM(pr.total) AS received_total
        FROM purchase_receipts pr
        WHERE pr.order_id = po.id
            AND pr.status = 'active'
    ) rc ON TRUE
WHERE po.company_id = sqlc.arg(company_id)
    AND (sqlc.narg(status)::purchase_order_status IS NULL OR po.status = sqlc.narg(status))
    AND (sqlc.narg(vendor_id)::UUID IS NULL OR po.vendor_id = sqlc.narg(vendor_id))
    AND (
        sqlc.narg(search)::TEXT IS NULL
        OR v.name ILIKE '%' || sqlc.narg(search) || '%'
        OR po.number::TEXT = sqlc.narg(search)
    )
ORDER BY po.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountPurchaseOrdersByStatus :many
SELECT status::TEXT AS status,
    COUNT(*)::INT AS count
FROM purchase_orders
WHERE company_id = $1
GROUP BY status;

-- name: GetPurchasesOverview :one
-- Indicadores da tela de compras: o que está a caminho e o que entrou no mês
SELECT (
        SELECT COALESCE(SUM((poi.quantity - poi.received_quantity) * poi.unit_cost), 0)
        FROM purchase_order_items poi
            INNER JOIN purchase_orders po ON po.id = poi.order_id
        WHERE po.company_id = sqlc.arg(company_id)
            AND po.status IN ('sent', 'partial')
    )::FLOAT AS pending_total,
    (
        SELECT COALESCE(SUM(pr.total), 0)
        FROM purchase_receipts pr
        WHERE pr.company_id = sqlc.arg(company_id)
            AND pr.status = 'active'
            AND pr.received_at >= sqlc.arg(month_start)::TIMESTAMPTZ
    )::FLOAT AS received_this_month,
    (
        SELECT COUNT(*)
        FROM purchase_orders po
        WHERE po.company_id = sqlc.arg(company_id)
            AND po.status IN ('sent', 'partial')
            AND po.expected_date < sqlc.arg(today)::DATE
    )::INT AS late_orders;

-- name: LockProductForPurchase :one
SELECT id,
    name,
    sell_in_bulk,
    COALESCE(quantity, 0)::INT AS quantity,
    COALESCE(cost_price, 0)::FLOAT AS cost_price,
    COALESCE(sale_price, 0)::FLOAT AS sale_price
FROM products
WHERE id = $1
    AND company_id = $2
    AND deleted_at IS NULL
FOR UPDATE;

-- name: ApplyPurchaseToProduct :exec
-- Soma (ou subtrai, no estorno) o saldo e grava o novo custo; preço de venda só quando informado
UPDATE products
SET quantity = CASE
        WHEN sell_in_bulk THEN quantity
        ELSE COALESCE(quantity, 0) + sqlc.arg(quantity_delta)::INT
    END,
    cost_price = sqlc.arg(cost_price),
    sale_price = COALESCE(sqlc.narg(sale_price), sale_price),
    updated_by = sqlc.arg(updated_by),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: CreatePurchaseStockMovement :exec
INSERT INTO stock_movements (
        company_id,
        product_id,
        purchase_receipt_id,
        type,
        quantity,
        reason,
        created_by
    )
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: AddReceivedQuantity :one
-- Nenhuma linha = passaria da quantidade pedida (ou item de outro pedido)
UPDATE purchase_order_items
SET received_quantity = received_quantity + sqlc.arg(quantity)::INT
WHERE id = sqlc.arg(id)
    AND order_id = sqlc.arg(order_id)
    AND received_quantity + sqlc.arg(quantity)::INT BETWEEN 0 AND quantity
RETURNING id;

-- name: CreatePurchaseReceipt :one
INSERT INTO purchase_receipts (
        company_id,
        order_id,
        received_at,
        invoice_number,
        invoice_series,
        invoice_key,
        items_total,
        freight,
        other_expenses,
        discount,
        total,
        notes,
        created_by
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id;

-- name: CreatePurchaseReceiptItem :exec
INSERT INTO purchase_receipt_items (
        receipt_id,
        order_item_id,
        product_id,
        quantity,
        unit_cost,
        landed_unit_cost,
        previous_cost,
        new_cost,
        previous_quantity
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CreatePurchaseBill :exec
INSERT INTO bills_payable (
        company_id,
        vendor_id,
        category_id,
        payment_method_id,
        amount,
        due_date,
        status,
        description,
        purchase_receipt_id
    )
VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8);

-- name: ListPurchaseReceipts :many
SELECT pr.id,
    pr.status::TEXT AS status,
    pr.received_at,
    COALESCE(pr.invoice_number, '')::TEXT AS invoice_number,
    COALESCE(pr.invoice_series, '')::TEXT AS invoice_series,
    COALESCE(pr.invoice_key, '')::TEXT AS invoice_key,
    pr.items_total::FLOAT AS items_total,
    pr.freight::FLOAT AS freight,
    pr.other_expenses::FLOAT AS other_expenses,
    pr.discount::FLOAT AS discount,
    pr.total::FLOAT AS total,
    pr.notes,
    COALESCE(u.name, '')::TEXT AS created_by_name,
    pr.reversed_at,
    COALESCE(ru.name, '')::TEXT AS reversed_by_name,
    pr.reversal_reason
FROM purchase_receipts pr
    LEFT JOIN users u ON u.id = pr.created_by
    LEFT JOIN users ru ON ru.id = pr.reversed_by
WHERE pr.order_id = $1
ORDER BY pr.received_at, pr.created_at;

-- name: ListPurchaseReceiptItemsByOrder :many
SELECT pri.receipt_id,
    pri.order_item_id,
    pri.product_id,
    p.name::TEXT AS product_name,
    pri.quantity,
    pri.unit_cost::FLOAT AS unit_cost,
    pri.landed_unit_cost::FLOAT AS landed_unit_cost,
    COALESCE(pri.previous_cost, 0)::FLOAT AS previous_cost,
    pri.new_cost::FLOAT AS new_cost
FROM purchase_receipt_items pri
    INNER JOIN purchase_receipts pr ON pr.id = pri.receipt_id
    INNER JOIN products p ON p.id = pri.product_id
WHERE pr.order_id = $1
ORDER BY p.name;

-- name: ListPurchaseBillsByOrder :many
SELECT b.id,
    b.purchase_receipt_id,
    b.amount::FLOAT AS amount,
    b.due_date,
    b.status::TEXT AS status,
    COALESCE(b.amount_paid, 0)::FLOAT AS amount_paid,
    b.payment_date,
    COALESCE(b.description, '')::TEXT AS description
FROM bills_payable b
    INNER JOIN purchase_receipts pr ON pr.id = b.purchase_receipt_id
WHERE pr.order_id = $1
ORDER BY b.due_date;

-- name: LockPurchaseReceipt :one
SELECT pr.id,
    pr.order_id,
    pr.status::TEXT AS status,
    po.number AS order_number
FROM purchase_receipts pr
    INNER JOIN purchase_orders po ON po.id = pr.order_id
WHERE pr.id = $1
    AND pr.company_id = $2
FOR UPDATE OF pr;

-- name: ListPurchaseReceiptItems :many
SELECT id,
    order_item_id,
    product_id,
    quantity,
    landed_unit_cost::FLOAT AS landed_unit_cost,
    COALESCE(previous_cost, 0)::FLOAT AS previous_cost,
    new_cost::FLOAT AS new_cost
FROM purchase_receipt_items
WHERE receipt_id = $1;

-- name: CountPaidPurchaseBills :one
-- Parcelas pagas (total ou parcialmente) travam o estorno do recebimento
SELECT COUNT(*)::INT
FROM bills_payable
WHERE purchase_receipt_id = $1
    AND status <> 'canceled'
    AND (status = 'paid' OR COALESCE(amount_paid, 0) > 0);

-- name: CancelPurchaseBills :exec
UPDATE bills_payable
SET status = 'canceled',
    notes = CONCAT_WS(E'\n', NULLIF(notes, ''), sqlc.arg(note)::TEXT),
    updated_at = NOW()
WHERE purchase_receipt_id = sqlc.arg(purchase_receipt_id)
    AND status <> 'canceled';

-- name: MarkPurchaseReceiptReversed :exec
UPDATE purchase_receipts
SET status = 'reversed',
    reversed_at = NOW(),
    reversed_by = sqlc.arg(reversed_by),
    reversal_reason = sqlc.narg(reversal_reason)
WHERE id = sqlc.arg(id);

-- name: BillCategoryBelongsToCompany :one
SELECT EXISTS (
        SELECT 1
        FROM bill_categories
        WHERE id = $1
            AND company_id = $2
    );

-- name: PaymentMethodBelongsToCompany :one
SELECT EXISTS (
        SELECT 1
        FROM payment_methods
        WHERE id = $1
            AND company_id = $2
    );

-- name: SearchPurchaseProducts :many
-- Busca de produtos para o pedido (Compras não depende do módulo Estoque)
SELECT p.id,
    p.name::TEXT AS name,
    COALESCE(p.model_name, '')::TEXT AS model_name,
    COALESCE(p.color, '')::TEXT AS color,
    COALESCE(p.size, '')::TEXT AS size,
    COALESCE(p.barcode, '')::TEXT AS barcode,
    p.unit::TEXT AS unit,
    p.sell_in_bulk,
    COALESCE(p.quantity, 0)::INT AS quantity,
    COALESCE(p.cost_price, 0)::FLOAT AS cost_price,
    COALESCE(p.sale_price, 0)::FLOAT AS sale_price,
    COUNT(*) OVER()::INT AS total_rows
FROM products p
WHERE p.company_id = sqlc.arg(company_id)
    AND p.deleted_at IS NULL
    AND (
        sqlc.arg(search)::TEXT = ''
        OR p.name ILIKE '%' || sqlc.arg(search)::TEXT || '%'
        OR p.barcode = sqlc.arg(search)::TEXT
    )
ORDER BY p.name
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: SearchPurchaseModels :many
-- Modelos com mais de uma variação que casam com a busca (atalho de recompra)
SELECT MIN(p.model_name)::TEXT AS model_name,
    COUNT(*)::INT AS variations
FROM products p
WHERE p.company_id = sqlc.arg(company_id)
    AND p.deleted_at IS NULL
    AND p.model_name IS NOT NULL
    AND p.model_name ILIKE '%' || sqlc.arg(search)::TEXT || '%'
GROUP BY lower(p.model_name)
HAVING COUNT(*) > 1
ORDER BY 1
LIMIT 5;

-- name: ListModelVariations :many
SELECT p.id,
    p.name::TEXT AS name,
    p.model_name::TEXT AS model_name,
    COALESCE(p.color, '')::TEXT AS color,
    COALESCE(p.size, '')::TEXT AS size,
    COALESCE(p.barcode, '')::TEXT AS barcode,
    p.unit::TEXT AS unit,
    p.sell_in_bulk,
    COALESCE(p.quantity, 0)::INT AS quantity,
    COALESCE(p.cost_price, 0)::FLOAT AS cost_price,
    COALESCE(p.sale_price, 0)::FLOAT AS sale_price
FROM products p
WHERE p.company_id = $1
    AND p.deleted_at IS NULL
    AND lower(p.model_name) = lower(sqlc.arg(model_name)::TEXT)
ORDER BY p.color, p.created_at;

-- name: ListPurchaseCategories :many
SELECT id, name
FROM product_categories
WHERE company_id = $1
    AND status = 'ACTIVE'
ORDER BY name;

-- name: GetPurchaseCategory :one
SELECT id
FROM product_categories
WHERE id = $1
    AND company_id = $2
    AND status = 'ACTIVE';

-- name: CountCompanyProducts :one
SELECT COUNT(*)::INT
FROM products
WHERE company_id = $1
    AND deleted_at IS NULL;

-- name: BarcodeExists :one
SELECT EXISTS (
        SELECT 1
        FROM products
        WHERE company_id = $1
            AND barcode = $2
            AND deleted_at IS NULL
    );

-- name: CreatePurchaseProduct :one
-- Variação criada pelo pedido: nasce sem estoque (entra no recebimento)
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
        unit,
        sell_in_bulk,
        created_by
    )
VALUES (
        sqlc.arg(company_id),
        sqlc.arg(category_id),
        sqlc.arg(name),
        sqlc.narg(description),
        sqlc.arg(model_name),
        sqlc.narg(color),
        sqlc.narg(size),
        sqlc.arg(barcode),
        0,
        sqlc.arg(cost_price),
        sqlc.arg(sale_price),
        'UN',
        FALSE,
        sqlc.arg(created_by)
    )
RETURNING id;

-- name: GetPurchaseEmailContext :one
-- Dados de empresa, fornecedor e usuário para montar o e-mail do pedido
SELECT po.id,
    po.number,
    po.status::TEXT AS status,
    COALESCE(v.name, '')::TEXT AS vendor_name,
    COALESCE(v.email, '')::TEXT AS vendor_email,
    COALESCE(NULLIF(co.trade_name, ''), co.name)::TEXT AS company_name,
    COALESCE(co.document, '')::TEXT AS company_document,
    COALESCE(co.email, '')::TEXT AS company_email,
    COALESCE(co.phone, '')::TEXT AS company_phone,
    COALESCE(u.name, '')::TEXT AS user_name,
    COALESCE(u.email, '')::TEXT AS user_email
FROM purchase_orders po
    INNER JOIN vendors v ON v.id = po.vendor_id
    INNER JOIN companies co ON co.id = po.company_id
    LEFT JOIN users u ON u.id = sqlc.arg(user_id)
    AND u.company_id = po.company_id
WHERE po.id = sqlc.arg(order_id)
    AND po.company_id = sqlc.arg(company_id);

-- name: CreatePurchaseOrderEmail :exec
INSERT INTO purchase_order_emails (company_id, order_id, kind, recipient, cc, sent_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListPurchaseOrderEmails :many
SELECT e.kind,
    e.recipient,
    e.created_at,
    COALESCE(u.name, '')::TEXT AS sent_by_name
FROM purchase_order_emails e
    LEFT JOIN users u ON u.id = e.sent_by
WHERE e.order_id = $1
ORDER BY e.created_at DESC
LIMIT 10;

-- name: GetPurchaseOrderVendorEmail :one
SELECT COALESCE(v.email, '')::TEXT
FROM purchase_orders po
    INNER JOIN vendors v ON v.id = po.vendor_id
WHERE po.id = $1;
