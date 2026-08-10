ALTER TABLE sales DROP CONSTRAINT chk_sales_customer_required_for_credit;
ALTER TABLE sales DROP COLUMN buyer_document;
ALTER TABLE sales ALTER COLUMN customer_id SET NOT NULL;