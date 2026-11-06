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

-- ---------- Emissão da NFC-e ----------

-- name: LockSaleFiscal :exec
-- Serializa emissões da mesma venda (duplo clique, PDV + worker)
SELECT pg_advisory_xact_lock(hashtextextended('fiscal-sale:' || @sale_id::text, 0));

-- name: GetSaleForNfce :one
SELECT s.id, s.company_id, s.status::text AS status, s.payment_method::text AS payment_method,
       s.subtotal, s.discount_amount, s.total_amount, s.down_payment, s.credit_amount,
       s.buyer_document, s.deleted_at, s.sale_at
FROM sales s
WHERE s.id = $1 AND s.company_id = $2;

-- name: ListSaleItemsForNfce :many
-- Cada item com o perfil tributário já resolvido: produto -> categoria -> padrão da empresa
SELECT si.id, si.quantity, si.unit_price, COALESCE(si.discount, 0)::numeric AS discount,
       p.id AS product_id, p.name, p.barcode, p.ncm, p.cest, p.unit, p.origem_mercadoria,
       COALESCE(p.tax_profile_id, pc.tax_profile_id, cfs.default_tax_profile_id) AS tax_profile_id
FROM sale_items si
JOIN products p ON p.id = si.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
LEFT JOIN company_fiscal_settings cfs ON cfs.company_id = p.company_id
WHERE si.sale_id = $1
ORDER BY si.created_at, si.id;

-- name: ListTaxProfilesByIds :many
-- Inclui perfis excluídos depois da venda? Não: perfil excluído some (o item cai no padrão)
SELECT * FROM tax_profiles
WHERE company_id = @company_id AND id = ANY(@ids::uuid[]) AND deleted_at IS NULL;

-- name: GetLiveFiscalDocumentBySale :one
SELECT * FROM fiscal_documents
WHERE sale_id = $1 AND model = 65 AND status NOT IN ('REJECTED', 'FAILED', 'ERROR');

-- name: GetLatestFiscalDocumentBySale :one
SELECT * FROM fiscal_documents
WHERE sale_id = $1 AND company_id = $2 AND model = 65
ORDER BY attempt DESC
LIMIT 1;

-- name: NextFiscalAttempt :one
SELECT (COALESCE(MAX(attempt), 0) + 1)::int FROM fiscal_documents WHERE sale_id = $1 AND model = 65;

-- name: CreateFiscalDocument :one
INSERT INTO fiscal_documents (
    company_id, sale_id, model, environment, attempt, idempotency_key, status,
    error_message, request_payload, created_by
) VALUES (
    @company_id, @sale_id, 65, @environment, @attempt, @idempotency_key, @status,
    sqlc.narg(error_message), @request_payload, @created_by
)
RETURNING *;

-- name: ApplyFiscalDocumentResult :one
-- Situação vinda da API fiscal (emissão, consulta ou webhook)
UPDATE fiscal_documents SET
    external_id = COALESCE(sqlc.narg(external_id), external_id),
    status = @status,
    number = COALESCE(sqlc.narg(number), number),
    access_key = COALESCE(sqlc.narg(access_key), access_key),
    protocol = COALESCE(sqlc.narg(protocol), protocol),
    status_code = sqlc.narg(status_code),
    reject_reason = sqlc.narg(reject_reason),
    error_message = sqlc.narg(error_message),
    emission_type = COALESCE(sqlc.narg(emission_type), emission_type),
    authorized_at = COALESCE(sqlc.narg(authorized_at), authorized_at),
    updated_at = NOW()
WHERE id = @id
RETURNING *;

-- name: MarkFiscalDocumentRetry :one
-- Falha transitória (API fiscal/SEFAZ fora): continua PENDING para o worker
UPDATE fiscal_documents SET
    error_message = @error_message,
    retries = retries + 1,
    updated_at = NOW()
WHERE id = @id
RETURNING *;

-- name: MarkFiscalDocumentCancelled :one
UPDATE fiscal_documents SET
    status = 'CANCELLED',
    cancel_reason = COALESCE(sqlc.narg(cancel_reason), cancel_reason),
    cancelled_at = COALESCE(cancelled_at, NOW()),
    error_message = NULL,
    updated_at = NOW()
