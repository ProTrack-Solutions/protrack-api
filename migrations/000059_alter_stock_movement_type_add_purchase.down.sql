-- O Postgres não remove valores de enum; as movimentações de compra viram ENTRY/ADJUSTMENT
UPDATE stock_movements SET type = 'ENTRY' WHERE type = 'PURCHASE';
UPDATE stock_movements SET type = 'ADJUSTMENT' WHERE type = 'PURCHASE_REVERSAL';
