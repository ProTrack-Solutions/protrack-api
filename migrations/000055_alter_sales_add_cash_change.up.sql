-- Dinheiro recebido no balcão e troco devolvido. Valem para a venda em dinheiro
-- (sobre o total) e para a entrada da venda a prazo. Nulos quando o operador não
-- informou o valor recebido (pagamento exato ou outra forma de pagamento).
ALTER TABLE sales
    ADD COLUMN IF NOT EXISTS amount_received NUMERIC(12, 2) NULL CHECK (amount_received >= 0),
    ADD COLUMN IF NOT EXISTS change_amount NUMERIC(12, 2) NULL CHECK (change_amount >= 0);
