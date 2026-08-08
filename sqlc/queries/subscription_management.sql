-- name: GetSubscriptionDetailsByCompanyID :one
SELECT
    s.id AS subscription_id,
    s.company_id,
    s.status AS subscription_status,
    s.current_period_start,
    s.current_period_end,
    s.canceled_at,
    s.external_subscription_id,
    s.created_at AS subscription_created_at,
    s.updated_at AS subscription_updated_at,

    p.id AS plan_id,
    p.external_id AS plan_external_id,
    p.external_price_id AS plan_external_price_id,
    p.name AS plan_name,
    p.description AS plan_description,
    p.price_cents AS plan_price_cents,
    p.currency AS plan_currency,
    p.billing_cycle AS plan_billing_cycle,
    p.active AS plan_active,
    p.highlight AS plan_highlight,
    p.icon AS plan_icon,

    pm.id AS payment_method_id,
    pm.type AS payment_method_type,
    pm.card_brand AS payment_method_card_brand,
    pm.card_last4 AS payment_method_card_last4,
    pm.card_exp_month AS payment_method_card_exp_month,
    pm.card_exp_year AS payment_method_card_exp_year,
    pm.is_default AS payment_method_is_default,

    COALESCE(
        (
            SELECT json_agg(
                json_build_object(
                    'id', pf.id,
                    'name', pf.name,
                    'is_enabled', pf.is_enabled,
                    'display_order', pf.display_order
                ) ORDER BY pf.display_order ASC
            )
            FROM plan_features pf
            WHERE pf.plan_id = p.id
        ),
        '[]'
    ) AS features
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
LEFT JOIN subscription_payment_methods pm ON pm.id = s.payment_method_id
WHERE s.company_id = $1;
