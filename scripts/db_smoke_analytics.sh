#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$PROJECT_DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  ENV_FILE="$PROJECT_DIR/.env.example"
fi

set -a
# shellcheck source=/dev/null
source "$ENV_FILE"
set +a

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

if ! counts_result="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -At <<'SQL'
SELECT
  (SELECT COUNT(*) FROM public.pos_product_dim) || '|' ||
  (SELECT COUNT(*) FROM public.pos_branch_dim) || '|' ||
  (SELECT COUNT(*) FROM public.pos_sales_hourly_fact);
SQL
)"; then
  echo "smoke failed: unable to read analytics table counts" >&2
  exit 1
fi

IFS='|' read -r product_count branch_count fact_count <<< "$counts_result"

if ! leaderboard_count="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -At <<'SQL'
WITH ranked_products AS (
    SELECT
        fact.owner_user_id,
        product.product_no,
        product.product_name,
        SUM(fact.qty_milli) AS qty_milli,
        SUM(fact.net_sales_milli) AS net_sales_milli
    FROM public.pos_sales_hourly_fact AS fact
    JOIN public.pos_product_dim AS product
      ON product.owner_user_id = fact.owner_user_id
     AND product.product_no = fact.product_no
    WHERE product.is_active = TRUE
      AND COALESCE(product.product_name, '') !~ '幣|券|折抵|折扣|點數|贈|服務費|運費|調整|測試'
      AND lower(COALESCE(product.product_name, '')) NOT LIKE '%test%'
    GROUP BY fact.owner_user_id, product.product_no, product.product_name
)
SELECT COUNT(*) FROM ranked_products;
SQL
)"; then
  echo "smoke failed: leaderboard query execution error" >&2
  exit 1
fi

echo "smoke_counts: pos_product_dim=$product_count pos_branch_dim=$branch_count pos_sales_hourly_fact=$fact_count"
echo "smoke_leaderboard_keywords_excluded: 幣, 券, 折抵, 折扣, 點數, 贈, 服務費, 運費, 調整, 測試, test"
echo "smoke_leaderboard_row_count_after_exclusion: $leaderboard_count"

smoke_failed=0

if (( product_count == 0 )); then
  echo "smoke failed: pos_product_dim is empty" >&2
  smoke_failed=1
fi

if (( branch_count == 0 )); then
  echo "smoke failed: pos_branch_dim is empty" >&2
  smoke_failed=1
fi

if (( fact_count == 0 )); then
  echo "smoke failed: pos_sales_hourly_fact is empty" >&2
  smoke_failed=1
fi

if (( leaderboard_count == 0 )); then
  echo "smoke failed: leaderboard query returned 0 rows after exclusion" >&2
  smoke_failed=1
fi

if (( smoke_failed != 0 )); then
  exit 1
fi

if leaderboard_preview="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -At <<'SQL'
WITH ranked_products AS (
    SELECT
        fact.owner_user_id,
        product.product_no,
        product.product_name,
        SUM(fact.qty_milli) AS qty_milli,
        SUM(fact.net_sales_milli) AS net_sales_milli
    FROM public.pos_sales_hourly_fact AS fact
    JOIN public.pos_product_dim AS product
      ON product.owner_user_id = fact.owner_user_id
     AND product.product_no = fact.product_no
    WHERE product.is_active = TRUE
      AND COALESCE(product.product_name, '') !~ '幣|券|折抵|折扣|點數|贈|服務費|運費|調整|測試'
      AND lower(COALESCE(product.product_name, '')) NOT LIKE '%test%'
    GROUP BY fact.owner_user_id, product.product_no, product.product_name
)
SELECT owner_user_id || '|' || product_no || '|' || product_name || '|' || qty_milli || '|' || net_sales_milli
FROM ranked_products
ORDER BY net_sales_milli DESC, qty_milli DESC, product_no ASC
LIMIT 5;
SQL
)"; then
  echo "smoke_leaderboard_top5:"
  if [[ -n "$leaderboard_preview" ]]; then
    echo "$leaderboard_preview"
  fi
fi

# Minimal product-summary grain check: join product dim with sales fact and aggregate per product
if product_summary_count="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -At <<'SQL'
SELECT COUNT(*) FROM (
  SELECT
    fact.owner_user_id,
    fact.product_no,
    product.product_name,
    SUM(fact.qty_milli) AS total_qty_milli,
    SUM(fact.net_sales_milli) AS total_net_sales_milli
  FROM public.pos_sales_hourly_fact AS fact
  JOIN public.pos_product_dim AS product
    ON product.owner_user_id = fact.owner_user_id
   AND product.product_no = fact.product_no
  GROUP BY fact.owner_user_id, fact.product_no, product.product_name
) AS t;
SQL
)"; then
  echo "smoke_product_summary_count: $product_summary_count"
else
  echo "smoke failed: product-summary aggregation query execution error" >&2
  exit 1
fi

if (( product_summary_count == 0 )); then
  echo "smoke failed: product-summary aggregation returned 0 rows" >&2
  smoke_failed=1
else
  # preview top 5 product-summary rows
  if product_summary_preview="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
    psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -At <<'SQL'
SELECT owner_user_id || '|' || product_no || '|' || product_name || '|' || SUM(qty_milli) || '|' || SUM(net_sales_milli)
FROM public.pos_sales_hourly_fact AS fact
JOIN public.pos_product_dim AS product
  ON product.owner_user_id = fact.owner_user_id
 AND product.product_no = fact.product_no
GROUP BY owner_user_id, product_no, product_name
ORDER BY SUM(net_sales_milli) DESC NULLS LAST
LIMIT 5;
SQL
)"; then
    echo "smoke_product_summary_top5:"
    if [[ -n "$product_summary_preview" ]]; then
      echo "$product_summary_preview"
    fi
  fi
fi

echo "smoke passed"