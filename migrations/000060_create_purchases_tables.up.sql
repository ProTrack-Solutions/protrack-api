-- Compras: pedido ao fornecedor e recebimentos (entradas de mercadoria).
-- O estoque, o custo médio e as contas a pagar só mudam no recebimento.
CREATE TYPE purchase_order_status AS ENUM (
    'draft',     -- rascunho, ainda editável
    'sent',      -- enviado ao fornecedor, aguardando mercadoria
    'partial',   -- recebido em parte
    'received',  -- tudo recebido
    'closed',    -- encerrado manualmente com saldo pendente descartado
    'canceled'
);

CREATE TYPE purchase_receipt_status AS ENUM ('active', 'reversed');

CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    number INT NOT NULL,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    status purchase_order_status NOT NULL DEFAULT 'draft',
    expected_date DATE NULL,
    notes TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    closed_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    close_reason TEXT NULL,
    CONSTRAINT uq_purchase_orders_number UNIQUE (company_id, number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_company_status ON purchase_orders (company_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_cost NUMERIC(12, 4) NOT NULL CHECK (unit_cost >= 0),
    received_quantity INT NOT NULL DEFAULT 0 CHECK (received_quantity >= 0 AND received_quantity <= quantity),
    CONSTRAINT uq_purchase_order_items_product UNIQUE (order_id, product_id)
);

CREATE TABLE IF NOT EXISTS purchase_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE RESTRICT,
    status purchase_receipt_status NOT NULL DEFAULT 'active',
    received_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Dados da nota do fornecedor (preenchidos à mão até a importação do XML)
    invoice_number VARCHAR(20) NULL,
    invoice_series VARCHAR(5) NULL,
    invoice_key VARCHAR(44) NULL,
    items_total NUMERIC(12, 2) NOT NULL CHECK (items_total >= 0),
    freight NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (freight >= 0),
    other_expenses NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (other_expenses >= 0),
    discount NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    total NUMERIC(12, 2) NOT NULL CHECK (total >= 0),
    notes TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reversed_at TIMESTAMPTZ NULL,
    reversed_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    reversal_reason TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_order ON purchase_receipts (order_id, received_at);
CREATE INDEX IF NOT EXISTS idx_purchase_receipts_company_received_at ON purchase_receipts (company_id, received_at);

-- Cada item guarda o retrato do custo antes/depois, usado no estorno
CREATE TABLE IF NOT EXISTS purchase_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES purchase_receipts(id) ON DELETE CASCADE,
    order_item_id UUID NOT NULL REFERENCES purchase_order_items(id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    -- Preço unitário da nota e custo unitário final com frete/despesas/desconto rateados
    unit_cost NUMERIC(12, 4) NOT NULL CHECK (unit_cost >= 0),
    landed_unit_cost NUMERIC(12, 4) NOT NULL CHECK (landed_unit_cost >= 0),
    previous_cost NUMERIC(12, 2) NULL,
    new_cost NUMERIC(12, 2) NOT NULL,
    previous_quantity INT NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_receipt_items_receipt ON purchase_receipt_items (receipt_id);

-- Contas a pagar geradas pelo recebimento
ALTER TABLE bills_payable
    ADD COLUMN IF NOT EXISTS purchase_receipt_id UUID NULL REFERENCES purchase_receipts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bills_payable_purchase_receipt ON bills_payable (purchase_receipt_id) WHERE purchase_receipt_id IS NOT NULL;

-- Movimentação de estoque aponta para o recebimento que a originou
ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS purchase_receipt_id UUID NULL REFERENCES purchase_receipts(id) ON DELETE SET NULL;

-- Módulo liberável por departamento
INSERT INTO modules (code, name) VALUES ('purchases', 'Compras') ON CONFLICT (code) DO NOTHING;
