CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_additions_parquet" (
    "id" varchar,
    "order_id" varchar,
    "tax_name" varchar,
    "tax_rate" real,
    "tax_type" varchar,
    "current_tax" real,
    "discount_name" varchar,
    "discount_rate" real,
    "discount_type" varchar,
    "current_discount" real,
    "surcharge_name" varchar,
    "surcharge_rate" real,
    "surcharge_type" varchar,
    "current_surcharge" real,
    "has_discount" boolean,
    "has_surcharge" boolean,
    "created" integer,
    "modified" integer,
    "include_tax" real,
    "terminal_no" varchar,
    "t_open_date" date
);