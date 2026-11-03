ALTER TABLE cash_session_movements DROP COLUMN IF EXISTS transfer_id;
DROP TABLE IF EXISTS cash_transfers;
DROP TYPE IF EXISTS cash_transfer_status;
