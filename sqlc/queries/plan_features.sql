-- name: CreatePlanFeature :exec
INSERT INTO plan_features (
    plan_id,
    name,
    is_enabled,
    display_order
) VALUES (
    $1, $2, $3, $4
);

-- name: ListFeaturesByPlanID :many
SELECT *
FROM plan_features
WHERE plan_id = $1
ORDER BY display_order ASC;

-- name: DeletePlanFeature :exec
DELETE FROM plan_features
WHERE id = $1;