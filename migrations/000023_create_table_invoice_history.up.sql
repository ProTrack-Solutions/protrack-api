CREATE TABLE IF NOT EXISTS invoice_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id),
    payment_method_id UUID REFERENCES subscription_payment_methods(id),        -- 'credit_card', 'pix'
    
    mp_payment_id VARCHAR(255) UNIQUE NOT NULL, -- ID único da transação gerado pelo MP (Payment ID)
    amount_cents INT NOT NULL,                   -- Valor cobrado (em centavos)
    status VARCHAR(50) NOT NULL,                 -- 'approved', 'rejected', 'refunded', 'in_process'
    
    paid_at TIMESTAMPTZ,                           -- Data/hora da confirmação do pagamento
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_invoice_history_company_id ON invoice_history(company_id);
CREATE INDEX idx_invoice_history_subscription ON invoice_history(subscription_id);
CREATE INDEX idx_invoice_history_status ON invoice_history(status);