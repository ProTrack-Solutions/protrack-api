CREATE TABLE IF NOT EXISTS bills_payable (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    vendor_id UUID,
    category_id UUID,
    payment_method_id UUID,
    amount DECIMAL(12, 2) NOT NULL,
    due_date DATE NOT NULL,
    status account_status_enum DEFAULT 'pending',
    description TEXT,
    scheduled_date DATE,
    payment_date DATE,
    amount_paid DECIMAL(12, 2),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_bills_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_bills_vendor FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_bills_category FOREIGN KEY (category_id) REFERENCES bill_categories(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_bills_payment_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE
    SET NULL
);