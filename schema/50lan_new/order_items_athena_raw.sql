-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_items` (
  `id` string,
  `order_id` string,
  `cate_no` string,
  `included_tax` float,
  `cate_name` string,
  `product_no` string,
  `product_barcode` string,
  `product_name` string,
  `current_qty` float,
  `current_price` float,
  `current_subtotal` float,
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
  `condiments` string,
  `current_condiment` float,
  `memo` string,
  `created` int,
  `modified` int,
  `destination` string,
  `parent_no` string,
  `weight` float,
  `sale_unit` string,
  `seat_no` string,
  `parent_index` string,
  `has_setitems` boolean,
  `stock_maintained` boolean,
  `current_service_charge` float,
  `included_service_charge` float,
  `price_level` int,
  `service_clerk` string,
  `service_clerk_displayname` string,
  `service_status` int,
  `service_status_time1` int,
  `service_status_time2` int,
  `service_status_time3` int,
  `service_status_time4` int,
  `service_status_time5` int,
  `item_sequence` int,
  `item_sequence_barcode` string,
  `org_order_sequence_no` string,
  `service_charge_name` string,
  `service_charge_rate` float,
  `service_charge_type` string,
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
  's3://s3-athena-vivicloud/50lan_new/orders_items/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485504'
);