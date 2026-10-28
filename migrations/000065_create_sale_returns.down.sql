ALTER TABLE stock_movements DROP COLUMN IF EXISTS sale_return_id;
DROP TABLE IF EXISTS customer_credit_movements;
DROP TABLE IF EXISTS sale_return_items;
DROP TABLE IF EXISTS sale_returns;
DROP TYPE IF EXISTS sale_return_kind;
ALTER TABLE sales DROP COLUMN IF EXISTS credit_amount;
ALTER TABLE customers DROP COLUMN IF EXISTS store_credit;
