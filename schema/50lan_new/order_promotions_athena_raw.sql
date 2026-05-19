-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_promotions` (
  `id` string,
  `order_id` string,
  `promotion_id` string,
  `name` string,
  `code` string,
  `alt_name1` string,
  `alt_name2` string,
  `trigger` string,
  `trigger_name` string,
  `trigger_level` string,
  `type` string,
  `type_name` string,
  `type_level` string,
  `matched_amount` int,
  `matched_items_qty` int,
  `matched_items_subtotal` float,
  `discount_subtotal` float,
  `tax_name` string,
  `current_tax` float,
  `included_tax` float,
  `created` int,
  `modified` int,
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
  's3://s3-athena-vivicloud/50lan_new/orders_promotions/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485465'
);