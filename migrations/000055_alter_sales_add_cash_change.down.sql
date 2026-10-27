ALTER TABLE sales
    DROP COLUMN IF EXISTS change_amount,
    DROP COLUMN IF EXISTS amount_received;
