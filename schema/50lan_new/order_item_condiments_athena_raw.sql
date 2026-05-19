-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_item_condiments` (
  `id` string,
  `order_id` string,
  `item_id` string,
  `name` string,
  `price` float,
  `created` int,
  `modified` int,
  `condiment_id` string,
  `condiment_group_id` string,
  `current_qty` int,
  `current_subtotal` float,
  `condiment_group_name` string,
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
  's3://s3-athena-vivicloud/50lan_new/orders_item_condiments/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485544'
);