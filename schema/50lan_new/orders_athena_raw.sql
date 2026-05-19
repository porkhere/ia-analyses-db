-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`orders` (
  `id` string,
  `sequence` string,
  `items_count` int,
  `total` float,
  `change` float,
  `tax_subtotal` float,
  `surcharge_subtotal` float,
  `discount_subtotal` float,
  `payment_subtotal` float,
  `rounding_prices` string,
  `precision_prices` string,
  `rounding_taxes` string,
  `precision_taxes` string,
  `status` int,
  `service_clerk` string,
  `service_clerk_displayname` string,
  `proceeds_clerk` string,
  `proceeds_clerk_displayname` string,
  `member` string,
  `member_displayname` string,
  `member_email` string,
  `member_cellphone` string,
  `invoice_type` string,
  `invoice_title` string,
  `invoice_no` string,
  `invoice_count` int,
  `destination` string,
  `table_no` int,
  `check_no` int,
  `no_of_customers` int,
  `terminal_no` string,
  `transaction_created` timestamp,
  `transaction_submitted` timestamp,
  `created` timestamp,
  `modified` timestamp,
  `included_tax_subtotal` float,
  `item_subtotal` float,
  `sale_period` date,
  `shift_number` int,
  `branch_id` string,
  `branch` string,
  `transaction_voided` timestamp,
  `promotion_subtotal` float,
  `void_clerk` string,
  `void_clerk_displayname` string,
  `void_sale_period` string,
  `void_shift_number` int,
  `revalue_subtotal` float,
  `qty_subtotal` int,
  `inherited_order_id` string,
  `inherited_desc` string,
  `item_surcharge_subtotal` float,
  `trans_surcharge_subtotal` float,
  `item_discount_subtotal` float,
  `trans_discount_subtotal` float,
  `service_charge_subtotal` float,
  `included_service_charge_subtotal` float,
  `s_day` string,
  `s_month` string,
  `s_year` string,
  `s_hour` string,
  `tr_date` date
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
  's3://s3-athena-vivicloud/50lan_new/orders/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485447'
);