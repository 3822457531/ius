-- 创建算力池充值订单表
CREATE TABLE IF NOT EXISTS `chatgpt_power_pool_recharge` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  `order_number` VARCHAR(100) NOT NULL COMMENT '订单号',
  `user_id` INT UNSIGNED NOT NULL COMMENT '用户ID',
  `power_amount` DOUBLE NOT NULL DEFAULT 0 COMMENT '购买的算力数量',
  `usdt_amount` DOUBLE NOT NULL DEFAULT 0 COMMENT '需要支付的USDT金额',
  `bsc_address` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'BSC收款地址',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '订单状态: 1-待支付, 2-已支付, 3-已完成, 4-已取消',
  `tx_hash` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '交易哈希',
  `pay_time` BIGINT NOT NULL DEFAULT 0 COMMENT '支付时间',
  `finish_time` BIGINT NOT NULL DEFAULT 0 COMMENT '完成时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_number` (`order_number`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='算力池充值订单表';

