-- Entradas por compra e seus estornos ganham tipo próprio no kardex.
-- Fica numa migration separada: valor novo de enum só pode ser usado depois do commit.
ALTER TYPE stock_movement_type ADD VALUE IF NOT EXISTS 'PURCHASE';
ALTER TYPE stock_movement_type ADD VALUE IF NOT EXISTS 'PURCHASE_REVERSAL';
