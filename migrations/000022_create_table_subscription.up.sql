CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    payment_method_id UUID REFERENCES subscription_payment_methods(id),
    
    -- Dados de Integração com Mercado Pago
    mp_subscription_id VARCHAR(255) UNIQUE,  -- ID do Preapproval gerado pela API do MP
    
    -- Controle de Status
    -- Valores possíveis: 'authorized' (ativa/paga), 'paused', 'cancelled', 'pending'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', 
    
    -- Datas de Controle do Período
    current_period_start TIMESTAMPTZ,           -- Início da vigência do mês atual
    current_period_end TIMESTAMPTZ,             -- Fim da vigência do mês atual (data de renovação)
    canceled_at TIMESTAMPTZ,                    -- Preenchido apenas se o cliente cancelar
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_company_id ON subscriptions(company_id);
CREATE INDEX idx_subscriptions_mp_id ON subscriptions(mp_subscription_id);