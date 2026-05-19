CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_payments_parquet" (
    "id" varchar,
    "order_id" varchar,
    "order_items_count" integer,
    "order_total" real,
    "name" varchar,
    "amount" real,
    "memo1" varchar,
    "memo2" varchar,
    "created" integer,
    "modified" integer,
    "origin_amount" real,
    "service_clerk" varchar,
    "proceeds_clerk" varchar,
    "service_clerk_displayname" varchar,
    "proceeds_clerk_displayname" varchar,
    "change" real,
    "sale_period" integer,
    "shift_number" integer,
    "terminal_no" varchar,
    "is_groupable" boolean,
    "t_open_date" date
);