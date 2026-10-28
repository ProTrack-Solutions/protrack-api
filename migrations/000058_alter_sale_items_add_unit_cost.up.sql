-- Custo unitário do produto no momento da venda. Sem ele o CMV dos relatórios
-- usaria o custo atual do cadastro, que muda a cada reajuste e distorce a
-- margem de vendas antigas. Itens já existentes recebem o custo atual como
-- melhor aproximação disponível.
ALTER TABLE sale_items
    ADD COLUMN IF NOT EXISTS unit_cost NUMERIC(12, 2) NULL CHECK (unit_cost >= 0);

UPDATE sale_items si
SET unit_cost = p.cost_price
FROM products p
WHERE p.id = si.product_id
    AND si.unit_cost IS NULL;

-- Relatórios filtram vendas por empresa e data da venda
CREATE INDEX IF NOT EXISTS idx_sales_company_sale_at ON sales (company_id, sale_at);
