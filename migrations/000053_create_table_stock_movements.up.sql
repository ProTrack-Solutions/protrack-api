CREATE TYPE stock_movement_type AS ENUM ('ENTRY', 'SALE', 'SALE_CANCEL', 'ADJUSTMENT');

-- Histórico de toda entrada e saída de estoque. products.quantity continua sendo o saldo atual;
-- esta tabela guarda o "porquê" de cada variação (quantity é positivo na entrada e negativo na saída).
CREATE TABLE IF NOT EXISTS stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sale_id UUID NULL REFERENCES sales(id) ON DELETE SET NULL,
    type stock_movement_type NOT NULL,
    quantity INT NOT NULL CHECK (quantity <> 0),
    reason TEXT NULL,
    created_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_created_at ON stock_movements (product_id, created_at);
CREATE INDEX IF NOT EXISTS idx_stock_movements_company_created_at ON stock_movements (company_id, created_at);
CREATE INDEX IF NOT EXISTS idx_stock_movements_sale ON stock_movements (sale_id) WHERE sale_id IS NOT NULL;

-- Backfill: reconstrói o histórico a partir das vendas já existentes (somente produtos com controle de estoque)
INSERT INTO stock_movements (company_id, product_id, sale_id, type, quantity, created_by, created_at)
SELECT s.company_id, si.product_id, s.id, 'SALE', -si.quantity, s.created_by, COALESCE(si.created_at, s.created_at)
FROM sale_items si
    INNER JOIN sales s ON s.id = si.sale_id
    INNER JOIN products p ON p.id = si.product_id
WHERE p.sell_in_bulk = FALSE;

INSERT INTO stock_movements (company_id, product_id, sale_id, type, quantity, created_by, created_at)
SELECT s.company_id, si.product_id, s.id, 'SALE_CANCEL', si.quantity, s.deleted_by, s.deleted_at
FROM sale_items si
    INNER JOIN sales s ON s.id = si.sale_id
    INNER JOIN products p ON p.id = si.product_id
WHERE p.sell_in_bulk = FALSE
    AND s.deleted_at IS NOT NULL;

-- Saldo inicial = estoque atual + o que saiu nas vendas não canceladas, para que a soma do histórico bata com products.quantity
INSERT INTO stock_movements (company_id, product_id, type, quantity, reason, created_by, created_at)
SELECT p.company_id, p.id, 'ENTRY', p.quantity + COALESCE(sold.total, 0), 'Saldo inicial (migração)', p.created_by, p.created_at
FROM products p
    LEFT JOIN (
        SELECT si.product_id, SUM(si.quantity) AS total
        FROM sale_items si
            INNER JOIN sales s ON s.id = si.sale_id
        WHERE s.deleted_at IS NULL
        GROUP BY si.product_id
    ) sold ON sold.product_id = p.id
WHERE p.sell_in_bulk = FALSE
    AND p.quantity + COALESCE(sold.total, 0) <> 0;
