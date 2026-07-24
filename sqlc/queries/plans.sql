-- name: CreatePlan :exec
INSERT INTO plans (name, description, price_cents, currency, billing_cycle, active, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

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
UPDATE plans SET name = $2, description = $3, price_cents = $4, currency = $5, billing_cycle = $6, updated_at = NOW()
WHERE id = $1;

-- name: TogglePlanActiveStatus :exec
UPDATE plans SET active = $2, updated_at = NOW()
WHERE id = $1;