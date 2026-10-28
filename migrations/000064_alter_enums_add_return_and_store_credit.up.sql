-- Valores novos ficam numa migration separada: o Postgres não deixa usar um
-- valor de enum na mesma transação em que ele foi criado.
ALTER TYPE stock_movement_type ADD VALUE IF NOT EXISTS 'RETURN';

-- payment_method_enum foi criado dentro de um bloco DO (o sqlc não o enxerga),
-- então a alteração também fica num bloco DO
DO $$
BEGIN
    ALTER TYPE payment_method_enum ADD VALUE IF NOT EXISTS 'store_credit';
END $$;
