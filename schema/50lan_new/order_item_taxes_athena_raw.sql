-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_item_taxes` (
  `id` string,
  `order_id` string,
  `order_item_id` string,
  `promotion_id` string,
  `tax_no` string,
  `tax_name` string,
  `tax_type` string,
  `tax_rate` float,
  `tax_rate_type` string,
  `tax_threshold` float,
  `tax_subtotal` float,
  `included_tax_subtotal` float,
  `item_count` int,
  `taxable_amount` float,
  `created` int,
  `modified` int,
  `order_addition_id` string,
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
  's3://s3-athena-vivicloud/50lan_new/orders_item_taxes/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485521'
);