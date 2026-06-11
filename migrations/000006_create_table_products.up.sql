CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    category_id UUID NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    barcode VARCHAR(50),
    quantity INT NOT NULL DEFAULT 0,
    size VARCHAR(50),
    cost_price NUMERIC(10, 2),
    sale_price NUMERIC(10, 2),
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_products_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE
    SET NULL,
        CONSTRAINT uq_product_name_per_company UNIQUE (company_id, name)
);