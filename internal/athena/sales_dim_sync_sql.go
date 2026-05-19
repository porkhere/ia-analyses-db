package athena

import "fmt"

const defaultConflictSampleLimit = 10

func buildSalesProductDimCTE(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s,
product_dim_source AS (
	SELECT
		cast(%d AS bigint) AS owner_user_id,
		coalesce(nullif(trim(oi.product_no), ''), 'UNKNOWN_PRODUCT') AS product_no,
		coalesce(nullif(trim(oi.product_name), ''), 'UNKNOWN_PRODUCT_NAME') AS product_name,
		nullif(trim(oi.cate_no), '') AS cate_no,
		nullif(trim(oi.cate_name), '') AS cate_name,
		max(cast(od.business_date AS timestamp)) AS last_seen_at,
		cast(count(*) AS bigint) AS source_row_count
	FROM status_dedup_order_destinations od
	JOIN order_items_parquet oi ON oi.order_id = od.order_id
	WHERE oi.t_open_date BETWEEN date '%s' AND date '%s'
	  AND coalesce(nullif(trim(oi.product_no), ''), '') <> ''
	GROUP BY 1, 2, 3, 4, 5
),
product_conflict_keys AS (
	SELECT
		product_no,
		cast(count(*) AS bigint) AS variant_count
	FROM product_dim_source
	GROUP BY 1
	HAVING count(*) > 1
),
product_ranked AS (
	SELECT
		owner_user_id,
		product_no,
		product_name,
		cate_no,
		cate_name,
		last_seen_at,
		source_row_count,
		row_number() OVER (
			PARTITION BY product_no
			ORDER BY source_row_count DESC, last_seen_at DESC NULLS LAST, product_name DESC, coalesce(cate_no, '') DESC, coalesce(cate_name, '') DESC
		) AS variant_rank
	FROM product_dim_source
)`, buildPreviewCTE(window), ownerUserID, window.StartDate.Format(dateLayout), window.EndDate.Format(dateLayout))
}

func BuildSalesProductDimCandidateSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s
SELECT
	cast(owner_user_id AS varchar) AS owner_user_id,
	product_no,
	product_name,
	coalesce(cate_no, '') AS cate_no,
	coalesce(cate_name, '') AS cate_name,
	cast(last_seen_at AS varchar) AS last_seen_at,
	cast(source_row_count AS varchar) AS source_row_count
FROM product_ranked
WHERE variant_rank = 1
ORDER BY product_no`, buildSalesProductDimCTE(window, ownerUserID))
}

func BuildSalesProductDimConflictCountSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s
SELECT cast(count(*) AS varchar) AS conflict_key_count
FROM product_conflict_keys`, buildSalesProductDimCTE(window, ownerUserID))
}

func BuildSalesProductDimConflictSampleSQL(window QueryWindow, ownerUserID int64, limit int) string {
	return fmt.Sprintf(`%s
SELECT
	pk.product_no,
	cast(pk.variant_count AS varchar) AS variant_count,
	pr.product_name AS chosen_product_name,
	coalesce(pr.cate_no, '') AS chosen_cate_no,
	coalesce(pr.cate_name, '') AS chosen_cate_name,
	cast(pr.source_row_count AS varchar) AS chosen_source_row_count,
	cast(pr.last_seen_at AS varchar) AS chosen_last_seen_at,
	array_join(
		slice(
			array_agg(
				format('%%s|%%s|%%s|rows=%%s', pds.product_name, coalesce(pds.cate_no, ''), coalesce(pds.cate_name, ''), cast(pds.source_row_count AS varchar))
				ORDER BY pds.source_row_count DESC, pds.last_seen_at DESC NULLS LAST, pds.product_name DESC, coalesce(pds.cate_no, '') DESC, coalesce(pds.cate_name, '') DESC
			),
			1,
			3
		),
		' ; '
	) AS sample_variants
