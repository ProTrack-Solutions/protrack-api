-- ---------- Configuração fiscal da empresa ----------

-- name: GetCompanyFiscalSettings :one
SELECT * FROM company_fiscal_settings WHERE company_id = $1;

-- name: UpsertCompanyFiscalSettings :one
INSERT INTO company_fiscal_settings (
    company_id, enabled, environment, issuer_id_test, issuer_id_live,
    tax_regime, default_tax_profile_id, certificate_expires_at, updated_by
) VALUES (
    @company_id, @enabled, @environment, @issuer_id_test, @issuer_id_live,
    @tax_regime, @default_tax_profile_id, @certificate_expires_at, @updated_by
)
ON CONFLICT (company_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    environment = EXCLUDED.environment,
    issuer_id_test = EXCLUDED.issuer_id_test,
    issuer_id_live = EXCLUDED.issuer_id_live,
    tax_regime = EXCLUDED.tax_regime,
    default_tax_profile_id = EXCLUDED.default_tax_profile_id,
    certificate_expires_at = EXCLUDED.certificate_expires_at,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING *;

-- name: LockCompanyFiscalSettings :exec
-- Serializa o cadastro do emitente (dois cliques não criam dois CNPJs)
SELECT pg_advisory_xact_lock(hashtextextended('fiscal-settings:' || @company_id::text, 0));

-- name: GetCompanyForFiscal :one
SELECT id, name, trade_name, document, phone,
       address_street, address_number, address_complement, address_neighborhood,
       address_city, address_state, address_zipcode, inscricao_estadual
FROM companies
WHERE id = $1 AND deleted_at IS NULL;

-- ---------- Perfis tributários ----------

-- name: ListTaxProfiles :many
SELECT tp.*,
       (SELECT COUNT(*) FROM products p WHERE p.tax_profile_id = tp.id AND p.deleted_at IS NULL)::bigint AS products_count,
       (SELECT COUNT(*) FROM product_categories pc WHERE pc.tax_profile_id = tp.id AND pc.deleted_at IS NULL)::bigint AS categories_count
FROM tax_profiles tp
WHERE tp.company_id = $1 AND tp.deleted_at IS NULL
ORDER BY lower(tp.name);

-- name: GetTaxProfile :one
SELECT * FROM tax_profiles
WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL;

-- name: CreateTaxProfile :one
INSERT INTO tax_profiles (
    company_id, name, icms_csosn, icms_cst, icms_rate, icms_base_reduction, icms_fcp_rate,
    cfop, benefit_code, pis_cst, pis_rate, cofins_cst, cofins_rate,
    ibs_cbs_cst, ibs_cbs_class_code, ibs_state_rate, ibs_city_rate, cbs_rate,
    created_by, updated_by
) VALUES (
    @company_id, @name, @icms_csosn, @icms_cst, @icms_rate, @icms_base_reduction, @icms_fcp_rate,
    @cfop, @benefit_code, @pis_cst, @pis_rate, @cofins_cst, @cofins_rate,
    @ibs_cbs_cst, @ibs_cbs_class_code, @ibs_state_rate, @ibs_city_rate, @cbs_rate,
    @created_by, @created_by
)
RETURNING *;

-- name: UpdateTaxProfile :one
UPDATE tax_profiles SET
    name = @name,
    icms_csosn = @icms_csosn,
    icms_cst = @icms_cst,
    icms_rate = @icms_rate,
    icms_base_reduction = @icms_base_reduction,
    icms_fcp_rate = @icms_fcp_rate,
    cfop = @cfop,
    benefit_code = @benefit_code,
    pis_cst = @pis_cst,
    pis_rate = @pis_rate,
    cofins_cst = @cofins_cst,
    cofins_rate = @cofins_rate,
    ibs_cbs_cst = @ibs_cbs_cst,
    ibs_cbs_class_code = @ibs_cbs_class_code,
    ibs_state_rate = @ibs_state_rate,
    ibs_city_rate = @ibs_city_rate,
    cbs_rate = @cbs_rate,
    updated_by = @updated_by,
    updated_at = NOW()
WHERE id = @id AND company_id = @company_id AND deleted_at IS NULL
RETURNING *;

-- name: DeleteTaxProfile :execrows
UPDATE tax_profiles SET deleted_at = NOW(), updated_by = @updated_by
WHERE id = @id AND company_id = @company_id AND deleted_at IS NULL;

-- name: DetachTaxProfile :exec
-- Perfil excluído: categorias e produtos voltam a herdar (empresa/categoria)
WITH cats AS (
    UPDATE product_categories SET tax_profile_id = NULL
    WHERE product_categories.tax_profile_id = @id AND product_categories.company_id = @company_id
), prods AS (
    UPDATE products SET tax_profile_id = NULL
    WHERE products.tax_profile_id = @id AND products.company_id = @company_id
)
UPDATE company_fiscal_settings SET default_tax_profile_id = NULL
WHERE company_fiscal_settings.default_tax_profile_id = @id AND company_fiscal_settings.company_id = @company_id;

-- name: TaxProfileNameExists :one
SELECT EXISTS (
    SELECT 1 FROM tax_profiles
    WHERE company_id = @company_id AND lower(name) = lower(@name) AND deleted_at IS NULL
      AND id <> COALESCE(sqlc.narg(except_id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
);

-- ---------- Tributação de categorias e produtos ----------

-- name: SetCategoryTaxProfile :execrows
UPDATE product_categories SET tax_profile_id = sqlc.narg(tax_profile_id), updated_at = NOW()
WHERE id = @id AND company_id = @company_id AND deleted_at IS NULL;

-- name: ListCategoriesFiscal :many
SELECT id, name, color, tax_profile_id
FROM product_categories
WHERE company_id = $1 AND deleted_at IS NULL
ORDER BY lower(name);

-- name: UpdateProductFiscal :execrows
-- Produto avulso ou, com model_name, todas as variações do modelo de grade
UPDATE products SET
    ncm = sqlc.narg(ncm),
    cest = sqlc.narg(cest),
    unit = COALESCE(sqlc.narg(unit)::unit_of_measure, unit),
    origem_mercadoria = @origin,
    tax_profile_id = sqlc.narg(tax_profile_id),
    updated_by = @updated_by,
    updated_at = NOW()
WHERE company_id = @company_id AND deleted_at IS NULL
  AND (
    id = sqlc.narg(product_id)
    OR (sqlc.narg(model_name)::text IS NOT NULL AND lower(model_name) = lower(sqlc.narg(model_name)::text))
  );

-- name: GetProductFiscal :one
SELECT p.id, p.name, p.model_name, p.ncm, p.cest, p.unit, p.origem_mercadoria, p.tax_profile_id,
       p.category_id, pc.tax_profile_id AS category_tax_profile_id
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE p.id = $1 AND p.company_id = $2 AND p.deleted_at IS NULL;

-- name: CountProductsMissingNcm :one
SELECT COUNT(*)::bigint FROM products
WHERE company_id = $1 AND deleted_at IS NULL AND (ncm IS NULL OR ncm !~ '^\d{8}$');
