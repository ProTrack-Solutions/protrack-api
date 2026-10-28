DROP INDEX IF EXISTS uq_products_model_variation;
DROP INDEX IF EXISTS idx_products_company_model;

ALTER TABLE products
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS model_name;
