-- name: CreatePlan :one
INSERT INTO plans (
    external_id,
    external_price_id,
    name,
    description,
    price_cents,
    currency,
    billing_cycle,
    active,
    highlight,
    icon,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
)
RETURNING id;

-- name: GetPlanByID :one
SELECT *
FROM plans
WHERE id = $1;

-- name: ListPlans :many
SELECT *
FROM plans;

-- name: ListPlansByActiveStatus :many
SELECT *
FROM plans
WHERE active = $1;

-- name: UpdatePlan :exec
UPDATE plans SET name = $2, description = $3, price_cents = $4, currency = $5, billing_cycle = $6, highlight=$7, icon=$8, external_price_id=$9, updated_at = NOW()
WHERE id = $1;

-- name: TogglePlanActiveStatus :exec
UPDATE plans SET active = $2, updated_at = NOW()
WHERE id = $1;