ALTER TABLE plans
  DROP CONSTRAINT IF EXISTS plans_trial_days_non_negative,
  DROP CONSTRAINT IF EXISTS plans_original_price_gt_price,
  DROP COLUMN IF EXISTS trial_days,
  DROP COLUMN IF EXISTS original_price_cents;
