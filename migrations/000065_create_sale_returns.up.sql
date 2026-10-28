-- Vale-troca do cliente: saldo atual. O extrato fica em customer_credit_movements.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS store_credit NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (store_credit >= 0);

-- Parte da venda paga com crédito (vale-troca do cliente ou valor dos itens
-- devolvidos numa troca). Não é dinheiro novo: o caixa e o fluxo descontam.
ALTER TABLE sales
    ADD COLUMN IF NOT EXISTS credit_amount NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (credit_amount >= 0);

CREATE TYPE sale_return_kind AS ENUM ('return', 'exchange');

-- Devolução ou troca de itens de uma venda. O valor devolvido (returned_total)
-- é distribuído nesta ordem: itens novos da troca (exchange_credit), parcelas
-- em aberto da venda (debt_reduction) e o resto vira vale-troca (credit_amount)
-- ou dinheiro de volta (refund_amount).
CREATE TABLE IF NOT EXISTS sale_returns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    number INT NOT NULL,
    kind sale_return_kind NOT NULL,
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE RESTRICT,
    -- Venda nova com os itens levados na troca
    exchange_sale_id UUID NULL REFERENCES sales(id) ON DELETE RESTRICT,
    -- Cliente que recebeu o vale-troca (ou o cliente da venda)
    customer_id UUID NULL REFERENCES customers(id) ON DELETE RESTRICT,
    returned_total NUMERIC(12, 2) NOT NULL CHECK (returned_total > 0),
    exchange_total NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (exchange_total >= 0),
    exchange_credit NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (exchange_credit >= 0),
    debt_reduction NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (debt_reduction >= 0),
    credit_amount NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (credit_amount >= 0),
    refund_amount NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (refund_amount >= 0),
    refund_method VARCHAR(20) NULL,
    -- Diferença paga pelo cliente quando os itens novos valem mais
    difference_paid NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (difference_paid >= 0),
    difference_method VARCHAR(20) NULL,
    reason VARCHAR(30) NOT NULL,
    notes TEXT NULL,
    -- Feita fora do prazo de troca por um administrador
    out_of_window BOOLEAN NOT NULL DEFAULT FALSE,
    override_reason TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_sale_returns_number UNIQUE (company_id, number),
    CONSTRAINT chk_sale_returns_refund_method CHECK ((refund_amount > 0) = (refund_method IS NOT NULL)),
    CONSTRAINT chk_sale_returns_difference_method CHECK ((difference_paid > 0) = (difference_method IS NOT NULL)),
    CONSTRAINT chk_sale_returns_credit_customer CHECK (credit_amount = 0 OR customer_id IS NOT NULL),
    CONSTRAINT chk_sale_returns_override CHECK (NOT out_of_window OR override_reason IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_sale_returns_company_created_at ON sale_returns (company_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sale_returns_sale ON sale_returns (sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_returns_exchange_sale ON sale_returns (exchange_sale_id) WHERE exchange_sale_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sale_returns_user_created_at ON sale_returns (company_id, created_by, created_at);

-- Itens devolvidos. unit_amount é o valor líquido pago por unidade (preço menos
-- o desconto rateado da venda). restock = FALSE para peça avariada.
CREATE TABLE IF NOT EXISTS sale_return_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    return_id UUID NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    sale_item_id UUID NOT NULL REFERENCES sale_items(id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_amount NUMERIC(12, 4) NOT NULL CHECK (unit_amount >= 0),
    total NUMERIC(12, 2) NOT NULL CHECK (total >= 0),
    restock BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_sale_return_items_return ON sale_return_items (return_id);
CREATE INDEX IF NOT EXISTS idx_sale_return_items_sale_item ON sale_return_items (sale_item_id);

-- Extrato do vale-troca. amount > 0 entra crédito, < 0 sai.
CREATE TABLE IF NOT EXISTS customer_credit_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    -- return: vale gerado na devolução; sale: usado numa venda;
    -- sale_cancel: devolvido ao cancelar a venda que usou o vale
    type VARCHAR(20) NOT NULL CHECK (type IN ('return', 'sale', 'sale_cancel')),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount <> 0),
    balance_after NUMERIC(12, 2) NOT NULL CHECK (balance_after >= 0),
    sale_return_id UUID NULL REFERENCES sale_returns(id) ON DELETE SET NULL,
    sale_id UUID NULL REFERENCES sales(id) ON DELETE SET NULL,
    created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_customer_credit_movements_customer ON customer_credit_movements (customer_id, created_at DESC);

ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS sale_return_id UUID NULL REFERENCES sale_returns(id) ON DELETE SET NULL;