FROM product_conflict_keys pk
JOIN product_ranked pr ON pr.product_no = pk.product_no AND pr.variant_rank = 1
JOIN product_dim_source pds ON pds.product_no = pk.product_no
GROUP BY pk.product_no, pk.variant_count, pr.product_name, pr.cate_no, pr.cate_name, pr.source_row_count, pr.last_seen_at
ORDER BY pk.variant_count DESC, pk.product_no
LIMIT %d`, buildSalesProductDimCTE(window, ownerUserID), normalizeConflictSampleLimit(limit))
}

func buildSalesBranchDimCTE(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s,
branch_dim_source AS (
	SELECT
		cast(%d AS bigint) AS owner_user_id,
		osc.branch_id,
		coalesce(nullif(trim(o.branch), ''), 'UNKNOWN_BRANCH_NAME') AS branch_name,
		cast(NULL AS varchar) AS group_code,
		max(coalesce(o.modified, o.transaction_submitted, o.transaction_created, cast(o.t_open_date AS timestamp))) AS last_seen_at,
		cast(count(*) AS bigint) AS source_row_count
	FROM orders_sales_candidate osc
	JOIN orders_parquet o ON o.id = osc.order_id
	WHERE o.t_open_date BETWEEN date '%s' AND date '%s'
	  AND o.t_open_date = osc.business_date
	  AND coalesce(o.status, -1) = 1
	  AND coalesce(nullif(trim(o.branch_id), ''), 'UNKNOWN_BRANCH') = osc.branch_id
	GROUP BY 1, 2, 3, 4
),
branch_conflict_keys AS (
	SELECT
		branch_id,
		cast(count(*) AS bigint) AS variant_count
	FROM branch_dim_source
	GROUP BY 1
	HAVING count(*) > 1
),
branch_ranked AS (
	SELECT
		owner_user_id,
		branch_id,
		branch_name,
		group_code,
		last_seen_at,
		source_row_count,
		row_number() OVER (
			PARTITION BY branch_id
			ORDER BY source_row_count DESC, last_seen_at DESC NULLS LAST, branch_name DESC
		) AS variant_rank
	FROM branch_dim_source
)`, buildPreviewCTE(window), ownerUserID, window.StartDate.Format(dateLayout), window.EndDate.Format(dateLayout))
}

func BuildSalesBranchDimCandidateSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s
SELECT
	cast(owner_user_id AS varchar) AS owner_user_id,
	branch_id,
	branch_name,
	coalesce(group_code, '') AS group_code,
	cast(last_seen_at AS varchar) AS last_seen_at,
	cast(source_row_count AS varchar) AS source_row_count
FROM branch_ranked
WHERE variant_rank = 1
ORDER BY branch_id`, buildSalesBranchDimCTE(window, ownerUserID))
}

func BuildSalesBranchDimConflictCountSQL(window QueryWindow, ownerUserID int64) string {
	return fmt.Sprintf(`%s
SELECT cast(count(*) AS varchar) AS conflict_key_count
FROM branch_conflict_keys`, buildSalesBranchDimCTE(window, ownerUserID))
}

func BuildSalesBranchDimConflictSampleSQL(window QueryWindow, ownerUserID int64, limit int) string {
	return fmt.Sprintf(`%s
SELECT
	bk.branch_id,
	cast(bk.variant_count AS varchar) AS variant_count,
	br.branch_name AS chosen_branch_name,
	coalesce(br.group_code, '') AS chosen_group_code,
	cast(br.source_row_count AS varchar) AS chosen_source_row_count,
	cast(br.last_seen_at AS varchar) AS chosen_last_seen_at,
	array_join(
		slice(
			array_agg(
				format('%%s|group=%%s|rows=%%s', bds.branch_name, coalesce(bds.group_code, ''), cast(bds.source_row_count AS varchar))
				ORDER BY bds.source_row_count DESC, bds.last_seen_at DESC NULLS LAST, bds.branch_name DESC
			),
			1,
			3
		),
		' ; '
	) AS sample_variants
FROM branch_conflict_keys bk
JOIN branch_ranked br ON br.branch_id = bk.branch_id AND br.variant_rank = 1
JOIN branch_dim_source bds ON bds.branch_id = bk.branch_id
GROUP BY bk.branch_id, bk.variant_count, br.branch_name, br.group_code, br.source_row_count, br.last_seen_at
ORDER BY bk.variant_count DESC, bk.branch_id
LIMIT %d`, buildSalesBranchDimCTE(window, ownerUserID), normalizeConflictSampleLimit(limit))
}

func normalizeConflictSampleLimit(limit int) int {
	if limit <= 0 {
		return defaultConflictSampleLimit
	}

	return limit
}
