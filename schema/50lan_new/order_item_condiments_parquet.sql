CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_item_condiments_parquet" (
    "id" varchar,
    "order_id" varchar,
    "item_id" varchar,
    "name" varchar,
    "price" real,
    "created" integer,
    "modified" integer,
    "condiment_id" varchar,
    "condiment_group_id" varchar,
    "current_qty" integer,
    "current_subtotal" real,
    "condiment_group_name" varchar,
    "terminal_no" varchar,
    "t_open_date" date
);