-- name: CreateSubscriptionPaymentMethod :exec
INSERT INTO subscription_payment_methods (
    company_id,
    gateway_payment_method_id,
    type,
    card_brand,
    card_last4,
    card_exp_month,
    card_exp_year,
    is_default,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
); 

-- name: ListSubscriptionPaymentMethodsByCompanyId :many
SELECT * FROM subscription_payment_methods WHERE company_id = $1;

-- name: GetSubscriptionPaymentMethodById :one
SELECT * FROM subscription_payment_methods WHERE id = $1;

-- name: UpdateSubscriptionPaymentMethod :exec
UPDATE subscription_payment_methods SET
    gateway_payment_method_id = $2,
    type = $3,
    card_brand = $4,
    card_last4 = $5,
    card_exp_month = $6,
    card_exp_year = $7,
    updated_at = NOW(),
    updated_by = $8
WHERE id = $1;

-- name: SetDefaultSubscriptionPaymentMethod :exec
UPDATE subscription_payment_methods SET is_default = $2, updated_by = $3 WHERE company_id = $1;

-- name: DeleteSubscriptionPaymentMethod :exec
DELETE FROM subscription_payment_methods WHERE id = $1;
