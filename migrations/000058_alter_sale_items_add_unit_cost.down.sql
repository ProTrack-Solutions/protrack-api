DROP INDEX IF EXISTS idx_sales_company_sale_at;

ALTER TABLE sale_items
    DROP COLUMN IF EXISTS unit_cost;
