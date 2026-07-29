-- name: CreateSubscription :exec
INSERT INTO subscriptions (
    company_id,
    plan_id,
    payment_method_id,
    mp_subscription_id,
    status,
    current_period_start,
    current_period_end
)
VALUES(
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);

-- name: GetSubscriptionById :one
SELECT * FROM subscriptions WHERE id = $1;

-- name: UpdateSubscriptionPlan :exec
UPDATE subscriptions
SET
    plan_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSubscriptionMethod :exec
UPDATE subscriptions
SET
    payment_method_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: CancelSubscription :exec
UPDATE subscriptions
SET
    status = 'cancelled',
    canceled_at = NOW(),
    updated_at = NOW()
WHERE id = $1;