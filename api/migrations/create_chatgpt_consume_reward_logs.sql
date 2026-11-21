CREATE TABLE IF NOT EXISTS `chatgpt_consume_reward_logs` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `source_key` VARCHAR(191) NOT NULL COMMENT '幂等键，如 mj:123 / sora2:456',
  `user_id` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '消费用户ID',
  `power` DOUBLE NOT NULL DEFAULT 0 COMMENT '成功消耗算力',
  `source_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '业务名称',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_consume_reward_source_key` (`source_key`),
  KEY `idx_consume_reward_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用广场算力消费分红幂等';
