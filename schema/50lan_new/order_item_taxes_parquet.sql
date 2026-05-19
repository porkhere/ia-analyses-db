CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_item_taxes_parquet" (
    "id" varchar,
    "order_id" varchar,
    "order_item_id" varchar,
    "promotion_id" varchar,
    "tax_no" varchar,
    "tax_name" varchar,
    "tax_type" varchar,
    "tax_rate" real,
    "tax_rate_type" varchar,
    "tax_threshold" real,
    "tax_subtotal" real,
    "included_tax_subtotal" real,
    "item_count" integer,
    "taxable_amount" real,
    "created" integer,
    "modified" integer,
    "order_addition_id" varchar,
    "terminal_no" varchar,
    "t_open_date" date
);