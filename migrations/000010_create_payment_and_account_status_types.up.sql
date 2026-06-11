-- payment_method_enum e account_status_enum (usados pela tabela sales)

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method_enum') THEN
        CREATE TYPE payment_method_enum AS ENUM (
            'cash',
            'credit_card',
            'debit_card',
            'pix',
            'bank_transfer',
            'installments',
            'other'
        );
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_status_enum') THEN
        CREATE TYPE account_status_enum AS ENUM (
            'pending',
            'paid',
            'overdue',
            'scheduled',
            'canceled',
            'partial'
        );
    END IF;
END $$;
