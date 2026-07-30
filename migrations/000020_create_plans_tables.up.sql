CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,              -- ex: 'Plano Pro'
    description TEXT,                        -- ex: 'Acesso total a todas as funcionalidades'
    price_cents INT NOT NULL,                -- ex: 4990 (R$ 49,90 - gravado em centavos para evitar erros de arredondamento)
    currency VARCHAR(3) DEFAULT 'BRL',       -- 'BRL'
    billing_cycle VARCHAR(20) NOT NULL,      -- 'monthly', 'yearly'
    active BOOLEAN DEFAULT true,             -- Se o plano ainda pode ser assinado por novos usuários
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);