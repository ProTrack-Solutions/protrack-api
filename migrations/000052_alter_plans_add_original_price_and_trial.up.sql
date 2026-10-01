ALTER TABLE plans
  ADD COLUMN original_price_cents INT,                 -- Preço "de" exibido riscado (somente exibição; a cobrança usa price_cents)
  ADD COLUMN trial_days INT NOT NULL DEFAULT 0;        -- Dias de teste grátis aplicados na criação da assinatura no Stripe

ALTER TABLE plans
  ADD CONSTRAINT plans_original_price_gt_price CHECK (original_price_cents IS NULL OR original_price_cents > price_cents),
  ADD CONSTRAINT plans_trial_days_non_negative CHECK (trial_days >= 0);

-- Padroniza a grafia do status de cancelamento com a do Stripe ('canceled')
UPDATE subscriptions SET status = 'canceled' WHERE status = 'cancelled';
