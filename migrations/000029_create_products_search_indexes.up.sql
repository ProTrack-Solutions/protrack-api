CREATE INDEX IF NOT EXISTS idx_products_search
ON products
USING GIN (
    to_tsvector(
        'portuguese_unaccent',
        coalesce(name, '') || ' ' || coalesce(description, '')
    )
);

CREATE INDEX IF NOT EXISTS idx_products_barcode_trgm
ON products
USING GIN (barcode gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_categories_name_trgm
ON product_categories
USING GIN (name gin_trgm_ops);