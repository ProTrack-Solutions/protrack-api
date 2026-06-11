CREATE TABLE IF NOT EXISTS payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    sale_id UUID,
    payment_method_id UUID,
    user_id UUID NOT NULL,
    amount_paid DECIMAL(12, 2) NOT NULL,
    payment_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    CONSTRAINT fk_payment_history_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_history_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_history_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_payment_history_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_payment_history_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE
    SET NULL
);