WHERE id = @id
RETURNING *;

-- name: SetFiscalDocumentError :exec
UPDATE fiscal_documents SET error_message = sqlc.narg(error_message), updated_at = NOW()
WHERE id = @id;

-- name: GetFiscalDocument :one
SELECT * FROM fiscal_documents WHERE id = $1 AND company_id = $2;

-- name: GetFiscalDocumentByExternalId :one
SELECT * FROM fiscal_documents WHERE external_id = $1;

-- name: ClaimFiscalDocumentsToReconcile :many
-- Lease: marca updated_at para outra instância do worker não pegar o mesmo lote
UPDATE fiscal_documents SET updated_at = NOW()
WHERE id IN (
    SELECT fd.id FROM fiscal_documents fd
    WHERE fd.status IN ('PENDING', 'PROCESSING', 'CONTINGENCY')
      AND fd.updated_at < NOW() - make_interval(secs => @min_age_seconds::int)
    ORDER BY fd.updated_at
    LIMIT @batch_size::int
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: ListFiscalDocuments :many
-- Notas das vendas (só a última tentativa de cada venda) e notas manuais
SELECT fd.*,
       COALESCE(s.total_amount, fd.total, 0)::numeric AS total_amount,
       COALESCE(s.sale_at, fd.created_at)::timestamptz AS sale_at,
       c.full_name AS customer_name,
       COUNT(*) OVER() AS total_count
FROM fiscal_documents fd
LEFT JOIN sales s ON s.id = fd.sale_id
LEFT JOIN customers c ON c.id = s.customer_id
WHERE fd.company_id = @company_id
  AND (sqlc.narg(status)::text IS NULL OR fd.status = sqlc.narg(status)::text)
  AND (sqlc.narg(model)::smallint IS NULL OR fd.model = sqlc.narg(model)::smallint)
  AND (sqlc.narg(source)::text IS NULL OR fd.source = sqlc.narg(source)::text)
  AND (NOT @problems_only::bool OR fd.status IN ('REJECTED', 'FAILED', 'ERROR', 'PENDING'))
  AND (
    fd.sale_id IS NULL
    OR fd.attempt = (SELECT MAX(x.attempt) FROM fiscal_documents x WHERE x.sale_id = fd.sale_id AND x.model = fd.model)
  )
ORDER BY fd.created_at DESC
LIMIT @page_size::int OFFSET @page_offset::int;

-- name: ListFiscalDocumentsBySales :many
-- Última nota de cada venda (lista de vendas)
SELECT DISTINCT ON (sale_id) *
FROM fiscal_documents
WHERE company_id = @company_id AND sale_id = ANY(@sale_ids::uuid[]) AND model = 65
ORDER BY sale_id, attempt DESC;

-- ---------- Emissão manual (NF-e e NFC-e avulsas) ----------

-- name: GetFiscalDocumentByIdempotencyKey :one
SELECT * FROM fiscal_documents WHERE idempotency_key = $1 AND company_id = $2;

-- name: CreateManualFiscalDocument :one
INSERT INTO fiscal_documents (
    company_id, source, model, environment, attempt, idempotency_key, status,
    request_payload, operation_nature, recipient_name, recipient_document, total, created_by
) VALUES (
    @company_id, 'manual', @model, @environment, 1, @idempotency_key, 'PENDING',
    @request_payload, sqlc.narg(operation_nature), sqlc.narg(recipient_name),
    sqlc.narg(recipient_document), @total, @created_by
)
RETURNING *;

-- name: ListProductsForManualDocument :many
-- Produtos da nota manual com o perfil tributário resolvido: produto -> categoria -> padrão
SELECT p.id, p.name, p.barcode, p.ncm, p.cest, p.unit, p.origem_mercadoria,
       COALESCE(p.tax_profile_id, pc.tax_profile_id, cfs.default_tax_profile_id) AS tax_profile_id
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
LEFT JOIN company_fiscal_settings cfs ON cfs.company_id = p.company_id
WHERE p.company_id = @company_id AND p.id = ANY(@ids::uuid[]) AND p.deleted_at IS NULL;

-- name: InsertFiscalWebhookEvent :execrows
INSERT INTO fiscal_webhook_events (id, type) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING;
