CREATE SCHEMA IF NOT EXISTS "50lan_new";

CREATE TABLE IF NOT EXISTS "50lan_new"."order_annotations_parquet" (
    "id" varchar,
    "type" varchar,
    "text" varchar,
    "created" integer,
    "modified" integer,
    "order_id" varchar,
    "terminal_no" varchar,
    "t_open_date" date
);