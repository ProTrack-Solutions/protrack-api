ALTER TABLE products
    ADD CONSTRAINT uq_product_name_per_company UNIQUE (company_id, name);
