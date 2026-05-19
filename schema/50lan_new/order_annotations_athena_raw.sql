-- Raw Athena external table DDL captured from current Glue metadata.
CREATE EXTERNAL TABLE IF NOT EXISTS `50lan_new`.`order_annotations` (
  `id` string,
  `type` string,
  `text` string,
  `created` int,
  `modified` int,
  `order_id` string,
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
  's3://s3-athena-vivicloud/50lan_new/orders_annotations/gz'
TBLPROPERTIES (
  'transient_lastDdlTime'='1703485558'
);