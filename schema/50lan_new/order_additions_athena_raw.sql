-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_additions` (
  `id` string,
  `order_id` string,
  `tax_name` string,
  `tax_rate` float,
  `tax_type` string,
  `current_tax` float,
  `discount_name` string,
  `discount_rate` float,
  `discount_type` string,
  `current_discount` float,
  `surcharge_name` string,
  `surcharge_rate` float,
  `surcharge_type` string,
  `current_surcharge` float,
  `has_discount` boolean,
  `has_surcharge` boolean,
  `created` int,
  `modified` int,
  `include_tax` float,
  `terminal_no` string
)
PARTITIONED BY (
  `t_open_date` date
)
ROW FORMAT DELIMITED
  FIELDS TERMINATED BY '|'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://s3-athena-vivicloud/50lan_new/orders_additions/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485580'
);