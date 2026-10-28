DELETE FROM department_modules WHERE module_code = 'purchases';
DELETE FROM modules WHERE code = 'purchases';

ALTER TABLE stock_movements DROP COLUMN IF EXISTS purchase_receipt_id;
DROP INDEX IF EXISTS idx_bills_payable_purchase_receipt;
ALTER TABLE bills_payable DROP COLUMN IF EXISTS purchase_receipt_id;

DROP TABLE IF EXISTS purchase_receipt_items;
DROP TABLE IF EXISTS purchase_receipts;
DROP TABLE IF EXISTS purchase_order_items;
DROP TABLE IF EXISTS purchase_orders;

DROP TYPE IF EXISTS purchase_receipt_status;
DROP TYPE IF EXISTS purchase_order_status;
