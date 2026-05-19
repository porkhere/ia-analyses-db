-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_payments` (
  `id` string,
  `order_id` string,
  `order_items_count` int,
  `order_total` float,
  `name` string,
  `amount` float,
  `memo1` string,
  `memo2` string,
  `created` int,
  `modified` int,
  `origin_amount` float,
  `service_clerk` string,
  `proceeds_clerk` string,
  `service_clerk_displayname` string,
  `proceeds_clerk_displayname` string,
  `change` float,
  `sale_period` int,
  `shift_number` int,
  `terminal_no` string,
  `is_groupable` boolean
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
  's3://s3-athena-vivicloud/50lan_new/orders_payments/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485486'
);