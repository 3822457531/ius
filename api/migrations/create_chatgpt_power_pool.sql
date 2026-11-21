-- 创建算力池表
CREATE TABLE IF NOT EXISTS `chatgpt_power_pool` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  `total_power` DOUBLE NOT NULL DEFAULT 0 COMMENT '平台总算力',
  `total_usdt` DOUBLE NOT NULL DEFAULT 0 COMMENT '平台总USDT',
  `buyback_usdt` DOUBLE NOT NULL DEFAULT 0 COMMENT '回购池USDT',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='算力池表';

-- 插入初始数据（id=1的默认记录）
INSERT INTO `chatgpt_power_pool` (`id`, `total_power`, `total_usdt`, `buyback_usdt`) 
VALUES (1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `id`=`id`;

