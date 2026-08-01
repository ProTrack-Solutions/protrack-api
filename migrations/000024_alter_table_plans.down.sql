ALTER TABLE plans
  DROP COLUMN IF EXISTS highlight,
  DROP COLUMN IF EXISTS icon;
  DROP COLUMN IF EXISTS external_price_id;