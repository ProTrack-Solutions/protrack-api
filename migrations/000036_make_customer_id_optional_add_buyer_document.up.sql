ALTER TABLE sales ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE sales ADD COLUMN buyer_document VARCHAR(20) NULL;
ALTER TABLE sales ADD CONSTRAINT chk_sales_customer_required_for_credit
  CHECK (payment_method <> 'installments' OR customer_id IS NOT NULL);