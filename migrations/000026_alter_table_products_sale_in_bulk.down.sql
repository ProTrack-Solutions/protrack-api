
ALTER TABLE products
    DROP CONSTRAINT IF EXISTS chk_quantity_required_when_not_bulk;

UPDATE products SET quantity = 0 WHERE quantity IS NULL;

ALTER TABLE products
    ALTER COLUMN quantity SET DEFAULT 0,
    ALTER COLUMN quantity SET NOT NULL;

ALTER TABLE products
    DROP COLUMN IF EXISTS sell_in_bulk,
    DROP COLUMN IF EXISTS unit;

DROP TYPE IF EXISTS unit_of_measure;