ALTER TABLE plan_features
    ADD COLUMN feature_key VARCHAR(100) NOT NULL DEFAULT 'unknown',
    ADD COLUMN limit_value INT;