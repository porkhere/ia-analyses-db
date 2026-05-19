CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_promotions_parquet" (
    "id" varchar,
    "order_id" varchar,
    "promotion_id" varchar,
    "name" varchar,
    "code" varchar,
    "alt_name1" varchar,
    "alt_name2" varchar,
    "trigger" varchar,
    "trigger_name" varchar,
    "trigger_level" varchar,
    "type" varchar,
    "type_name" varchar,
    "type_level" varchar,
    "matched_amount" integer,
    "matched_items_qty" integer,
    "matched_items_subtotal" real,
    "discount_subtotal" real,
    "tax_name" varchar,
    "current_tax" real,
    "included_tax" real,
    "created" integer,
    "modified" integer,
    "terminal_no" varchar,
    "t_open_date" date
);