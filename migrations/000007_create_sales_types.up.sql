-- Status da Conta/Venda
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'sale_status_enum') THEN
        CREATE TYPE sale_status_enum AS ENUM (
            'pending', 
            'paid', 
            'overdue', 
            'scheduled',
            'canceled'
        );
    END IF;
END $$;

-- Métodos de Pagamento
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method_enum') THEN
        CREATE TYPE payment_method_enum AS ENUM (
            'cash',
            'credit_card',
            'debit_card',
            'pix',
            'bank_transfer',
            'installments', -- Correspondente ao 'aprazo'
            'other'
        );
    END IF;
END $$;