CREATE TYPE unit_of_measure AS ENUM ('UN', 'KG', 'G', 'L', 'ML');

ALTER TABLE products
    ADD COLUMN unit unit_of_measure NOT NULL DEFAULT 'UN',
    ADD COLUMN sell_in_bulk BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE products
    ALTER COLUMN quantity DROP NOT NULL,
    ALTER COLUMN quantity DROP DEFAULT;

ALTER TABLE products
    ADD CONSTRAINT chk_quantity_required_when_not_bulk
    CHECK (sell_in_bulk = TRUE OR quantity IS NOT NULL);