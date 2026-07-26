CREATE TABLE IF NOT EXISTS  subscription_payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    gateway_payment_method_id VARCHAR(255) NOT NULL, -- ID do cartão no Gateway
    type VARCHAR(20) NOT NULL,                       -- 'credit_card', 'pix', etc.
    card_brand VARCHAR(50),                          -- 'visa', 'mastercard'
    card_last4 VARCHAR(4),                           -- '4432'
    card_exp_month INT,                              -- 12
    card_exp_year INT,                               -- 2028
    is_default BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);